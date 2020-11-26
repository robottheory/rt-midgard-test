package stat_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/thorchain/midgard/event"
	"gitlab.com/thorchain/midgard/internal/timeseries/testdb"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

// Testing conversion between different pools and gapfill
func TestSwapsHistoryE2E(t *testing.T) {
	testdb.SetupTestDB(t)
	testdb.MustExec(t, "DELETE FROM swap_events")
	testdb.MustExec(t, "DELETE FROM block_pool_depths")

	// Adding two entries to fix the exchange rate, 25 BTCB-1DE = 1 RUNE and 1 BNB = 2 RUNE
	testdb.InsertBlockPoolDepth(t, "BNB.BTCB-1DE", 25, 1, "2020-09-03 12:00:00")
	testdb.InsertBlockPoolDepth(t, "BNB.BNB", 1, 2, "2020-09-05 12:00:00")

	// Swapping 200 BTCB-1DE to rune at exchange rate of 1/25 = 8 RUNE and selling 15 RUNE on 3rd of September
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.BTCB-1DE", FromAsset: "BNB.BTCB-1DE", FromE8: 200, BlockTimestamp: "2020-09-03 12:00:00"})
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.BTCB-1DE", FromAsset: event.RuneAsset(), FromE8: 15, BlockTimestamp: "2020-09-03 12:00:00"})

	// Swapping 10 BNB to rune at exchange rate of 2/1 = 20 RUNE and selling 50 RUNE on 5th of September
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.BNB", FromAsset: "BNB.BNB", FromE8: 10, BlockTimestamp: "2020-09-05 12:00:00"})
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.BNB", FromAsset: event.RuneAsset(), FromE8: 50, BlockTimestamp: "2020-09-05 12:00:00"})

	from := testdb.ToTime("2020-09-02 12:00:00").Unix()
	to := testdb.ToTime("2020-09-05 23:00:00").Unix()
	body := testdb.CallV1(t, fmt.Sprintf("http://localhost:8080/v2/history/swaps?interval=day&from=%d&to=%d", from, to))

	var swapHistory oapigen.SwapHistoryResponse
	testdb.MustUnmarshal(t, body, &swapHistory)

	var expectedIntervals = make(oapigen.SwapHistoryIntervals, 3)
	expectedIntervals[0] = oapigen.SwapHistoryInterval{
		ToRuneVolume:  "8",
		ToAssetVolume: "15",
		Time:          unixStr("2020-09-03 00:00:00"),
		TotalVolume:   "23"}
	expectedIntervals[1] = oapigen.SwapHistoryInterval{
		ToRuneVolume:  "0",
		ToAssetVolume: "0",
		Time:          unixStr("2020-09-04 00:00:00"),
		TotalVolume:   "0"}
	expectedIntervals[2] = oapigen.SwapHistoryInterval{
		ToRuneVolume:  "20",
		ToAssetVolume: "50",
		Time:          unixStr("2020-09-05 00:00:00"),
		TotalVolume:   "70"}

	assert.Equal(t, expectedIntervals, swapHistory.Intervals)
	assert.Equal(t, unixStr("2020-09-03 00:00:00"), swapHistory.Meta.FirstTime)
	assert.Equal(t, unixStr("2020-09-05 00:00:00"), swapHistory.Meta.LastTime)
	assert.Equal(t, "28", swapHistory.Meta.ToRuneVolume)
	assert.Equal(t, "65", swapHistory.Meta.ToAssetVolume)
	assert.Equal(t, intStr(28+65), swapHistory.Meta.TotalVolume)
}

func TestSwapsCloseToBoundaryE2E(t *testing.T) {
	testdb.SetupTestDB(t)
	testdb.MustExec(t, "DELETE FROM swap_events")
	testdb.MustExec(t, "DELETE FROM block_pool_depths")

	testdb.InsertBlockPoolDepth(t, "BNB.BTCB-1DE", 2000, 1000, "2020-01-01 00:00:00")

	// Swapping 300 at price 1/2 = 150 RUNE, in the beginning of the year and at the end of the year
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.BTCB-1DE", FromAsset: "BNB.BTCB-1DE", FromE8: 100, BlockTimestamp: "2020-01-01 00:01:00"})
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.BTCB-1DE", FromAsset: "BNB.BTCB-1DE", FromE8: 200, BlockTimestamp: "2020-12-31 23:59:00"})

	from := testdb.ToTime("2019-01-01 00:00:00").Unix()
	to := testdb.ToTime("2022-01-01 00:00:00").Unix()
	body := testdb.CallV1(t, fmt.Sprintf("http://localhost:8080/v2/history/swaps?interval=year&from=%d&to=%d", from, to))

	var swapHistory oapigen.SwapHistoryResponse
	testdb.MustUnmarshal(t, body, &swapHistory)

	// We check if both first and last minute was attributed to the same year
	assert.Equal(t, "150", swapHistory.Meta.ToRuneVolume)
	assert.Equal(t, 3, len(swapHistory.Intervals))
	assert.Equal(t, unixStr("2020-01-01 00:00:00"), swapHistory.Intervals[1].Time)
	assert.Equal(t, "150", swapHistory.Intervals[1].ToRuneVolume)
}

func TestSwapsYearCountE2E(t *testing.T) {
	testdb.SetupTestDB(t)
	testdb.MustExec(t, "DELETE FROM swap_events")
	testdb.MustExec(t, "DELETE FROM block_pool_depths")

	from := testdb.ToTime("2015-01-01 00:00:00").Unix()
	to := testdb.ToTime("2018-01-01 00:00:00").Unix()
	body := testdb.CallV1(t, fmt.Sprintf("http://localhost:8080/v2/history/swaps?interval=year&from=%d&to=%d", from, to))

	var swapHistory oapigen.SwapHistoryResponse
	testdb.MustUnmarshal(t, body, &swapHistory)

	assert.Equal(t, 3, len(swapHistory.Intervals))
	assert.Equal(t, unixStr("2017-01-01 00:00:00"), swapHistory.Intervals[2].Time)
}

func unixStr(t string) string {
	return intStr(testdb.ToTime(t).Unix())
}

func intStr(v int64) string {
	return strconv.FormatInt(v, 10)
}
