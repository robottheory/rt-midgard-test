package timeseries_test

import (
	"math"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/internal/fetch/notinchain"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

// TODO(muninn): split up to separate tests, migrate to fakeblocks.
func TestNetwork(t *testing.T) {
	defer testdb.StartMockThornode()()
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
	testdb.RegisterThornodeNodes(nodeAccounts)

	setupTotalReserve := int64(10000)
	testdb.RegisterThornodeReserve(setupTotalReserve)

	blocks := testdb.InitTestBlocks(t)

	const setupEmissionCurve = 2
	const setupBlocksPerYear = 2000000
	setupPoolRuneDepth := int64(200)
	setupTotalWeeklyFees := int64(10)

	blocks.NewBlock(t, "2020-09-01 00:00:00",
		testdb.SetMimir{Key: "EmissionCurve", Value: setupEmissionCurve},
		testdb.SetMimir{Key: "BlocksPerYear", Value: setupBlocksPerYear},
		testdb.AddLiquidity{
			Pool: "BNB.TWT-123", AssetAmount: 110, RuneAmount: 180,
		}, testdb.PoolActivate{Pool: "BNB.TWT-123"},
	)

	blocks.NewBlock(t, "2020-09-01 00:10:00",
		testdb.Swap{
			Pool:               "BNB.TWT-123",
			Coin:               "20 THOR.RUNE",
			EmitAsset:          "10 BNB.BNB",
			LiquidityFeeInRune: setupTotalWeeklyFees,
		},
	)

	body := testdb.CallJSON(t, "http://localhost:8080/v2/network")

	var jsonApiResult oapigen.Network
	testdb.MustUnmarshal(t, body, &jsonApiResult)

	// specified in ThorNode
	require.Equal(t, "1", jsonApiResult.ActiveNodeCount)
	require.Equal(t, "1", jsonApiResult.StandbyNodeCount)
	require.Equal(t, strconv.FormatInt(setupActiveBond, 10), jsonApiResult.BondMetrics.TotalActiveBond)
	require.Equal(t, strconv.FormatInt(setupStandbyBond, 10), jsonApiResult.BondMetrics.TotalStandbyBond)
	require.Equal(t, strconv.FormatInt(setupTotalReserve, 10), jsonApiResult.TotalReserve)
	require.Equal(t, strconv.FormatInt(setupPoolRuneDepth, 10), jsonApiResult.TotalPooledRune)

	// TODO(muninn): find a better setup that the block reward is a sane value
	// expectedBlockReward := int64(float64(setupTotalReserve) / float64(setupEmissionCurve*setupBlocksPerYear))
	expectedBlockReward := int64(0)

	require.Equal(t, strconv.FormatInt(expectedBlockReward, 10), jsonApiResult.BlockRewards.BlockReward)

	expectedPoolShareFactor := float64(setupActiveBond-setupPoolRuneDepth) / float64(setupActiveBond+setupPoolRuneDepth)
	expectedWeeklyTotalIncome := float64(expectedBlockReward + setupTotalWeeklyFees)
	expectedLiquidityIncome := expectedPoolShareFactor * float64(expectedWeeklyTotalIncome)
	expectedBondingIncome := (float64(1) - expectedPoolShareFactor) * expectedWeeklyTotalIncome
	expectedLiquidityAPY := math.Pow(1+(expectedLiquidityIncome/float64(2*setupPoolRuneDepth)), 52) - 1
	expectedBondingAPY := math.Pow(1+(expectedBondingIncome/float64(setupActiveBond)), 52) - 1
	require.Equal(t, floatStr(expectedPoolShareFactor), jsonApiResult.PoolShareFactor)
	require.Equal(t, floatStr(expectedLiquidityAPY), jsonApiResult.LiquidityAPY)
	require.Equal(t, floatStr(expectedBondingAPY), jsonApiResult.BondingAPY)
}

func TestNetworkNextChurnHeight(t *testing.T) {
	defer testdb.StartMockThornode()()
	blocks := testdb.InitTestBlocks(t)

	// ChurnInterval = 20 ; ChurnRetryInterval = 10
	blocks.NewBlock(t, "2020-09-01 00:00:00",
		testdb.SetMimir{Key: "ChurnInterval", Value: 20},
		testdb.SetMimir{Key: "ChurnRetryInterval", Value: 10},
	)

	// Last churn at block 2
	blocks.NewBlock(t, "2020-09-01 00:10:00", testdb.ActiveVault{AddVault: "addr"})

	body := testdb.CallJSON(t, "http://localhost:8080/v2/network")
	var result oapigen.Network
	testdb.MustUnmarshal(t, body, &result)

	require.Equal(t, "22", result.NextChurnHeight)

	blocks.EmptyBlocksBefore(t, 23) // Churn didn't happen at block 22

	body = testdb.CallJSON(t, "http://localhost:8080/v2/network")
	testdb.MustUnmarshal(t, body, &result)

	require.Equal(t, "32", result.NextChurnHeight)
}

func TestNetworkPoolCycle(t *testing.T) {
	defer testdb.StartMockThornode()()
	blocks := testdb.InitTestBlocks(t)

	// PoolCycle = 10
	blocks.NewBlock(t, "2020-09-01 00:00:00",
		testdb.SetMimir{Key: "PoolCycle", Value: 10},
	)

	// last block = 13
	blocks.EmptyBlocksBefore(t, 14)

	body := testdb.CallJSON(t, "http://localhost:8080/v2/network")
	var result oapigen.Network
	testdb.MustUnmarshal(t, body, &result)
	require.Equal(t, "7", result.PoolActivationCountdown)
}

func floatStr(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
