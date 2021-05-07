package testdb

import (
	"fmt"
	"testing"

	abci "github.com/tendermint/tendermint/abci/types"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"

	"github.com/stretchr/testify/require"
	"gitlab.com/thorchain/midgard/internal/fetch/chain"
	"gitlab.com/thorchain/midgard/internal/fetch/record"
	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/internal/util"
)

type blockCreator struct {
	lastHeight int64
	demux      record.Demux
}

type FakeEvent interface {
	ToTendermint() abci.Event
}

func (bc *blockCreator) NewBlock(t *testing.T, timeStr string, events ...FakeEvent) {
	bc.lastHeight++

	block := chain.Block{
		Height:  bc.lastHeight,
		Time:    StrToSec(timeStr).ToTime(),
		Hash:    []byte(fmt.Sprintf("hash%d", bc.lastHeight)),
		Results: &coretypes.ResultBlockResults{}}

	for _, event := range events {
		block.Results.EndBlockEvents = append(block.Results.EndBlockEvents, event.ToTendermint())
	}

	bc.demux.Block(block)
	err := timeseries.CommitBlock(block.Height, block.Time, block.Hash)
	require.NoError(t, err)
}

func toAttributes(attrs map[string]string) (ret []abci.EventAttribute) {
	for k, v := range attrs {
		ret = append(ret, abci.EventAttribute{Index: true, Key: []byte(k), Value: []byte(v)})
	}
	return
}

type Swap struct {
	Pool         string
	Coin         string
	EmitAsset    string
	LiquidityFee int64
}

func (x Swap) ToTendermint() abci.Event {
	return abci.Event{Type: "swap", Attributes: toAttributes(map[string]string{
		"pool":                  x.Pool,
		"memo":                  "doesntmatter",
		"coin":                  x.Coin,
		"emit_asset":            x.EmitAsset,
		"from":                  "addressfrom",
		"to":                    "addressto",
		"chain":                 "chain",
		"id":                    "txid",
		"swap_target":           "0",
		"swap_slip":             "1",
		"liquidity_fee":         "1",
		"liquidity_fee_in_rune": util.IntStr(x.LiquidityFee),
	})}
}

type AddLiquidity struct {
	Pool        string
	AssetAmount int64
	RuneAmount  int64
}

func (x AddLiquidity) ToTendermint() abci.Event {
	return abci.Event{Type: "add_liquidity", Attributes: toAttributes(map[string]string{
		"pool":                     x.Pool,
		"liquidity_provider_units": "1",
		"rune_address":             "runeAddress",
		"rune_amount":              util.IntStr(x.RuneAmount),
		"asset_amount":             util.IntStr(x.AssetAmount),
		"asset_address":            "assetAddress",
		"THOR_txid":                "chain",
		"BNB_txid":                 "txid",
	})}
}

type PoolActivate struct {
	Pool string
}

func (x PoolActivate) ToTendermint() abci.Event {
	return abci.Event{Type: "pool", Attributes: toAttributes(map[string]string{
		"pool":        x.Pool,
		"pool_status": "Available",
	})}
}
