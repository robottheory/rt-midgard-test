// Package chain provides a blockchain client.
package chain

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/pascaldekloe/metrics"
	"github.com/rs/zerolog"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	"gitlab.com/thorchain/midgard/config"
	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/util/timer"
)

const (
	// NOTE(huginn): numbers are chosen to give a good performance on an "average" desktop
	// machine with a 4 core CPU. With more cores it might make sense to increase the
	// parallelism, though care should be taken to not overload the Thornode.
	// See `docs/parallel_batch_bench.md` for measurments to guide selection of these parameters.
	defaultFetchBatchSize   int = 100 // must be divisible by BlockFetchParallelism
	defaultFetchParallelism int = 4
)

var logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).With().Timestamp().Str("module", "chain").Logger()

// CursorHeight is the Tendermint chain position [sequence identifier].
var CursorHeight = metrics.Must1LabelInteger("midgard_chain_cursor_height", "node")

// NodeHeight is the latest Tendermint chain position [sequence identifier]
// reported by the node.
var NodeHeight = metrics.Must1LabelRealSample("midgard_chain_height", "node")

func init() {
	metrics.MustHelp("midgard_chain_cursor_height", "The Tendermint sequence identifier that is next in line.")
	metrics.MustHelp("midgard_chain_height", "The latest Tendermint sequence identifier reported by the node.")
}

// Block is a chain record.
type Block struct {
	Height  int64     // sequence identifier
	Time    time.Time // establishment timestamp
	Hash    []byte    // content identifier
	Results *coretypes.ResultBlockResults
}

// Client provides Tendermint access.
type Client struct {
	// Single RPC access
	client *rpchttp.HTTP

	// Parallel / batched access
	batchClients []*rpchttp.BatchHTTP

	batchSize   int // divisible by parallelism
	parallelism int
}

func (c *Client) DebugFetchResults(ctx context.Context, height int64) (*coretypes.ResultBlockResults, error) {
	return c.client.BlockResults(ctx, &height)
}

func (c *Client) BatchSize() int {
	return c.batchSize
}

// NewClient configures a new instance. Timeout applies to all requests on endpoint.
func NewClient(cfg *config.Config) (*Client, error) {
	var timeout time.Duration = cfg.ThorChain.ReadTimeout.WithDefault(8 * time.Second)

	endpoint, err := url.Parse(cfg.ThorChain.TendermintURL)
	if err != nil {
		logger.Fatal().Err(err).Msg("Exit on malformed Tendermint RPC URL")
	}

	batchSize := config.IntWithDefault(cfg.ThorChain.FetchBatchSize, defaultFetchBatchSize)
	parallelism := config.IntWithDefault(cfg.ThorChain.Parallelism, defaultFetchParallelism)
	if batchSize%parallelism != 0 {
		logger.Fatal().Msgf("BatchSize=%d must be divisible by Parallelism=%d", batchSize, parallelism)
	}

	// need the path separate from the URL for some reason
	path := endpoint.Path
	endpoint.Path = ""
	remote := endpoint.String()

	var client *rpchttp.HTTP
	var batchClients []*rpchttp.BatchHTTP
	for i := 0; i < parallelism; i++ {
		// rpchttp.NewWithTimeout rounds to seconds for some reason
		client, err = rpchttp.NewWithClient(remote, path, &http.Client{Timeout: timeout})
		if err != nil {
			return nil, fmt.Errorf("tendermint RPC client instantiation: %w", err)
		}
		batchClients = append(batchClients, client.NewBatch())
	}

	return &Client{
		client:       client,
		batchClients: batchClients,
		batchSize:    batchSize,
		parallelism:  parallelism,
	}, nil
}

// ErrNoData is an up-to-date status.
var ErrNoData = errors.New("no more data on blockchain")

// ErrQuit accepts an abort request.
var ErrQuit = errors.New("receive on quit channel")

func reportProgress(nextHeightToFetch, thornodeHeight int64) {
	midgardHeight := nextHeightToFetch - 1
	if midgardHeight < 0 {
		midgardHeight = 0
	}
	if midgardHeight == thornodeHeight {
		logger.Info().Int64("height", midgardHeight).Msg("Fully synced")
	} else {
		progress := 100 * float64(midgardHeight) / float64(thornodeHeight)
		logger.Info().Str("progress", fmt.Sprintf("%.2f%%", progress)).Int64("height", midgardHeight).Msg("Syncing")
	}
}

