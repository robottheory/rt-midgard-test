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

	"gitlab.com/thorchain/midgard/internal/fetch/sync/chain"

	"github.com/rs/cors"

	"github.com/pascaldekloe/metrics/gostat"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"gitlab.com/thorchain/midgard/config"
	"gitlab.com/thorchain/midgard/internal/api"
	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/fetch/notinchain"
	"gitlab.com/thorchain/midgard/internal/fetch/record"
	"gitlab.com/thorchain/midgard/internal/fetch/sync"
	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/internal/util/jobs"
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

	config.ReadGlobal()

	setupDB()

	// TODO(muninn): Don't start the jobs immediately, but wait till they are _all_ done
	// with their setups (and potentially log.Fatal()ed) and then start them together.

	mainContext, mainCancel := context.WithCancel(context.Background())

	blocks, fetchJob := sync.StartBlockFetch(mainContext)

	httpServerJob := startHTTPServer(mainContext)

	websocketsJob := startWebsockets(mainContext)

	blockWriteJob := startBlockWrite(mainContext, blocks)

	aggregatesRefreshJob := db.StartAggregatesRefresh(mainContext)

	cacheJob := api.GlobalCacheStore.StartBackgroundRefresh(mainContext)

	responseCacheJob := api.NewResponseCache(mainContext, &config.Global)

	signal := <-signals
	timeout := config.Global.ShutdownTimeout.Value()
	log.Info().Msgf("Shutting down services initiated with timeout in %s", timeout)
	mainCancel()
	finishCTX, finishCancel := context.WithTimeout(context.Background(), timeout)
	defer finishCancel()

	jobs.WaitAll(finishCTX,
		websocketsJob,
		fetchJob,
		httpServerJob,
		blockWriteJob,
		aggregatesRefreshJob,
		cacheJob,
		responseCacheJob,
	)

	log.Fatal().Msgf("Exit on signal %s", signal)
}

func startWebsockets(ctx context.Context) *jobs.Job {
	if !config.Global.Websockets.Enable {
		log.Info().Msg("Websockets are not enabled")
		return nil
	}
	db.CreateWebsocketChannel()
	quitWebsockets, err := websockets.Start(ctx, config.Global.Websockets.ConnectionLimit)
	if err != nil {
		log.Fatal().Err(err).Msg("Websockets failure")
	}
	return quitWebsockets
}

func startHTTPServer(ctx context.Context) *jobs.Job {
	c := &config.Global
	if c.ListenPort == 0 {
		c.ListenPort = 8080
		log.Info().Msgf("Default HTTP server listen port to %d", c.ListenPort)
	}
	whiteListIPs := strings.Split(c.WhiteListIps, ",")
	api.InitHandler(c.ThorChain.ThorNodeURL, c.ThorChain.ProxiedWhitelistedEndpoints, c.MaxReqPerSec, whiteListIPs, c.DisabledEndpoints, c.ApiCacheConfig.DefaultOHCLVCount)
	if c.AllowedOrigins == nil || len(c.AllowedOrigins) == 0 {
		c.AllowedOrigins = []string{"*"}
	}
	corsLimiter := cors.New(cors.Options{
		AllowedOrigins:   c.AllowedOrigins,
		AllowCredentials: true,
	})
	api.Handler = corsLimiter.Handler(api.Handler)
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

func startBlockWrite(ctx context.Context, blocks <-chan chain.Block) *jobs.Job {
	db.LoadFirstBlockFromDB(context.Background())

	record.LoadCorrections(db.ChainID())

	err := notinchain.LoadConstants()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to read constants")
	}
	var lastHeightWritten int64
	blockBatch := int64(config.Global.TimeScale.CommitBatchSize)

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

				if commit && db.FetchCaughtUp() {
					db.RequestAggregatesRefresh()
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

func setupDB() {
	db.Setup()
	err := timeseries.Setup(config.Global.UsdPools)
	if err != nil {
		log.Fatal().Err(err).Msg("Error durring reading last block from DB")
	}
}
