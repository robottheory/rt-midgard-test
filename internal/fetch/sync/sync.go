package sync

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
	"gitlab.com/thorchain/midgard/config"
	"gitlab.com/thorchain/midgard/internal/api"
	"gitlab.com/thorchain/midgard/internal/fetch/chain"
	"gitlab.com/thorchain/midgard/internal/fetch/notinchain"
	"gitlab.com/thorchain/midgard/internal/util/jobs"
)

var liveFirstHash string

// startBlockFetch launches the synchronisation routine.
// Stops fetching when ctx is cancelled.
func StartBlockFetch(ctx context.Context, c *config.Config, lastFetchedHeight int64, noMoreData func()) (<-chan chain.Block, *jobs.Job, string) {
	notinchain.BaseURL = c.ThorChain.ThorNodeURL

	// instantiate client
	client, err := chain.NewClient(ctx, c)
	if err != nil {
		// error check does not include network connectivity
		log.Fatal().Err(err).Msg("Exit on Tendermint RPC client instantiation")
	}

	api.DebugFetchResults = client.DebugFetchResults

	liveFirstHash, err = client.FirstBlockHash()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to fetch first block hash from live chain")
	}
	log.Info().Msgf("First block hash on live chain: %s", liveFirstHash)
	log.Info().Msgf("Starting with previous blockchain height %d", lastFetchedHeight)

	var lastNoData atomic.Value
	api.InSync = func() bool {
		lastTime, ok := lastNoData.Load().(time.Time)
		if !ok {
			// first node didn't load yet.
			return false
		}
		return time.Since(lastTime) < 2*c.ThorChain.LastChainBackoff.WithDefault(7*time.Second)
	}

	// launch read routine
	ch := make(chan chain.Block, client.BatchSize())
	job := jobs.Start("BlockFetch", func() {
		var nextHeightToFetch int64 = lastFetchedHeight + 1
		backoff := time.NewTicker(c.ThorChain.LastChainBackoff.WithDefault(7 * time.Second))
		defer backoff.Stop()

		// TODO(pascaldekloe): Could use a limited number of
		// retries with skip block logic perhaps?
		for {
			if ctx.Err() != nil {
				return
			}
			nextHeightToFetch, err = client.CatchUp(ch, nextHeightToFetch)
			switch err {
			case chain.ErrNoData:
				lastNoData.Store(time.Now())
				noMoreData()
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
	})

	return ch, &job, liveFirstHash
}
