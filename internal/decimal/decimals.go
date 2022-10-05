package decimal

import (
	_ "embed"
	"encoding/json"

	"gitlab.com/thorchain/midgard/internal/util/midlog"
)

//go:embed decimals.json
var decimalString string
var poolsDecimal map[string]SingleResult

type SingleResult struct {
	NativeDecimals int64    `json:"decimals"` // -1 means that only the asset name was observed without the decimal count.
	AssetSeen      []string `json:"asset_seen"`
	DecimalSource  []string `json:"decimal_source"`
}

func init() {
	err := json.Unmarshal([]byte(decimalString), &poolsDecimal)
	if err != nil {
		midlog.FatalE(err, "There is no decimals.json file to open. please run the decimal script first: `go run ./cmd/decimal` to get the native decimal values in the pools endpoint")
	}
}

func PoolsDecimal() map[string]SingleResult {
	return poolsDecimal
}
