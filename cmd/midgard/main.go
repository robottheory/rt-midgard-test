package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/rs/cors"

	"github.com/pascaldekloe/metrics/gostat"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"gitlab.com/thorchain/midgard/config"
	"gitlab.com/thorchain/midgard/internal/api"
	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/fetch/chain"
	"gitlab.com/thorchain/midgard/internal/fetch/notinchain"
	"gitlab.com/thorchain/midgard/internal/fetch/record"
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

	db.Setup(&c.TimeScale)

	mainContext, mainCancel := context.WithCancel(context.Background())

	blocks, fetchJob := startBlockFetch(mainContext, &c)

	httpServerJob := startHTTPServer(mainContext, &c)

	websocketsJob := startWebsockets(mainContext, &c)

	blockWriteJob := startBlockWrite(mainContext, &c, blocks)

	cacheJob := api.GlobalCacheStore.StartBackgroundRefresh(mainContext)

	responseCacheJob := api.NewResponseCache(mainContext)

	signal := <-signals
	timeout := c.ShutdownTimeout.WithDefault(5 * time.Second)
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
		responseCacheJob,
	)

	log.Fatal().Msgf("Exit on signal %s", signal)
}

func startWebsockets(ctx context.Context, c *config.Config) *jobs.Job {
	if !c.Websockets.Enable {
		log.Info().Msg("Websockets are not enabled")
		return nil
	}
	chain.CreateWebsocketChannel()
	quitWebsockets, err := websockets.Start(ctx, c.Websockets.ConnectionLimit)
	if err != nil {
		log.Fatal().Err(err).Msg("Websockets failure")
	}
	return quitWebsockets
}

// startBlockFetch launches the synchronisation routine.
// Stops fetching when ctx is cancelled.
func startBlockFetch(ctx context.Context, c *config.Config) (<-chan chain.Block, *jobs.Job) {
	notinchain.BaseURL = c.ThorChain.ThorNodeURL

	// instantiate client
	client, err := chain.NewClient(c)
	if err != nil {
		// error check does not include network connectivity
		log.Fatal().Err(err).Msg("Exit on Tendermint RPC client instantiation")
	}

	api.DebugFetchResults = client.DebugFetchResults

	// fetch current position (from commit log)
	lastFetchedHeight, _, _, err := timeseries.Setup(c.UsdPools)
	if err != nil {
		// no point in running without a database
		log.Fatal().Err(err).Msg("Exit on RDB unavailable")
	}
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
			nextHeightToFetch, err = client.CatchUp(ctx, ch, nextHeightToFetch)
			switch err {
			case chain.ErrNoData:
				db.SetInSync(true)
				lastNoData.Store(time.Now())
				setCaughtUp()
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

	return ch, &job
}

func startHTTPServer(ctx context.Context, c *config.Config) *jobs.Job {
	if c.ListenPort == 0 {
		c.ListenPort = 8080
		log.Info().Msgf("Default HTTP server listen port to %d", c.ListenPort)
	}
	api.InitHandler(c.ThorChain.ThorNodeURL, c.ThorChain.ProxiedWhitelistedEndpoints, c.MaxReqPerSec)
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
		ReadTimeout:  c.ReadTimeout.WithDefault(20 * time.Second),
		WriteTimeout: c.WriteTimeout.WithDefault(20 * time.Second),
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
	blockBatch := int64(config.IntWithDefault(c.TimeScale.CommitBatchSize, 100))

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

				commit := immediate || hasCaughtUp() || block.Height%blockBatch == 0
				err = timeseries.ProcessBlock(block, commit)
				if err != nil {
					break loop
				}

				if commit {
					db.RefreshAggregates(ctx, hasCaughtUp(), false)
				}

				lastHeightWritten = block.Height
				t()
			}
		}
		log.Error().Err(err).Msg("Unrecoverable error in BlockWriter, terminating")
		signals <- syscall.SIGABRT
	})
	return &ret
}

var caughtUpWithChain int32

func init() {
	caughtUpWithChain = 0
}

func setCaughtUp() {
	atomic.StoreInt32(&caughtUpWithChain, 1)
}

func hasCaughtUp() bool {
	v := atomic.LoadInt32(&caughtUpWithChain)
	if v != 0 {
		return true
	} else {
		return false
	}
}
