package graphql_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"gitlab.com/thorchain/midgard/event"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/stretchr/testify/assert"
	"gitlab.com/thorchain/midgard/internal/graphql"
	"gitlab.com/thorchain/midgard/internal/graphql/generated"
	"gitlab.com/thorchain/midgard/internal/graphql/model"
	"gitlab.com/thorchain/midgard/internal/timeseries/testdb"
)

// Returns json representation with indentation.
func NiceStr(v interface{}) string {
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

func TestVolumeHistoryE2E(t *testing.T) {
	testdb.SetupTestDB(t)
	schema := generated.NewExecutableSchema(generated.Config{Resolvers: &graphql.Resolver{}})
	gqlClient := client.New(handler.NewDefaultServer(schema))
	testdb.MustExec(t, "DELETE FROM swap_events")
	testdb.MustExec(t, "DELETE FROM block_pool_depths")

	// Adding entry to fix the exchange rate, 1 BNB = 2 RUNE
	testdb.InsertBlockPoolDepth(t, "BNB.BNB", 1, 2, "2020-09-03 12:00:00")

	// Swapping 10 BNB to rune at exchange rate of 2/1 = 20 RUNE and selling 50 RUNE on 3rd and 5th of September
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.BNB", FromAsset: "BNB.BNB", FromE8: 10, BlockTimestamp: testdb.ToTime("2020-09-03 15:00:00").UnixNano()})
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.BNB", FromAsset: event.RuneAsset(), FromE8: 50, BlockTimestamp: testdb.ToTime("2020-09-03 16:00:00").UnixNano()})
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.BNB", FromAsset: "BNB.BNB", FromE8: 10, BlockTimestamp: testdb.ToTime("2020-09-05 12:00:00").UnixNano()})
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.BNB", FromAsset: event.RuneAsset(), FromE8: 50, BlockTimestamp: testdb.ToTime("2020-09-05 12:00:00").UnixNano()})

	// Lower limit is inclusive, upper limit is exclusive
	from := testdb.ToTime("2020-09-03 00:00:00").Unix()
	until := testdb.ToTime("2020-09-06 00:00:00").Unix()

	queryString := fmt.Sprintf(`{
		volumeHistory(pool: "BNB.BNB", from: %d, until: %d, interval: DAY) {
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
		}`, from, until)

	type Result struct {
		VolumeHistory model.PoolVolumeHistory
	}
	var actual Result
	gqlClient.MustPost(queryString, &actual)

	// Fee is fixed at 4 RUNE per swap
	expected := Result{model.PoolVolumeHistory{
		Meta: &model.PoolVolumeHistoryMeta{
			First: testdb.ToTime("2020-09-03 00:00:00").Unix(),
			Last:  testdb.ToTime("2020-09-05 00:00:00").Unix(),
			ToRune: &model.VolumeStats{
				Count:        2,
				VolumeInRune: 40,
				FeesInRune:   8,
			},
			ToAsset: &model.VolumeStats{
				Count:        2,
				VolumeInRune: 100,
				FeesInRune:   8,
			},
			Combined: &model.VolumeStats{
				Count:        4,
				VolumeInRune: 140,
				FeesInRune:   16,
			},
		},
		Intervals: []*model.PoolVolumeHistoryBucket{
			{
				Time: testdb.ToTime("2020-09-03 00:00:00").Unix(),
				ToRune: &model.VolumeStats{
					Count:        1,
					VolumeInRune: 20,
					FeesInRune:   4,
				},
				ToAsset: &model.VolumeStats{
					Count:        1,
					VolumeInRune: 50,
					FeesInRune:   4,
				},
				Combined: &model.VolumeStats{
					Count:        2,
					VolumeInRune: 70,
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
					FeesInRune:   4,
				},
				ToAsset: &model.VolumeStats{
					Count:        1,
					VolumeInRune: 50,
					FeesInRune:   4,
				},
				Combined: &model.VolumeStats{
					Count:        2,
					VolumeInRune: 70,
					FeesInRune:   8,
				},
			},
		},
	}}
	assert.Equal(t, expected, actual)
}
