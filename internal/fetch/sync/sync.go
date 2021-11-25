package sync

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/pascaldekloe/metrics"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gitlab.com/thorchain/midgard/config"
	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/fetch/chain"
	"gitlab.com/thorchain/midgard/internal/fetch/notinchain"
	"gitlab.com/thorchain/midgard/internal/util/jobs"
	"gitlab.com/thorchain/midgard/internal/util/miderr"

	coretypes "github.com/tendermint/tendermint/rpc/core/types"
)

// TODO(freki): migrate chain and blockstore under sync in a subdirectory. Preferably if possible:
//     sync/sync.go
//     sync/chain/chain.go
//     sync/blockstore/blockstore.go

var logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).With().Timestamp().Str("module", "sync").Logger()

// CursorHeight is the Tendermint chain position [sequence identifier].
var CursorHeight = metrics.Must1LabelInteger("midgard_chain_cursor_height", "node")

// NodeHeight is the latest Tendermint chain position [sequence identifier]
// reported by the node.
var NodeHeight = metrics.Must1LabelRealSample("midgard_chain_height", "node")

type Sync struct {
	chainClient *chain.Client
	blockStore  *chain.BlockStore

	ctx          context.Context
	status       *coretypes.ResultStatus
	cursorHeight *metrics.Integer
}

const CheckBlockStoreBlocks = false

func (s *Sync) DebugFetchBlock(height int64) (*coretypes.ResultBlockResults, error) {
	if s.blockStore.HasHeight(height) {
		block, err := s.blockStore.SingleBlock(height)
		if err != nil {
			return nil, err
		}
		ret := block.Results
		if CheckBlockStoreBlocks {
			fromChain, err := s.chainClient.DebugFetchBlock(height)
			if err != nil {
				return nil, err
			}
			if reflect.DeepEqual(ret, fromChain) {
				return nil, miderr.InternalErr("Blockstore blocks blocks don't match chain blocks")
			}
		}
		return ret, nil
	}
	return s.chainClient.DebugFetchBlock(height)
}

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
func (s *Sync) reportDetailed(offset int64, timeoutMinutes int) {
	currentTime := db.TimeToSecond(time.Now())
	if db.Second(timeoutMinutes*60) <= currentTime-lastReportDetailedTime {
		lastReportDetailedTime = currentTime
		logger.Info().Msgf("Connected to Tendermint node %q [%q] on chain %q",
			s.status.NodeInfo.DefaultNodeID, s.status.NodeInfo.ListenAddr, s.status.NodeInfo.Network)
		logger.Info().Msgf("Thornode blocks %d - %d from %s to %s",
			s.status.SyncInfo.EarliestBlockHeight,
			s.status.SyncInfo.LatestBlockHeight,
			s.status.SyncInfo.EarliestBlockTime.Format("2006-01-02"),
			s.status.SyncInfo.LatestBlockTime.Format("2006-01-02"))
		reportProgress(offset, s.status.SyncInfo.LatestBlockHeight)
	}
}

func (s *Sync) refreshStatus() (finalBlockHeight int64, err error) {
	status, err := s.chainClient.RefreshStatus()
	if err != nil {
		return 0, fmt.Errorf("Status() RPC failed: %w", err)
	}
	s.status = status

	finalBlockHeight = s.status.SyncInfo.LatestBlockHeight

	statusTime := time.Now()
	node := string(s.status.NodeInfo.DefaultNodeID)
	s.cursorHeight = CursorHeight(node)
	s.cursorHeight.Set(s.status.SyncInfo.EarliestBlockHeight)
	nodeHeight := NodeHeight(node)
	nodeHeight.Set(float64(finalBlockHeight), statusTime)

	return finalBlockHeight, nil
}

// ErrNoData is an up-to-date status.
var ErrNoData = errors.New("no more data on blockchain")

