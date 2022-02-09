package sync

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/pascaldekloe/metrics"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gitlab.com/thorchain/midgard/config"
	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/fetch/notinchain"
	"gitlab.com/thorchain/midgard/internal/fetch/sync/blockstore"
	"gitlab.com/thorchain/midgard/internal/fetch/sync/chain"
	"gitlab.com/thorchain/midgard/internal/util/jobs"
	"gitlab.com/thorchain/midgard/internal/util/miderr"
	"gitlab.com/thorchain/midgard/internal/util/timer"

	coretypes "github.com/tendermint/tendermint/rpc/core/types"
)

var logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).With().Timestamp().Str("module", "sync").Logger()

// CursorHeight is the Tendermint chain position [sequence identifier].
var CursorHeight = metrics.Must1LabelInteger("midgard_chain_cursor_height", "node")

// NodeHeight is the latest Tendermint chain position [sequence identifier]
// reported by the node.
var NodeHeight = metrics.Must1LabelRealSample("midgard_chain_height", "node")

type Sync struct {
	chainClient *chain.Client
	blockStore  *blockstore.BlockStore

	ctx          context.Context
	status       *coretypes.ResultStatus
	cursorHeight *metrics.Integer
}

var CheckBlockStoreBlocks = false

func (s *Sync) FetchSingle(height int64) (*coretypes.ResultBlockResults, error) {
	if s.blockStore != nil && s.blockStore.HasHeight(height) {
		block, err := s.blockStore.SingleBlock(height)
		if err != nil {
			return nil, err
		}
		ret := block.Results
		if CheckBlockStoreBlocks {
			fromChain, err := s.chainClient.FetchSingle(height)
			if err != nil {
				return nil, err
			}
			if !reflect.DeepEqual(ret, fromChain) {
				return nil, miderr.InternalErr("Blockstore blocks blocks don't match chain blocks")
			}
		}
		return ret, nil
	}
	return s.chainClient.FetchSingle(height)
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
func (s *Sync) reportDetailed(offset int64, force bool) {
	currentTime := db.TimeToSecond(time.Now())
	if force || db.Second(60*5) <= currentTime-lastReportDetailedTime {
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
	db.LastThorNodeBlock.Set(s.status.SyncInfo.LatestBlockHeight,
		db.TimeToNano(s.status.SyncInfo.LatestBlockTime))

	statusTime := time.Now()
	node := string(s.status.NodeInfo.DefaultNodeID)
	s.cursorHeight = CursorHeight(node)
	s.cursorHeight.Set(s.status.SyncInfo.EarliestBlockHeight)
	nodeHeight := NodeHeight(node)
	nodeHeight.Set(float64(finalBlockHeight), statusTime)

	return finalBlockHeight, nil
}

var loopTimer = timer.NewTimer("sync_next")

// CatchUp reads the latest block height from Status then it fetches all blocks from offset to
// that height.
// The error return is never nil. See ErrQuit and ErrNoData for normal exit.
func (s *Sync) CatchUp(out chan<- chain.Block, startHeight int64) (
	height int64, inSync bool, err error) {
	originalStartHeight := startHeight

	finalBlockHeight, err := s.refreshStatus()
	if err != nil {
		return startHeight, false, fmt.Errorf("Status() RPC failed: %w", err)
	}
	s.reportDetailed(startHeight, false)

	i := NewIterator(s, startHeight, finalBlockHeight)

	// If there are not many blocks to fetch we are probably in sync with ThorNode
	const heightEpsilon = 10
	inSync = finalBlockHeight < originalStartHeight+heightEpsilon

	for {
		if s.ctx.Err() != nil {
			// Job was cancelled.
			return startHeight, false, nil
		}

		t := loopTimer.One()
		block, err := i.Next()
		t()
		if err != nil {
			return startHeight, false, err
		}

		if block == nil {
			if !inSync {
				// Force report when there was a long CatchUp
				s.reportDetailed(startHeight, true)
			}
			s.reportDetailed(startHeight, false)
			return startHeight, inSync, nil
		}

		select {
		case <-s.ctx.Done():
			return startHeight, false, nil
		case out <- *block:
			startHeight = block.Height + 1
			s.cursorHeight.Set(startHeight)
			db.LastFetchedBlock.Set(block.Height, db.TimeToNano(block.Time))

			// report every so often in batch mode too.
			if !inSync && startHeight%10000 == 1 {
				reportProgress(startHeight, finalBlockHeight)
			}
		}
	}
}

func (s *Sync) KeepInSync(ctx context.Context, out chan chain.Block) {
	heightOnStart := db.LastCommittedBlock.Get().Height
	log.Info().Msgf("Starting chain read from previous height in DB %d", heightOnStart)

	var nextHeightToFetch int64 = heightOnStart + 1

	for {
		if ctx.Err() != nil {
			// Requested to stop
			return
		}
		var err error
		var inSync bool
		nextHeightToFetch, inSync, err = s.CatchUp(out, nextHeightToFetch)
		if err != nil {
			log.Info().Err(err).Msgf("Block fetch error, retrying")
			db.SleepWithContext(ctx, config.Global.ThorChain.LastChainBackoff.Value())
		}
		if inSync {
			db.SetFetchCaughtUp()
			db.SleepWithContext(ctx, 2*time.Second)
		}
	}
}

func (s *Sync) BlockStoreHeight() int64 {
	return s.blockStore.LastFetchedHeight()
}

var GlobalSync *Sync

func InitGlobalSync(ctx context.Context) {
	var err error
	notinchain.BaseURL = config.Global.ThorChain.ThorNodeURL
	GlobalSync = &Sync{ctx: ctx}
	GlobalSync.chainClient, err = chain.NewClient(ctx)
	if err != nil {
		// error check does not include network connectivity
		log.Fatal().Err(err).Msg("Exit on Tendermint RPC client instantiation")
	}

	_, err = GlobalSync.refreshStatus()
	if err != nil {
		log.Fatal().Err(err).Msg("Error fetching ThorNode status")
	}

	hash := string(GlobalSync.status.SyncInfo.EarliestBlockHash)
	log.Info().Msgf("Tendermint chain ID: %s", db.PrintableHash(hash))
	db.SetChainId(hash)
	db.FirstBlock.Set(1, db.TimeToNano(GlobalSync.status.SyncInfo.EarliestBlockTime))

	GlobalSync.blockStore = blockstore.NewBlockStore(ctx, config.Global.BlockStore, db.ChainID())
}

func InitBlockFetch(ctx context.Context) (<-chan chain.Block, jobs.NamedFunction) {
	InitGlobalSync(ctx)

	ch := make(chan chain.Block, GlobalSync.chainClient.BatchSize())
	return ch, jobs.Later("BlockFetch", func() {
		GlobalSync.KeepInSync(ctx, ch)
	})
}