var lastReportDetailedTime db.Second

// Reports every 5 min when in sync.
func reportDetailed(status *coretypes.ResultStatus, offset int64, timeoutMinutes int) {
	currentTime := db.TimeToSecond(time.Now())
	if db.Second(timeoutMinutes*60) <= currentTime-lastReportDetailedTime {
		lastReportDetailedTime = currentTime
		logger.Info().Msgf("Connected to Tendermint node %q [%q] on chain %q",
			status.NodeInfo.DefaultNodeID, status.NodeInfo.ListenAddr, status.NodeInfo.Network)
		logger.Info().Msgf("Thornode blocks %d - %d from %s to %s",
			status.SyncInfo.EarliestBlockHeight,
			status.SyncInfo.LatestBlockHeight,
			status.SyncInfo.EarliestBlockTime.Format("2006-01-02"),
			status.SyncInfo.LatestBlockTime.Format("2006-01-02"))
		reportProgress(offset, status.SyncInfo.LatestBlockHeight)
	}
}

var WebsocketNotify *chan struct{}

// Create websockets channel, called if enabled by config.
func CreateWebsocketChannel() {
	websocketChannel := make(chan struct{}, 2)
	WebsocketNotify = &websocketChannel
}

// CatchUp reads the latest block height from Status then it fetches all blocks from offset to
// that height.
// The error return is never nil. See ErrQuit and ErrNoData for normal exit.
func (c *Client) CatchUp(ctx context.Context, out chan<- Block, nextHeight int64) (
	height int64, err error) {
	originalNextHeight := nextHeight
	status, err := c.client.Status(ctx)
	if err != nil {
		return nextHeight, fmt.Errorf("Status() RPC failed: %w", err)
	}
	// Prints out only the first time, because we have shorter timeout later.
	reportDetailed(status, nextHeight, 10)

	statusTime := time.Now()
	node := string(status.NodeInfo.DefaultNodeID)
	cursorHeight := CursorHeight(node)
	cursorHeight.Set(status.SyncInfo.EarliestBlockHeight)
	nodeHeight := NodeHeight(node)
	nodeHeight.Set(float64(status.SyncInfo.LatestBlockHeight), statusTime)

	for {
		if ctx.Err() != nil {
			// Job was cancelled.
			return nextHeight, nil
		}
		if status.SyncInfo.LatestBlockHeight < nextHeight {
			if 10 < nextHeight-originalNextHeight {
				// Force report when finishing syncing
				reportDetailed(status, nextHeight, 0)
			}
			reportDetailed(status, nextHeight, 5)
			return nextHeight, ErrNoData
		}

		batchSize := int64(c.batchSize)
		parallelism := c.parallelism
		remaining := status.SyncInfo.LatestBlockHeight - nextHeight + 1
		if remaining < batchSize {
			batchSize = remaining
			parallelism = 1
		}
		batch := make([]Block, batchSize)

		err := c.fetchBlocksParallel(ctx, batch, nextHeight, parallelism)
		if err != nil {
			return nextHeight, err
		}

		for _, block := range batch {
			select {
			case <-ctx.Done():
				return nextHeight, nil
			case out <- block:
				nextHeight = block.Height + 1
				cursorHeight.Set(nextHeight)

				// report every so often in batch mode too.
				if 1 < batchSize && nextHeight%10000 == 1 {
					reportProgress(nextHeight, status.SyncInfo.LatestBlockHeight)
				}
			}
		}

		// Notify websockets if we already passed batch mode.
		// TODO(huginn): unify with `hasCaughtUp()` in main.go
		if batchSize < int64(c.batchSize) && WebsocketNotify != nil {
			select {
			case *WebsocketNotify <- struct{}{}:
			default:
			}
		}
	}
}

var (
	fetchTimerBatch    = timer.NewTimer("block_fetch_batch")
	fetchTimerParallel = timer.NewTimer("block_fetch_parallel")
	fetchTimerSingle   = timer.NewTimer("block_fetch_single")
)

