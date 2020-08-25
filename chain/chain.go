// Package chain provides blockchain synchronisation.
package chain

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/pascaldekloe/metrics"
	"github.com/tendermint/tendermint/rpc/client"
	"github.com/tendermint/tendermint/rpc/client/http"
	"github.com/tendermint/tendermint/rpc/core/types"
)

// CursorHeight is the Tendermint chain position [sequence identifier].
var CursorHeight *metrics.Integer

// NodeHeight is the latest Tendermint chain position [sequence identifier]
// reported by the node.
var NodeHeight *metrics.Sample

// FetchRetries is the number of times tried again after error.
var FetchRetries *metrics.Counter

// Setup initializes the connection, including CursorHeight and NodeHeight.
// Metrics initialization panics on second invocation.
func Setup(scheme, addr string) error {
	client, err := http.New(fmt.Sprintf("%s://%s", scheme, addr), "/websocket")
	if err != nil {
		return fmt.Errorf("Tendermint RPC client instantiation: %w", err)
	}
	statusClient = client
	historyClient = client
	batchClient := client.NewBatch()
	signClient = batchClient
	signClientTrigger = batchClient.Send

	status, err := statusClient.Status()
	if err != nil {
		return fmt.Errorf("Tendermint RPC status unavailable: %w", err)
	}
	log.Printf("connected to node %q [%q] on chain %q",
		status.NodeInfo.DefaultNodeID, status.NodeInfo.ListenAddr, status.NodeInfo.Network)
	log.Printf("earliest block 0x%X height %d from %s", status.SyncInfo.EarliestBlockHash,
		status.SyncInfo.EarliestBlockHeight, status.SyncInfo.EarliestBlockTime)
	log.Printf("latest block 0x%X height %d from %s", status.SyncInfo.LatestBlockHash,
		status.SyncInfo.LatestBlockHeight, status.SyncInfo.LatestBlockTime)

	CursorHeight = metrics.Must1LabelInteger("midgard_chain_cursor_height", "node")(string(status.NodeInfo.DefaultNodeID))
	metrics.MustHelp("midgard_chain_cursor_height", "The The Tendermint sequence identifier which is next in line.")
	CursorHeight.Set(status.SyncInfo.EarliestBlockHeight)

	NodeHeight = metrics.Must1LabelRealSample("midgard_chain_height", "node")(string(status.NodeInfo.DefaultNodeID))
	metrics.MustHelp("midgard_chain_height", "The latest Tendermint sequence identifier reported by the node.")
	NodeHeight.Set(float64(status.SyncInfo.LatestBlockHeight), time.Now())

	FetchRetries = metrics.Must1LabelCounter("midgard_chain_fetch_retries", "node")(string(status.NodeInfo.DefaultNodeID))
	metrics.MustHelp("midgard_chain_fetch_retries", "Number of times tried again after fetch error.")

	return nil
}

// StatusClient has a Tendermint connection.
var statusClient client.StatusClient

// HistoryClient has a Tendermint connection.
var historyClient client.HistoryClient

// SignClient has a Tendermint connection in batch mode.
var signClient client.SignClient

// SignClientTrigger executes enqueued requests (on SignClient).
// See github.com/tendermint/tendermint/rpc/client/http BatchHTTP.
var signClientTrigger func() ([]interface{}, error)

// ReadSingleton is a protective lock to prevent coding mistakes.
var readSingleton = make(chan struct{}, 1)

// Block is a chain record.
type Block struct {
	Height  int64     // sequence identifier
	Time    time.Time // header timestamp
	Results *coretypes.ResultBlockResults
}

