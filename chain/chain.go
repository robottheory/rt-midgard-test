// Package chain provides a blockchain client.
package chain

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/pascaldekloe/metrics"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	"gitlab.com/thorchain/midgard/internal/util/timer"
)

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

// NewClient configures a new instance. Timeout applies to all requests on endpoint.
func NewClient(endpoint *url.URL, timeout time.Duration) (*Client, error) {
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

// Follow reads blocks in chronological order starting at the offset height.
// The error return is never nil. See ErrQuit and ErrNoData for normal exit.
// Height points to the next block in line, which is offset + the number of
// blocks send to out.
func (c *Client) Follow(ctx context.Context, out chan<- Block, offset int64, quit <-chan struct{}) (
	height int64, err error) {

	status, err := c.statusClient.Status(ctx)
	if err != nil {
		return offset, fmt.Errorf("Tendermint RPC status unavailable: %w", err)
	}
	statusTime := time.Now()
	log.Printf("connected to Tendermint node %q [%q] on chain %q",
		status.NodeInfo.DefaultNodeID, status.NodeInfo.ListenAddr, status.NodeInfo.Network)
	log.Printf("earliest Tendermint block 0x%X height %d from %s", status.SyncInfo.EarliestBlockHash,
		status.SyncInfo.EarliestBlockHeight, status.SyncInfo.EarliestBlockTime)
	log.Printf("latest Tendermint block 0x%X height %d from %s", status.SyncInfo.LatestBlockHash,
		status.SyncInfo.LatestBlockHeight, status.SyncInfo.LatestBlockTime)

	node := string(status.NodeInfo.DefaultNodeID)
	cursorHeight := CursorHeight(node)
	cursorHeight.Set(status.SyncInfo.EarliestBlockHeight)
	nodeHeight := NodeHeight(node)
	nodeHeight.Set(float64(status.SyncInfo.LatestBlockHeight), statusTime)

	for {
		// Tendermint does not provide a no-data status; need to poll ourselves
		if offset > status.SyncInfo.LatestBlockHeight {
			status, err = c.statusClient.Status(ctx)
			if err != nil {
				return offset, fmt.Errorf("Tendermint RPC status unavailable: %w", err)
			}
			nodeHeight.Set(float64(status.SyncInfo.LatestBlockHeight), time.Now())

			if offset > status.SyncInfo.LatestBlockHeight {
				return offset, ErrNoData
			}
		}

		// The maximum batch size is 20, because the limit of the historyClient.
		// https://github.com/tendermint/tendermint/issues/5339
		// If this turns out to be slow we can increase the batch size by calling
		// batch history client.
		batchSize := int64(20)
		remaining := status.SyncInfo.LatestBlockHeight - offset + 1
		if remaining < batchSize {
			batchSize = remaining
		}
		batch := make([]Block, batchSize)

		n, err := c.fetchBlocks(ctx, batch, offset)
		if err != nil {
			return offset, err
		}

		if n == 0 {
			select { // must check quit, even on no data
			default:
				continue
			case <-quit:
				return offset, ErrNoData
			}
		}

		// submit batch[:n]
		for i := 0; i < n; i++ {
			select {
			case <-quit:
				return offset, nil
			case out <- batch[i]:
				offset = batch[i].Height + 1
				cursorHeight.Set(offset)
			}
		}
	}
}

var fetchTimerBatch = timer.NewNano("block_fetch_batch")
var fetchTimerSingle = timer.NewNano("block_fetch_single")

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
