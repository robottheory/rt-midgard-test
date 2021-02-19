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

	"github.com/kelseyhightower/envconfig"
	"github.com/pascaldekloe/metrics/gostat"

	"gitlab.com/thorchain/midgard/chain"
	"gitlab.com/thorchain/midgard/chain/notinchain"
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

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.LUTC | log.Lshortfile)
	log.Print("Daemon launch as ", strings.Join(os.Args, " "))

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	// include Go runtime metrics
	gostat.CaptureEvery(5 * time.Second)

	var c Config = ReadConfig()

	miderr.SetFailOnError(c.FailOnError)

	// override config with env variables
	err := envconfig.Process("midgard", &c)
	if err != nil {
		log.Fatal("Failed to process config environment variables, ", err)
	}

	// apply configuration
	db.Setup(&c.TimeScale)
	blocks := SetupBlockchain(&c)
	if c.ListenPort == 0 {
		c.ListenPort = 8080
		log.Printf("Default HTTP server listen port to %d", c.ListenPort)
	}
	api.InitHandler(c.ThorChain.ThorNodeURL, c.ThorChain.ProxiedWhitelistedEndpoints, c.Websockets.Enable)
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

	mainContext, mainCancel := context.WithCancel(context.Background())

	quitWebsockets := startWebsockets(mainContext, &c)

	db.LoadFirstBlockTimestampFromDB(context.Background())
	err = notinchain.LoadConstants()
	if err != nil {
		log.Fatal("Failed to read constants", err)
	}

	// launch blockchain reading
	go func() {
		m := event.Demux{Listener: timeseries.EventListener}

		for block := range blocks {
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
		log.Print("Timeseries feed stopped")
		signals <- syscall.SIGABRT
	}()

	signal := <-signals
	timeout := c.ShutdownTimeout.WithDefault(5 * time.Second)
	log.Print("Shutting down services initiated with timeout in ", timeout)
	mainCancel()
	finishCTX, finishCancel := context.WithTimeout(context.Background(), timeout)
	defer finishCancel()

	// TODO(acsaba): stop block fetching properly.
	// TODO(acsaba): stop block writing properly.
	// TODO(acsaba): refactor http server startup and shutdown to use jobs.
	if err := srv.Shutdown(finishCTX); err != nil {
		log.Print("HTTP shutdown: ", err)
	}
	quitWebsockets.Wait(finishCTX)

	log.Fatal("exit on signal ", signal)
}

func startWebsockets(ctx context.Context, c *Config) *jobs.Job {
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

// SetupBlockchain launches the synchronisation routine.
func SetupBlockchain(c *Config) <-chan chain.Block {
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
	offset, _, _, err := timeseries.Setup()
	if err != nil {
		// no point in running without a database
		log.Fatal("exit on RDB unavailable: ", err)
	}
	if offset != 0 {
		offset++
		log.Print("Starting with previous blockchain height ", offset)
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
	go func() {
		backoff := time.NewTicker(c.ThorChain.LastChainBackoff.WithDefault(7 * time.Second))
		defer backoff.Stop()

		// TODO(pascaldekloe): Could use a limited number of
		// retries with skip block logic perhaps?
		for {
			offset, err = client.CatchUp(context.Background(), ch, offset, nil)
			switch err {
			case chain.ErrNoData:
				lastNoData.Store(time.Now())
			default:
				log.Print("follow blockchain retry on ", err)
			}
			<-backoff.C
		}
	}()

	return ch
}
