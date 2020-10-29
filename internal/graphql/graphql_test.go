package graphql

import (
	"testing"
	"time"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/stretchr/testify/require"
	"gitlab.com/thorchain/midgard/internal/graphql/generated"
	"gitlab.com/thorchain/midgard/internal/graphql/internal/mocks"
	"gitlab.com/thorchain/midgard/internal/graphql/model"
	"gitlab.com/thorchain/midgard/internal/timeseries/stat"
)

var (
	schema   = generated.NewExecutableSchema(generated.Config{Resolvers: &Resolver{}})
	c        = client.New(handler.NewDefaultServer(schema))
	testData = mocks.TestData
)

func setupStubs(t *testing.T) {
	mocks.T = t
	getAssetAndRuneDepths = mocks.MockAssetAndRuneDepths
	getPools = mocks.MockGetPools
	getPoolStatus = mocks.MockGetPoolStatus
	poolStakesLookup = mocks.MockPoolStakesLookup
	poolUnstakesLookup = mocks.MockPoolUnstakesLookup
	poolSwapsFromRuneBucketsLookup = mocks.MockPoolSwapsFromRuneBucketsLookup
	poolSwapsToRuneBucketsLookup = mocks.MockPoolSwapsToRuneBucketsLookup
	poolStakesBucketsLookup = mocks.MockPoolStakesBucketsLookup

	allPoolStakesAddrLookup = mocks.MockAllPoolStakesAddrLookup
	stakeAddrs = mocks.MockStakeAddrs

	stakesLookup = mocks.MockStakesLookup
	unstakesLookup = mocks.MockUnstakesLookup

	swapsFromRuneLookup = mocks.MockSwapsFromRuneLookup
	swapsToRuneLookup = mocks.MockSwapsToRuneLookup

	nodesSecpAndEd = mocks.MockNodesSecpAndEd

	lastBlock = mocks.MockLastBlock

	cachedNodeAccountsLookup = mocks.MockCachedNodeAccountsLookup
	cachedNodeAccountLookup = mocks.MockCachedNodeAccountLookup
}

