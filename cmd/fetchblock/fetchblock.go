package main

// Same as fetching block_results from tendermint, but unwraps the base64 encoded fields
// to readable format.
// For original form one can check:
// http://<tendermint>:26657/block_results?height=1000

import (
	"context"
	"encoding/base64"
	"encoding/json"

	"github.com/sirupsen/logrus"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	"gitlab.com/thorchain/midgard/config"
	"gitlab.com/thorchain/midgard/internal/fetch/chain"
)

const HEIGHT = 1000

var fieldsToUnwrap = map[string]bool{"key": true, "value": true}

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: "2006-01-02 15:04:05", FullTimestamp: true})
	logrus.SetLevel(logrus.InfoLevel)
	// logrus.SetLevel(logrus.DebugLevel)

	var c config.Config = config.ReadConfig()

	client, err := chain.NewClient(&c)
	if err != nil {
		logrus.Fatal("exit on Tendermint RPC client instantiation: ", err)
	}
	ctx := context.Background()

	var results *coretypes.ResultBlockResults
	results, err = client.DebugFetchResults(ctx, HEIGHT)
	if err != nil {
		logrus.Fatal("Failed to fetch block: ", err)
	}

	var original interface{} = results
	any := decodeFields(original)
	buf2, _ := json.MarshalIndent(any, "", "\t")
	logrus.Info(string(buf2))
}

func decodeFields(original interface{}) interface{} {
	buf, _ := json.Marshal(original)
	var any interface{}
	err := json.Unmarshal(buf, &any)
	if err != nil {
		logrus.Fatal(err)
	}
	changeFields(any)
	return any
}

func changeFields(any interface{}) {
	msgMap, ok := any.(map[string]interface{})
	if ok {
		for k, v := range msgMap {
			if fieldsToUnwrap[k] {
				s, err := base64.StdEncoding.DecodeString(v.(string))
				if err != nil {
					logrus.Fatal(err)
				}
				msgMap[k] = string(s)
			} else {
				changeFields(v)
			}
		}
	}
	msgSlice, ok := any.([]interface{})
	if ok {
		for i := range msgSlice {
			changeFields(msgSlice[i])
		}
	}
}