func (c *Client) fetchBlock(ctx context.Context, block *Block, height int64) error {
	defer fetchTimerSingle.One()()

	info, err := c.client.BlockchainInfo(ctx, height, height)
	if err != nil {
		return fmt.Errorf("BlockchainInfo for %d, failed: %w", height, err)
	}

	if len(info.BlockMetas) != 1 {
		return fmt.Errorf("BlockchainInfo for %d, wrong number of results: %d",
			height, len(info.BlockMetas))
	}

	header := &info.BlockMetas[0].Header
	if header.Height != height {
		return fmt.Errorf("BlockchainInfo for %d, wrong height: %d", height, header.Height)
	}

	block.Height = height
	block.Time = header.Time
	block.Hash = []byte(info.BlockMetas[0].BlockID.Hash)

	block.Results, err = c.client.BlockResults(ctx, &block.Height)
	if err != nil {
		return fmt.Errorf("BlockResults for %d, failed: %w", height, err)
	}

	// Validate that heights in the response match the request
	if block.Height != height || block.Results.Height != height {
		return fmt.Errorf("BlockResults for %d, got Height=%d Results.Height=%d",
			height, block.Height, block.Results.Height)
	}

	return nil
}

func (c *Client) fetchBlocks(ctx context.Context, clientIdx int, batch []Block, height int64) error {
	// Note: n > 1 is required
	n := len(batch)
	var err error
	client := c.batchClients[clientIdx]
	defer fetchTimerBatch.Batch(n)()

	last := height + int64(n) - 1
	infos := make([]*coretypes.ResultBlockchainInfo, n)
	for i := 0; i < n; i++ {
		h := height + int64(i)
		// Note(huginn): we could do "batched batched" request: asking for 20 BlockchainInfos
		// at a time. Measurements suggest that this wouldn't improve the speed, the BlockchainInfo
		// requests take about 4% of the BlockResults, and it would complicate the logic significantly.
		infos[i], err = client.BlockchainInfo(ctx, h, h)
		if err != nil {
			return fmt.Errorf("BlockchainInfo batch for %d: %w", h, err)
		}
	}
	_, err = client.Send(ctx)
	if err != nil {
		return fmt.Errorf("BlockchainInfo batch Send for %d-%d: %w", height, last, err)
	}

	for i, info := range infos {
		h := height + int64(i)

		if len(info.BlockMetas) != 1 {
			return fmt.Errorf("BlockchainInfo batch for %d, wrong number of results: %d",
				height, len(info.BlockMetas))
		}

		header := &info.BlockMetas[0].Header
		if header.Height != h {
			return fmt.Errorf("BlockchainInfo for %d, wrong height: %d", h, header.Height)
		}

		block := &batch[i]
		block.Height = header.Height
		block.Time = header.Time
		block.Hash = []byte(info.BlockMetas[0].BlockID.Hash)
	}

	for i := range batch {
		block := &batch[i]
		block.Results, err = client.BlockResults(ctx, &block.Height)
		if err != nil {
			return fmt.Errorf("BlockResults batch for %d: %w", block.Height, err)
		}
	}

	_, err = client.Send(ctx)
	if err != nil {
		return fmt.Errorf("BlockResults batch Send for %d-%d: %w", height, last, err)
	}

	// Validate that heights in the response match the request
	for i := range batch {
		h := height + int64(i)
		block := &batch[i]
		if block.Height != h || block.Results.Height != h {
			return fmt.Errorf("BlockResults batch for %d, got Height=%d Results.Height=%d",
				h, block.Height, block.Results.Height)
		}
	}

	return nil
}

func (c *Client) fetchBlocksParallel(ctx context.Context, batch []Block, height int64, parallelism int) error {
	n := len(batch)
	if n == 1 {
		return c.fetchBlock(ctx, &batch[0], height)
	}

	if parallelism == 1 {
		return c.fetchBlocks(ctx, 0, batch, height)
	}

	k := n / parallelism
	if k*parallelism != n {
		return fmt.Errorf("batch size %d not divisible into %d parallel parts", n, parallelism)
	}

	defer fetchTimerParallel.Batch(n)()

	done := make(chan error, parallelism)
	for i := 0; i < parallelism; i++ {
		clientIdx := i
		start := i * k
		go func() {
			err := c.fetchBlocks(ctx, clientIdx, batch[start:start+k], height+int64(start))
			done <- err
		}()
	}

	var err error
	for i := 0; i < parallelism; i++ {
		e := <-done
		if e != nil {
			err = e
		}
	}
	return err
}
