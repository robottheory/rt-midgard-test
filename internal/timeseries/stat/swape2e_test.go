package stat_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/stretchr/testify/require"

	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/internal/fetch/record"
	"gitlab.com/thorchain/midgard/internal/graphql"
	"gitlab.com/thorchain/midgard/internal/graphql/generated"
	"gitlab.com/thorchain/midgard/internal/graphql/model"
	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/internal/timeseries/stat"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

func TestSwapHistoryGraphqlFailures(t *testing.T) {
	schema := generated.NewExecutableSchema(generated.Config{Resolvers: &graphql.Resolver{}})
	gqlClient := client.New(handler.NewDefaultServer(schema))

	queryString := `{
		volumeHistory(from: 1599696000, until: 1600560000) {
		  meta {
			first
		  }
		}
	}`

	type GraphqlResult struct {
		Pool model.Pool
	}
	var graphqlResult GraphqlResult

	err := gqlClient.Post(queryString, &graphqlResult)
	if err == nil {
		t.Fatal("Query was expected to fail, but didn't:", queryString)
	}

	queryString = `{
		volumeHistory(from: 1599696000, interval: DAY) {
		  meta {
			first
		  }
		}
	}`
	err = gqlClient.Post(queryString, &graphqlResult)
	if err == nil {
		t.Fatal("Query was expected to fail, but didn't:", queryString)
	}

	queryString = `{
		volumeHistory(until: 1600560000, interval: DAY) {
		  meta {
			first
		  }
		}
	}`
	err = gqlClient.Post(queryString, &graphqlResult)
	if err == nil {
		t.Fatal("Query was expected to fail, but didn't:", queryString)
	}
}

func graphqlSwapsQuery(from, to db.Second) string {
	return fmt.Sprintf(`{
		volumeHistory(from: %d, until: %d, interval: DAY) {
		  meta {
			first
        	last
        	toRune {
			  count
			  feesInRune
			  volumeInRune
        	}
        	toAsset {
          	  count
          	  feesInRune
          	  volumeInRune
        	}
        	combined {
          	  count
          	  feesInRune
          	  volumeInRune
        	}
		  }
		  intervals {
			time
			toRune {
			  count
			  feesInRune
			  volumeInRune
        	}
        	toAsset {
          	  count
          	  feesInRune
          	  volumeInRune
        	}
        	combined {
          	  count
          	  feesInRune
          	  volumeInRune
        	}
		  }
		}
		}`, from, to)
}

// Checks that JSON and GraphQL results are consistent.
func CheckSameSwaps(t *testing.T, jsonResult oapigen.SwapHistoryResponse, gqlQuery string) {
	type Result struct {
		VolumeHistory model.PoolVolumeHistory
	}
	var gqlResult Result

	schema := generated.NewExecutableSchema(generated.Config{Resolvers: &graphql.Resolver{}})
	gqlClient := client.New(handler.NewDefaultServer(schema))
	gqlClient.MustPost(gqlQuery, &gqlResult)

	require.Equal(t, jsonResult.Meta.StartTime, intStr(gqlResult.VolumeHistory.Meta.First))
	require.Equal(t, jsonResult.Meta.EndTime, intStr(gqlResult.VolumeHistory.Meta.Last))
	require.Equal(t, jsonResult.Meta.ToAssetVolume, intStr(gqlResult.VolumeHistory.Meta.ToAsset.VolumeInRune))
	require.Equal(t, jsonResult.Meta.ToRuneVolume, intStr(gqlResult.VolumeHistory.Meta.ToRune.VolumeInRune))
	require.Equal(t, jsonResult.Meta.TotalVolume, intStr(gqlResult.VolumeHistory.Meta.Combined.VolumeInRune))

	require.Equal(t, len(jsonResult.Intervals), len(gqlResult.VolumeHistory.Intervals))
	for i := 0; i < len(jsonResult.Intervals); i++ {
		jr := jsonResult.Intervals[i]
		gr := gqlResult.VolumeHistory.Intervals[i]
		require.Equal(t, jr.StartTime, intStr(gr.Time))
		require.Equal(t, jr.ToAssetVolume, intStr(gr.ToAsset.VolumeInRune))
		require.Equal(t, jr.ToRuneVolume, intStr(gr.ToRune.VolumeInRune))
		require.Equal(t, jr.TotalVolume, intStr(gr.Combined.VolumeInRune))
	}
}

