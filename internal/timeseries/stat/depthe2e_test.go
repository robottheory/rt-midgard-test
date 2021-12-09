package stat_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/stretchr/testify/require"
	"gitlab.com/thorchain/midgard/config"
	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/internal/graphql"
	"gitlab.com/thorchain/midgard/internal/graphql/generated"
	"gitlab.com/thorchain/midgard/internal/graphql/model"
	"gitlab.com/thorchain/midgard/internal/util"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

func graphqlDepthsQuery(from, to db.Second) string {
	return fmt.Sprintf(`{
		poolHistory(pool: "BNB.BNB", from: %d, until: %d, interval: DAY) {
			meta {
			first
			last
			runeLast
			runeFirst
			assetLast
			assetFirst
			priceFirst
			priceLast
			}
			intervals {
			time
			rune
			asset
			price
			}
		}
		}`, from, to)
}

// Checks that JSON and GraphQL results are consistent.
// TODO(acsaba): check all fields once graphql is corrected.
func CheckSameDepths(t *testing.T, jsonResult oapigen.DepthHistoryResponse, gqlQuery string) {
	schema := generated.NewExecutableSchema(generated.Config{Resolvers: &graphql.Resolver{}})
	gqlClient := client.New(handler.NewDefaultServer(schema))

	type Result struct {
		PoolHistory model.PoolHistoryDetails
	}
	var gqlResult Result
	gqlClient.MustPost(gqlQuery, &gqlResult)

	require.Equal(t, jsonResult.Meta.StartTime, util.IntStr(gqlResult.PoolHistory.Meta.First))

	require.Equal(t, len(jsonResult.Intervals), len(gqlResult.PoolHistory.Intervals))
	for i := 0; i < len(jsonResult.Intervals); i++ {
		jr := jsonResult.Intervals[i]
		gr := gqlResult.PoolHistory.Intervals[i]
		require.Equal(t, jr.StartTime, util.IntStr(gr.Time))
		require.Equal(t, jr.AssetDepth, util.IntStr(gr.Asset))
		require.Equal(t, jr.RuneDepth, util.IntStr(gr.Rune))
		require.Equal(t, jr.AssetPrice, floatStr(gr.Price))
	}
}

func TestDepthHistoryE2E(t *testing.T) {
	testdb.InitTest(t)
	testdb.DeclarePools("BNB.BNB")

	// This will be skipped because we query 01-09 to 01-13
	testdb.InsertBlockPoolDepth(t, "BNB.BNB", 1000, 1, "2020-01-13 12:00:00")

	// This will be the initial value
	testdb.InsertBlockPoolDepth(t, "BNB.BNB", 30, 3, "2020-01-05 12:00:00")

	testdb.InsertBlockPoolDepth(t, "BNB.BNB", 10, 20, "2020-01-10 12:00:05")
	testdb.InsertBlockPoolDepth(t, "BNB.BNB", 20, 30, "2020-01-10 14:00:00")
	testdb.InsertBlockPoolDepth(t, "BNB.BNB", 2, 5, "2020-01-12 09:00:00")
	testdb.InsertBlockPoolDepth(t, "BNB.BNB", 6, 18, "2020-01-12 10:00:00")

	from := testdb.StrToSec("2020-01-09 00:00:00")
	to := testdb.StrToSec("2020-01-13 00:00:00")

	body := testdb.CallJSON(t, fmt.Sprintf(
		"http://localhost:8080/v2/history/depths/BNB.BNB?interval=day&from=%d&to=%d", from, to))

	var jsonResult oapigen.DepthHistoryResponse
	testdb.MustUnmarshal(t, body, &jsonResult)

	require.Equal(t, oapigen.DepthHistoryMeta{
		StartTime: epochStr("2020-01-09 00:00:00"),
		EndTime:   epochStr("2020-01-13 00:00:00"),
	}, jsonResult.Meta)
	require.Equal(t, 4, len(jsonResult.Intervals))
	require.Equal(t, epochStr("2020-01-09 00:00:00"), jsonResult.Intervals[0].StartTime)
	require.Equal(t, epochStr("2020-01-10 00:00:00"), jsonResult.Intervals[0].EndTime)
	require.Equal(t, epochStr("2020-01-13 00:00:00"), jsonResult.Intervals[3].EndTime)

	// initial value correct
	jan9 := jsonResult.Intervals[0]
	require.Equal(t, "3", jan9.RuneDepth)

	jan10 := jsonResult.Intervals[1]
	require.Equal(t, "30", jan10.RuneDepth)
	require.Equal(t, "20", jan10.AssetDepth)
	require.Equal(t, "1.5", jan10.AssetPrice)

	// gapfill works.
	jan11 := jsonResult.Intervals[2]
	require.Equal(t, "1.5", jan11.AssetPrice)
	CheckSameDepths(t, jsonResult, graphqlDepthsQuery(from, to))
}

