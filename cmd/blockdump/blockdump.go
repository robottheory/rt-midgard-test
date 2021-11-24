// Tool for dumping to a json structure the blocks received from ThorNode.
//
// The Output path is configured with the "block_store_folder" configuration parameter
// Each output file contains exactly blocksPerFile number of block events (block batch)
// sent from ThorNode
// Partially fetched block batches are stored in a temporary file.
//
// Each block batch file is named after the last contained block height (padded with zeros to 12 width)
//
// The tool is restartable, and will resume the dump from the last successfully fetched block
// batch (unfinished block batches are discarded)
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
	"gitlab.com/thorchain/midgard/internal/fetch/chain"
	"gitlab.com/thorchain/midgard/internal/fetch/sync"
	"gitlab.com/thorchain/midgard/internal/util/jobs"
	"gitlab.com/thorchain/midgard/internal/util/miderr"
)

func main() {
	// TODO(muninn) refactor main into utility functions, use them from here
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
	log.Info().Msgf("Daemon launch as %s", strings.Join(os.Args, " "))

	signals := make(chan os.Signal, 10)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	// include Go runtime metrics
	gostat.CaptureEvery(5 * time.Second)

	c := config.ReadConfig()

	miderr.SetFailOnError(true)

	mainContext, mainCancel := context.WithCancel(context.Background())
	blockStore := chain.NewBlockStore(mainContext, c.BlockStoreFolder)

	blocks, fetchJob, _ := sync.StartBlockFetch(mainContext, &c, blockStore.LastFetchedHeight())

	blockStoreJob := jobs.Start("BlockStore", func() {
		defer blockStore.Close()
		for {
			if mainContext.Err() != nil {
				log.Info().Msgf("Error: shutdown process")
				return
			}
			select {
			case <-mainContext.Done():
				log.Info().Msgf("Done: shutdown process")
				return
			case block := <-blocks:
				blockStore.Dump(&block)
			}
		}
	})

	signal := <-signals
	timeout := c.ShutdownTimeout.WithDefault(5 * time.Second)
	log.Info().Msgf("Shutting down services initiated with timeout in %s", timeout)
	mainCancel()
	finishCTX, finishCancel := context.WithTimeout(context.Background(), timeout)
	defer finishCancel()

	jobs.WaitAll(finishCTX,
		fetchJob,
		&blockStoreJob,
	)

	log.Fatal().Msgf("Exit on signal %s", signal)

}