// Testing conversion between different pools and gapfill
func TestSwapsHistoryE2E(t *testing.T) {
	testdb.InitTest(t)

	// Swapping BTCB-1DE to 8 rune (4 to, 4 fee) and selling 15 rune on 3rd of September/
	// total fee=4; average slip=2
	testdb.InsertSwapEvent(t, testdb.FakeSwap{
		Pool: "BNB.BTCB-1DE", FromAsset: "BNB.BTCB-1DE",
		ToE8: 8 - 2, LiqFeeInRuneE8: 2, SwapSlipBP: 1,
		BlockTimestamp: "2020-09-03 12:00:00"})

	testdb.InsertSwapEvent(t, testdb.FakeSwap{
		Pool: "BNB.BTCB-1DE", FromAsset: record.RuneAsset(),
		FromE8: 15, LiqFeeInRuneE8: 4, SwapSlipBP: 3,
		BlockTimestamp: "2020-09-03 12:00:00"})

	// Swapping BNB to 20 RUNE and selling 50 RUNE on 5th of September
	// total fee=13; average slip=3
	testdb.InsertSwapEvent(t, testdb.FakeSwap{
		Pool:      "BNB.BNB",
		FromAsset: "BNB.BNB",
		ToE8:      20 - 5, LiqFeeInRuneE8: 5, SwapSlipBP: 1,
		BlockTimestamp: "2020-09-05 12:00:00"})

	testdb.InsertSwapEvent(t, testdb.FakeSwap{
		Pool: "BNB.BNB", FromAsset: record.RuneAsset(),
		FromE8: 50, LiqFeeInRuneE8: 8, SwapSlipBP: 5,
		BlockTimestamp: "2020-09-05 12:00:00"})

	from := testdb.StrToSec("2020-09-03 00:00:00")
	to := testdb.StrToSec("2020-09-05 23:00:00")
	{
		// Check all pools
		body := testdb.CallJSON(t, fmt.Sprintf(
			"http://localhost:8080/v2/history/swaps?interval=day&from=%d&to=%d", from, to))

		var jsonResult oapigen.SwapHistoryResponse
		testdb.MustUnmarshal(t, body, &jsonResult)

		require.Equal(t, epochStr("2020-09-03 00:00:00"), jsonResult.Meta.StartTime)
		require.Equal(t, epochStr("2020-09-06 00:00:00"), jsonResult.Meta.EndTime)
		require.Equal(t, "28", jsonResult.Meta.ToRuneVolume)
		require.Equal(t, "65", jsonResult.Meta.ToAssetVolume)
		require.Equal(t, intStr(28+65), jsonResult.Meta.TotalVolume)

		require.Equal(t, 3, len(jsonResult.Intervals))
		require.Equal(t, epochStr("2020-09-03 00:00:00"), jsonResult.Intervals[0].StartTime)
		require.Equal(t, epochStr("2020-09-04 00:00:00"), jsonResult.Intervals[0].EndTime)
		require.Equal(t, epochStr("2020-09-05 00:00:00"), jsonResult.Intervals[2].StartTime)

		require.Equal(t, "15", jsonResult.Intervals[0].ToAssetVolume)
		require.Equal(t, "8", jsonResult.Intervals[0].ToRuneVolume)
		require.Equal(t, "23", jsonResult.Intervals[0].TotalVolume)

		require.Equal(t, "0", jsonResult.Intervals[1].TotalVolume)

		require.Equal(t, "50", jsonResult.Intervals[2].ToAssetVolume)
		require.Equal(t, "20", jsonResult.Intervals[2].ToRuneVolume)

		// fees were 2,4 ; 5,8
		require.Equal(t, "4", jsonResult.Intervals[0].ToAssetFees)
		require.Equal(t, "2", jsonResult.Intervals[0].ToRuneFees)
		require.Equal(t, "6", jsonResult.Intervals[0].TotalFees)
		require.Equal(t, "19", jsonResult.Meta.TotalFees)

		require.Equal(t, "3", jsonResult.Intervals[0].ToAssetAverageSlip)
		require.Equal(t, "1", jsonResult.Intervals[0].ToRuneAverageSlip)
		require.Equal(t, "2", jsonResult.Intervals[0].AverageSlip)
		require.Equal(t, "2.5", jsonResult.Meta.AverageSlip)

		CheckSameSwaps(t, jsonResult, graphqlSwapsQuery(from, to))
	}

	{
		// Check only BNB.BNB pool
		body := testdb.CallJSON(t, fmt.Sprintf(
			"http://localhost:8080/v2/history/swaps?interval=day&from=%d&to=%d&pool=BNB.BNB", from, to))

		var jsonResult oapigen.SwapHistoryResponse
		testdb.MustUnmarshal(t, body, &jsonResult)

		require.Equal(t, 3, len(jsonResult.Intervals))
		require.Equal(t, "0", jsonResult.Intervals[0].TotalVolume)
		require.Equal(t, "50", jsonResult.Intervals[2].ToAssetVolume)
		require.Equal(t, "20", jsonResult.Intervals[2].ToRuneVolume)
		// TODO(acsaba): check graphql pool filter
	}

	// TODO(acsaba): check graphql and v1 errors on the same place.
}

