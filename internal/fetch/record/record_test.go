package record_test

import (
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"

	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/internal/fetch/chain"
	"gitlab.com/thorchain/midgard/internal/fetch/record"
	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

func toAttributes(attrs map[string]string) (ret []abci.EventAttribute) {
	for k, v := range attrs {
		ret = append(ret, abci.EventAttribute{Index: true, Key: []byte(k), Value: []byte(v)})
	}
	return
}

func fakeSwap(pool, coin, emitAsset string) abci.Event {
	return abci.Event{Type: "swap", Attributes: toAttributes(map[string]string{
		"pool":                  pool,
		"memo":                  "doesntmatter",
		"coin":                  coin,
		"emit_asset":            emitAsset,
		"from":                  "addressfrom",
		"to":                    "addressto",
		"chain":                 "chain",
		"id":                    "txid",
		"swap_target":           "0",
		"swap_slip":             "1",
		"liquidity_fee":         "1",
		"liquidity_fee_in_rune": "1",
	})}
}

func createBlock(height int64, timeStr string) chain.Block {
	block := chain.Block{
		Height:  height,
		Time:    testdb.StrToSec(timeStr).ToTime(),
		Hash:    []byte(fmt.Sprintf("hash%d", height)),
		Results: &coretypes.ResultBlockResults{}}
	return block
}

var demux = record.Demux{}

func commitBlock(t *testing.T, block chain.Block) {
	demux.Block(block)

	err := timeseries.CommitBlock(block.Height, block.Time, block.Hash)
	require.NoError(t, err)
}

func checkDepths(t *testing.T, pool string, assetE8, runeE8 int64) {
	body := testdb.CallJSON(t, "http://localhost:8080/v2/pool/"+pool)
	var jsonApiResponse oapigen.PoolResponse
	testdb.MustUnmarshal(t, body, &jsonApiResponse)

	require.Equal(t, "BTC.BTC", jsonApiResponse.Asset)

	assert.Equal(t, strconv.FormatInt(assetE8, 10), jsonApiResponse.AssetDepth, "Bad Asset depth")
	assert.Equal(t, strconv.FormatInt(runeE8, 10), jsonApiResponse.RuneDepth, "Bad Rune depth")
}

func TestSimpleSwap(t *testing.T) {
	testdb.InitTest(t)

	testdb.InsertPoolEvents(t, "BTC.BTC", "Enabled")
	record.Recorder.SetAssetDepth("BTC.BTC", 1000)
	record.Recorder.SetRuneDepth("BTC.BTC", 2000)

	block := createBlock(1, "2021-01-01 00:00:00")
	commitBlock(t, block)

	checkDepths(t, "BTC.BTC", 1000, 2000)

	block = createBlock(2, "2021-01-02 00:00:00")
	block.Results.EndBlockEvents = append(block.Results.EndBlockEvents,
		fakeSwap("BTC.BTC", "100 BTC.BTC", "200 THOR.RUNE"))
	commitBlock(t, block)

	checkDepths(t, "BTC.BTC", 1100, 1800)

	block = createBlock(3, "2021-01-03 00:00:00")
	block.Results.EndBlockEvents = append(block.Results.EndBlockEvents,
		fakeSwap("BTC.BTC", "20 THOR.RUNE", "10 BTC.BTC"))
	commitBlock(t, block)

	checkDepths(t, "BTC.BTC", 1090, 1820)
}

func TestSynthSwap(t *testing.T) {
	testdb.InitTest(t)

	testdb.InsertPoolEvents(t, "BTC.BTC", "Enabled")
	record.Recorder.SetAssetDepth("BTC.BTC", 1000)
	record.Recorder.SetRuneDepth("BTC.BTC", 2000)

	block := createBlock(1, "2021-01-01 00:00:00")
	commitBlock(t, block)

	checkDepths(t, "BTC.BTC", 1000, 2000)

	block = createBlock(2, "2021-01-02 00:00:00")
	block.Results.EndBlockEvents = append(block.Results.EndBlockEvents,
		fakeSwap("BTC.BTC", "100 BTC/BTC", "200 THOR.RUNE"))
	commitBlock(t, block)

	checkDepths(t, "BTC.BTC", 1000, 1800)

	block = createBlock(3, "2021-01-03 00:00:00")
	block.Results.EndBlockEvents = append(block.Results.EndBlockEvents,
		fakeSwap("BTC.BTC", "100 THOR.RUNE", "50 BTC/BTC"))
	commitBlock(t, block)

	checkDepths(t, "BTC.BTC", 1000, 1900)
}

func TestSwapErrors(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	testdb.InitTest(t)

	testdb.InsertPoolEvents(t, "BTC.BTC", "Enabled")
	record.Recorder.SetAssetDepth("BTC.BTC", 1000)
	record.Recorder.SetRuneDepth("BTC.BTC", 2000)

	block := createBlock(1, "2021-01-01 00:00:00")
	commitBlock(t, block)

	checkDepths(t, "BTC.BTC", 1000, 2000)

	block = createBlock(2, "2021-01-02 00:00:00")

	// Unkown from pool
	block.Results.EndBlockEvents = append(block.Results.EndBlockEvents,
		fakeSwap("BTC.BTC", "1 BTC?BTC", "2 THOR.RUNE"))

	// Both is rune
	block.Results.EndBlockEvents = append(block.Results.EndBlockEvents,
		fakeSwap("BTC.BTC", "10 THOR.RUNE", "20 THOR.RUNE"))

	// None is rune
	block.Results.EndBlockEvents = append(block.Results.EndBlockEvents,
		fakeSwap("BTC.BTC", "100 BTC.BTC", "200 BTC/BTC"))

	commitBlock(t, block)

	checkDepths(t, "BTC.BTC", 1000, 2000)
}
