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
	rpcclient "github.com/tendermint/tendermint/rpc/client"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	"gitlab.com/thorchain/midgard/config"
	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/util/miderr"
	"gitlab.com/thorchain/midgard/internal/util/timer"
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
	// StatusClient has a Tendermint connection.
	statusClient rpcclient.StatusClient

	// HistoryClient has a Tendermint connection.
	historyClient rpcclient.HistoryClient

	// SignClient has a Tendermint connection.
	signClient rpcclient.SignClient

	// SignBatchClient has a Tendermint connection in batch mode.
	signBatchClient rpcclient.SignClient

	// SignBatchClientTrigger executes enqueued requests (on SignClient).
	// See github.com/tendermint/tendermint/rpchttp/client/http BatchHTTP.
	signBatchClientTrigger func(ctx context.Context) ([]interface{}, error)
}

func (client *Client) DebugFetchResults(ctx context.Context, height int64) (*coretypes.ResultBlockResults, error) {
	return client.signClient.BlockResults(ctx, &height)
}

// NewClient configures a new instance. Timeout applies to all requests on endpoint.
func NewClient(c *config.Config) (*Client, error) {
	var timeout time.Duration = c.ThorChain.ReadTimeout.WithDefault(8 * time.Second)

	endpoint, err := url.Parse(c.ThorChain.TendermintURL)
	if err != nil {
		logger.Fatal().Err(err).Msg("Exit on malformed Tendermint RPC URL")
	}
	// need the path seperate from the URL for some reason
	path := endpoint.Path
	endpoint.Path = ""
	remote := endpoint.String()

	// rpchttp.NewWithTimeout rounds to seconds for some reason
	client, err := rpchttp.NewWithClient(remote, path, &http.Client{Timeout: timeout})
	if err != nil {
		return nil, fmt.Errorf("Tendermint RPC client instantiation: %w", err)
	}
	batchClient := client.NewBatch()
	return &Client{
		statusClient:           client,
		historyClient:          client,
		signClient:             client,
		signBatchClient:        batchClient,
		signBatchClientTrigger: batchClient.Send,
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
	status, err := c.statusClient.Status(ctx)
	if err != nil {
		return nextHeight, fmt.Errorf("Tendermint RPC status unavailable: %w", err)
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

		// The maximum batch size is 20, because the limit of the historyClient.
		// https://github.com/tendermint/tendermint/issues/5339
		// If this turns out to be slow we can increase the batch size by calling
		// batch history client.
		const maxBatchSize = 20
		batchSize := int64(maxBatchSize)
		remaining := status.SyncInfo.LatestBlockHeight - nextHeight + 1
		if remaining < batchSize {
			batchSize = remaining
		}
		batch := make([]Block, batchSize)

		n, err := c.fetchBlocks(ctx, batch, nextHeight)
		if err != nil {
			return nextHeight, err
		}

		if n == 0 {
			select { // must check quit, even on no data
			default:
				return nextHeight, miderr.InternalErrF(
					"Faild to fetch blocks, was expecting %d blocks", batchSize)
			case <-ctx.Done():
				return nextHeight, nil
			}
		}

		// submit batch[:n]
		for i := 0; i < n; i++ {
			select {
			case <-ctx.Done():
				return nextHeight, nil
			case out <- batch[i]:
				nextHeight = batch[i].Height + 1
				cursorHeight.Set(nextHeight)

				// report every so often in batch mode too.
				if 1 < batchSize && nextHeight%1000 == 1 {
					reportProgress(nextHeight, status.SyncInfo.LatestBlockHeight)
				}
			}
		}

		// Notify websockets if we already passed batch mode.
		if batchSize < maxBatchSize-1 && WebsocketNotify != nil {
			select {
			case *WebsocketNotify <- struct{}{}:
			default:
			}
		}
	}
}

var (
	fetchTimerBatch  = timer.NewTimer("block_fetch_batch")
	fetchTimerSingle = timer.NewTimer("block_fetch_single")
)

// FetchBlocks resolves n blocks into batch, starting at the offset (height).
func (c *Client) fetchBlocks(ctx context.Context, batch []Block, offset int64) (n int, err error) {
	if 1 == len(batch) {
		defer fetchTimerSingle.One()()
	} else {
		defer fetchTimerBatch.Batch(len(batch))()
	}

	last := offset + int64(len(batch)-1)
	info, err := c.historyClient.BlockchainInfo(ctx, offset, last)
	if err != nil {
		return 0, fmt.Errorf("Tendermint RPC BlockchainInfo %d–%d: %w", offset, last, err)
	}

	if len(info.BlockMetas) == 0 {
		return 0, nil
	}

	// validate descending [!] order
	for i := 1; i < len(info.BlockMetas); i++ {
		height := info.BlockMetas[i].Header.Height
		previous := info.BlockMetas[i-1].Header.Height
		if height >= previous {
			return 0, fmt.Errorf("Tendermint RPC BlockchainInfo %d–%d got chain %d after %d", offset, last, previous, height)
		}
	}
	// validate range
	if high, low := info.BlockMetas[0].Header.Height, info.BlockMetas[len(info.BlockMetas)-1].Header.Height; high > last || low < offset {
		return 0, fmt.Errorf("Tendermint RPC BlockchainInfo %d–%d got %d–%d", offset, last, low, high)
	}

	// setup blocks for batch request
	for i := len(info.BlockMetas) - 1; i >= 0; i-- {
		batch[n].Height = info.BlockMetas[i].Header.Height
		batch[n].Time = info.BlockMetas[i].Header.Time
		batch[n].Hash = []byte(info.BlockMetas[i].BlockID.Hash)

		// We get unmarshalling error from the batch client if we have one call only.
		// For this reason we call signClient when there is one call only.
		if 1 < len(batch) {
			batch[n].Results, err = c.signBatchClient.BlockResults(ctx, &info.BlockMetas[i].Header.Height)
		} else {
			batch[n].Results, err = c.signClient.BlockResults(ctx, &info.BlockMetas[i].Header.Height)
		}
		if err != nil {
			return 0, fmt.Errorf("enqueue BlockResults(%d) for Tendermint RPC batch: %w", batch[n].Height, err)
		}

		n++
	}

	if 1 < len(batch) {
		if _, err := c.signBatchClientTrigger(ctx); err != nil {
			return 0, fmt.Errorf("Tendermint RPC batch BlockResults %d–%d: %w", offset, last, err)
		}
	}
	// validate response matching batch request
	for i := range batch[:n] {
		if got, requested := batch[i].Results.Height, batch[i].Height; got != requested {
			return 0, fmt.Errorf("Tendermint RPC BlockResults(%d) got chain %d instead", requested, got)
		}
	}

	return n, nil
}
