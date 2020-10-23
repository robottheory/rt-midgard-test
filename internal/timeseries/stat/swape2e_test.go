package stat_test

import (
	"encoding/json"
	"fmt"
	"gitlab.com/thorchain/midgard/event"
	"gitlab.com/thorchain/midgard/internal/timeseries/stat"
	"gitlab.com/thorchain/midgard/internal/timeseries/testdb"
	"reflect"
	"testing"
)

// Testing conversion between different pools and no gapfill yet
func TestTotalVolumeChangesE2E(t *testing.T) {
	testdb.SetupTestDB(t)
	testdb.MustExec(t, "DELETE FROM swap_events")
	testdb.MustExec(t, "DELETE FROM block_pool_depths")

	// Adding two entries to fix the exchange rate, 25 BTCB-1DE = 1 RUNE and 1 BNB = 2 RUNE
	testdb.InsertBlockPoolDepth(t, "BNB.BTCB-1DE", 25, 1, testdb.ToTime("2020-09-03 12:00:00").UnixNano())
	testdb.InsertBlockPoolDepth(t, "BNB.BNB", 1, 2, testdb.ToTime("2020-09-05 12:00:00").UnixNano())

	// Swapping 200 BTCB-1DE to rune at exchange rate of 1/25 = 8 RUNE and selling 15 RUNE on 3rd of September
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.BTCB-1DE", FromAsset: "BNB.BTCB-1DE", FromE8: 200, BlockTimestamp: testdb.ToTime("2020-09-03 12:00:00").UnixNano()})
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.BTCB-1DE", FromAsset: event.RuneAsset(), FromE8: 15, BlockTimestamp: testdb.ToTime("2020-09-03 12:00:00").UnixNano()})

	// Swapping 10 BNB to rune at exchange rate of 2/1 = 20 RUNE and selling 50 RUNE on 5th of September
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.BNB", FromAsset: "BNB.BNB", FromE8: 10, BlockTimestamp: testdb.ToTime("2020-09-05 12:00:00").UnixNano()})
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.BNB", FromAsset: event.RuneAsset(), FromE8: 50, BlockTimestamp: testdb.ToTime("2020-09-05 12:00:00").UnixNano()})

	from := testdb.ToTime("2020-09-03 12:00:00").Unix()
	to := testdb.ToTime("2020-09-05 23:00:00").Unix()
	body := testdb.CallV1(t, fmt.Sprintf("http://localhost:8080/v1/history/total_volume?interval=day&from=%d&to=%d", from, to))

	var swaps []stat.SwapVolumeChanges
	json.Unmarshal(body, &swaps)

	var expected = make([]stat.SwapVolumeChanges, 2)
	expected[0] = stat.SwapVolumeChanges{BuyVolume: "8.000000", SellVolume: "15", Time: testdb.ToTime("2020-09-03 00:00:00").Unix(), TotalVolume: "23.000000"}
	expected[1] = stat.SwapVolumeChanges{BuyVolume: "20.000000", SellVolume: "50", Time: testdb.ToTime("2020-09-05 00:00:00").Unix(), TotalVolume: "70.000000"}

	if !reflect.DeepEqual(swaps, expected) {
		t.Fatalf("/v1/history/total_volume returned unexpected results (actual: %v, expected: %v", swaps, expected)
	}
}
