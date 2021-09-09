package timeseries_test

import (
	"math"
	"strconv"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/require"

	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/internal/fetch/notinchain"
	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

// TODO(muninn): split up to separate tests, migrate to fakeblocks.
func TestNetwork(t *testing.T) {
	testdb.InitTest(t)

	setupLastChurnBlock := int64(1)
	setupLastChurnBlockTimeStr := "2020-09-01 00:00:00"
	setupLastBlock := int64(2)
	setupLastBlockTimeStr := "2020-09-01 00:10:00"

	timeseries.SetLastTimeForTest(testdb.StrToSec(setupLastBlockTimeStr))
	timeseries.SetLastHeightForTest(setupLastBlock)

	setupPoolAssetDepth := int64(100)
	setupPoolRuneDepth := int64(200)
	setupPoolSynthDepth := int64(0)
	timeseries.SetDepthsForTest([]timeseries.Depth{{"BNB.TWT-123", setupPoolAssetDepth, setupPoolRuneDepth, setupPoolSynthDepth}})
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

	expectedBlockReward := int64(float64(setupTotalReserve) / float64(setupConstants.EmissionCurve*setupConstants.BlocksPerYear))
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

	expectedNextChurnHeight := setupLastChurnBlock + setupConstants.ChurnInterval
	require.Equal(t, strconv.FormatInt(expectedNextChurnHeight, 10), jsonApiResult.NextChurnHeight)

	expectedPoolActivationCountdown := setupConstants.PoolCycle - setupLastBlock%setupConstants.PoolCycle
	require.Equal(t, strconv.FormatInt(expectedPoolActivationCountdown, 10), jsonApiResult.PoolActivationCountdown)

	require.Equal(t, strconv.FormatInt(setupTotalReserve, 10), jsonApiResult.TotalReserve)
	require.Equal(t, strconv.FormatInt(setupPoolRuneDepth, 10), jsonApiResult.TotalPooledRune)
}

func floatStr(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
