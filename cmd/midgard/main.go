package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/pascaldekloe/metrics/gostat"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"gitlab.com/thorchain/midgard/config"
	"gitlab.com/thorchain/midgard/internal/api"
	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/fetch/chain"
	"gitlab.com/thorchain/midgard/internal/fetch/notinchain"
	"gitlab.com/thorchain/midgard/internal/fetch/record"
	"gitlab.com/thorchain/midgard/internal/fetch/sync"
	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/internal/timeseries/stat"
	"gitlab.com/thorchain/midgard/internal/util/jobs"
	"gitlab.com/thorchain/midgard/internal/util/miderr"
	"gitlab.com/thorchain/midgard/internal/util/timer"
	"gitlab.com/thorchain/midgard/internal/websockets"
)

var writeTimer = timer.NewTimer("block_write_total")

var signals chan os.Signal

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
	log.Info().Msgf("Daemon launch as %s", strings.Join(os.Args, " "))

	signals = make(chan os.Signal, 10)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	// include Go runtime metrics
	gostat.CaptureEvery(5 * time.Second)

	var c config.Config = config.ReadConfig()

	miderr.SetFailOnError(c.FailOnError)

	stat.SetUsdPools(c.UsdPools)

	setupDB(&c)

	mainContext, mainCancel := context.WithCancel(context.Background())

	blocks, fetchJob := sync.StartBlockFetch(mainContext, &c)

	httpServerJob := startHTTPServer(mainContext, &c)

	websocketsJob := startWebsockets(mainContext, &c)

	blockWriteJob := startBlockWrite(mainContext, &c, blocks)

	cacheJob := api.GlobalCacheStore.StartBackgroundRefresh(mainContext)

	signal := <-signals
	timeout := c.ShutdownTimeout.Value()
	log.Info().Msgf("Shutting down services initiated with timeout in %s", timeout)
	mainCancel()
	finishCTX, finishCancel := context.WithTimeout(context.Background(), timeout)
	defer finishCancel()

	jobs.WaitAll(finishCTX,
		websocketsJob,
		fetchJob,
		httpServerJob,
		blockWriteJob,
		cacheJob,
	)

	log.Fatal().Msgf("Exit on signal %s", signal)
}

func startWebsockets(ctx context.Context, c *config.Config) *jobs.Job {
	if !c.Websockets.Enable {
		log.Info().Msg("Websockets are not enabled")
		return nil
	}
	db.CreateWebsocketChannel()
	quitWebsockets, err := websockets.Start(ctx, c.Websockets.ConnectionLimit)
	if err != nil {
		log.Fatal().Err(err).Msg("Websockets failure")
	}
	return quitWebsockets
}

func startHTTPServer(ctx context.Context, c *config.Config) *jobs.Job {
	if c.ListenPort == 0 {
		c.ListenPort = 8080
		log.Info().Msgf("Default HTTP server listen port to %d", c.ListenPort)
	}
	api.InitHandler(c.ThorChain.ThorNodeURL, c.ThorChain.ProxiedWhitelistedEndpoints)
	srv := &http.Server{
		Handler:      api.Handler,
		Addr:         fmt.Sprintf(":%d", c.ListenPort),
		ReadTimeout:  c.ReadTimeout.Value(),
		WriteTimeout: c.WriteTimeout.Value(),
	}

	// launch HTTP server
	go func() {
		err := srv.ListenAndServe()
		log.Error().Err(err).Msg("HTTP stopped")
		signals <- syscall.SIGABRT
	}()

	ret := jobs.Start("HTTPserver", func() {
		<-ctx.Done()
		if err := srv.Shutdown(context.Background()); err != nil {
			log.Error().Err(err).Msg("HTTP failed shutdown")
		}
	})
	return &ret
}

func startBlockWrite(ctx context.Context, c *config.Config, blocks <-chan chain.Block) *jobs.Job {
	db.LoadFirstBlockFromDB(context.Background())

	record.LoadCorrections(db.ChainID())

	err := notinchain.LoadConstants()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to read constants")
	}
	var lastHeightWritten int64
	blockBatch := int64(c.TimeScale.CommitBatchSize)

	ret := jobs.Start("BlockWrite", func() {
		var err error
		// TODO(huginn): replace loop label with some logic
	loop:
		for {
			if ctx.Err() != nil {
				log.Info().Msgf("Shutdown db write process, last height processed: %d", lastHeightWritten)
				return
			}
			select {
			case <-ctx.Done():
				log.Info().Msgf("Shutdown db write process, last height processed: %d", lastHeightWritten)
				return
			case block := <-blocks:
				if block.Height == 0 {
					// Default constructed block, height should be at least 1.
					log.Error().Msg("Block height of 0 is invalid")
					break loop
				}
				t := writeTimer.One()

				// When using the ImmediateInserter we can commit after every block, since it
				// flushes at the end of every block.
				_, immediate := db.Inserter.(*db.ImmediateInserter)

				commit := immediate || db.FetchCaughtUp() || block.Height%blockBatch == 0
				err = timeseries.ProcessBlock(block, commit)
				if err != nil {
					break loop
				}

				if commit {
					db.RefreshAggregates(ctx, db.FetchCaughtUp(), false)
				}

				// TODO(huginn): ping after aggregates finished
				db.WebsocketsPing()

				lastHeightWritten = block.Height
				t()
			}
		}
		log.Error().Err(err).Msg("Unrecoverable error in BlockWriter, terminating")
		signals <- syscall.SIGABRT
	})
	return &ret
}

func setupDB(c *config.Config) {
	db.Setup(&c.TimeScale)
	err := timeseries.Setup()
	if err != nil {
		log.Fatal().Err(err).Msg("Error durring reading last block from DB")
	}
}
