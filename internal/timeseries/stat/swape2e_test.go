package stat_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"gitlab.com/thorchain/midgard/internal/graphql"
	"gitlab.com/thorchain/midgard/internal/graphql/generated"
	"gitlab.com/thorchain/midgard/internal/graphql/model"

	"github.com/stretchr/testify/assert"
	"gitlab.com/thorchain/midgard/event"
	"gitlab.com/thorchain/midgard/internal/timeseries/testdb"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

func callSwapHistoryGraphqlFail(t *testing.T, gqlClient *client.Client) {
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

// Testing conversion between different pools and gapfill
func TestSwapsHistoryE2E(t *testing.T) {
	testdb.SetupTestDB(t)
	schema := generated.NewExecutableSchema(generated.Config{Resolvers: &graphql.Resolver{}})
	gqlClient := client.New(handler.NewDefaultServer(schema))

	testdb.MustExec(t, "DELETE FROM swap_events")
	testdb.MustExec(t, "DELETE FROM block_pool_depths")

	// Adding two entries to fix the exchange rate, 25 BTCB-1DE = 1 RUNE and 1 BNB = 2 RUNE
	testdb.InsertBlockPoolDepth(t, "BNB.BTCB-1DE", 25, 1, "2020-09-03 12:00:00")
	testdb.InsertBlockPoolDepth(t, "BNB.BNB", 1, 2, "2020-09-05 12:00:00")

	// Swapping 200 BTCB-1DE to rune at exchange rate of 1/25 = 8 RUNE and selling 15 RUNE on 3rd of September
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.BTCB-1DE", FromAsset: "BNB.BTCB-1DE", ToE8: 8 - 4, LiqFeeInRuneE8: 4, BlockTimestamp: "2020-09-03 12:00:00"})
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.BTCB-1DE", FromAsset: event.RuneAsset(), FromE8: 15, LiqFeeInRuneE8: 4, BlockTimestamp: "2020-09-03 12:00:00"})

	// Swapping 10 BNB to rune at exchange rate of 2/1 = 20 RUNE and selling 50 RUNE on 5th of September
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.BNB", FromAsset: "BNB.BNB", ToE8: 20 - 4, LiqFeeInRuneE8: 4, BlockTimestamp: "2020-09-05 12:00:00"})
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.BNB", FromAsset: event.RuneAsset(), FromE8: 50, LiqFeeInRuneE8: 4, BlockTimestamp: "2020-09-05 12:00:00"})

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

	expectedGraphqlIntervals := []*model.PoolVolumeHistoryBucket{
		{
			Time: testdb.ToTime("2020-09-03 00:00:00").Unix(),
			ToRune: &model.VolumeStats{
				Count:        1,
				VolumeInRune: 8,
				FeesInRune:   0,
			},
			ToAsset: &model.VolumeStats{
				Count:        1,
				VolumeInRune: 15,
				FeesInRune:   0,
			},
			Combined: &model.VolumeStats{
				Count:        2,
				VolumeInRune: 23,
				FeesInRune:   8,
			},
		},
		{
			Time: testdb.ToTime("2020-09-04 00:00:00").Unix(),
			ToRune: &model.VolumeStats{
				Count:        0,
				VolumeInRune: 0,
				FeesInRune:   0,
			},
			ToAsset: &model.VolumeStats{
				Count:        0,
				VolumeInRune: 0,
				FeesInRune:   0,
			},
			Combined: &model.VolumeStats{
				Count:        0,
				VolumeInRune: 0,
				FeesInRune:   0,
			},
		},
		{
			Time: testdb.ToTime("2020-09-05 00:00:00").Unix(),
			ToRune: &model.VolumeStats{
				Count:        1,
				VolumeInRune: 20,
				FeesInRune:   0,
			},
			ToAsset: &model.VolumeStats{
				Count:        1,
				VolumeInRune: 50,
				FeesInRune:   0,
			},
			Combined: &model.VolumeStats{
				Count:        2,
				VolumeInRune: 70,
				FeesInRune:   8,
			},
		}}

	queryString := fmt.Sprintf(`{
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

	type GraphqlResult struct {
		VolumeHistory model.PoolVolumeHistory
	}
	var graphqlResult GraphqlResult
	gqlClient.MustPost(queryString, &graphqlResult)

	assert.Equal(t, expectedIntervals, swapHistory.Intervals)
	assert.Equal(t, expectedGraphqlIntervals, graphqlResult.VolumeHistory.Intervals)
	assert.Equal(t, unixStr("2020-09-03 00:00:00"), swapHistory.Meta.FirstTime)
	assert.Equal(t, testdb.ToTime("2020-09-03 00:00:00").Unix(), graphqlResult.VolumeHistory.Meta.First)
	assert.Equal(t, unixStr("2020-09-05 00:00:00"), swapHistory.Meta.LastTime)
	assert.Equal(t, testdb.ToTime("2020-09-05 00:00:00").Unix(), graphqlResult.VolumeHistory.Meta.Last)
	assert.Equal(t, "28", swapHistory.Meta.ToRuneVolume)
	assert.Equal(t, int64(28), graphqlResult.VolumeHistory.Meta.ToRune.VolumeInRune)
	assert.Equal(t, "65", swapHistory.Meta.ToAssetVolume)
	assert.Equal(t, int64(65), graphqlResult.VolumeHistory.Meta.ToAsset.VolumeInRune)
	assert.Equal(t, intStr(28+65), swapHistory.Meta.TotalVolume)
	assert.Equal(t, int64(28+65), graphqlResult.VolumeHistory.Meta.Combined.VolumeInRune)

	// Check for failure
	testdb.CallV1Fail(t, fmt.Sprintf("http://localhost:8080/v2/history/swaps?interval=year&from=%d", from))
	testdb.CallV1Fail(t, fmt.Sprintf("http://localhost:8080/v2/history/swaps?interval=year&to=%d", to))
	testdb.CallV1Fail(t, fmt.Sprintf("http://localhost:8080/v2/history/swaps?from=%d&to=%d", from, to))
	callSwapHistoryGraphqlFail(t, gqlClient)
}

func TestSwapsCloseToBoundaryE2E(t *testing.T) {
	testdb.SetupTestDB(t)
	testdb.MustExec(t, "DELETE FROM swap_events")

	// Swapping to rune 50 in the beginning of the year and 100 at the end of the year
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.BTCB-1DE", FromAsset: "BNB.BTCB-1DE", ToE8: 49, LiqFeeInRuneE8: 1, BlockTimestamp: "2020-01-01 00:01:00"})
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.BTCB-1DE", FromAsset: "BNB.BTCB-1DE", ToE8: 97, LiqFeeInRuneE8: 3, BlockTimestamp: "2020-12-31 23:59:00"})

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

func TestMinute5(t *testing.T) {
	testdb.SetupTestDB(t)
	testdb.MustExec(t, "DELETE FROM swap_events")

	// Swapping 50 and 100 rune
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.BTCB-1DE", FromAsset: "BNB.BTCB-1DE", ToE8: 49, LiqFeeInRuneE8: 1, BlockTimestamp: "2020-01-01 00:01:00"})
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.BTCB-1DE", FromAsset: "BNB.BTCB-1DE", ToE8: 97, LiqFeeInRuneE8: 3, BlockTimestamp: "2020-01-01 00:12:00"})

	from := testdb.ToTime("2020-01-01 00:00:00").Unix()
	to := testdb.ToTime("2020-01-01 00:15:00").Unix()
	body := testdb.CallV1(t, fmt.Sprintf("http://localhost:8080/v2/history/swaps?interval=5min&from=%d&to=%d", from, to))

	var swapHistory oapigen.SwapHistoryResponse
	testdb.MustUnmarshal(t, body, &swapHistory)

	assert.Equal(t, "150", swapHistory.Meta.ToRuneVolume)
	assert.Equal(t, 3, len(swapHistory.Intervals))
	assert.Equal(t, unixStr("2020-01-01 00:00:00"), swapHistory.Intervals[0].Time)
	assert.Equal(t, unixStr("2020-01-01 00:05:00"), swapHistory.Intervals[1].Time)
	assert.Equal(t, unixStr("2020-01-01 00:10:00"), swapHistory.Intervals[2].Time)
	assert.Equal(t, "50", swapHistory.Intervals[0].ToRuneVolume)
	assert.Equal(t, "100", swapHistory.Intervals[2].ToRuneVolume)
}

func unixStr(t string) string {
	return intStr(testdb.ToTime(t).Unix())
}

func intStr(v int64) string {
	return strconv.FormatInt(v, 10)
}
