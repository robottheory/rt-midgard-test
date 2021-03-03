package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/pascaldekloe/metrics/gostat"

	"gitlab.com/thorchain/midgard/chain"
	"gitlab.com/thorchain/midgard/chain/notinchain"
	"gitlab.com/thorchain/midgard/config"
	"gitlab.com/thorchain/midgard/event"
	"gitlab.com/thorchain/midgard/internal/api"
	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/internal/util/jobs"
	"gitlab.com/thorchain/midgard/internal/util/miderr"
	"gitlab.com/thorchain/midgard/internal/util/timer"
	"gitlab.com/thorchain/midgard/internal/websockets"
)

var writeTimer = timer.NewNano("block_write_total")

var signals chan os.Signal

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.LUTC | log.Lshortfile)
	log.Print("Daemon launch as ", strings.Join(os.Args, " "))

	signals = make(chan os.Signal, 10)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	// include Go runtime metrics
	gostat.CaptureEvery(5 * time.Second)

	var c config.Config = config.ReadConfig()

	miderr.SetFailOnError(c.FailOnError)

	db.Setup(&c.TimeScale)

	mainContext, mainCancel := context.WithCancel(context.Background())

	blocks, fetchJob := startBlockFetch(mainContext, &c)

	httpServerJob := startHTTPServer(mainContext, &c)

	websocketsJob := startWebsockets(mainContext, &c)

	blockWriteJob := startBlockWrite(mainContext, &c, blocks)

	signal := <-signals
	timeout := c.ShutdownTimeout.WithDefault(5 * time.Second)
	log.Print("Shutting down services initiated with timeout in ", timeout)
	mainCancel()
	finishCTX, finishCancel := context.WithTimeout(context.Background(), timeout)
	defer finishCancel()

	jobs.WaitAll(finishCTX,
		websocketsJob,
		fetchJob,
		httpServerJob,
		blockWriteJob)

	log.Fatal("exit on signal ", signal)
}

func startWebsockets(ctx context.Context, c *config.Config) *jobs.Job {
	if !c.Websockets.Enable {
		log.Println("Websockets are not enabled.")
		return nil
	}
	chain.CreateWebsocketChannel()
	quitWebsockets, err := websockets.Start(ctx, c.Websockets.ConnectionLimit)
	if err != nil {
		log.Fatal("Websockets failure:", err)
	}
	return quitWebsockets
}

// startBlockFetch launches the synchronisation routine.
// Stops fetching when ctx is cancelled.
func startBlockFetch(ctx context.Context, c *config.Config) (<-chan chain.Block, *jobs.Job) {
	// normalize & validate configuration
	if c.ThorChain.ThorNodeURL == "" {
		c.ThorChain.ThorNodeURL = "http://localhost:1317/thorchain"
		log.Printf("default THOR node REST URL to %q", c.ThorChain.ThorNodeURL)
	} else {
		log.Printf("THOR node REST URL is set to %q", c.ThorChain.ThorNodeURL)
	}
	if _, err := url.Parse(c.ThorChain.ThorNodeURL); err != nil {
		log.Fatal("exit on malformed THOR node REST URL: ", err)
	}
	notinchain.BaseURL = c.ThorChain.ThorNodeURL

	if c.ThorChain.TendermintURL == "" {
		c.ThorChain.TendermintURL = "http://localhost:26657/websocket"
		log.Printf("default Tendermint RPC URL to %q", c.ThorChain.TendermintURL)
	} else {
		log.Printf("Tendermint RPC URL is set to %q", c.ThorChain.TendermintURL)
	}
	endpoint, err := url.Parse(c.ThorChain.TendermintURL)
	if err != nil {
		log.Fatal("exit on malformed Tendermint RPC URL: ", err)
	}

	// instantiate client
	client, err := chain.NewClient(endpoint, c.ThorChain.ReadTimeout.WithDefault(2*time.Second))
	if err != nil {
		// error check does not include network connectivity
		log.Fatal("exit on Tendermint RPC client instantiation: ", err)
	}

	// fetch current position (from commit log)
	lastOffset, _, _, err := timeseries.Setup()
	if err != nil {
		// no point in running without a database
		log.Fatal("exit on RDB unavailable: ", err)
	}
	if lastOffset != 0 {
		lastOffset++
		log.Print("Starting with previous blockchain height ", lastOffset)
	}

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
	ch := make(chan chain.Block, 99)
	job := jobs.Start("BlockFetch", func() {
		var offset int64 = lastOffset
		backoff := time.NewTicker(c.ThorChain.LastChainBackoff.WithDefault(7 * time.Second))
		defer backoff.Stop()

		// TODO(pascaldekloe): Could use a limited number of
		// retries with skip block logic perhaps?
		for {
			if ctx.Err() != nil {
				return
			}
			offset, err = client.CatchUp(ctx, ch, offset)
			switch err {
			case chain.ErrNoData:
				lastNoData.Store(time.Now())
			default:
				log.Print("follow blockchain retry on ", err)
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
		log.Printf("Default HTTP server listen port to %d", c.ListenPort)
	}
	api.InitHandler(c.ThorChain.ThorNodeURL, c.ThorChain.ProxiedWhitelistedEndpoints)
	srv := &http.Server{
		Handler:      api.CORS(api.Handler),
		Addr:         fmt.Sprintf(":%d", c.ListenPort),
		ReadTimeout:  c.ReadTimeout.WithDefault(2 * time.Second),
		WriteTimeout: c.WriteTimeout.WithDefault(3 * time.Second),
	}

	// launch HTTP server
	go func() {
		err := srv.ListenAndServe()
		log.Print("HTTP stopped on ", err)
		signals <- syscall.SIGABRT
	}()

	ret := jobs.Start("HTTPserver", func() {
		<-ctx.Done()
		if err := srv.Shutdown(context.Background()); err != nil {
			log.Print("HTTP failed shutdown: ", err)
		}
	})
	return &ret
}

func startBlockWrite(ctx context.Context, c *config.Config, blocks <-chan chain.Block) *jobs.Job {
	db.LoadFirstBlockTimestampFromDB(context.Background())
	err := notinchain.LoadConstants()
	if err != nil {
		log.Fatal("Failed to read constants", err)
	}

	ret := jobs.Start("blockWrite", func() {
		m := event.Demux{Listener: timeseries.EventListener}

		for {
			if ctx.Err() != nil {
				return
			}
			select {
			case <-ctx.Done():
				return
			case block := <-blocks:
				if block.Height == 0 {
					// Default constructed block, height should be at least 1.
					log.Print("Timeseries feed stopped")
					signals <- syscall.SIGABRT
					return
				}
				t := writeTimer.One()
				m.Block(block)
				err := timeseries.CommitBlock(block.Height, block.Time, block.Hash)
				if err != nil {
					log.Print("Timeseries feed stopped on ", err)
					signals <- syscall.SIGABRT
					return
				}
				t()
			}
		}
	})
	return &ret
}
