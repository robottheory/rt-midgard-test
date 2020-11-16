package mocks

import (
	"context"
	"testing"
	"time"

	"gitlab.com/thorchain/midgard/chain/notinchain"
	"gitlab.com/thorchain/midgard/internal/graphql/model"
	"gitlab.com/thorchain/midgard/internal/timeseries/stat"
)

var T *testing.T

type ExpectedResponse struct {
	Nodes         []model.Node
	Stats         model.Stats
	Pool          model.Pool
	VolumeHistory model.PoolVolumeHistory
	StakeHistory  model.PoolStakeHistory
	Stakers       []model.Staker
	DepthHistory  model.PoolHistoryDetails
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
	StakeFirst           time.Time
	StakeLast            time.Time

	UnstakeTxCount          int64
	UnstakeAssetE8Total     int64
	UnstakeRuneE8Total      int64
	UnstakeStakeUnitsTotal  int64
	UnstakeBasisPointsTotal int64

	SwapsFromRuneBucket []stat.PoolSwaps
	SwapsToRuneBucket   []stat.PoolSwaps
	StakeHistory        []stat.PoolStakes

	Expected     ExpectedResponse
	NodeAccounts []*notinchain.NodeAccount
	PoolDepths   []stat.PoolDepth
}

type NodesSecpAndEdData struct {
	Secp map[string]string
	Ed   map[string]string
}

type Data struct {
	NodesSecpAndEdData NodesSecpAndEdData
	Pools              []Pool
	Timestamp          string
	LastBlockHeight    int64
	LastBlockTimestamp string
	LastBlockHash      []byte
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
	return &stat.PoolStakes{
		Asset:           p.Asset,
		TxCount:         p.StakeTxCount,
		AssetE8Total:    p.StakeAssetE8Total,
		RuneE8Total:     p.StakeRuneE8Total,
		StakeUnitsTotal: p.StakeStakeUnitsTotal,
		First:           p.StakeFirst,
		Last:            p.StakeLast,
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

func MockAllPoolStakesAddrLookup(ctx context.Context, addr string, w stat.Window) ([]stat.PoolStakes, error) {
	p := TestData.Pool(addr)
	return p.StakeHistory, nil
}

func MockStakeAddrs(ctx context.Context, moment time.Time) (addrs []string, err error) {
	return []string{"TEST.COIN"}, nil
}

func MockStakesLookup(ctx context.Context, w stat.Window) (*stat.Stakes, error) {
	ts1, _ := time.Parse(time.RFC3339, "2020-08-26 17:52:47.685651618 +0900 JST")
	ts2, _ := time.Parse(time.RFC3339, "2020-08-27 07:04:46.31075222 +0900 JST")
	return &stat.Stakes{

		TxCount:         7,
		RuneAddrCount:   4,
		RuneE8Total:     2658849927846,
		StakeUnitsTotal: 4860500000000,
		First:           ts1,
		Last:            ts2,
	}, nil
}
func MockUnstakesLookup(ctx context.Context, w stat.Window) (*stat.Unstakes, error) {
	return &stat.Unstakes{
		TxCount:       100,
		RuneAddrCount: 100,
		RuneE8Total:   100,
	}, nil
}
func MockSwapsFromRuneLookup(ctx context.Context, w stat.Window) (*stat.Swaps, error) {
	return &stat.Swaps{
		TxCount:       0,
		RuneAddrCount: 0,
		RuneE8Total:   0,
	}, nil
}
func MockSwapsToRuneLookup(ctx context.Context, w stat.Window) (*stat.Swaps, error) {
	return &stat.Swaps{
		TxCount:       1,
		RuneAddrCount: 1,
		RuneE8Total:   100000000,
	}, nil
}

func MockNodesSecpAndEd(ctx context.Context, t time.Time) (secp256k1Addrs, ed25519Addrs map[string]string, err error) {
	return TestData.NodesSecpAndEdData.Secp, TestData.NodesSecpAndEdData.Ed, nil
}
func MockLastBlock() (height int64, timestamp time.Time, hash []byte) {
	ts, _ := time.Parse(time.RFC3339, TestData.LastBlockTimestamp)
	return TestData.LastBlockHeight, ts, TestData.LastBlockHash
}

func MockCachedNodeAccountsLookup() ([]*notinchain.NodeAccount, error) {
	p := TestData.Pool("TEST.COIN")
	return p.NodeAccounts, nil
}

func MockCachedNodeAccountLookup(addr string) (*notinchain.NodeAccount, error) {
	p := TestData.Pool("TEST.COIN")
	return p.NodeAccounts[0], nil
}
func MockPoolDepthBucketsLookup(ctx context.Context, asset string, bucketSize time.Duration, w stat.Window) ([]stat.PoolDepth, error) {
	p := TestData.Pool(asset)
	return p.PoolDepths, nil
}
