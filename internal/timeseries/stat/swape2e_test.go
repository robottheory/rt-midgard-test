package stat_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/stretchr/testify/assert"

	"gitlab.com/thorchain/midgard/event"
	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/internal/graphql"
	"gitlab.com/thorchain/midgard/internal/graphql/generated"
	"gitlab.com/thorchain/midgard/internal/graphql/model"
	"gitlab.com/thorchain/midgard/internal/timeseries"
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

	assert.Equal(t, jsonResult.Meta.StartTime, intStr(gqlResult.VolumeHistory.Meta.First))
	assert.Equal(t, jsonResult.Meta.EndTime, intStr(gqlResult.VolumeHistory.Meta.Last))
	assert.Equal(t, jsonResult.Meta.ToAssetVolume, intStr(gqlResult.VolumeHistory.Meta.ToAsset.VolumeInRune))
	assert.Equal(t, jsonResult.Meta.ToRuneVolume, intStr(gqlResult.VolumeHistory.Meta.ToRune.VolumeInRune))
	assert.Equal(t, jsonResult.Meta.TotalVolume, intStr(gqlResult.VolumeHistory.Meta.Combined.VolumeInRune))

	assert.Equal(t, len(jsonResult.Intervals), len(gqlResult.VolumeHistory.Intervals))
	for i := 0; i < len(jsonResult.Intervals); i++ {
		jr := jsonResult.Intervals[i]
		gr := gqlResult.VolumeHistory.Intervals[i]
		assert.Equal(t, jr.StartTime, intStr(gr.Time))
		assert.Equal(t, jr.ToAssetVolume, intStr(gr.ToAsset.VolumeInRune))
		assert.Equal(t, jr.ToRuneVolume, intStr(gr.ToRune.VolumeInRune))
		assert.Equal(t, jr.TotalVolume, intStr(gr.Combined.VolumeInRune))
	}
}

// Testing conversion between different pools and gapfill
func TestSwapsHistoryE2E(t *testing.T) {
	testdb.SetupTestDB(t)

	testdb.MustExec(t, "DELETE FROM swap_events")

	// Swapping BTCB-1DE to 8 rune (4 to, 4 fee) and selling 15 rune on 3rd of September/
	// total fee=4; average slip=2
	testdb.InsertSwapEvent(t, testdb.FakeSwap{
		Pool: "BNB.BTCB-1DE", FromAsset: "BNB.BTCB-1DE",
		ToE8: 8 - 2, LiqFeeInRuneE8: 2, TradeSlipBP: 1,
		BlockTimestamp: "2020-09-03 12:00:00"})

	testdb.InsertSwapEvent(t, testdb.FakeSwap{
		Pool: "BNB.BTCB-1DE", FromAsset: event.RuneAsset(),
		FromE8: 15, LiqFeeInRuneE8: 4, TradeSlipBP: 3,
		BlockTimestamp: "2020-09-03 12:00:00"})

	// Swapping BNB to 20 RUNE and selling 50 RUNE on 5th of September
	// total fee=13; average slip=3
	testdb.InsertSwapEvent(t, testdb.FakeSwap{
		Pool:      "BNB.BNB",
		FromAsset: "BNB.BNB",
		ToE8:      20 - 5, LiqFeeInRuneE8: 5, TradeSlipBP: 1,
		BlockTimestamp: "2020-09-05 12:00:00"})

	testdb.InsertSwapEvent(t, testdb.FakeSwap{
		Pool: "BNB.BNB", FromAsset: event.RuneAsset(),
		FromE8: 50, LiqFeeInRuneE8: 8, TradeSlipBP: 5,
		BlockTimestamp: "2020-09-05 12:00:00"})

	from := testdb.StrToSec("2020-09-03 00:00:00")
	to := testdb.StrToSec("2020-09-05 23:00:00")
	{
		// Check all pools
		body := testdb.CallV1(t, fmt.Sprintf(
			"http://localhost:8080/v2/history/swaps?interval=day&from=%d&to=%d", from, to))

		var jsonResult oapigen.SwapHistoryResponse
		testdb.MustUnmarshal(t, body, &jsonResult)

		assert.Equal(t, epochStr("2020-09-03 00:00:00"), jsonResult.Meta.StartTime)
		assert.Equal(t, epochStr("2020-09-06 00:00:00"), jsonResult.Meta.EndTime)
		assert.Equal(t, "28", jsonResult.Meta.ToRuneVolume)
		assert.Equal(t, "65", jsonResult.Meta.ToAssetVolume)
		assert.Equal(t, intStr(28+65), jsonResult.Meta.TotalVolume)

		assert.Equal(t, 3, len(jsonResult.Intervals))
		assert.Equal(t, epochStr("2020-09-03 00:00:00"), jsonResult.Intervals[0].StartTime)
		assert.Equal(t, epochStr("2020-09-04 00:00:00"), jsonResult.Intervals[0].EndTime)
		assert.Equal(t, epochStr("2020-09-05 00:00:00"), jsonResult.Intervals[2].StartTime)

		assert.Equal(t, "15", jsonResult.Intervals[0].ToAssetVolume)
		assert.Equal(t, "8", jsonResult.Intervals[0].ToRuneVolume)
		assert.Equal(t, "23", jsonResult.Intervals[0].TotalVolume)

		assert.Equal(t, "0", jsonResult.Intervals[1].TotalVolume)

		assert.Equal(t, "50", jsonResult.Intervals[2].ToAssetVolume)
		assert.Equal(t, "20", jsonResult.Intervals[2].ToRuneVolume)

		// fees were 2,4 ; 5,8
		assert.Equal(t, "6", jsonResult.Intervals[0].TotalFees)
		assert.Equal(t, "19", jsonResult.Meta.TotalFees)

		//
		assert.Equal(t, "2", jsonResult.Intervals[0].AverageSlip)
		assert.Equal(t, "2.5", jsonResult.Meta.AverageSlip)

		CheckSameSwaps(t, jsonResult, graphqlSwapsQuery(from, to))
	}

	{
		// Check only BNB.BNB pool
		body := testdb.CallV1(t, fmt.Sprintf(
			"http://localhost:8080/v2/history/swaps?interval=day&from=%d&to=%d&pool=BNB.BNB", from, to))

		var jsonResult oapigen.SwapHistoryResponse
		testdb.MustUnmarshal(t, body, &jsonResult)

		assert.Equal(t, 3, len(jsonResult.Intervals))
		assert.Equal(t, "0", jsonResult.Intervals[0].TotalVolume)
		assert.Equal(t, "50", jsonResult.Intervals[2].ToAssetVolume)
		assert.Equal(t, "20", jsonResult.Intervals[2].ToRuneVolume)
		// TODO(acsaba): check graphql pool filter
	}

	// TODO(acsaba): check graphql and v1 errors on the same place.
}

