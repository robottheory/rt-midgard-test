package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
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

	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/pascaldekloe/metrics/gostat"

	"gitlab.com/thorchain/midgard/chain"
	"gitlab.com/thorchain/midgard/chain/notinchain"
	"gitlab.com/thorchain/midgard/event"
	"gitlab.com/thorchain/midgard/internal/api"
	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/internal/util/timer"
)

var writeTimer = timer.NewNano("block_write_total")

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.LUTC)
	log.Print("daemon launch as ", strings.Join(os.Args, " "))

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	// include Go runtime metrics
	gostat.CaptureEvery(5 * time.Second)

	// read configuration
	var c Config
	switch len(os.Args) {
	case 1:
		break // refer to defaults
	case 2:
		c = *MustLoadConfigFile(os.Args[1])
	default:
		log.Fatal("one optional configuration file argument only—no flags")
	}

	// apply configuration
	SetupDatabase(&c)
	blocks := SetupBlockchain(&c)
	if c.ListenPort == 0 {
		c.ListenPort = 8080
		log.Printf("default HTTP server listen port to %d", c.ListenPort)
	}
	api.InitHandler(c.ThorChain.NodeURL, c.ThorChain.ProxiedWhitelistedEndpoints)
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

	// launch blockchain reading
	go func() {
		m := event.Demux{Listener: timeseries.EventListener}

		for block := range blocks {
			t := writeTimer.One()
			m.Block(block)
			err := timeseries.CommitBlock(block.Height, block.Time, block.Hash)
			if err != nil {
				log.Print("timeseries feed stopped on ", err)
				signals <- syscall.SIGABRT
				return
			}
			t()
		}
		log.Print("timeseries feed stopped")
		signals <- syscall.SIGABRT
	}()

	signal := <-signals
	timeout := c.ShutdownTimeout.WithDefault(10 * time.Millisecond)
	log.Print("HTTP shutdown initiated with timeout in ", timeout)
	ctx, _ := context.WithTimeout(context.Background(), timeout)
	if err := srv.Shutdown(ctx); err != nil {
		log.Print("HTTP shutdown: ", err)
	}

	log.Fatal("exit on signal ", signal)
}

func SetupDatabase(c *Config) {
	dbObj, err := sql.Open("pgx",
		fmt.Sprintf("user=%s dbname=%s sslmode=%s password=%s host=%s port=%d",
			c.TimeScale.UserName, c.TimeScale.Database, c.TimeScale.Sslmode,
			c.TimeScale.Password, c.TimeScale.Host, c.TimeScale.Port))
	if err != nil {
		log.Fatal("exit on PostgreSQL client instantiation: ", err)
	}

	db.Exec = dbObj.Exec
	db.Query = dbObj.QueryContext
}

// SetupBlockchain launches the synchronisation routine.
func SetupBlockchain(c *Config) <-chan chain.Block {
	// normalize & validate configuration
	if c.ThorChain.NodeURL == "" {
		c.ThorChain.NodeURL = "http://localhost:1317/thorchain"
		log.Printf("default THOR node REST URL to %q", c.ThorChain.NodeURL)
	} else {
		log.Printf("THOR node REST URL is set to %q", c.ThorChain.NodeURL)
	}
	if _, err := url.Parse(c.ThorChain.NodeURL); err != nil {
		log.Fatal("exit on malformed THOR node REST URL: ", err)
	}
	notinchain.BaseURL = c.ThorChain.NodeURL

	if c.ThorChain.URL == "" {
		c.ThorChain.URL = "http://localhost:26657/websocket"
		log.Printf("default Tendermint RPC URL to %q", c.ThorChain.URL)
	} else {
		log.Printf("Tendermint RPC URL is set to %q", c.ThorChain.URL)
	}
	endpoint, err := url.Parse(c.ThorChain.URL)
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
		log.Print("starting with previous blockchain height ", offset)
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
			offset, err = client.Follow(ch, offset, nil)
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

func MustLoadConfigFile(path string) *Config {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal("exit on configuration file unavailable: ", err)
	}
	defer f.Close()

	dec := json.NewDecoder(f)

	// prevent config not used due typos
	dec.DisallowUnknownFields()

	var c Config
	if err := dec.Decode(&c); err != nil {
		log.Fatal("exit on malformed configuration: ", err)
	}
	return &c
}

type Config struct {
	ListenPort      int      `json:"listen_port"`
	ShutdownTimeout Duration `json:"shutdown_timeout"`
	ReadTimeout     Duration `json:"read_timeout"`
	WriteTimeout    Duration `json:"write_timeout"`

	TimeScale struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		UserName string `json:"user_name"`
		Password string `json:"password"`
		Database string `json:"database"`
		Sslmode  string `json:"sslmode"`
	} `json:"timescale"`

	ThorChain struct {
		URL                         string   `json:"url"`
		NodeURL                     string   `json:"node_url"`
		ReadTimeout                 Duration `json:"read_timeout"`
		LastChainBackoff            Duration `json:"last_chain_backoff"`
		ProxiedWhitelistedEndpoints []string `json:"proxied_whitelisted_endpoints"`
	} `json:"thorchain"`
}

type Duration time.Duration

func (d Duration) WithDefault(def time.Duration) time.Duration {
	if d == 0 {
		return def
	}
	return time.Duration(d)
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case string:
		v, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		*d = Duration(v)
	default:
		return errors.New("duration not a string")
	}
	return nil
}
