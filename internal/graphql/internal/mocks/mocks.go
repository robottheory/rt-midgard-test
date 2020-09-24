package mocks

import (
	"context"
	"testing"
	"time"

	"gitlab.com/thorchain/midgard/internal/graphql/model"
	"gitlab.com/thorchain/midgard/internal/timeseries/stat"
)

var T *testing.T

type ExpectedResponse struct {
	Pool         model.Pool
	SwapHistory  model.PoolSwapHistory
	StakeHistory model.PoolStakeHistory
}

type Pool struct {
	Asset          string
	Status         string
	Ae8pp          int64
	Re8pp          int64
	Price          float64
	DepthTimestamp string

	StakeTxCount         int64
	StakeAssetE8Total    int64
	StakeRuneE8Total     int64
	StakeStakeUnitsTotal int64
	StakeFirst           string
	StakeLast            string

	UnstakeTxCount          int64
	UnstakeAssetE8Total     int64
	UnstakeRuneE8Total      int64
	UnstakeStakeUnitsTotal  int64
	UnstakeBasisPointsTotal int64

	SwapsFromRuneBucket []stat.PoolSwaps
	SwapsToRuneBucket   []stat.PoolSwaps
	StakeHistory        []stat.PoolStakes

	Expected ExpectedResponse
}

type Data struct {
	Pools     []Pool
	Timestamp string
}

func (d Data) Pool(asset string) *Pool {
	var p Pool
	for _, item := range TestData.Pools {
		if item.Asset == asset {
			p = item
			break
		}
	}
	return &p
}

func MockAssetAndRuneDepths() (assetE8PerPool, runeE8PerPool map[string]int64, timestamp time.Time) {
	ae8pp := map[string]int64{}
	re8pp := map[string]int64{}
	for _, item := range TestData.Pools {
		ae8pp[item.Asset] = item.Ae8pp
		re8pp[item.Asset] = item.Re8pp
	}
	ts, _ := time.Parse(time.RFC3339, TestData.Timestamp)
	return ae8pp, re8pp, ts
}

func MockGetPools(context.Context, time.Time) ([]string, error) {
	var result []string
	for _, item := range TestData.Pools {
		result = append(result, item.Asset)
	}
	return result, nil
}
func MockGetPoolStatus(ctx context.Context, asset string, ts time.Time) (string, error) {
	p := TestData.Pool(asset)
	return p.Status, nil
}

func MockPoolStakesLookup(ctx context.Context, asset string, w stat.Window) (*stat.PoolStakes, error) {
	p := TestData.Pool(asset)
	f, _ := time.Parse(time.RFC3339, p.StakeFirst)
	l, _ := time.Parse(time.RFC3339, p.StakeLast)
	return &stat.PoolStakes{
		Asset:           p.Asset,
		TxCount:         p.StakeTxCount,
		AssetE8Total:    p.StakeAssetE8Total,
		RuneE8Total:     p.StakeRuneE8Total,
		StakeUnitsTotal: p.StakeStakeUnitsTotal,
		First:           f,
		Last:            l,
	}, nil
}

func MockPoolUnstakesLookup(ctx context.Context, pool string, w stat.Window) (*stat.PoolUnstakes, error) {
	p := TestData.Pool(pool)
	return &stat.PoolUnstakes{
		TxCount:          p.UnstakeTxCount,
		AssetE8Total:     p.UnstakeAssetE8Total,
		RuneE8Total:      p.UnstakeRuneE8Total,
		StakeUnitsTotal:  p.UnstakeStakeUnitsTotal,
		BasisPointsTotal: p.UnstakeBasisPointsTotal,
	}, nil
}

func MockPoolSwapsFromRuneBucketsLookup(ctx context.Context, pool string, bucketSize time.Duration, w stat.Window) ([]stat.PoolSwaps, error) {
	return TestData.Pool(pool).SwapsFromRuneBucket, nil

}

func MockPoolSwapsToRuneBucketsLookup(ctx context.Context, pool string, bucketSize time.Duration, w stat.Window) ([]stat.PoolSwaps, error) {
	return TestData.Pool(pool).SwapsToRuneBucket, nil

}
func MockPoolStakesBucketsLookup(ctx context.Context, asset string, bucketSize time.Duration, w stat.Window) ([]stat.PoolStakes, error) {
	p := TestData.Pool(asset)
	return p.StakeHistory, nil
}
