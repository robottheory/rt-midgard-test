package config

import (
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog"
)

type Duration time.Duration

type Config struct {
	ListenPort      int      `json:"listen_port" split_words:"true"`
	ShutdownTimeout Duration `json:"shutdown_timeout" split_words:"true"`

	// ReadTimeout and WriteTimeout refer to the webserver timeouts
	ReadTimeout  Duration `json:"read_timeout" split_words:"true"`
	WriteTimeout Duration `json:"write_timeout" split_words:"true"`

	BlockStore BlockStore

	// Only for development.
	FailOnError bool `json:"fail_on_error" split_words:"true"`

	ThorChain ThorChain `json:"thorchain"`

	TimeScale TimeScale `json:"timescale"`

	Websockets Websockets `json:"websockets" split_words:"true"`

	UsdPools []string `json:"usdpools" split_words:"true"`
}

type BlockStore struct {
	Local  string `json:"local" split_words:"true"`
	Remote string `json:"remote" split_words:"true"`
}

type ThorChain struct {
	TendermintURL               string   `json:"tendermint_url" split_words:"true"`
	ThorNodeURL                 string   `json:"thornode_url" split_words:"true"`
	ReadTimeout                 Duration `json:"read_timeout" split_words:"true"`
	LastChainBackoff            Duration `json:"last_chain_backoff" split_words:"true"`
	ProxiedWhitelistedEndpoints []string `json:"proxied_whitelisted_endpoints" split_words:"true"`
	FetchBatchSize              int      `json:"fetch_batch_size" split_words:"true"`
	Parallelism                 int      `json:"parallelism" split_words:"true"`
}

type TimeScale struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	UserName string `json:"user_name"`
	Password string `json:"password"`
	Database string `json:"database"`
	Sslmode  string `json:"sslmode"`

	// -1 sets it to infinite
	MaxOpenConns    int `json:"max_open_conns"`
	CommitBatchSize int `json:"commit_batch_size"`
}

type Websockets struct {
	Enable          bool `json:"enable" split_words:"true"`
	ConnectionLimit int  `json:"connection_limit" split_words:"true"`
}

var defaultConfig = Config{
	ThorChain: ThorChain{
		ThorNodeURL:      "http://localhost:1317/thorchain",
		TendermintURL:    "http://localhost:26657/websocket",
		ReadTimeout:      Duration(8 * time.Second),
		LastChainBackoff: Duration(7 * time.Second),

		// NOTE(huginn): numbers are chosen to give a good performance on an "average" desktop
		// machine with a 4 core CPU. With more cores it might make sense to increase the
		// parallelism, though care should be taken to not overload the Thornode.
		// See `docs/parallel_batch_bench.md` for measurments to guide selection of these parameters.
		FetchBatchSize: 100, // must be divisible by BlockFetchParallelism
		Parallelism:    4,
	},
	TimeScale: TimeScale{
		MaxOpenConns:    80,
		CommitBatchSize: 100,
	},
	ShutdownTimeout: Duration(20 * time.Second),
	ReadTimeout:     Duration(20 * time.Second),
	WriteTimeout:    Duration(20 * time.Second),
}

var logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).With().Timestamp().Str("module", "config").Logger()

func (d Duration) Value() time.Duration {
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

func (d *Duration) Decode(value string) error {
	v, err := time.ParseDuration(value)
	if err != nil {
		return err
	}
	*d = Duration(v)
	return nil
}

func MustLoadConfigFile(path string, c *Config) {
	f, err := os.Open(path)
	if err != nil {
		logger.Fatal().Err(err).Msg("Exit on configuration file unavailable")
	}
	defer f.Close()

	dec := json.NewDecoder(f)

	// prevent config not used due typos
	dec.DisallowUnknownFields()

	if err := dec.Decode(&c); err != nil {
		logger.Fatal().Err(err).Msg("Exit on malformed configuration")
	}
}

func logAndcheckUrls(c *Config) {
	urls := []struct {
		url, name string
	}{
		{c.ThorChain.ThorNodeURL, "THORNode REST URL"},
		{c.ThorChain.TendermintURL, "Tendermint RPC URL"},
		{c.BlockStore.Remote, "BlockStore Remote URL"},
	}
	for _, v := range urls {
		logger.Info().Msgf(v.name+": %q", v.url)
		if _, err := url.Parse(v.url); err != nil {
			logger.Fatal().Err(err).Msgf("Exit on malformed %s", v.url)
		}
	}
}

func ReadConfigFrom(filename string) Config {
	var ret Config = defaultConfig
	if filename != "" {
		MustLoadConfigFile(filename, &ret)
	}

	// override config with env variables
	err := envconfig.Process("midgard", &ret)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to process config environment variables")
	}

	logAndcheckUrls(&ret)

	return ret
}

func ReadConfig() Config {
	switch len(os.Args) {
	case 1:
		return ReadConfigFrom("")
	case 2:
		return ReadConfigFrom(os.Args[1])
	default:
		logger.Fatal().Msg("One optional configuration file argument only-no flags")
		return Config{}
	}
}