func TestUSDHistoryE2E(t *testing.T) {
	testdb.InitTest(t)
	testdb.DeclarePools("BNB.BNB", "USDA", "USDB")

	config.Global.UsdPools = []string{"USDA", "USDB"}

	// assetPrice: 2, runePriceUSD: 2
	testdb.InsertBlockPoolDepth(t, "BNB.BNB", 1, 2, "2020-01-05 12:00:00")
	testdb.InsertBlockPoolDepth(t, "USDA", 200, 100, "2020-01-05 12:00:00")
	testdb.InsertBlockPoolDepth(t, "USDB", 30, 10, "2020-01-05 12:00:00")

	// runePriceUSD 3
	testdb.InsertBlockPoolDepth(t, "USDB", 3000, 1000, "2020-01-10 12:00:05")

	// runePriceUSD 2, back to USDA
	testdb.InsertBlockPoolDepth(t, "USDB", 10, 10, "2020-01-11 12:00:05")

	// assetPrice: 10
	testdb.InsertBlockPoolDepth(t, "BNB.BNB", 1, 10, "2020-01-13 12:00:00")

	from := testdb.StrToSec("2020-01-09 00:00:00")
	to := testdb.StrToSec("2020-01-14 00:00:00")

	body := testdb.CallJSON(t, fmt.Sprintf(
		"http://localhost:8080/v2/history/depths/BNB.BNB?interval=day&from=%d&to=%d", from, to))

	var jsonResult oapigen.DepthHistoryResponse
	testdb.MustUnmarshal(t, body, &jsonResult)

	require.Equal(t, 5, len(jsonResult.Intervals))
	require.Equal(t, epochStr("2020-01-09 00:00:00"), jsonResult.Intervals[0].StartTime)

	require.Equal(t, "2", jsonResult.Intervals[0].AssetPrice)

	require.Equal(t, "4", jsonResult.Intervals[0].AssetPriceUSD)
	require.Equal(t, "6", jsonResult.Intervals[1].AssetPriceUSD)
	require.Equal(t, "4", jsonResult.Intervals[2].AssetPriceUSD)
	require.Equal(t, "4", jsonResult.Intervals[3].AssetPriceUSD)
	require.Equal(t, "20", jsonResult.Intervals[4].AssetPriceUSD)
}

func TestLiquidityUnitsHistoryE2E(t *testing.T) {
	testdb.InitTest(t)
	testdb.DeclarePools("BTC.BTC", "BNB.BNB")

	testdb.InsertStakeEvent(t, testdb.FakeStake{
		Pool:           "BTC.BTC",
		StakeUnits:     10,
		BlockTimestamp: "2020-01-10 12:00:00",
	})

	testdb.InsertStakeEvent(t, testdb.FakeStake{
		Pool:           "BTC.BTC",
		StakeUnits:     10, // total 20
		BlockTimestamp: "2020-01-20 12:00:00",
	})

	testdb.InsertUnstakeEvent(t, testdb.FakeUnstake{
		Pool:           "BTC.BTC",
		StakeUnits:     5, // total 15
		BlockTimestamp: "2020-01-21 12:00:00",
	})

	// This will be skipped because it's a different pool
	testdb.InsertStakeEvent(t, testdb.FakeStake{
		Pool:           "BNB.BNB",
		StakeUnits:     1000,
		BlockTimestamp: "2020-01-20 12:00:00",
	})

	from := testdb.StrToSec("2020-01-19 00:00:00")
	to := testdb.StrToSec("2020-01-22 00:00:00")

	body := testdb.CallJSON(t, fmt.Sprintf(
		"http://localhost:8080/v2/history/depths/BTC.BTC?interval=day&from=%d&to=%d", from, to))

	var jsonResult oapigen.DepthHistoryResponse
	testdb.MustUnmarshal(t, body, &jsonResult)

	require.Equal(t, 3, len(jsonResult.Intervals))
	require.Equal(t, epochStr("2020-01-20 00:00:00"), jsonResult.Intervals[0].EndTime)
	require.Equal(t, "10", jsonResult.Intervals[0].LiquidityUnits)

	require.Equal(t, epochStr("2020-01-21 00:00:00"), jsonResult.Intervals[1].EndTime)
	require.Equal(t, "20", jsonResult.Intervals[1].LiquidityUnits)

	require.Equal(t, epochStr("2020-01-22 00:00:00"), jsonResult.Intervals[2].EndTime)
	require.Equal(t, "15", jsonResult.Intervals[2].LiquidityUnits)
}

func TestDepthAggregateE2E(t *testing.T) {
	testdb.InitTest(t)
	testdb.DeclarePools("A.A", "B.B")

	// This is to test that "sewing" together data from the continuous aggregate and
	// the live table works correctly.
	//
	// We have two relevant time buckets for this test:
	// - one ending on 2020-01-02 00:00:00 - data here should be coming from the continuous aggregate
	// - one starting on 2020-01-02 00:00:00 - data here should be coming only from the live table

	testdb.InsertBlockPoolDepth(t, "A.A", 1, 30, "2020-01-01 23:57:00")
	testdb.InsertBlockPoolDepth(t, "A.A", 1, 10, "2020-01-02 00:02:00")
	testdb.InsertBlockPoolDepth(t, "A.A", 1, 20, "2020-01-02 00:03:00")

	testdb.InsertBlockPoolDepth(t, "B.B", 1, 10, "2020-01-01 23:57:00")
	testdb.InsertBlockPoolDepth(t, "B.B", 1, 20, "2020-01-02 00:03:00")

	db.RefreshAggregatesForTests()

	to := testdb.StrToSec("2020-01-02 00:02:30")
	var jsonResult oapigen.DepthHistoryResponse

	body := testdb.CallJSON(t, fmt.Sprintf(
		"http://localhost:8080/v2/history/depths/A.A?&to=%d", to))
	testdb.MustUnmarshal(t, body, &jsonResult)

	require.Equal(t, "10", jsonResult.Intervals[0].RuneDepth)

	body = testdb.CallJSON(t, fmt.Sprintf(
		"http://localhost:8080/v2/history/depths/B.B?&to=%d", to))
	testdb.MustUnmarshal(t, body, &jsonResult)

	require.Equal(t, "10", jsonResult.Intervals[0].RuneDepth)
}

func floatStr(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
