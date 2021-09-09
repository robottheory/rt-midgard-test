package timeseries_test

import (
	"math"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/internal/fetch/notinchain"
	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

// TODO(muninn): split up to separate tests, migrate to fakeblocks.
func TestNetwork(t *testing.T) {
	defer testdb.StartMockThornode()()

	testdb.InitTest(t)

	setupLastBlock := int64(2)
	setupLastBlockTimeStr := "2020-09-01 00:10:00"

	timeseries.SetLastTimeForTest(testdb.StrToSec(setupLastBlockTimeStr))
	timeseries.SetLastHeightForTest(setupLastBlock)

	setupPoolAssetDepth := int64(100)
	setupPoolRuneDepth := int64(200)
	setupPoolSynthDepth := int64(0)
	timeseries.SetDepthsForTest([]timeseries.Depth{{"BNB.TWT-123", setupPoolAssetDepth, setupPoolRuneDepth, setupPoolSynthDepth}})

	const setupEmissionCurve = 2
	const setupBlocksPerYear = 2000000
	const setupPoolCycle = 10

	testdb.SetThornodeConstant(t, "EmissionCurve", setupEmissionCurve, setupLastBlockTimeStr)
	testdb.SetThornodeConstant(t, "BlocksPerYear", setupBlocksPerYear, setupLastBlockTimeStr)
	testdb.SetThornodeConstant(t, "PoolCycle", setupPoolCycle, setupLastBlockTimeStr)

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
	testdb.RegisterThornodeNodes(nodeAccounts)

	setupTotalReserve := int64(10000)
	testdb.RegisterThornodeReserve(setupTotalReserve)

	testdb.InsertBlockLog(t, 1, "2020-09-01 00:00:00")
	testdb.InsertBlockLog(t, setupLastBlock, setupLastBlockTimeStr)

	setupTotalWeeklyFees := int64(10)
	testdb.InsertSwapEvent(t,
		testdb.FakeSwap{Pool: "BNB.TWT-123", FromAsset: "BNB.RUNE", FromE8: 10,
			LiqFeeInRuneE8: setupTotalWeeklyFees, BlockTimestamp: setupLastBlockTimeStr})
	testdb.InsertStakeEvent(t,
		testdb.FakeStake{Pool: "BNB.TWT-123", BlockTimestamp: setupLastBlockTimeStr})

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

	expectedBlockReward := int64(float64(setupTotalReserve) / float64(setupEmissionCurve*setupBlocksPerYear))
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

	expectedPoolActivationCountdown := setupPoolCycle - setupLastBlock%setupPoolCycle
	require.Equal(t, strconv.FormatInt(expectedPoolActivationCountdown, 10), jsonApiResult.PoolActivationCountdown)
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

func floatStr(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
