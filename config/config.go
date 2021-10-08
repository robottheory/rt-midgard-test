package config

import (
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog/log"
	"gitlab.com/thorchain/midgard/internal/db"
)

type Duration time.Duration

type Config struct {
	ListenPort      int      `json:"listen_port" split_words:"true"`
	MaxReqPerSec    float64  `json:"max_req_per_sec" split_words:"true"`
	AllowedOrigins  []string `json:"allowed_origins" split_words:"true"`
	ShutdownTimeout Duration `json:"shutdown_timeout" split_words:"true"`
	ReadTimeout     Duration `json:"read_timeout" split_words:"true"`
	WriteTimeout    Duration `json:"write_timeout" split_words:"true"`
	ApiCacheConfig  struct {
		ShortTermLifetime int `json:"short_term_lifetime" split_words:"true"`
		MidTermLifetime   int `json:"mid_term_lifetime" split_words:"true"`
		LongTermLifetime  int `json:"long_term_lifetime" split_words:"true"`
	} `json:"api_cache_config" split_words:"true"`

	// Only for development.
	FailOnError bool `json:"fail_on_error" split_words:"true"`

	TimeScale db.Config `json:"timescale"`

	ThorChain struct {
		TendermintURL               string   `json:"tendermint_url" split_words:"true"`
		ThorNodeURL                 string   `json:"thornode_url" split_words:"true"`
		ReadTimeout                 Duration `json:"read_timeout" split_words:"true"`
		LastChainBackoff            Duration `json:"last_chain_backoff" split_words:"true"`
		ProxiedWhitelistedEndpoints []string `json:"proxied_whitelisted_endpoints" split_words:"true"`
		FetchBatchSize              int      `json:"fetch_batch_size" split_words:"true"`
		Parallelism                 int      `json:"parallelism" split_words:"true"`
	} `json:"thorchain"`

	Websockets struct {
		Enable          bool `json:"enable" split_words:"true"`
		ConnectionLimit int  `json:"connection_limit" split_words:"true"`
	} `json:"websockets" split_words:"true"`

	UsdPools []string `json:"usdpools" split_words:"true"`
}

func IntWithDefault(v int, def int) int {
	if v == 0 {
		return def
	}
	return v
}

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

func (d *Duration) Decode(value string) error {
	v, err := time.ParseDuration(value)
	if err != nil {
		return err
	}
	*d = Duration(v)
	return nil
}

func MustLoadConfigFile(path string) *Config {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal().Err(err).Msg("Exit on configuration file unavailable")
	}
	defer f.Close()

	dec := json.NewDecoder(f)

	// prevent config not used due typos
	dec.DisallowUnknownFields()

	var c Config
	if err := dec.Decode(&c); err != nil {
		log.Fatal().Err(err).Msg("Exit on malformed configuration")
	}
	return &c
}

func setDefaultCacheLifetime(c *Config) {
	if c.ApiCacheConfig.ShortTermLifetime == 0 {
		c.ApiCacheConfig.ShortTermLifetime = 10
	}
	if c.ApiCacheConfig.MidTermLifetime == 0 {
		c.ApiCacheConfig.MidTermLifetime = 60
	}
	if c.ApiCacheConfig.LongTermLifetime == 0 {
		c.ApiCacheConfig.LongTermLifetime = 5 * 60
	}
}

func setDefaultUrls(c *Config) {
	if c.ThorChain.ThorNodeURL == "" {
		c.ThorChain.ThorNodeURL = "http://localhost:1317/thorchain"
		log.Info().Msgf("Default THORNode REST URL to %q", c.ThorChain.ThorNodeURL)
	} else {
		log.Info().Msgf("THORNode REST URL is set to %q", c.ThorChain.ThorNodeURL)
	}
	if _, err := url.Parse(c.ThorChain.ThorNodeURL); err != nil {
		log.Fatal().Err(err).Msg("Exit on malformed THORNode REST URL")
	}

	if c.ThorChain.TendermintURL == "" {
		c.ThorChain.TendermintURL = "http://localhost:26657/websocket"
		log.Info().Msgf("Default Tendermint RPC URL to %q", c.ThorChain.TendermintURL)
	} else {
		log.Info().Msgf("Tendermint RPC URL is set to %q", c.ThorChain.TendermintURL)
	}
	if c.TimeScale.MaxOpenConns == 0 {
		c.TimeScale.MaxOpenConns = 80
		log.Info().Msgf("Default TimeScale.MaxOpenConnections: %d",
			c.TimeScale.MaxOpenConns)
	}
	_, err := url.Parse(c.ThorChain.TendermintURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Exit on malformed Tendermint RPC URL")
	}
}

func ReadConfigFrom(filename string) Config {
	var ret Config
	if filename != "" {
		ret = *MustLoadConfigFile(filename)
	}

	// override config with env variables
	err := envconfig.Process("midgard", &ret)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to process config environment variables")
	}

	setDefaultUrls(&ret)
	setDefaultCacheLifetime(&ret)
	return ret
}

func ReadConfig() Config {
	switch len(os.Args) {
	case 1:
		return ReadConfigFrom("")
	case 2:
		return ReadConfigFrom(os.Args[1])
	default:
		log.Fatal().Msg("One optional configuration file argument only-no flags")
		return Config{}
	}
}