// Follow reads blocks in chronological order starting at the CursorHeight.
// [Tendermint height]. Operation continues until either a critical read error
// or a receive on quit occurs. The error return is never nil. The out(bound)
// channel is closed on return.
func Follow(out chan<- Block, quit <-chan os.Signal) error {
	defer close(out)

	select {
	case readSingleton <- struct{}{}:
		defer func() {
			<-readSingleton // release lock
		}()
	default:
		return errors.New("chain follow already active")
	}

	// prevent request loops
	backoffTicker := time.NewTicker(time.Second)
	defer backoffTicker.Stop()

	// request up to 40 chains at a time
	batch := make([]Block, 40)
	for {
		offset := CursorHeight.Get()

		// Tendermint does not provide a no-data status; need to poll ourselves
		if height, _ := NodeHeight.Get(); height < float64(offset) {
			select {
			case <-backoffTicker.C:
				status, err := statusClient.Status()
				if err != nil {
					return fmt.Errorf("Tendermint RPC status unavailable: %w", err)
				}
				NodeHeight.Set(float64(status.SyncInfo.LatestBlockHeight), time.Now())
				continue

			case signal := <-quit:
				return fmt.Errorf("abort blockchain reading on signal %s", signal)
			}
		}

		n, heightLag, err := fetchBlocks(batch, offset)
		for retryCount := 1; err != nil; retryCount++ {
			FetchRetries.Add(1)
			log.Printf("retry %d on %s", retryCount, err)
			select {
			case <-backoffTicker.C:
				n, heightLag, err = fetchBlocks(batch, offset)
				continue

			case signal := <-quit:
				return fmt.Errorf("abort blockchain reading on signal %s", signal)
			}
		}

		if heightLag < 0 { // shouldn't happen 😏
			log.Printf("fetch height %d got %d blocks, which is %d blocks ahead of the chain height", offset, n, -heightLag)
			n += int(heightLag)
			heightLag = 0
			if n < 0 {
				offset += int64(n)
				CursorHeight.Set(offset)
				continue
			}
		}

		if n == 0 {
			select { // must check quit, even on no data
			default:
				continue
			case signal := <-quit:
				return fmt.Errorf("abort blockchain reading on signal %s", signal)
			}
		}

		CursorHeight.Set(batch[n-1].Height + 1)

		// submit batch[:n]
		for i := 0; i < n; i++ {
			select {
			case signal := <-quit:
				return fmt.Errorf("abort blockchain reading on signal %s", signal)
			case out <- batch[i]:
				continue
			}
		}
	}
}

// FetchBlocks resolves n blocks into batch, starting at the offset (height).
// The heightLag is the reported number of blocks remaining on the node.
func fetchBlocks(batch []Block, offset int64) (n int, heightLag int64, err error) {
	last := offset + int64(len(batch)-1)
	info, err := historyClient.BlockchainInfo(offset, last)
	if err != nil {
		return 0, 0, fmt.Errorf("Tendermint RPC BlockchainInfo %d–%d: %w", offset, last, err)
	}
	NodeHeight.Set(float64(info.LastHeight), time.Now())

	if len(info.BlockMetas) == 0 {
		return 0, info.LastHeight - offset, nil
	}

	// validate descending [!] order
	for i := 1; i < len(info.BlockMetas); i++ {
		height := info.BlockMetas[i].Header.Height
		previous := info.BlockMetas[i-1].Header.Height
		if height >= previous {
			return 0, 0, fmt.Errorf("Tendermint RPC BlockchainInfo %d–%d got chain %d after %d", offset, last, previous, height)
		}
	}
	// validate range
	if high, low := info.BlockMetas[0].Header.Height, info.BlockMetas[len(info.BlockMetas)-1].Header.Height; high > last || low < offset {
		return 0, 0, fmt.Errorf("Tendermint RPC BlockchainInfo %d–%d got %d–%d", offset, last, low, high)
	}

	// setup blocks for batch request
	for i := len(info.BlockMetas) - 1; i >= 0; i-- {
		batch[n].Height = info.BlockMetas[i].Header.Height
		batch[n].Time = info.BlockMetas[i].Header.Time

		// Why the pointer receiver? 🤨 Using BlockMeta.Header field (after extraction)
		// out of precaution, as it is no longer needed for anything else form here on.
		batch[n].Results, err = signClient.BlockResults(&info.BlockMetas[i].Header.Height)
		if err != nil {
			return 0, 0, fmt.Errorf("enqueue BlockResults(%d) for Tendermint RPC batch: %w", batch[n].Height, err)
		}

		n++
	}

	if _, err := signClientTrigger(); err != nil {
		return 0, 0, fmt.Errorf("Tendermint RPC batch %d–%d: %w", offset, last, err)
	}
	// validate response matching batch request
	for i := range batch[:n] {
		if got, requested := batch[i].Results.Height, batch[i].Height; got != requested {
			return 0, 0, fmt.Errorf("Tendermint RPC BlockResults(%d) got chain %d instead", requested, got)
		}
	}

	return n, info.LastHeight - batch[n-1].Height, nil
}
