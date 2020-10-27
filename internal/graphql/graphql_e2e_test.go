package graphql_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/stretchr/testify/assert"
	"gitlab.com/thorchain/midgard/internal/graphql"
	"gitlab.com/thorchain/midgard/internal/graphql/generated"
	"gitlab.com/thorchain/midgard/internal/graphql/model"
	"gitlab.com/thorchain/midgard/internal/timeseries/testdb"
)

// Returns json representation with indentation.
func niceStr(v interface{}) string {
	buf, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic("Unmarshal failed")
	}
	return string(buf)
}

func TestDepthHistoryE2E(t *testing.T) {
	testdb.SetupTestDB(t)
	schema := generated.NewExecutableSchema(generated.Config{Resolvers: &graphql.Resolver{}})
	gqlClient := client.New(handler.NewDefaultServer(schema))
	testdb.MustExec(t, "DELETE FROM block_pool_depths")

	// This will be skipped because we query 01-10 to 02-10
	testdb.InsertBlockPoolDepth(t, "BNB.BTCB-1DE", 25, 1, "2020-01-05 12:00:00")

	testdb.InsertBlockPoolDepth(t, "BNB.BNB", 10, 20, "2020-01-10 12:00:05")
	testdb.InsertBlockPoolDepth(t, "BNB.BNB", 20, 30, "2020-01-10 14:00:00")
	testdb.InsertBlockPoolDepth(t, "BNB.BNB", 2, 5, "2020-01-13 09:00:00")
	testdb.InsertBlockPoolDepth(t, "BNB.BNB", 6, 18, "2020-01-13 10:00:00")

	queryString := fmt.Sprintf(`{
		depthHistory(asset: "BNB.BNB", from: %d, until: %d, interval: DAY) {
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
			first
			last
			runeLast
			runeFirst
			assetLast
			assetFirst
			priceFirst
			priceLast
		  }
		}
		}`, testdb.ToTime("2020-01-10 00:00:00").Unix(), testdb.ToTime("2020-02-10 00:00:00").Unix())

	type Result struct {
		DepthHistory model.PoolDepthHistory
	}
	var actual Result
	gqlClient.MustPost(queryString, &actual)

	expected := Result{model.PoolDepthHistory{
		Meta: &model.PoolDepthHistoryBucket{
			First:      testdb.ToTime("2020-01-10 12:00:05").Unix(),
			Last:       testdb.ToTime("2020-01-13 10:00:00").Unix(),
			RuneFirst:  20,
			RuneLast:   18,
			AssetFirst: 10,
			AssetLast:  6,
			PriceFirst: 2, // 20 / 10
			PriceLast:  3, // 18 / 6
		},
		Intervals: []*model.PoolDepthHistoryBucket{
			{
				First:      testdb.ToTime("2020-01-10 12:00:05").Unix(),
				Last:       testdb.ToTime("2020-01-10 14:00:00").Unix(),
				RuneFirst:  20,
				RuneLast:   30,
				AssetFirst: 10,
				AssetLast:  20,
				PriceFirst: 2,
				PriceLast:  1.5,
			},
			{
				First:      testdb.ToTime("2020-01-13 09:00:00").Unix(),
				Last:       testdb.ToTime("2020-01-13 10:00:00").Unix(),
				RuneFirst:  5,
				RuneLast:   18,
				AssetFirst: 2,
				AssetLast:  6,
				PriceFirst: 2.5,
				PriceLast:  3,
			},
		},
	}}
	assert.Equal(t, expected, actual)
}

// TODO(acsaba): Looks like PriceHistory is just a subset of DepthHistory
//     Let's get rid of it!
func TestPriceHistoryE2E(t *testing.T) {
	testdb.SetupTestDB(t)
	schema := generated.NewExecutableSchema(generated.Config{Resolvers: &graphql.Resolver{}})
	gqlClient := client.New(handler.NewDefaultServer(schema))
	testdb.MustExec(t, "DELETE FROM block_pool_depths")

	// This will be skipped because we query 01-10 to 02-10
	testdb.InsertBlockPoolDepth(t, "BNB.BTCB-1DE", 25, 1, "2020-01-05 12:00:00")

	testdb.InsertBlockPoolDepth(t, "BNB.BNB", 10, 20, "2020-01-10 12:00:05")
	testdb.InsertBlockPoolDepth(t, "BNB.BNB", 20, 30, "2020-01-10 14:00:00")
	testdb.InsertBlockPoolDepth(t, "BNB.BNB", 2, 5, "2020-01-13 09:00:00")
	testdb.InsertBlockPoolDepth(t, "BNB.BNB", 6, 18, "2020-01-13 10:00:00")

	queryString := fmt.Sprintf(`{
		priceHistory(asset: "BNB.BNB", from: %d, until: %d, interval: DAY) {
		  meta {
			first
			last
			priceFirst
			priceLast
		  }
		  intervals {
			first
			last
			priceFirst
			priceLast
		  }
		}
		}`, testdb.ToTime("2020-01-10 00:00:00").Unix(), testdb.ToTime("2020-02-10 00:00:00").Unix())

	type Result struct {
		PriceHistory model.PoolPriceHistory
	}
	var actual Result
	gqlClient.MustPost(queryString, &actual)

	expected := Result{model.PoolPriceHistory{
		Meta: &model.PoolPriceHistoryBucket{
			First:      testdb.ToTime("2020-01-10 12:00:05").Unix(),
			Last:       testdb.ToTime("2020-01-13 10:00:00").Unix(),
			PriceFirst: 2, // 20 / 10
			PriceLast:  3, // 18 / 6
		},
		Intervals: []*model.PoolPriceHistoryBucket{
			{
				First:      testdb.ToTime("2020-01-10 12:00:05").Unix(),
				Last:       testdb.ToTime("2020-01-10 14:00:00").Unix(),
				PriceFirst: 2,
				PriceLast:  1.5,
			},
			{
				First:      testdb.ToTime("2020-01-13 09:00:00").Unix(),
				Last:       testdb.ToTime("2020-01-13 10:00:00").Unix(),
				PriceFirst: 2.5,
				PriceLast:  3,
			},
		},
	}}
	assert.Equal(t, expected, actual)
}