// CatchUp reads the latest block height from Status then it fetches all blocks from offset to
// that height.
// The error return is never nil. See ErrQuit and ErrNoData for normal exit.
func (s *Sync) CatchUp(out chan<- chain.Block, startHeight int64) (
	height int64, err error) {
	originalStartHeight := startHeight

	finalBlockHeight, err := s.refreshStatus()
	if err != nil {
		return startHeight, fmt.Errorf("Status() RPC failed: %w", err)
	}
	// Prints out only the first time, because we have shorter timeout later.
	s.reportDetailed(startHeight, 10)

	i := s.chainClient.Iterator(startHeight, finalBlockHeight)
	endReached := finalBlockHeight < originalStartHeight+10

	for {
		if s.ctx.Err() != nil {
			// Job was cancelled.
			return startHeight, nil
		}
		block, err := i.Next()
		if err != nil {
			return startHeight, err
		}

		if block == nil {
			if 10 < startHeight-originalStartHeight {
				// Force report when finishing syncing
				s.reportDetailed(startHeight, 0)
			}
			s.reportDetailed(startHeight, 5)
			return startHeight, ErrNoData
		}

		if block.Height != startHeight {
			return startHeight, miderr.InternalErrF(
				"Block height not incremented by one. Actual: %d Expected: %d",
				block.Height,
				startHeight,
			)
		}

		select {
		case <-s.ctx.Done():
			return startHeight, nil
		case out <- *block:
			startHeight = block.Height + 1
			s.cursorHeight.Set(startHeight)

			// report every so often in batch mode too.
			if !endReached && startHeight%10000 == 1 {
				reportProgress(startHeight, finalBlockHeight)
			}
		}

		// Notify websockets if we already passed batch mode.
		// TODO(huginn): unify with `hasCaughtUp()` in main.go
		if endReached && chain.WebsocketNotify != nil {
			select {
			case *chain.WebsocketNotify <- struct{}{}:
			default:
			}
		}
	}
}

// TODO(muninn): iterrate over blockstore too
func (s *Sync) KeepInSync(ctx context.Context, c *config.Config, out chan chain.Block) {
	lastFetchedHeight := db.LastBlockHeight()
	log.Info().Msgf("Starting chain read from previous height in DB %d", lastFetchedHeight)

	var nextHeightToFetch int64 = lastFetchedHeight + 1
	backoff := time.NewTicker(c.ThorChain.LastChainBackoff.Value())
	defer backoff.Stop()

	for {
		if ctx.Err() != nil {
			// Requested to stop
			return
		}
		var err error
		nextHeightToFetch, err = s.CatchUp(out, nextHeightToFetch)
		switch err {
		case chain.ErrNoData:
			db.SetFetchCaughtUp()
		default:
			log.Info().Err(err).Msgf("Block fetch error, retrying")
		}
		select {
		case <-backoff.C:
			// Noop
		case <-ctx.Done():
			return
		}
	}
}

var GlobalSync *Sync

// startBlockFetch launches the synchronisation routine.
// Stops fetching when ctx is cancelled.
func StartBlockFetch(ctx context.Context, c *config.Config) (<-chan chain.Block, *jobs.Job) {

	notinchain.BaseURL = c.ThorChain.ThorNodeURL

	var err error
	GlobalSync = &Sync{ctx: ctx}
	GlobalSync.blockStore = chain.NewBlockStore(ctx, c.BlockStoreFolder)

	GlobalSync.chainClient, err = chain.NewClient(ctx, c)
	if err != nil {
		// error check does not include network connectivity
		log.Fatal().Err(err).Msg("Exit on Tendermint RPC client instantiation")
	}

	liveFirstHash, err := GlobalSync.chainClient.FirstBlockHash()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to fetch first block hash from live chain")
	}
	log.Info().Msgf("First block hash on live chain: %s", liveFirstHash)

	dbChainID := db.ChainID()
	if dbChainID != "" && dbChainID != liveFirstHash {
		log.Fatal().Str("liveHash", liveFirstHash).Str("dbHash", dbChainID).Msg(
			"Live and DB first hash mismatch. Choose correct DB instance or wipe the DB Manually")
	}

	lastFetchedHeight := db.LastBlockHeight()
	log.Info().Msgf("Starting chain read from previous height in DB %d", lastFetchedHeight)

	ch := make(chan chain.Block, GlobalSync.chainClient.BatchSize())
	job := jobs.Start("BlockFetch", func() {
		GlobalSync.KeepInSync(ctx, c, ch)
	})

	return ch, &job
}