func TestGraphQL(t *testing.T) {
	setupStubs(t)
	t.Run("makeBucketSizeAndDurationWindow", func(t *testing.T) {

		var (
			from     int64
			until    int64
			interval model.LegacyInterval
			bs       time.Duration
			dur      stat.Window
			err      error
		)

		bs, dur, err = makeBucketSizeAndDurationWindow(nil, nil, nil)
		require.Equal(t, bs, 24*time.Hour)
		require.Equal(t, dur.Until.Sub(dur.Since), 24*time.Hour)
		require.Equal(t, err, nil)

		from = 100000
		until = 200000
		bs, dur, err = makeBucketSizeAndDurationWindow(&from, &until, nil)
		require.Equal(t, bs, 24*time.Hour)
		require.Equal(t, dur.Until.Sub(dur.Since), time.Duration(until-from)*time.Second)
		require.Nil(t, err)

		from = 300000
		until = 200000
		bs, dur, err = makeBucketSizeAndDurationWindow(&from, &until, nil)
		require.NotNil(t, err)

		from = 100000
		until = 200000
		interval = model.LegacyIntervalDay
		bs, dur, err = makeBucketSizeAndDurationWindow(&from, &until, &interval)
		require.Equal(t, bs, 24*time.Hour)
		require.Equal(t, dur.Until.Sub(dur.Since), time.Duration(until-from)*time.Second)
		require.Nil(t, err)

		from = 100000
		until = 200000
		interval = model.LegacyIntervalWeek
		bs, _, _ = makeBucketSizeAndDurationWindow(&from, &until, &interval)
		require.Equal(t, bs, 7*24*time.Hour)
		require.Equal(t, dur.Until.Sub(dur.Since), time.Duration(until-from)*time.Second)
		require.Nil(t, err)

		from = 100000
		until = 200000
		interval = model.LegacyIntervalMonth
		bs, _, _ = makeBucketSizeAndDurationWindow(&from, &until, &interval)
		require.Equal(t, bs, 30*24*time.Hour)
		require.Equal(t, dur.Until.Sub(dur.Since), time.Duration(until-from)*time.Second)
		require.Nil(t, err)
	})

	t.Run("fetch_pools", func(t *testing.T) {
		var resp struct {
			Pools []*model.Pool
		}
		c.MustPost(`{
				  pools {
					asset
					status
					price
					units
				    depth {
					  assetDepth
					  runeDepth
					  poolDepth
					}
					stakes {
					  assetStaked
					  runeStaked
					  poolStaked
					}
					roi {
					  assetROI
					  runeROI
					}
				  }
				}`, &resp)

		expected := testData.Pools[0].Expected.Pool

		require.Equal(t, 1, len(resp.Pools))
		require.Equal(t, expected.Asset, resp.Pools[0].Asset)
		require.Equal(t, expected.Status, resp.Pools[0].Status)
		require.Equal(t, expected.Price, resp.Pools[0].Price)
		require.Equal(t, expected.Units, resp.Pools[0].Units)
		require.Equal(t, expected.Depth, resp.Pools[0].Depth)
		require.Equal(t, expected.Stakes, resp.Pools[0].Stakes)
		require.Equal(t, expected.Roi, resp.Pools[0].Roi)
	})
	t.Run("fetch_pool_by_id", func(t *testing.T) {
		var resp struct {
			Pool *model.Pool
		}
		c.MustPost(`{
				  pool(asset: "TEST.COIN") {
					asset
					status
					price
					units
				    depth {
					  assetDepth
					  runeDepth
					  poolDepth
					}
					stakes {
					  assetStaked
					  runeStaked
					  poolStaked
					}
					roi {
					  assetROI
					  runeROI
					}
				  }
				}`, &resp)

		expected := testData.Pool("TEST.COIN").Expected.Pool

		require.Equal(t, expected.Asset, resp.Pool.Asset)
		require.Equal(t, expected.Status, resp.Pool.Status)
		require.Equal(t, expected.Price, resp.Pool.Price)
		require.Equal(t, expected.Units, resp.Pool.Units)
		require.Equal(t, expected.Depth, resp.Pool.Depth)
		require.Equal(t, expected.Stakes, resp.Pool.Stakes)
		require.Equal(t, expected.Roi, resp.Pool.Roi)
	})
	t.Run("fetch_unknown_pool", func(t *testing.T) {
		var resp struct {
			Pool *model.Pool
		}
		c.MustPost(`{
				  pool(asset: "UNKNOWN") {
					asset
					status
					price
					units
				    depth {
					  assetDepth
					  runeDepth
					  poolDepth
					}
					stakes {
					  assetStaked
					  runeStaked
					  poolStaked
					}
					roi {
					  assetROI
					  runeROI
					}
				  }
				}`, &resp)

		require.Equal(t, "UNKNOWN", resp.Pool.Asset)
		require.Equal(t, "", resp.Pool.Status)
		require.Equal(t, float64(0), resp.Pool.Price)
		require.Equal(t, int64(0), resp.Pool.Units)
		require.Equal(t, &model.PoolDepth{}, resp.Pool.Depth)
		require.Equal(t, &model.PoolStakes{}, resp.Pool.Stakes)
		require.Equal(t, &model.Roi{}, resp.Pool.Roi)
	})
	t.Run("fetch_pool_limit_fields", func(t *testing.T) {
		var resp struct {
			Pool *model.Pool
		}
		c.MustPost(`{
				  pool(asset: "TEST.COIN") {
					asset
					status
					price
					units
				  }
				}`, &resp)

		//Fields not requested shouldn't be fetchedk
		require.Nil(t, resp.Pool.Depth)
		require.Nil(t, resp.Pool.Stakes)
		require.Nil(t, resp.Pool.Roi)
	})

	t.Run("fetch_pool_stake_history", func(t *testing.T) {
		var resp struct {
			StakeHistory model.PoolStakeHistory
		}
		c.MustPost(`{
					  stakeHistory(asset: "TEST.COIN") {
						  intervals{
							first
							last
							count
							volumeInRune
							volumeInAsset
							units
						  }
					  }
				}`, &resp)

		expected := testData.Pool("TEST.COIN").Expected.StakeHistory

		require.Equal(t, expected, resp.StakeHistory)
	})

	t.Run("fetch_stats", func(t *testing.T) {
		var resp struct {
			Stats model.Stats
		}
		c.MustPost(`{
					  stats {
						dailyActiveUsers
						dailyTx
						monthlyActiveUsers
						monthlyTx
						totalAssetBuys
						totalAssetSells
						totalDepth
						totalStakeTx
						totalStaked
						totalTx
						totalUsers
						totalVolume
						totalWithdrawTx
					  }
				}`, &resp)

		expected := testData.Pool("TEST.COIN").Expected.Stats

		require.Equal(t, expected, resp.Stats)
	})

	t.Run("fetch_assets", func(t *testing.T) {
		var resp struct {
			Assets []model.Asset
		}
		c.MustPost(`{
					  assets(query: ["TEST.COIN", "BTC"]) {
						asset
						created
						price
					  }
				}`, &resp)

		expected := testData.Pool("TEST.COIN").Expected.Assets

		require.Equal(t, 1, len(resp.Assets))
		require.Equal(t, expected, resp.Assets)
	})

	t.Run("fetch_nodes", func(t *testing.T) {
		var resp struct {
			Nodes []model.Node
		}
		c.MustPost(`{
					nodes(status: STANDBY) {
					  address
					  forcedToLeave
					  requestedToLeave
					  status
					  bond
					  leaveHeight
					  version
					  ipAddress
					  slashPoints
					  currentAward
					  jail {
						nodeAddr
						releaseHeight
						reason
					  }
					  publicKeys {
						secp256k1
						ed25519
					  }
					}
				}`, &resp)

		expected := testData.Pool("TEST.COIN").Expected.Nodes

		require.Equal(t, 1, len(resp.Nodes))
		require.Equal(t, expected, resp.Nodes)
	})

	t.Run("fetch_node_by_addr", func(t *testing.T) {
		var resp struct {
			Node model.Node
		}
		c.MustPost(`{
					node(address: "1234") {
					  address
					  forcedToLeave
					  requestedToLeave
					  status
					  bond
					  leaveHeight
					  version
					  ipAddress
					  slashPoints
					  currentAward
					  jail {
						nodeAddr
						releaseHeight
						reason
					  }
					  publicKeys {
						secp256k1
						ed25519
					  }
					}
				}`, &resp)

		expected := testData.Pool("TEST.COIN").Expected.Nodes[0]

		require.Equal(t, expected, resp.Node)
	})

	t.Run("fetch_stakers", func(t *testing.T) {
		var resp struct {
			Stakers []model.Staker
		}
		c.MustPost(`{
					  stakers {
						address
					  }
				}`, &resp)

		expected := testData.Pool("TEST.COIN").Expected.Stakers

		require.Equal(t, expected, resp.Stakers)
	})

	t.Run("fetch_staker_by_addr", func(t *testing.T) {
		var resp struct {
			Staker model.Staker
		}
		c.MustPost(`{
					staker(address: "TEST.COIN") {
						address
					  }
				}`, &resp)

		expected := testData.Pool("TEST.COIN").Expected.Stakers[0]

		require.Equal(t, expected, resp.Staker)
	})
}
