package stat_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

func TestUnstakesLookupE2E(t *testing.T) {
	testdb.InitTest(t)

	timeseries.SetDepthsForTest([]timeseries.Depth{
		{Pool: "BNB.BNB", AssetDepth: 1, RuneDepth: 10},
		{Pool: "BTC.BTC", AssetDepth: 1, RuneDepth: 100},
	})

	testdb.InsertUnstakeEvent(t, testdb.FakeUnstake{
		Pool: "BTC.BTC", EmitAssetE8: 2, EmitRuneE8: 2, BlockTimestamp: "2021-01-10 12:30:00"})
	testdb.InsertBlockPoolDepth(t, "BTC.BTC", 1, 100, "2021-01-10 12:30:00")

	testdb.InsertUnstakeEvent(t, testdb.FakeUnstake{
		Pool: "BNB.BNB", EmitAssetE8: 3, EmitRuneE8: 3, BlockTimestamp: "2021-01-12 12:30:00"})
	testdb.InsertBlockPoolDepth(t, "BNB.BNB", 1, 10, "2021-01-12 12:30:00")

	body := testdb.CallV1(t, fmt.Sprintf(
		"http://localhost:8080/v2/stats"))
	var jsonResult oapigen.StatsData
	testdb.MustUnmarshal(t, body, &jsonResult)

	assert.Equal(t, "2", jsonResult.WithdrawCount)
	assert.Equal(t, "235", jsonResult.WithdrawVolume)
}

func TestUnstakesEmpty(t *testing.T) {
	testdb.InitTest(t)

	timeseries.SetDepthsForTest([]timeseries.Depth{
		{Pool: "BNB.BNB", AssetDepth: 1, RuneDepth: 10},
		{Pool: "BTC.BTC", AssetDepth: 1, RuneDepth: 100},
	})

	body := testdb.CallV1(t, fmt.Sprintf(
		"http://localhost:8080/v2/stats"))
	var jsonResult oapigen.StatsData
	testdb.MustUnmarshal(t, body, &jsonResult)

	assert.Equal(t, "0", jsonResult.WithdrawCount)
	assert.Equal(t, "0", jsonResult.WithdrawVolume)
}
