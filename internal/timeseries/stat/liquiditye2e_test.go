package stat_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

func TestLiquidityHistoryE2E(t *testing.T) {
	testdb.InitTest(t)

	timeseries.SetDepthsForTest([]timeseries.Depth{
		{Pool: "BTC.BTC", AssetDepth: 1, RuneDepth: 1},
		{Pool: "BNB.BNB", AssetDepth: 1, RuneDepth: 1},
	})

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
	body := testdb.CallJSON(t, fmt.Sprintf(
		"http://localhost:8080/v2/history/liquidity_changes?interval=day&from=%d&to=%d", from, to))

	var jsonResult oapigen.LiquidityHistoryResponse
	testdb.MustUnmarshal(t, body, &jsonResult)

	require.Equal(t, epochStr("2020-09-03 00:00:00"), jsonResult.Meta.StartTime)
	require.Equal(t, intStr(to), jsonResult.Meta.EndTime)
	require.Equal(t, intStr(expectedBTCDeposits+expectedBNBDeposits), jsonResult.Meta.AddLiquidityVolume)
	require.Equal(t, intStr(expectedBTCWithdrawals+expectedBNBWithdrawals), jsonResult.Meta.WithdrawVolume)
	require.Equal(t, "3", jsonResult.Meta.AddLiquidityCount)
	require.Equal(t, "3", jsonResult.Meta.WithdrawCount)

	require.Equal(t, 3, len(jsonResult.Intervals))
	require.Equal(t, epochStr("2020-09-03 00:00:00"), jsonResult.Intervals[0].StartTime)
	require.Equal(t, epochStr("2020-09-04 00:00:00"), jsonResult.Intervals[0].EndTime)
	require.Equal(t, epochStr("2020-09-05 00:00:00"), jsonResult.Intervals[2].StartTime)
	require.Equal(t, intStr(to), jsonResult.Intervals[2].EndTime)

	require.Equal(t, intStr(expectedBTCDeposits), jsonResult.Intervals[0].AddLiquidityVolume)
	require.Equal(t, intStr(expectedBTCWithdrawals), jsonResult.Intervals[0].WithdrawVolume)
	require.Equal(t, "2", jsonResult.Intervals[0].AddLiquidityCount)
	require.Equal(t, "1", jsonResult.Intervals[0].WithdrawCount)

	require.Equal(t, "0", jsonResult.Intervals[1].AddLiquidityVolume)
	require.Equal(t, "0", jsonResult.Intervals[1].WithdrawVolume)

	require.Equal(t, intStr(expectedBNBDeposits), jsonResult.Intervals[2].AddLiquidityVolume)
	require.Equal(t, intStr(expectedBNBWithdrawals), jsonResult.Intervals[2].WithdrawVolume)

	// Check single pool
	body = testdb.CallJSON(t, fmt.Sprintf(
		"http://localhost:8080/v2/history/liquidity_changes?interval=day&from=%d&to=%d&pool=BNB.BNB", from, to))

	testdb.MustUnmarshal(t, body, &jsonResult)
	require.Equal(t, intStr(expectedBNBDeposits), jsonResult.Meta.AddLiquidityVolume)
	require.Equal(t, intStr(expectedBNBWithdrawals), jsonResult.Meta.WithdrawVolume)
}

func TestLiquidityAddOnePoolOnly(t *testing.T) {
	testdb.InitTest(t)

	testdb.InsertBlockPoolDepth(t, "BTC.BTC", 100, 200, "2020-01-01 12:00:00")
	testdb.InsertBlockPoolDepth(t, "BNB.BNB", 100, 300, "2020-01-01 12:00:00")

	testdb.InsertStakeEvent(t, testdb.FakeStake{Pool: "BTC.BTC", AssetE8: 1, RuneE8: 2, BlockTimestamp: "2020-01-01 12:00:00"})
	from := testdb.StrToSec("2020-01-01 00:00:00").ToI()
	to := testdb.StrToSec("2020-01-02 00:00:00").ToI()

	body := testdb.CallJSON(t, fmt.Sprintf(
		"http://localhost:8080/v2/history/liquidity_changes?interval=day&from=%d&to=%d", from, to))

	var jsonResult oapigen.LiquidityHistoryResponse
	testdb.MustUnmarshal(t, body, &jsonResult)

	require.Equal(t, "4", jsonResult.Meta.AddLiquidityVolume)
	require.Equal(t, "1", jsonResult.Meta.AddLiquidityCount)
}

