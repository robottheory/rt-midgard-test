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
	"gitlab.com/thorchain/midgard/event"
	"gitlab.com/thorchain/midgard/internal/api"
	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/internal/timeseries/stat"
)

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
	srv := &http.Server{
		Handler:      api.CORS(api.Handler),
		Addr:         fmt.Sprintf(":%d", c.ListenPort),
		ReadTimeout:  c.ReadTimeout.WithDefault(2 * time.Second),
		WriteTimeout: c.WriteTimeout.WithDefault(2 * time.Second),
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
			m.Block(block)
			err := timeseries.CommitBlock(block.Height, block.Time, block.Hash)
			if err != nil {
				log.Print("timeseries feed stopped on ", err)
				signals <- syscall.SIGABRT
				return
			}
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
	db, err := sql.Open("pgx", fmt.Sprintf("user=%s dbname=%s sslmode=%s password=%s host=%s port=%d", c.TimeScale.UserName, c.TimeScale.Database, c.TimeScale.Sslmode, c.TimeScale.Password, c.TimeScale.Host, c.TimeScale.Port))
	if err != nil {
		log.Fatal("exit on PostgreSQL client instantiation: ", err)
	}

	stat.DBQuery = db.Query
	timeseries.DBExec = db.Exec
	timeseries.DBQuery = db.Query
}

// SetupBlockchain launches the synchronisation routine.
func SetupBlockchain(c *Config) <-chan chain.Block {
	// normalize configuration
	if c.ThorChain.Scheme == "" {
		c.ThorChain.Scheme = "http"
		log.Printf("default Tendermint RPC scheme to %q", c.ThorChain.Scheme)
	}
	if c.ThorChain.Host == "" {
		c.ThorChain.Host = "localhost:26657"
		log.Printf("default Tendermint RPC host to %q", c.ThorChain.Host)
	}

	// instantiate client
	endpoint := &url.URL{Scheme: c.ThorChain.Scheme, Host: c.ThorChain.RPCHost, Path: "/websocket"}
	log.Print("Tendermint enpoint set to ", endpoint.String())
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
		return time.Since(lastNoData.Load().(time.Time)) < 2*c.ThorChain.NoEventsBackoff.WithDefault(7*time.Second)
	}

	// launch read routine
	ch := make(chan chain.Block, 99)
	go func() {
		// BUG(pascaldekloe): NoEventsBackoff is a misnommer
		// as chains without events do not cause any backoff.
		backoff := time.NewTicker(c.ThorChain.NoEventsBackoff.WithDefault(7 * time.Second))
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
	ListenPort      int      `json:"listen_port" mapstructure:"listen_port"`
	ShutdownTimeout Duration `json:"shutdown_timeout" mapstructure:"shutdown_timeout"`
	ReadTimeout     Duration `json:"read_timeout" mapstructure:"read_timeout"`
	WriteTimeout    Duration `json:"write_timeout" mapstructure:"write_timeout"`

	TimeScale struct {
		Host                  string   `json:"host" mapstructure:"host"`
		Port                  int      `json:"port" mapstructure:"port"`
		UserName              string   `json:"user_name" mapstructure:"user_name"`
		Password              string   `json:"password" mapstructure:"password"`
		Database              string   `json:"database" mapstructure:"database"`
		Sslmode               string   `json:"sslmode" mapstructure:"sslmode"`
	} `json:"timescale" mapstructure:"timescale"`

	ThorChain struct {
		Scheme                      string   `json:"scheme" mapstructure:"scheme"`
		Host                        string   `json:"host" mapstructure:"host"`
		RPCHost                     string   `json:"rpc_host" mapstructure:"rpc_host"`
		ReadTimeout                 Duration `json:"read_timeout" mapstructure:"read_timeout"`
		NoEventsBackoff             Duration `json:"no_events_backoff" mapstructure:"no_events_backoff"`
	} `json:"thorchain" mapstructure:"thorchain"`
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
