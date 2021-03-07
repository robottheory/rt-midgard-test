package timeseries_test

import (
	"math"
	"strconv"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/require"

	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/internal/fetch/chain/notinchain"
	"gitlab.com/thorchain/midgard/internal/graphql"
	"gitlab.com/thorchain/midgard/internal/graphql/generated"
	"gitlab.com/thorchain/midgard/internal/graphql/model"
	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

func TestNetwork(t *testing.T) {
	testdb.InitTest(t)
	schema := generated.NewExecutableSchema(generated.Config{Resolvers: &graphql.Resolver{}})
	gqlClient := client.New(handler.NewDefaultServer(schema))

	setupLastChurnBlock := int64(1)
	setupLastChurnBlockTimeStr := "2020-09-01 00:00:00"
	setupLastBlock := int64(2)
	setupLastBlockTimeStr := "2020-09-01 00:10:00"

	timeseries.SetLastTimeForTest(testdb.StrToSec(setupLastBlockTimeStr))
	timeseries.SetLastHeightForTest(setupLastBlock)

	setupPoolAssetDepth := int64(100)
	setupPoolRuneDepth := int64(200)
	timeseries.SetDepthsForTest([]timeseries.Depth{{"BNB.TWT-123", setupPoolAssetDepth, setupPoolRuneDepth}})
	testdb.InsertActiveVaultEvent(t, "addr", setupLastChurnBlockTimeStr)
	setupConstants := testdb.FakeThornodeConstants{
		EmissionCurve: 2,
		BlocksPerYear: 2000000,
		ChurnInterval: 10,
		PoolCycle:     10,
	}
	testdb.SetThornodeConstants(t, &setupConstants, setupLastBlockTimeStr)

	// Setting number of bonds, nodes  and totalReserve in the mocked ThorNode
	setupActiveBond := int64(500)
	setupStandbyBond := int64(100)
	nodeAccounts := make([]notinchain.NodeAccount, 2)
	nodeAccounts[0] = notinchain.NodeAccount{
		Status: "Active",
		Bond:   setupActiveBond,
	}
	nodeAccounts[1] = notinchain.NodeAccount{
		Status: "Standby",
		Bond:   setupStandbyBond,
	}

	setupTotalReserve := int64(10000)
	testdb.MockThorNode(setupTotalReserve, nodeAccounts)
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	testdb.InsertBlockLog(t, setupLastChurnBlock, setupLastChurnBlockTimeStr)
	testdb.InsertBlockLog(t, setupLastBlock, setupLastBlockTimeStr)

	setupTotalWeeklyFees := int64(10)
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.TWT-123", FromAsset: "BNB.RUNE", FromE8: 10, LiqFeeInRuneE8: setupTotalWeeklyFees, BlockTimestamp: setupLastBlockTimeStr})
	testdb.InsertStakeEvent(t, testdb.FakeStake{Pool: "BNB.TWT-123", BlockTimestamp: setupLastBlockTimeStr})

	body := testdb.CallV1(t, "http://localhost:8080/v2/network")

	var jsonApiResult oapigen.Network
	testdb.MustUnmarshal(t, body, &jsonApiResult)

	queryString := `{
		network {
			activeBonds,
			activeNodeCount
			standbyBonds
			standbyNodeCount
			bondMetrics {
				active {
					averageBond
					totalBond
					medianBond
					maximumBond
				}
				standby {
					averageBond
					totalBond
					medianBond
					maximumBond
				}
			}
			blockRewards {
				blockReward
				bondReward
				poolReward
			}
			liquidityAPY
			bondingAPY
			nextChurnHeight
			poolActivationCountdown
			poolShareFactor
			totalReserve
			totalPooledRune
		}
	}`

	type Result struct {
		Network model.Network
	}
	var graphqlResult Result
	gqlClient.MustPost(queryString, &graphqlResult)

	// specified in ThorNode
	require.Equal(t, "1", jsonApiResult.ActiveNodeCount)
	require.Equal(t, int64(1), graphqlResult.Network.ActiveNodeCount)
	require.Equal(t, "1", jsonApiResult.StandbyNodeCount)
	require.Equal(t, int64(1), graphqlResult.Network.StandbyNodeCount)
	require.Equal(t, strconv.FormatInt(setupActiveBond, 10), jsonApiResult.BondMetrics.TotalActiveBond)
	require.Equal(t, setupActiveBond, graphqlResult.Network.BondMetrics.Active.TotalBond)
	require.Equal(t, strconv.FormatInt(setupStandbyBond, 10), jsonApiResult.BondMetrics.TotalStandbyBond)
	require.Equal(t, setupStandbyBond, graphqlResult.Network.BondMetrics.Standby.TotalBond)
	require.Equal(t, strconv.FormatInt(setupTotalReserve, 10), jsonApiResult.TotalReserve)
	require.Equal(t, setupTotalReserve, graphqlResult.Network.TotalReserve)

	expectedBlockReward := int64(float64(setupTotalReserve) / float64(setupConstants.EmissionCurve*setupConstants.BlocksPerYear))
	require.Equal(t, strconv.FormatInt(expectedBlockReward, 10), jsonApiResult.BlockRewards.BlockReward)
	require.Equal(t, expectedBlockReward, graphqlResult.Network.BlockRewards.BlockReward)

	expectedPoolShareFactor := float64(setupActiveBond-setupPoolRuneDepth) / float64(setupActiveBond+setupPoolRuneDepth)
	expectedWeeklyTotalIncome := float64(expectedBlockReward + setupTotalWeeklyFees)
	expectedLiquidityIncome := expectedPoolShareFactor * float64(expectedWeeklyTotalIncome)
	expectedBondingIncome := (float64(1) - expectedPoolShareFactor) * expectedWeeklyTotalIncome
	expectedLiquidityAPY := math.Pow(1+(expectedLiquidityIncome/float64(2*setupPoolRuneDepth)), 52) - 1
	expectedBondingAPY := math.Pow(1+(expectedBondingIncome/float64(setupActiveBond)), 52) - 1
	require.Equal(t, floatStr(expectedPoolShareFactor), jsonApiResult.PoolShareFactor)
	require.Equal(t, expectedPoolShareFactor, graphqlResult.Network.PoolShareFactor)
	require.Equal(t, floatStr(expectedLiquidityAPY), jsonApiResult.LiquidityAPY)
	require.Equal(t, expectedLiquidityAPY, graphqlResult.Network.LiquidityApy)
	require.Equal(t, floatStr(expectedBondingAPY), jsonApiResult.BondingAPY)
	require.Equal(t, expectedBondingAPY, graphqlResult.Network.BondingApy)

	expectedNextChurnHeight := setupLastChurnBlock + setupConstants.ChurnInterval
	require.Equal(t, strconv.FormatInt(expectedNextChurnHeight, 10), jsonApiResult.NextChurnHeight)
	require.Equal(t, expectedNextChurnHeight, graphqlResult.Network.NextChurnHeight)

	expectedPoolActivationCountdown := setupConstants.PoolCycle - setupLastBlock%setupConstants.PoolCycle
	require.Equal(t, strconv.FormatInt(expectedPoolActivationCountdown, 10), jsonApiResult.PoolActivationCountdown)
	require.Equal(t, expectedPoolActivationCountdown, graphqlResult.Network.PoolActivationCountdown)

	require.Equal(t, strconv.FormatInt(setupTotalReserve, 10), jsonApiResult.TotalReserve)
	require.Equal(t, setupTotalReserve, graphqlResult.Network.TotalReserve)
	require.Equal(t, strconv.FormatInt(setupPoolRuneDepth, 10), jsonApiResult.TotalPooledRune)
	require.Equal(t, setupPoolRuneDepth, graphqlResult.Network.TotalPooledRune)
}

func floatStr(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