func TestLiquidityAssymetric(t *testing.T) {
	testdb.InitTest(t)

	testdb.InsertBlockPoolDepth(t, "BTC.BTC", 100, 200, "2020-01-01 12:00:00")

	testdb.InsertStakeEvent(t, testdb.FakeStake{
		Pool:    "BTC.BTC",
		AssetE8: 10, RuneE8: 2,
		BlockTimestamp: "2020-01-01 12:00:00"})

	testdb.InsertUnstakeEvent(t, testdb.FakeUnstake{
		Pool:        "BTC.BTC",
		EmitAssetE8: 1, EmitRuneE8: 1,
		BlockTimestamp: "2020-01-01 12:00:00"})

	from := testdb.StrToSec("2020-01-01 00:00:00").ToI()
	to := testdb.StrToSec("2020-01-02 00:00:00").ToI()

	body := testdb.CallJSON(t, fmt.Sprintf(
		"http://localhost:8080/v2/history/liquidity_changes?interval=day&from=%d&to=%d", from, to))

	var jsonResult oapigen.LiquidityHistoryResponse
	testdb.MustUnmarshal(t, body, &jsonResult)

	require.Equal(t, "20", jsonResult.Meta.AddAssetLiquidityVolume)
	require.Equal(t, "2", jsonResult.Meta.AddRuneLiquidityVolume)
	require.Equal(t, "22", jsonResult.Meta.AddLiquidityVolume)
	require.Equal(t, "1", jsonResult.Meta.AddLiquidityCount)

	require.Equal(t, "2", jsonResult.Meta.WithdrawAssetVolume)
	require.Equal(t, "1", jsonResult.Meta.WithdrawRuneVolume)
	require.Equal(t, "3", jsonResult.Meta.WithdrawVolume)
	require.Equal(t, "1", jsonResult.Meta.WithdrawCount)
}

func TestImpermanentLoss(t *testing.T) {
	testdb.InitTest(t)

	testdb.InsertBlockPoolDepth(t, "BTC.BTC", 100, 200, "2020-01-01 12:00:00")

	testdb.InsertStakeEvent(t, testdb.FakeStake{
		Pool:    "BTC.BTC",
		AssetE8: 10, RuneE8: 2,
		BlockTimestamp: "2020-01-01 12:00:00"})

	testdb.InsertUnstakeEvent(t, testdb.FakeUnstake{
		Pool:        "BTC.BTC",
		EmitAssetE8: 5, EmitRuneE8: 100,
		ImpLossProtectionE8: 42,
		BlockTimestamp:      "2020-01-01 12:00:00"})

	from := testdb.StrToSec("2020-01-01 00:00:00").ToI()
	to := testdb.StrToSec("2020-01-02 00:00:00").ToI()

	body := testdb.CallJSON(t, fmt.Sprintf(
		"http://localhost:8080/v2/history/liquidity_changes?interval=day&from=%d&to=%d", from, to))

	var jsonResult oapigen.LiquidityHistoryResponse
	testdb.MustUnmarshal(t, body, &jsonResult)

	require.Len(t, jsonResult.Intervals, 1)
	require.Equal(t, "42", jsonResult.Intervals[0].ImpermanentLossProtectionPaid)

	require.Equal(t, "42", jsonResult.Meta.ImpermanentLossProtectionPaid)
}
