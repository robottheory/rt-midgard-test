package stat_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

func TestEarningsHistoryE2E(t *testing.T) {
	testdb.SetupTestDB(t)

	testdb.MustExec(t, "DELETE FROM swap_events")
	testdb.MustExec(t, "DELETE FROM rewards_events")
	testdb.MustExec(t, "DELETE FROM rewards_event_entries")
	testdb.MustExec(t, "DELETE FROM update_node_account_status_events")

	// Before Interval
	testdb.InsertUpdateNodeAccountStatusEvent(t, "standby", "active", "2020-09-02 12:00:00")
	testdb.InsertUpdateNodeAccountStatusEvent(t, "standby", "active", "2020-09-02 12:00:00")

	// 3rd of September
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.BTCB-1DE", LiqFeeInRuneE8: 1, BlockTimestamp: "2020-09-03 12:00:00"})
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.BTCB-1DE", LiqFeeInRuneE8: 2, BlockTimestamp: "2020-09-03 12:30:00"})
	testdb.InsertUpdateNodeAccountStatusEvent(t, "active", "standby", "2020-09-03 12:30:00")
	testdb.InsertRewardsEvent(t, 3, "2020-09-03 13:00:00")
	testdb.InsertRewardsEventEntry(t, 4, "BNB.BTCB-1DE", "2020-09-03 13:00:00")

	// 5th of September
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.BNB", LiqFeeInRuneE8: 5, BlockTimestamp: "2020-09-05 12:00:00"})
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.BNB", LiqFeeInRuneE8: 6, BlockTimestamp: "2020-09-05 12:20:00"})
	testdb.InsertRewardsEvent(t, 7, "2020-09-05 13:00:00")
	testdb.InsertRewardsEventEntry(t, 8, "BNB.BNB", "2020-09-05 13:00:00")
	testdb.InsertUpdateNodeAccountStatusEvent(t, "standby", "active", "2020-09-05 14:00:00")
	testdb.InsertUpdateNodeAccountStatusEvent(t, "standby", "active", "2020-09-05 14:00:00")
	testdb.InsertUpdateNodeAccountStatusEvent(t, "standby", "active", "2020-09-05 14:00:00")

	// TODO(acsaba): the values reported change based on the from-to window. Fix.
	from := testdb.StrToSec("2020-09-03 00:00:00")
	to := testdb.StrToSec("2020-09-06 00:00:00")

	// Check all pools
	body := testdb.CallV1(t, fmt.Sprintf(
		"http://localhost:8080/v2/history/earnings?interval=day&from=%d&to=%d", from, to))

	var jsonResult oapigen.EarningsHistoryResponse
	testdb.MustUnmarshal(t, body, &jsonResult)

	// Node count weights
	// 3 Sep
	expectedNodeCountWeight1 := 2 * (toUnix("2020-09-03 12:30:00") - toUnix("2020-09-03 00:00:00"))
	expectedNodeCountWeight2 := 1 * (testdb.StrToSec("2020-09-04 00:00:00") - testdb.StrToSec("2020-09-03 12:30:00")).ToI()

	// 4 Sep
	expectedNodeCountWeight3 := 1 * (testdb.StrToSec("2020-09-05 00:00:00") - testdb.StrToSec("2020-09-04 00:00:00")).ToI()

	// 5 Sep
	expectedNodeCountWeight4 := 1 * (testdb.StrToSec("2020-09-05 14:00:00") - testdb.StrToSec("2020-09-05 00:00:00")).ToI()
	expectedNodeCountWeight5 := 4 * (to - testdb.StrToSec("2020-09-05 14:00:00")).ToI()

	expectedNodeCountTotalWeight := expectedNodeCountWeight1 + expectedNodeCountWeight2 + expectedNodeCountWeight3 + expectedNodeCountWeight4 + expectedNodeCountWeight5

	expectedMetaLiquidityFees := intStr(1 + 2 + 5 + 6)
	expectedMetaBondingEarnings := intStr(3 + 7)
	expectedMetaLiquidityEarnings := intStr(1 + 2 + 5 + 6 + 4 + 8)
	expectedMetaAvgNodeCount := floatStr2Digits(float64(expectedNodeCountTotalWeight) / float64(to-testdb.StrToSec("2020-09-03 00:00:00")))
	assert.Equal(t, epochStr("2020-09-03 00:00:00"), jsonResult.Meta.StartTime)
	assert.Equal(t, epochStr("2020-09-06 00:00:00"), jsonResult.Meta.EndTime)
	assert.Equal(t, expectedMetaLiquidityFees, jsonResult.Meta.LiquidityFees)
	assert.Equal(t, expectedMetaBondingEarnings, jsonResult.Meta.BondingEarnings)
	assert.Equal(t, expectedMetaLiquidityEarnings, jsonResult.Meta.LiquidityEarnings)
	assert.Equal(t, expectedMetaAvgNodeCount, jsonResult.Meta.AvgNodeCount)
	assert.Equal(t, 2, len(jsonResult.Meta.Pools))
	for _, p := range jsonResult.Meta.Pools {
		switch p.Pool {
		case "BNB.BTCB-1DE":
			assert.Equal(t, intStr(1+2+4), p.Earnings)
		case "BNB.BNB":
			assert.Equal(t, intStr(5+6+8), p.Earnings)
		}
	}

	assert.Equal(t, 3, len(jsonResult.Intervals))
	assert.Equal(t, epochStr("2020-09-03 00:00:00"), jsonResult.Intervals[0].StartTime)
	assert.Equal(t, epochStr("2020-09-04 00:00:00"), jsonResult.Intervals[0].EndTime)
	assert.Equal(t, epochStr("2020-09-05 00:00:00"), jsonResult.Intervals[2].StartTime)
	assert.Equal(t, intStr(to.ToI()), jsonResult.Intervals[2].EndTime)

	assert.Equal(t, intStr(1+2), jsonResult.Intervals[0].LiquidityFees)
	assert.Equal(t, "3", jsonResult.Intervals[0].BondingEarnings)
	assert.Equal(t, intStr(1+2+4), jsonResult.Intervals[0].LiquidityEarnings)
	assert.Equal(t, floatStr2Digits(float64(expectedNodeCountWeight1+expectedNodeCountWeight2)/float64(toUnix("2020-09-04 00:00:00")-toUnix("2020-09-03 00:00:00"))), jsonResult.Intervals[0].AvgNodeCount)

	assert.Equal(t, "0", jsonResult.Intervals[1].LiquidityFees)
	assert.Equal(t, "1.00", jsonResult.Intervals[1].AvgNodeCount)

	assert.Equal(t, intStr(5+6), jsonResult.Intervals[2].LiquidityFees)
	assert.Equal(t, "7", jsonResult.Intervals[2].BondingEarnings)
	assert.Equal(t, intStr(5+6+8), jsonResult.Intervals[2].LiquidityEarnings)
	assert.Equal(t, floatStr2Digits(float64(expectedNodeCountWeight4+expectedNodeCountWeight5)/float64(to.ToI()-toUnix("2020-09-05 00:00:00"))), jsonResult.Intervals[2].AvgNodeCount)
}

func toUnix(str string) int64 {
	return testdb.StrToSec(str).ToI()
}

func floatStr2Digits(v float64) string {
	return strconv.FormatFloat(v, 'f', 2, 64)
}
