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
	getPools = mocks.MockGetPools
	getPoolStatus = mocks.MockGetPoolStatus
	poolSwapsFromRuneBucketsLookup = mocks.MockPoolSwapsFromRuneBucketsLookup
	poolSwapsToRuneBucketsLookup = mocks.MockPoolSwapsToRuneBucketsLookup

	allPoolStakesAddrLookup = mocks.MockAllPoolStakesAddrLookup
	memberAddrs = mocks.MockMemberAddrs

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
		require.Equal(t, dur.Until.Sub(dur.From), 24*time.Hour)
		require.Equal(t, err, nil)

		from = 100000
		until = 200000
		bs, dur, err = makeBucketSizeAndDurationWindow(&from, &until, nil)
		require.Equal(t, bs, 24*time.Hour)
		require.Equal(t, dur.Until.Sub(dur.From), time.Duration(until-from)*time.Second)
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
		require.Equal(t, dur.Until.Sub(dur.From), time.Duration(until-from)*time.Second)
		require.Nil(t, err)

		from = 100000
		until = 200000
		interval = model.LegacyIntervalWeek
		bs, _, _ = makeBucketSizeAndDurationWindow(&from, &until, &interval)
		require.Equal(t, bs, 7*24*time.Hour)
		require.Equal(t, dur.Until.Sub(dur.From), time.Duration(until-from)*time.Second)
		require.Nil(t, err)

		from = 100000
		until = 200000
		interval = model.LegacyIntervalMonth
		bs, _, _ = makeBucketSizeAndDurationWindow(&from, &until, &interval)
		require.Equal(t, bs, 30*24*time.Hour)
		require.Equal(t, dur.Until.Sub(dur.From), time.Duration(until-from)*time.Second)
		require.Nil(t, err)
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