func TestSwapsCloseToBoundaryE2E(t *testing.T) {
	testdb.InitTest(t)

	// Swapping to rune 50 in the beginning of the year and 100 at the end of the year
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.BTCB-1DE", FromAsset: "BNB.BTCB-1DE", ToE8: 49, LiqFeeInRuneE8: 1, BlockTimestamp: "2020-01-01 00:01:00"})
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.BTCB-1DE", FromAsset: "BNB.BTCB-1DE", ToE8: 97, LiqFeeInRuneE8: 3, BlockTimestamp: "2020-12-31 23:59:00"})

	from := testdb.StrToSec("2019-01-01 00:00:00")
	to := testdb.StrToSec("2022-01-01 00:00:00")
	body := testdb.CallJSON(t, fmt.Sprintf("http://localhost:8080/v2/history/swaps?interval=year&from=%d&to=%d", from, to))

	var swapHistory oapigen.SwapHistoryResponse
	testdb.MustUnmarshal(t, body, &swapHistory)

	// We check if both first and last minute was attributed to the same year
	require.Equal(t, "150", swapHistory.Meta.ToRuneVolume)
	require.Equal(t, 3, len(swapHistory.Intervals))
	require.Equal(t, epochStr("2020-01-01 00:00:00"), swapHistory.Intervals[1].StartTime)
	require.Equal(t, "150", swapHistory.Intervals[1].ToRuneVolume)
}

