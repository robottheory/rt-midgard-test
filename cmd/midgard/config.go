package main

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"time"
)

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
