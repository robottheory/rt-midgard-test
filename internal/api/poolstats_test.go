package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

func TestPoolsStatsDepthAndSwaps(t *testing.T) {
	// The code under test uses default times.
	// All times should be between db.startOfChain and time.Now
	testdb.SetupTestDB(t)
	timeseries.SetLastTimeForTest(testdb.StrToSec("2020-12-20 23:00:00"))
	timeseries.SetDepthsForTest([]timeseries.Depth{{
		Pool: "BNB.BNB", AssetDepth: 1000, RuneDepth: 2000}})

	testdb.MustExec(t, "DELETE FROM swap_events")

	// Swapping BTCB-1DE to 10, fee 2
	testdb.InsertSwapEvent(t, testdb.FakeSwap{
		Pool: "BNB.BNB", FromAsset: "BNB.BNB",
		ToE8: 10 - 2, LiqFeeInRuneE8: 2, TradeSlipBP: 1,
		BlockTimestamp: "2020-12-03 12:00:00"})

	// Swap 30, fee 2
	testdb.InsertSwapEvent(t, testdb.FakeSwap{
		Pool: "BNB.BNB", FromAsset: "BNB.BNB",
		ToE8: 30 - 2, LiqFeeInRuneE8: 2, TradeSlipBP: 1,
		BlockTimestamp: "2020-12-03 13:00:00"})

	{
		body := testdb.CallV1(t,
			"http://localhost:8080/v2/pool/BNB.BNB/stats")

		var result oapigen.PoolStatsResponse
		testdb.MustUnmarshal(t, body, &result)

		assert.Equal(t, "1000", result.AssetDepth)
		assert.Equal(t, "2", result.SwapCount)
		assert.Equal(t, "40", result.ToRuneVolume)
		assert.Equal(t, "4", result.TotalFees)
	}
	{
		body := testdb.CallV1(t,
			"http://localhost:8080/v2/pool/BNB.BNB/stats/legacy")

		var result oapigen.PoolLegacyDetail
		testdb.MustUnmarshal(t, body, &result)

		assert.Equal(t, "1000", result.AssetDepth)
		assert.Equal(t, "2", result.SwappingTxCount)
		assert.Equal(t, "40", result.SellVolume)
		assert.Equal(t, "4", result.PoolFeesTotal)
	}
}
