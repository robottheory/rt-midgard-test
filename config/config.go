package config

import (
	"encoding/json"
	"errors"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/kelseyhightower/envconfig"
	"gitlab.com/thorchain/midgard/internal/db"
)

type Duration time.Duration

type Config struct {
	ListenPort      int      `json:"listen_port" split_words:"true"`
	ShutdownTimeout Duration `json:"shutdown_timeout" split_words:"true"`
	ReadTimeout     Duration `json:"read_timeout" split_words:"true"`
	WriteTimeout    Duration `json:"write_timeout" split_words:"true"`

	// Only for development.
	FailOnError bool `json:"fail_on_error" split_words:"true"`

	TimeScale db.Config `json:"timescale"`

	ThorChain struct {
		TendermintURL               string   `json:"tendermint_url" split_words:"true"`
		ThorNodeURL                 string   `json:"thornode_url" split_words:"true"`
		ReadTimeout                 Duration `json:"read_timeout" split_words:"true"`
		LastChainBackoff            Duration `json:"last_chain_backoff" split_words:"true"`
		ProxiedWhitelistedEndpoints []string `json:"proxied_whitelisted_endpoints" split_words:"true"`
	} `json:"thorchain"`

	Websockets struct {
		Enable          bool `json:"enable" split_words:"true"`
		ConnectionLimit int  `json:"connection_limit" split_words:"true"`
	} `json:"websockets" split_words:"true"`

	UsdPools []string `json:"usdpools" split_words:"true"`
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

func setDefaultUrls(c *Config) {
	if c.ThorChain.ThorNodeURL == "" {
		c.ThorChain.ThorNodeURL = "http://localhost:1317/thorchain"
		log.Printf("default THOR node REST URL to %q", c.ThorChain.ThorNodeURL)
	} else {
		log.Printf("THOR node REST URL is set to %q", c.ThorChain.ThorNodeURL)
	}
	if _, err := url.Parse(c.ThorChain.ThorNodeURL); err != nil {
		log.Fatal("exit on malformed THOR node REST URL: ", err)
	}

	if c.ThorChain.TendermintURL == "" {
		c.ThorChain.TendermintURL = "http://localhost:26657/websocket"
		log.Printf("default Tendermint RPC URL to %q", c.ThorChain.TendermintURL)
	} else {
		log.Printf("Tendermint RPC URL is set to %q", c.ThorChain.TendermintURL)
	}
	_, err := url.Parse(c.ThorChain.TendermintURL)
	if err != nil {
		log.Fatal("exit on malformed Tendermint RPC URL: ", err)
	}
}

func ReadConfig() Config {
	var ret Config
	switch len(os.Args) {
	case 1:
		break // refer to defaults
	case 2:
		ret = *MustLoadConfigFile(os.Args[1])
	default:
		log.Fatal("One optional configuration file argument only—no flags")
	}

	// override config with env variables
	err := envconfig.Process("midgard", &ret)
	if err != nil {
		log.Fatal("Failed to process config environment variables, ", err)
	}

	setDefaultUrls(&ret)
	return ret
}
