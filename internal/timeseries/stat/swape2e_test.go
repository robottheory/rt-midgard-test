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
func TestTotalVolumeChangesE2E(t *testing.T) {
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

	type volumeItem oapigen.SwapHistoryInterval
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

	assert.Equal(t, swapHistory.Intervals, expectedIntervals)
	assert.Equal(t, swapHistory.Meta.FirstTime, unixStr("2020-09-03 00:00:00"))
	assert.Equal(t, swapHistory.Meta.LastTime, unixStr("2020-09-05 00:00:00"))
	assert.Equal(t, swapHistory.Meta.ToRuneVolume, "28")
	assert.Equal(t, swapHistory.Meta.ToAssetVolume, "65")
	assert.Equal(t, swapHistory.Meta.TotalVolume, intStr(28+65))
}

func unixStr(t string) string {
	return intStr(testdb.ToTime(t).Unix())
}

func intStr(v int64) string {
	return strconv.FormatInt(v, 10)
}
