package stat_test

import (
	"fmt"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/stretchr/testify/assert"
	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/internal/graphql"
	"gitlab.com/thorchain/midgard/internal/graphql/generated"
	"gitlab.com/thorchain/midgard/internal/graphql/model"
)

func TestDepthHistoryE2E(t *testing.T) {
	testdb.SetupTestDB(t)
	schema := generated.NewExecutableSchema(generated.Config{Resolvers: &graphql.Resolver{}})
	gqlClient := client.New(handler.NewDefaultServer(schema))
	testdb.MustExec(t, "DELETE FROM block_pool_depths")

	// This will be skipped because we query 01-10 to 02-10
	testdb.InsertBlockPoolDepth(t, "BNB.BTCB-1DE", 1000, 1, "2020-01-11 12:00:00")

	// This will be the inicial value
	testdb.InsertBlockPoolDepth(t, "BNB.BNB", 30, 3, "2020-01-05 12:00:00")

	testdb.InsertBlockPoolDepth(t, "BNB.BNB", 10, 20, "2020-01-10 12:00:05")
	testdb.InsertBlockPoolDepth(t, "BNB.BNB", 20, 30, "2020-01-10 14:00:00")
	testdb.InsertBlockPoolDepth(t, "BNB.BNB", 2, 5, "2020-01-12 09:00:00")
	testdb.InsertBlockPoolDepth(t, "BNB.BNB", 6, 18, "2020-01-12 10:00:00")

	queryString := fmt.Sprintf(`{
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
	}`, testdb.ToTime("2020-01-10 00:00:00").Unix(), testdb.ToTime("2020-01-14 00:00:00").Unix())

	type Result struct {
		PoolHistory model.PoolHistoryDetails
	}
	var actual Result
	gqlClient.MustPost(queryString, &actual)

	assert.Equal(t, &model.PoolHistoryMeta{
		First:      testdb.ToTime("2020-01-10 00:00:00").Unix(),
		Last:       testdb.ToTime("2020-01-13 00:00:00").Unix(),
		RuneFirst:  3,
		RuneLast:   18,
		AssetFirst: 30,
		AssetLast:  6,
		PriceFirst: 0.1, // 30 / 3
		PriceLast:  3,   // 18 / 6
	}, actual.PoolHistory.Meta)

	assert.Equal(t, 4, len(actual.PoolHistory.Intervals))
	assert.Equal(t, testdb.ToTime("2020-01-10 00:00:00").Unix(), actual.PoolHistory.Intervals[0].Time)
	assert.Equal(t, testdb.ToTime("2020-01-13 00:00:00").Unix(), actual.PoolHistory.Intervals[3].Time)

	jan11 := actual.PoolHistory.Intervals[1]
	assert.Equal(t, int64(30), jan11.Rune)
	assert.Equal(t, int64(20), jan11.Asset)
	assert.Equal(t, 1.5, jan11.Price)

	// gapfill works.
	jan12 := actual.PoolHistory.Intervals[2]
	assert.Equal(t, testdb.ToTime("2020-01-12 00:00:00").Unix(), jan12.Time)
	assert.Equal(t, 1.5, jan12.Price)
}
