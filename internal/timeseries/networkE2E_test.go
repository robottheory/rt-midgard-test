package timeseries_test

import (
	"github.com/stretchr/testify/assert"
	"gitlab.com/thorchain/midgard/chain/notinchain"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
	"testing"

	"github.com/jarcoal/httpmock"
	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/internal/timeseries/testdb"
)

func TestNetwork(t *testing.T) {
	testdb.SetupTestDB(t)
	testdb.MustExec(t, "DELETE FROM stake_events")
	testdb.MustExec(t, "DELETE FROM block_log")
	testdb.MustExec(t, "DELETE FROM swap_events")
	testdb.MustExec(t, "DELETE FROM block_pool_depths")
	testdb.MustExec(t, "DELETE FROM active_vault_events")
	testdb.MustExec(t, "DELETE FROM set_mimir_events")

	timeseries.SetLastTimeForTest(testdb.ToTime("2020-09-01 00:00:00"))
	timeseries.SetDepthsForTest("BNB.TWT-123", 30000000000000, 2240582804123679)
	testdb.InsertActiveVaultEvent(t, "addr", "2020-09-01 00:00:00")
	testdb.SetThornodeConstants(t)

	// Setting number of bonds, nodes  and totalReserve in the mocked ThorNode
	nodeAccounts := make([]notinchain.NodeAccount, 2)
	nodeAccounts[0] = notinchain.NodeAccount{
		Status: "active",
		Bond:   22772603677970,
	}
	nodeAccounts[1] = notinchain.NodeAccount{
		Status: "standby",
		Bond:   9999990,
	}
	testdb.MockThorNode(108915513107, nodeAccounts)
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	testdb.InsertBlockLog(t, 1, "2020-09-01 00:00:00")
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.TWT-123", FromAsset: "BNB.RUNE", FromE8: 300000000, LiqFeeInRuneE8: 3908720129799, BlockTimestamp: "2020-09-01 00:00:00"})
	testdb.InsertStakeEvent(t, testdb.FakeStake{Pool: "BNB.TWT-123", BlockTimestamp: "2020-09-01 00:00:00"})

	body := testdb.CallV1(t, "http://localhost:8080/v2/network")

	var actual oapigen.Network
	testdb.MustUnmarshal(t, body, &actual)

	// specified in ThorNode
	assert.Equal(t, "1", actual.ActiveNodeCount)
	assert.Equal(t, "1", actual.StandbyNodeCount)
	assert.Equal(t, "22772603677970", actual.BondMetrics.TotalActiveBond)
	assert.Equal(t, "9999990", actual.BondMetrics.TotalStandbyBond)
	assert.Equal(t, "108915513107", actual.TotalReserve)

	assert.Equal(t, "17256", actual.BlockRewards.BlockReward)

	assert.Equal(t, "0", actual.LiquidityAPY)
	assert.Equal(t, "3879.8255319373584", actual.BondingAPY)
	assert.Equal(t, "2161", actual.NextChurnHeight)
	assert.Equal(t, "49999", actual.PoolActivationCountdown)
	assert.Equal(t, "0", actual.PoolShareFactor)
	assert.Equal(t, "108915513107", actual.TotalReserve)
	assert.Equal(t, "2240582804123679", actual.TotalPooledRune)

}
