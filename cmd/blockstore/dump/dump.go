// Tool for dumping to a json structure the blocks received from ThorNode.
//
// The Output path is configured with the "block_store_folder" configuration parameter
// Each output file contains exactly blocksPerTrunk number of block events (block trunk)
// sent from ThorNode
// Partially fetched block trunks are stored in a temporary file.
//
// Each block trunk file is named after the last contained block height (padded with zeros to 12 width)
//
// The tool is restartable, and will resume the dump from the last successfully fetched block
// trunk (unfinished block trnks are discarded)
package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/pascaldekloe/metrics/gostat"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gitlab.com/thorchain/midgard/config"
	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/fetch/sync/blockstore"
	"gitlab.com/thorchain/midgard/internal/fetch/sync/chain"
	"gitlab.com/thorchain/midgard/internal/util/jobs"
)

func main() {
	// TODO(muninn) refactor main into utility functions, use them from here
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
	log.Info().Msgf("BlockStore: dump daemon launch as %s", strings.Join(os.Args, " "))

	signals := make(chan os.Signal, 10)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	// include Go runtime metrics
	gostat.CaptureEvery(5 * time.Second)

	config.ReadGlobal()
	config.Global.FailOnError = true

	mainContext, mainCancel := context.WithCancel(context.Background())

	// TODO(freki): create folder if doesn't exist inside blocksoter
	blockStore := blockstore.NewBlockStore(mainContext, config.Global.BlockStore)
	startHeight := blockStore.LastFetchedHeight() + 1

	chainClient, err := chain.NewClient(mainContext)
	if err != nil {
		log.Fatal().Err(err).Msg("Error durring chain client initialization")
	}

	status, err := chainClient.RefreshStatus()
	if err != nil {
		log.Fatal().Err(err).Msg("Error durring fetching chain status")
	}
	endHeight := status.SyncInfo.LatestBlockHeight
	it := chainClient.Iterator(startHeight, endHeight)

	log.Info().Msgf("BlockStore: start fetching from %d to %d", startHeight, endHeight)

	// TODO(freki): log height on flush to have some progress report
	blockStoreJob := jobs.Start("BlockStore", func() {
		defer blockStore.Close()
		for {
			if mainContext.Err() != nil {
				log.Info().Msgf("BlockStore: write shutdown")
				return
			}
			block, err := it.Next()
			if err != nil {
				log.Warn().Err(err).Msgf("BlockStore: error while fetching at height %d", startHeight)
				db.SleepWithContext(mainContext, 7*time.Second)
				it = chainClient.Iterator(startHeight, endHeight)
			}
			if block == nil {
				// TODO(freki): backoff and continue when in synch
				signals <- syscall.SIGABRT
				return
			}
			if block.Height != startHeight {
				log.Error().Err(err).Msgf(
					"BlockStore: height not incremented by one. Expected: %d Actual: %d",
					startHeight, block.Height)
				return
			}
			blockStore.Dump(block)
			startHeight++
		}
	})

	signal := <-signals
	timeout := config.Global.ShutdownTimeout.Value()
	log.Info().Msgf("BlockStore: shutting down services initiated with timeout in %s", timeout)
	mainCancel()
	finishCTX, finishCancel := context.WithTimeout(context.Background(), timeout)
	defer finishCancel()

	jobs.WaitAll(finishCTX,
		&blockStoreJob,
	)

	log.Fatal().Msgf("Exit on signal %s", signal)

}
