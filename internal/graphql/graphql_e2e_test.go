package graphql_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"gitlab.com/thorchain/midgard/internal/timeseries"

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

func TestVolumeHistoryE2E(t *testing.T) {
	testdb.SetupTestDB(t)
	schema := generated.NewExecutableSchema(generated.Config{Resolvers: &graphql.Resolver{}})
	gqlClient := client.New(handler.NewDefaultServer(schema))
	testdb.MustExec(t, "DELETE FROM swap_events")
	testdb.MustExec(t, "DELETE FROM block_pool_depths")

	// Adding entry to fix the exchange rate, 1 BNB = 2 RUNE
	testdb.InsertBlockPoolDepth(t, "BNB.BNB", 1, 2, "2020-09-03 12:00:00")

	// Swapping 10 BNB to rune at exchange rate of 2/1 = 20 RUNE and selling 50 RUNE on 2nd, 3rd and 5th of September
	// Swaps on the 2nd will be skipped
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.BNB", FromAsset: "BNB.BNB", FromE8: 10, LiqFeeInRuneE8: 4, BlockTimestamp: "2020-09-02 13:00:00"})
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.BNB", FromAsset: event.RuneAsset(), FromE8: 50, LiqFeeInRuneE8: 4, BlockTimestamp: "2020-09-02 13:00:00"})

	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.BNB", FromAsset: "BNB.BNB", FromE8: 10, LiqFeeInRuneE8: 4, BlockTimestamp: "2020-09-03 15:00:00"})
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.BNB", FromAsset: event.RuneAsset(), FromE8: 50, LiqFeeInRuneE8: 4, BlockTimestamp: "2020-09-03 16:00:00"})
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.BNB", FromAsset: "BNB.BNB", FromE8: 10, LiqFeeInRuneE8: 4, BlockTimestamp: "2020-09-05 12:00:00"})
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.BNB", FromAsset: event.RuneAsset(), FromE8: 50, LiqFeeInRuneE8: 4, BlockTimestamp: "2020-09-05 12:00:00"})

	// Lower limit is rounded to the first whole period, upper limit is exclusive
	from := testdb.ToTime("2020-09-02 00:00:01").Unix()
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

func TestStakeHistoryE2E(t *testing.T) {
	testdb.SetupTestDB(t)
	schema := generated.NewExecutableSchema(generated.Config{Resolvers: &graphql.Resolver{}})
	gqlClient := client.New(handler.NewDefaultServer(schema))
	testdb.MustExec(t, "DELETE FROM stake_events")

	// first one is skipped
	testdb.InsertStakeEvent(t, testdb.FakeStake{Pool: "BNB.TWT-123", BlockTimestamp: "2020-09-02 08:00:00", RuneE8: 200, AssetE8: 1000, StakeUnits: 50})
	testdb.InsertStakeEvent(t, testdb.FakeStake{Pool: "BNB.TWT-123", BlockTimestamp: "2020-09-03 10:00:00", RuneE8: 1000, AssetE8: 5000, StakeUnits: 200})
	testdb.InsertStakeEvent(t, testdb.FakeStake{Pool: "BNB.TWT-123", BlockTimestamp: "2020-09-05 11:00:00", RuneE8: 3000, AssetE8: 2000, StakeUnits: 100})
	testdb.InsertStakeEvent(t, testdb.FakeStake{Pool: "BNB.TWT-123", BlockTimestamp: "2020-09-05 12:00:00", RuneE8: 1500, AssetE8: 4000, StakeUnits: 300})

	from := testdb.ToTime("2020-09-02 00:00:01").Unix()
	until := testdb.ToTime("2020-09-06 00:00:00").Unix()

	queryString := fmt.Sprintf(`{
		stakeHistory(pool: "BNB.TWT-123", from: %d, until: %d, interval: DAY) {
		  	meta {
				count
				first
				last
				runeVolume
				assetVolume
				units
			}
			intervals {
				time
				count
				runeVolume
				assetVolume
				units
		  }
		}
	}`, from, until)

	type Result struct {
		StakeHistory model.PoolStakeHistory
	}
	var actual Result
	gqlClient.MustPost(queryString, &actual)

	assert.Equal(t, testdb.ToTime("2020-09-05 00:00:00").Unix(), actual.StakeHistory.Meta.Last)
	assert.Equal(t, int64(11000), actual.StakeHistory.Meta.AssetVolume)
	assert.Equal(t, int64(600), actual.StakeHistory.Meta.Units)

	assert.Equal(t, int64(1), actual.StakeHistory.Intervals[0].Count)
	assert.Equal(t, int64(5000), actual.StakeHistory.Intervals[0].AssetVolume)
	assert.Equal(t, int64(1000), actual.StakeHistory.Intervals[0].RuneVolume)
	assert.Equal(t, int64(200), actual.StakeHistory.Intervals[0].Units)

	// gapfill
	assert.Equal(t, testdb.ToTime("2020-09-04 00:00:00").Unix(), actual.StakeHistory.Intervals[1].Time)
	assert.Equal(t, int64(0), actual.StakeHistory.Intervals[1].Count)
	assert.Equal(t, int64(0), actual.StakeHistory.Intervals[1].RuneVolume)

	assert.Equal(t, int64(2), actual.StakeHistory.Intervals[2].Count)
	assert.Equal(t, testdb.ToTime("2020-09-05 00:00:00").Unix(), actual.StakeHistory.Intervals[2].Time)
	assert.Equal(t, int64(6000), actual.StakeHistory.Intervals[2].AssetVolume)
	assert.Equal(t, int64(4500), actual.StakeHistory.Intervals[2].RuneVolume)
	assert.Equal(t, int64(400), actual.StakeHistory.Intervals[2].Units)
}

func TestPoolE2E(t *testing.T) {
	testdb.SetupTestDB(t)
	schema := generated.NewExecutableSchema(generated.Config{Resolvers: &graphql.Resolver{}})
	gqlClient := client.New(handler.NewDefaultServer(schema))
	timeseries.SetLastTimeForTest(testdb.ToTime("2020-09-01 23:00:00"))
	timeseries.SetDepthsForTest("BNB.TWT-123", 30000000000000, 2240582804123679)

	testdb.MustExec(t, "DELETE FROM stake_events")
	testdb.MustExec(t, "DELETE FROM unstake_events")
	testdb.MustExec(t, "DELETE FROM swap_events")
	testdb.MustExec(t, "DELETE FROM block_pool_depths")

	testdb.InsertBlockPoolDepth(t, "BNB.TWT-123", 4, 5, "2020-09-01 00:00:00")
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.TWT-123", FromAsset: "BNB.RUNE", FromE8: 30000, LiqFeeInRuneE8: 39087201297999, BlockTimestamp: "2020-09-01 00:00:00"})
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.TWT-123", FromAsset: "BNB.TWT-123", FromE8: 20000, LiqFeeInRuneE8: 39087201297999, BlockTimestamp: "2020-09-01 13:00:00"})
	testdb.InsertStakeEvent(t, testdb.FakeStake{Pool: "BNB.TWT-123", BlockTimestamp: "2020-09-01 00:00:00", StakeUnits: 80})
	testdb.InsertUnstakeEvent(t, testdb.FakeUnstake{Pool: "BNB.TWT-123", Asset: "BNB.TWT-123", BlockTimestamp: "2020-09-01 00:00:00", StakeUnits: 30})

	queryString := `{
		pool(asset: "BNB.TWT-123") {
			asset
			status
			units
			price
			poolAPY
			volume24h
	    }
	}`

	type Result struct {
		Pool model.Pool
	}
	var actual Result
	gqlClient.MustPost(queryString, &actual)

	assert.Equal(t, "BNB.TWT-123", actual.Pool.Asset)
	// stake - unstake -> 80 - 30 = 50
	assert.Equal(t, int64(50), actual.Pool.Units)
	// runeDepth / assetDepth
	assert.Equal(t, 74.6860934707893, actual.Pool.Price)
	assert.Equal(t, 1.4640203880094016, actual.Pool.PoolApy)
	// 30000 + 5/4 * 20000
	assert.Equal(t, int64(55000), actual.Pool.Volume24h)
}
