package stat_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

func TestUnstakesLookupE2E(t *testing.T) {
	testdb.InitTest(t)

	testdb.InsertUnstakeEvent(t, testdb.FakeUnstake{
		Pool: "BTC.BTC", EmitAssetE8: 2, EmitRuneE8: 2, BlockTimestamp: "2021-01-10 12:30:00"})
	testdb.InsertBlockPoolDepth(t, "BTC.BTC", 1, 100, "2021-01-10 12:30:00")

	testdb.InsertUnstakeEvent(t, testdb.FakeUnstake{
		Pool: "BNB.BNB", EmitAssetE8: 3, EmitRuneE8: 3, BlockTimestamp: "2021-01-12 12:30:00"})
	testdb.InsertBlockPoolDepth(t, "BNB.BNB", 1, 10, "2021-01-12 12:30:00")

	body := testdb.CallJSON(t, "http://localhost:8080/v2/stats")
	var jsonResult oapigen.StatsData
	testdb.MustUnmarshal(t, body, &jsonResult)

	assert.Equal(t, "2", jsonResult.WithdrawCount)
	assert.Equal(t, "235", jsonResult.WithdrawVolume)
}

func TestUnstakesEmpty(t *testing.T) {
	testdb.InitTest(t)

	body := testdb.CallJSON(t, "http://localhost:8080/v2/stats")
	var jsonResult oapigen.StatsData
	testdb.MustUnmarshal(t, body, &jsonResult)

	assert.Equal(t, "0", jsonResult.WithdrawCount)
	assert.Equal(t, "0", jsonResult.WithdrawVolume)
}

func TestWithdrawAllAssets(t *testing.T) {
	testdb.InitTest(t)

	testdb.InsertUnstakeEvent(t, testdb.FakeUnstake{
		Pool: "BNB.BNB", EmitAssetE8: 10, EmitRuneE8: 0, BlockTimestamp: "2021-01-12 12:30:00"})
	testdb.InsertBlockPoolDepth(t, "BNB.BNB", 0, 20, "2021-01-12 12:30:00")

	body := testdb.CallJSON(t, "http://localhost:8080/v2/stats")
	var jsonResult oapigen.StatsData
	testdb.MustUnmarshal(t, body, &jsonResult)

	assert.Equal(t, "1", jsonResult.WithdrawCount)
	assert.Equal(t, "0", jsonResult.WithdrawVolume)
}

func TestStakesLookupE2E(t *testing.T) {
	testdb.InitTest(t)

	testdb.InsertStakeEvent(t, testdb.FakeStake{
		Pool: "BTC.BTC", AssetE8: 2, RuneE8: 2, BlockTimestamp: "2021-01-10 12:30:00"})
	testdb.InsertBlockPoolDepth(t, "BTC.BTC", 1, 100, "2021-01-10 12:30:00")

	testdb.InsertStakeEvent(t, testdb.FakeStake{
		Pool: "BNB.BNB", AssetE8: 3, RuneE8: 3, BlockTimestamp: "2021-01-12 12:30:00"})
	testdb.InsertBlockPoolDepth(t, "BNB.BNB", 1, 10, "2021-01-12 12:30:00")

	body := testdb.CallJSON(t, "http://localhost:8080/v2/stats")
	var jsonResult oapigen.StatsData
	testdb.MustUnmarshal(t, body, &jsonResult)

	assert.Equal(t, "2", jsonResult.AddLiquidityCount)
	assert.Equal(t, "235", jsonResult.AddLiquidityVolume)
}
