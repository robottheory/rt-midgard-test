package timeseries_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/internal/fetch/notinchain"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

func TestNetworkAPY(t *testing.T) {
	defer testdb.StartMockThornode()()
	blocks := testdb.InitTestBlocks(t)

	// Active bond amount = 1500 rune
	testdb.RegisterThornodeNodes([]notinchain.NodeAccount{
		{Status: "Active", Bond: 1500},
		{Status: "Standby", Bond: 123}})

	// reserve=5200
	// blocks per year = 520 (10 weekly)
	// emission curve = 2
	// rewards per block: 5200 / (520 * 2) = 5
	testdb.RegisterThornodeReserve(5200)
	blocks.NewBlock(t, "2020-09-01 00:00:00",
		testdb.SetMimir{Key: "EmissionCurve", Value: 2},
		testdb.SetMimir{Key: "BlocksPerYear", Value: 520},

		testdb.AddLiquidity{Pool: "BNB.TWT-123", AssetAmount: 550, RuneAmount: 900},
		testdb.PoolActivate{Pool: "BNB.TWT-123"},
	)

	blocks.NewBlock(t, "2020-09-01 00:10:00",
		testdb.Swap{
			Pool:               "BNB.TWT-123",
			Coin:               "100 THOR.RUNE",
			EmitAsset:          "50 BNB.BNB",
			LiquidityFeeInRune: 10,
		},
	)
	// Final depths: Rune = 1000 (900 + 100) ; Asset = 500 (550 - 50)
	// LP pooled amount is considered 2000 (double the rune amount)

	body := testdb.CallJSON(t, "http://localhost:8080/v2/network")

	var jsonApiResult oapigen.Network
	testdb.MustUnmarshal(t, body, &jsonApiResult)

	require.Equal(t, "1", jsonApiResult.ActiveNodeCount)
	require.Equal(t, "1", jsonApiResult.StandbyNodeCount)
	require.Equal(t, "1500", jsonApiResult.BondMetrics.TotalActiveBond)
	require.Equal(t, "123", jsonApiResult.BondMetrics.TotalStandbyBond)
	require.Equal(t, "5200", jsonApiResult.TotalReserve)
	require.Equal(t, "1000", jsonApiResult.TotalPooledRune)

	require.Equal(t, "5", jsonApiResult.BlockRewards.BlockReward)

	// (Bond - Pooled) / (Bond + Pooled)
	// (1500 - 1000) / (1500 + 1000) = 500 / 2500 = 0.2
	require.Equal(t, "0.2", jsonApiResult.PoolShareFactor)

	// Weekly income = 60 (block reward * weekly blocks + liquidity fees)
	// LP earning weekly = 12 (60 * 0.2)
	// LP weekly yield = 0.6% (weekly earning / 2*rune depth = 12 / 2*1000)
	// LP cumulative yearly yield ~ 36% ( 1.006 ** 52)
	require.Contains(t, jsonApiResult.LiquidityAPY, "0.36")

	// Bonding earning = 48 (60 * 0.2)
	// Bonding weekly yield = 3.2% (weekly earning / active bond)
	// Bonding cumulative yearly yield ~ 414% ( 1.032 ** 52)
	require.Contains(t, jsonApiResult.BondingAPY, "4.14")
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