func TestMinute5(t *testing.T) {
	testdb.InitTest(t)

	// Swapping 50 and 100 rune
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.BTCB-1DE", FromAsset: "BNB.BTCB-1DE", ToE8: 49, LiqFeeInRuneE8: 1, BlockTimestamp: "2020-01-01 00:01:00"})
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.BTCB-1DE", FromAsset: "BNB.BTCB-1DE", ToE8: 97, LiqFeeInRuneE8: 3, BlockTimestamp: "2020-01-01 00:12:00"})

	from := testdb.StrToSec("2020-01-01 00:00:00")
	to := testdb.StrToSec("2020-01-01 00:15:00")
	body := testdb.CallJSON(t, fmt.Sprintf("http://localhost:8080/v2/history/swaps?interval=5min&from=%d&to=%d", from, to))

	var swapHistory oapigen.SwapHistoryResponse
	testdb.MustUnmarshal(t, body, &swapHistory)

	require.Equal(t, "150", swapHistory.Meta.ToRuneVolume)
	require.Equal(t, 3, len(swapHistory.Intervals))
	require.Equal(t, epochStr("2020-01-01 00:00:00"), swapHistory.Intervals[0].StartTime)
	require.Equal(t, epochStr("2020-01-01 00:05:00"), swapHistory.Intervals[1].StartTime)
	require.Equal(t, epochStr("2020-01-01 00:10:00"), swapHistory.Intervals[2].StartTime)
	require.Equal(t, "50", swapHistory.Intervals[0].ToRuneVolume)
	require.Equal(t, "100", swapHistory.Intervals[2].ToRuneVolume)
}

func TestSwapUsdPrices(t *testing.T) {
	testdb.InitTest(t)
	stat.SetUsdPoolsForTests([]string{"USDA", "USDB"})
	testdb.InsertBlockPoolDepth(t, "USDB", 30, 10, "2019-12-25 12:00:00")
	testdb.InsertBlockPoolDepth(t, "USDA", 200, 100, "2020-01-02 12:00:00")

	testdb.InsertSwapEvent(t, testdb.FakeSwap{
		Pool: "BTC.BTC", FromAsset: "BTC.BTC", ToE8: 2, LiqFeeInRuneE8: 1,
		BlockTimestamp: "2020-01-01 13:00:00"})
	testdb.InsertSwapEvent(t, testdb.FakeSwap{
		Pool: "BTC.BTC", FromAsset: "BTC.BTC", ToE8: 4, LiqFeeInRuneE8: 2,
		BlockTimestamp: "2020-01-03 13:00:00"})

	from := testdb.StrToSec("2020-01-01 00:00:00")
	to := testdb.StrToSec("2020-01-06 00:00:00")
	body := testdb.CallJSON(t, fmt.Sprintf("http://localhost:8080/v2/history/swaps?interval=day&from=%d&to=%d", from, to))

	var swapHistory oapigen.SwapHistoryResponse
	testdb.MustUnmarshal(t, body, &swapHistory)

	require.Equal(t, 5, len(swapHistory.Intervals))
	require.Equal(t, epochStr("2020-01-01 00:00:00"), swapHistory.Intervals[0].StartTime)
	require.Equal(t, "3", swapHistory.Intervals[0].ToRuneVolume)
	require.Equal(t, "3", swapHistory.Intervals[0].RunePriceUSD) // 30 / 10
	require.Equal(t, epochStr("2020-01-02 00:00:00"), swapHistory.Intervals[1].StartTime)
	require.Equal(t, "2", swapHistory.Intervals[1].RunePriceUSD)
	require.Equal(t, epochStr("2020-01-03 00:00:00"), swapHistory.Intervals[2].StartTime)
	require.Equal(t, "2", swapHistory.Intervals[2].RunePriceUSD)
	require.Equal(t, "2", swapHistory.Meta.RunePriceUSD)
}

func TestAverageNaN(t *testing.T) {
	testdb.InitTest(t)

	// No swaps
	from := testdb.StrToSec("2020-01-01 00:00:00")
	to := testdb.StrToSec("2020-01-02 00:00:00")
	body := testdb.CallJSON(t, fmt.Sprintf("http://localhost:8080/v2/history/swaps?interval=day&from=%d&to=%d", from, to))

	var swapHistory oapigen.SwapHistoryResponse
	testdb.MustUnmarshal(t, body, &swapHistory)

	require.Equal(t, "0", swapHistory.Meta.AverageSlip)
}

