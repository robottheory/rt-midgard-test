package stat_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/stretchr/testify/assert"
	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/internal/graphql"
	"gitlab.com/thorchain/midgard/internal/graphql/generated"
	"gitlab.com/thorchain/midgard/internal/graphql/model"
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

	assert.Equal(t, jsonResult.Meta.StartTime, intStr(gqlResult.PoolHistory.Meta.First))

	assert.Equal(t, len(jsonResult.Intervals), len(gqlResult.PoolHistory.Intervals))
	for i := 0; i < len(jsonResult.Intervals); i++ {
		jr := jsonResult.Intervals[i]
		gr := gqlResult.PoolHistory.Intervals[i]
		assert.Equal(t, jr.StartTime, intStr(gr.Time))
		assert.Equal(t, jr.AssetDepth, intStr(gr.Asset))
		assert.Equal(t, jr.RuneDepth, intStr(gr.Rune))
		assert.Equal(t, jr.AssetPrice, floatStr(gr.Price))
	}
}

func TestDepthHistoryE2E(t *testing.T) {
	testdb.SetupTestDB(t)
	testdb.MustExec(t, "DELETE FROM block_pool_depths")

	// This will be skipped because we query 01-10 to 02-10
	testdb.InsertBlockPoolDepth(t, "BNB.BTCB-1DE", 1000, 1, "2020-01-11 12:00:00")

	// This will be the inicial value
	testdb.InsertBlockPoolDepth(t, "BNB.BNB", 30, 3, "2020-01-05 12:00:00")

	testdb.InsertBlockPoolDepth(t, "BNB.BNB", 10, 20, "2020-01-10 12:00:05")
	testdb.InsertBlockPoolDepth(t, "BNB.BNB", 20, 30, "2020-01-10 14:00:00")
	testdb.InsertBlockPoolDepth(t, "BNB.BNB", 2, 5, "2020-01-12 09:00:00")
	testdb.InsertBlockPoolDepth(t, "BNB.BNB", 6, 18, "2020-01-12 10:00:00")

	from := testdb.StrToSec("2020-01-09 00:00:00")
	to := testdb.StrToSec("2020-01-13 00:00:00")

	body := testdb.CallV1(t, fmt.Sprintf(
		"http://localhost:8080/v2/history/depths/BNB.BNB?interval=day&from=%d&to=%d", from, to))

	var jsonResult oapigen.DepthHistoryResponse
	testdb.MustUnmarshal(t, body, &jsonResult)

	assert.Equal(t, jsonResult.Meta, oapigen.DepthHistoryMeta{
		StartTime: epochStr("2020-01-09 00:00:00"),
		EndTime:   epochStr("2020-01-13 00:00:00"),
	})
	assert.Equal(t, 4, len(jsonResult.Intervals))
	assert.Equal(t, epochStr("2020-01-09 00:00:00"), jsonResult.Intervals[0].StartTime)
	assert.Equal(t, epochStr("2020-01-10 00:00:00"), jsonResult.Intervals[0].EndTime)
	assert.Equal(t, epochStr("2020-01-13 00:00:00"), jsonResult.Intervals[3].EndTime)

	jan11 := jsonResult.Intervals[1]
	assert.Equal(t, "30", jan11.RuneDepth)
	assert.Equal(t, "20", jan11.AssetDepth)
	assert.Equal(t, "1.5", jan11.AssetPrice)

	// gapfill works.
	jan12 := jsonResult.Intervals[2]
	assert.Equal(t, "1.5", jan12.AssetPrice)
	CheckSameDepths(t, jsonResult, graphqlDepthsQuery(from, to))
}

func TestLiquidityUnitsHistoryE2E(t *testing.T) {
	testdb.SetupTestDB(t)
	testdb.MustExec(t, "DELETE FROM stake_events")
	testdb.MustExec(t, "DELETE FROM unstake_events")

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

	body := testdb.CallV1(t, fmt.Sprintf(
		"http://localhost:8080/v2/history/depths/BTC.BTC?interval=day&from=%d&to=%d", from, to))

	var jsonResult oapigen.DepthHistoryResponse
	testdb.MustUnmarshal(t, body, &jsonResult)

	assert.Equal(t, 3, len(jsonResult.Intervals))
	assert.Equal(t, epochStr("2020-01-20 00:00:00"), jsonResult.Intervals[0].EndTime)
	assert.Equal(t, "10", jsonResult.Intervals[0].Units)

	assert.Equal(t, epochStr("2020-01-21 00:00:00"), jsonResult.Intervals[1].EndTime)
	assert.Equal(t, "20", jsonResult.Intervals[1].Units)

	assert.Equal(t, epochStr("2020-01-22 00:00:00"), jsonResult.Intervals[2].EndTime)
	assert.Equal(t, "15", jsonResult.Intervals[2].Units)
}

func floatStr(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
