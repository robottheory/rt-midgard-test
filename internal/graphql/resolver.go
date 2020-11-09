//go:generate go run github.com/99designs/gqlgen

package graphql

import (
	"gitlab.com/thorchain/midgard/chain/notinchain"
	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/internal/timeseries/stat"
)

var (
	getAssetAndRuneDepths = timeseries.AssetAndRuneDepths
	getPoolStatus         = timeseries.PoolStatus
	getPools              = timeseries.Pools

	poolStakesLookup   = stat.PoolStakesLookup
	poolUnstakesLookup = stat.PoolUnstakesLookup

	poolSwapsFromRuneBucketsLookup = stat.PoolSwapsFromRuneBucketsLookup
	poolSwapsToRuneBucketsLookup   = stat.PoolSwapsToRuneBucketsLookup

	allPoolStakesAddrLookup = stat.AllPoolStakesAddrLookup
	stakeAddrs              = timeseries.StakeAddrs

	stakesLookup   = stat.StakesLookup
	unstakesLookup = stat.UnstakesLookup

	swapsFromRuneLookup = stat.SwapsFromRuneLookup
	swapsToRuneLookup   = stat.SwapsToRuneLookup

	nodesSecpAndEd = timeseries.NodesSecpAndEd

	lastBlock = timeseries.LastBlock

	cachedNodeAccountsLookup = notinchain.CachedNodeAccountsLookup
	cachedNodeAccountLookup  = notinchain.CachedNodeAccountLookup
)

type Resolver struct {
}

//TODO cache repeated db calls to improve efficiency like stat.PoolStakesLookup, UnstakeLookup etc