func TestSwapsCloseToBoundaryE2E(t *testing.T) {
	testdb.SetupTestDB(t)
	testdb.MustExec(t, "DELETE FROM swap_events")

	// Swapping to rune 50 in the beginning of the year and 100 at the end of the year
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.BTCB-1DE", FromAsset: "BNB.BTCB-1DE", ToE8: 49, LiqFeeInRuneE8: 1, BlockTimestamp: "2020-01-01 00:01:00"})
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.BTCB-1DE", FromAsset: "BNB.BTCB-1DE", ToE8: 97, LiqFeeInRuneE8: 3, BlockTimestamp: "2020-12-31 23:59:00"})

	from := testdb.StrToSec("2019-01-01 00:00:00")
	to := testdb.StrToSec("2022-01-01 00:00:00")
	body := testdb.CallV1(t, fmt.Sprintf("http://localhost:8080/v2/history/swaps?interval=year&from=%d&to=%d", from, to))

	var swapHistory oapigen.SwapHistoryResponse
	testdb.MustUnmarshal(t, body, &swapHistory)

	// We check if both first and last minute was attributed to the same year
	assert.Equal(t, "150", swapHistory.Meta.ToRuneVolume)
	assert.Equal(t, 3, len(swapHistory.Intervals))
	assert.Equal(t, epochStr("2020-01-01 00:00:00"), swapHistory.Intervals[1].StartTime)
	assert.Equal(t, "150", swapHistory.Intervals[1].ToRuneVolume)
}

func TestMinute5(t *testing.T) {
	testdb.SetupTestDB(t)
	testdb.MustExec(t, "DELETE FROM swap_events")

	// Swapping 50 and 100 rune
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.BTCB-1DE", FromAsset: "BNB.BTCB-1DE", ToE8: 49, LiqFeeInRuneE8: 1, BlockTimestamp: "2020-01-01 00:01:00"})
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.BTCB-1DE", FromAsset: "BNB.BTCB-1DE", ToE8: 97, LiqFeeInRuneE8: 3, BlockTimestamp: "2020-01-01 00:12:00"})

	from := testdb.StrToSec("2020-01-01 00:00:00")
	to := testdb.StrToSec("2020-01-01 00:15:00")
	body := testdb.CallV1(t, fmt.Sprintf("http://localhost:8080/v2/history/swaps?interval=5min&from=%d&to=%d", from, to))

	var swapHistory oapigen.SwapHistoryResponse
	testdb.MustUnmarshal(t, body, &swapHistory)

	assert.Equal(t, "150", swapHistory.Meta.ToRuneVolume)
	assert.Equal(t, 3, len(swapHistory.Intervals))
	assert.Equal(t, epochStr("2020-01-01 00:00:00"), swapHistory.Intervals[0].StartTime)
	assert.Equal(t, epochStr("2020-01-01 00:05:00"), swapHistory.Intervals[1].StartTime)
	assert.Equal(t, epochStr("2020-01-01 00:10:00"), swapHistory.Intervals[2].StartTime)
	assert.Equal(t, "50", swapHistory.Intervals[0].ToRuneVolume)
	assert.Equal(t, "100", swapHistory.Intervals[2].ToRuneVolume)
}

func TestAverageNaN(t *testing.T) {
	testdb.SetupTestDB(t)
	testdb.MustExec(t, "DELETE FROM swap_events")
	// No swaps

	from := testdb.StrToSec("2020-01-01 00:00:00")
	to := testdb.StrToSec("2020-01-02 00:00:00")
	body := testdb.CallV1(t, fmt.Sprintf("http://localhost:8080/v2/history/swaps?interval=day&from=%d&to=%d", from, to))

	var swapHistory oapigen.SwapHistoryResponse
	testdb.MustUnmarshal(t, body, &swapHistory)

	assert.Equal(t, "0", swapHistory.Meta.AverageSlip)
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

	// Check all pools
	body := testdb.CallV1(t,
		"http://localhost:8080/v2/pool/BNB.BNB/stats/legacy")

	var result oapigen.PoolLegacyResponse
	testdb.MustUnmarshal(t, body, &result)

	assert.Equal(t, "1000", result.AssetDepth)
	assert.Equal(t, "2000", result.RuneDepth)
	assert.Equal(t, "4000", result.PoolDepth)
	assert.Equal(t, "2", result.SwappingTxCount)
	assert.Equal(t, "20", result.PoolTxAverage)
	assert.Equal(t, "4", result.PoolFeesTotal)
}