// Parse string as date and return the unix epoch int value as string.
func epochStr(t string) string {
	return intStr(testdb.StrToSec(t).ToI())
}

func intStr(v int64) string {
	return strconv.FormatInt(v, 10)
}

func TestPoolsStatsLegacyE2E(t *testing.T) {
	// The code under test uses default times.
	// All times should be between db.startOfChain and time.Now
	testdb.InitTest(t)
	timeseries.SetDepthsForTest([]timeseries.Depth{{
		Pool: "BNB.BNB", AssetDepth: 1000, RuneDepth: 2000}})

	testdb.MustExec(t, "DELETE FROM swap_events")

	// Swapping BTCB-1DE to 10, fee 2
	testdb.InsertSwapEvent(t, testdb.FakeSwap{
		Pool: "BNB.BNB", FromAsset: "BNB.BNB",
		ToE8: 10 - 2, LiqFeeInRuneE8: 2, SwapSlipBP: 1,
		BlockTimestamp: "2020-12-03 12:00:00"})

	// Swap 30, fee 2
	testdb.InsertSwapEvent(t, testdb.FakeSwap{
		Pool: "BNB.BNB", FromAsset: "BNB.BNB",
		ToE8: 30 - 2, LiqFeeInRuneE8: 2, SwapSlipBP: 1,
		BlockTimestamp: "2020-12-03 13:00:00"})

	// Check all pools
	body := testdb.CallJSON(t,
		"http://localhost:8080/v2/pool/BNB.BNB/stats/legacy")

	var result oapigen.PoolLegacyResponse
	testdb.MustUnmarshal(t, body, &result)

	require.Equal(t, "1000", result.AssetDepth)
	require.Equal(t, "2000", result.RuneDepth)
	require.Equal(t, "4000", result.PoolDepth)
	require.Equal(t, "2", result.SwappingTxCount)
	require.Equal(t, "20", result.PoolTxAverage)
	require.Equal(t, "4", result.PoolFeesTotal)
}

func TestVolume24h(t *testing.T) {
	testdb.InitTest(t)

	timeseries.SetLastTimeForTest(testdb.StrToSec("2021-01-02 13:00:00"))
	timeseries.SetDepthsForTest([]timeseries.Depth{{
		Pool: "BNB.BNB", AssetDepth: 1000, RuneDepth: 2000}})

	testdb.InsertPoolEvents(t, "BNB.BNB", "Available")
	testdb.InsertStakeEvent(t, testdb.FakeStake{
		Pool: "BNB.BNB",
	})

	// swap 25h ago
	testdb.InsertSwapEvent(t, testdb.FakeSwap{
		Pool: "BNB.BNB", FromAsset: "BNB.BNB",
		ToE8: 10 - 2, LiqFeeInRuneE8: 2, SwapSlipBP: 1,
		BlockTimestamp: "2021-01-01 12:00:00"})

	// swap 22h ago
	testdb.InsertSwapEvent(t, testdb.FakeSwap{
		Pool: "BNB.BNB", FromAsset: "BNB.BNB",
		ToE8: 30 - 2, LiqFeeInRuneE8: 2, SwapSlipBP: 1,
		BlockTimestamp: "2021-01-01 15:00:00"})

	// swap 22h ago
	testdb.InsertSwapEvent(t, testdb.FakeSwap{
		Pool: "BNB.BNB", FromAsset: "RUNE",
		FromE8: 40, LiqFeeInRuneE8: 2, SwapSlipBP: 1,
		BlockTimestamp: "2021-01-01 15:00:00"})

	var pools oapigen.PoolsResponse
	testdb.MustUnmarshal(t, testdb.CallJSON(t,
		"http://localhost:8080/v2/pools"), &pools)
	require.Len(t, pools, 1)
	require.Equal(t, "BNB.BNB", pools[0].Asset)
	require.Equal(t, "70", pools[0].Volume24h)
}
