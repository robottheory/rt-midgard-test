package stat_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

func TestLiquidityHistoryE2E(t *testing.T) {
	testdb.SetupTestDB(t)

	testdb.MustExec(t, "DELETE FROM stake_events")
	testdb.MustExec(t, "DELETE FROM unstake_events")
	testdb.MustExec(t, "DELETE FROM block_pool_depths")

	// 3rd of September
	testdb.InsertBlockPoolDepth(t, "BTC.BTC", 100, 200, "2020-09-03 12:30:00")
	testdb.InsertStakeEvent(t, testdb.FakeStake{Pool: "BTC.BTC", AssetE8: 1, RuneE8: 2, BlockTimestamp: "2020-09-03 12:30:00"})
	testdb.InsertStakeEvent(t, testdb.FakeStake{Pool: "BTC.BTC", AssetE8: 3, RuneE8: 4, BlockTimestamp: "2020-09-03 12:30:00"})
	testdb.InsertUnstakeEvent(t, testdb.FakeUnstake{Pool: "BTC.BTC", EmitAssetE8: 5, EmitRuneE8: 6, BlockTimestamp: "2020-09-03 12:30:00"})

	// 5th of September
	testdb.InsertBlockPoolDepth(t, "BNB.BNB", 100, 300, "2020-09-05 12:30:00")
	testdb.InsertStakeEvent(t, testdb.FakeStake{Pool: "BNB.BNB", AssetE8: 7, RuneE8: 8, BlockTimestamp: "2020-09-05 12:30:00"})
	testdb.InsertUnstakeEvent(t, testdb.FakeUnstake{Pool: "BNB.BNB", EmitAssetE8: 9, EmitRuneE8: 10, BlockTimestamp: "2020-09-05 12:30:00"})
	testdb.InsertUnstakeEvent(t, testdb.FakeUnstake{Pool: "BNB.BNB", EmitAssetE8: 11, EmitRuneE8: 12, BlockTimestamp: "2020-09-05 12:30:00"})

	from := testdb.StrToSec("2020-09-03 00:00:00").ToI()
	to := testdb.StrToSec("2020-09-06 00:00:00").ToI()

	expectedBTCDeposits := int64(1*2 + 2 + 3*2 + 4)
	expectedBNBDeposits := int64(7*3 + 8)
	expectedBTCWithdrawals := int64(5*2 + 6)
	expectedBNBWithdrawals := int64(9*3 + 10 + 11*3 + 12)
	// Check all pools
	body := testdb.CallV1(t, fmt.Sprintf(
		"http://localhost:8080/v2/history/liquidity_changes?interval=day&from=%d&to=%d", from, to))

	var jsonResult oapigen.LiquidityHistoryResponse
	testdb.MustUnmarshal(t, body, &jsonResult)

	assert.Equal(t, epochStr("2020-09-03 00:00:00"), jsonResult.Meta.StartTime)
	assert.Equal(t, intStr(to), jsonResult.Meta.EndTime)
	assert.Equal(t, intStr(expectedBTCDeposits+expectedBNBDeposits), jsonResult.Meta.AddLiqudityVolume)
	assert.Equal(t, intStr(expectedBTCWithdrawals+expectedBNBWithdrawals), jsonResult.Meta.WithdrawVolume)

	assert.Equal(t, 3, len(jsonResult.Intervals))
	assert.Equal(t, epochStr("2020-09-03 00:00:00"), jsonResult.Intervals[0].StartTime)
	assert.Equal(t, epochStr("2020-09-04 00:00:00"), jsonResult.Intervals[0].EndTime)
	assert.Equal(t, epochStr("2020-09-05 00:00:00"), jsonResult.Intervals[2].StartTime)
	assert.Equal(t, intStr(to), jsonResult.Intervals[2].EndTime)

	assert.Equal(t, intStr(expectedBTCDeposits), jsonResult.Intervals[0].AddLiqudityVolume)
	assert.Equal(t, intStr(expectedBTCWithdrawals), jsonResult.Intervals[0].WithdrawVolume)

	assert.Equal(t, "0", jsonResult.Intervals[1].AddLiqudityVolume)
	assert.Equal(t, "0", jsonResult.Intervals[1].WithdrawVolume)

	assert.Equal(t, intStr(expectedBNBDeposits), jsonResult.Intervals[2].AddLiqudityVolume)
	assert.Equal(t, intStr(expectedBNBWithdrawals), jsonResult.Intervals[2].WithdrawVolume)

	// Check single pool
	body = testdb.CallV1(t, fmt.Sprintf(
		"http://localhost:8080/v2/history/liquidity_changes?interval=day&from=%d&to=%d&pool=BNB.BNB", from, to))

	testdb.MustUnmarshal(t, body, &jsonResult)
	assert.Equal(t, intStr(expectedBNBDeposits), jsonResult.Meta.AddLiqudityVolume)
	assert.Equal(t, intStr(expectedBNBWithdrawals), jsonResult.Meta.WithdrawVolume)
}

func TestLiquidityAddOnePoolOnly(t *testing.T) {
	testdb.SetupTestDB(t)

	testdb.MustExec(t, "DELETE FROM stake_events")
	testdb.MustExec(t, "DELETE FROM unstake_events")
	testdb.MustExec(t, "DELETE FROM block_pool_depths")

	testdb.InsertBlockPoolDepth(t, "BTC.BTC", 100, 200, "2020-01-01 12:00:00")
	testdb.InsertBlockPoolDepth(t, "BNB.BNB", 100, 300, "2020-01-01 12:00:00")

	testdb.InsertStakeEvent(t, testdb.FakeStake{Pool: "BTC.BTC", AssetE8: 1, RuneE8: 2, BlockTimestamp: "2020-01-01 12:00:00"})
	from := testdb.StrToSec("2020-01-01 00:00:00").ToI()
	to := testdb.StrToSec("2020-01-02 00:00:00").ToI()

	body := testdb.CallV1(t, fmt.Sprintf(
		"http://localhost:8080/v2/history/liquidity_changes?interval=day&from=%d&to=%d", from, to))

	var jsonResult oapigen.LiquidityHistoryResponse
	testdb.MustUnmarshal(t, body, &jsonResult)

	assert.Equal(t, "4", jsonResult.Meta.AddLiqudityVolume)
}
