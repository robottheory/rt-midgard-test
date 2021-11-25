package sync

import (
	"context"
	"reflect"
	"time"

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

type Sync struct {
	chainClient *chain.Client
	blockStore  *chain.BlockStore
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
		nextHeightToFetch, err = GlobalSync.chainClient.CatchUp(out, nextHeightToFetch)
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
	GlobalSync = &Sync{}
	GlobalSync.blockStore = chain.NewBlockStore(ctx, c.BlockStoreFolder)

	GlobalSync.chainClient, err = chain.NewClient(ctx, c, GlobalSync.blockStore)
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
