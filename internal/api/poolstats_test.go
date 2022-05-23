package api_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

func TestPoolsStatsDepthAndSwaps(t *testing.T) {
	blocks := testdb.InitTestBlocks(t)

	blocks.NewBlock(t, "2010-01-01 00:00:00",
		testdb.AddLiquidity{Pool: "BNB.BNB", AssetAmount: 1000, RuneAmount: 2000},
	)

	// Swapping to 10 rune, fee 2
	blocks.NewBlock(t, "2020-12-03 12:00:00", testdb.Swap{
		Pool:               "BNB.BNB",
		EmitAsset:          "8 THOR.RUNE",
		Coin:               "0 BNB.BNB",
		LiquidityFeeInRune: 2,
		Slip:               1,
	})

	// Swap 30, fee 2
	blocks.NewBlock(t, "2020-12-03 13:00:00", testdb.Swap{
		Pool:               "BNB.BNB",
		EmitAsset:          "28 THOR.RUNE",
		Coin:               "0 BNB.BNB",
		LiquidityFeeInRune: 2,
		Slip:               2,
	})

	blocks.NewBlock(t, "2020-12-20 23:00:00")

	body := testdb.CallJSON(t,
		"http://localhost:8080/v2/pool/BNB.BNB/stats")

	var result oapigen.PoolStatsResponse
	testdb.MustUnmarshal(t, body, &result)

	require.Equal(t, "1000", result.AssetDepth)
	require.Equal(t, "2", result.SwapCount)
	require.Equal(t, "40", result.ToRuneVolume)
	require.Equal(t, "4", result.TotalFees)
	require.Equal(t, "4", result.ToRuneFees)
	require.Equal(t, "0", result.ToAssetFees)
	require.Equal(t, "1.5", result.AverageSlip)
	require.Equal(t, "1.5", result.ToRuneAverageSlip)
	require.Equal(t, "0", result.ToAssetAverageSlip)
}

func TestPoolStatsLiquidity(t *testing.T) {
	blocks := testdb.InitTestBlocks(t)

	blocks.NewBlock(t, "2000-01-01 00:00:00",
		testdb.AddLiquidity{Pool: "BNB.BNB", AssetAmount: 1000000, RuneAmount: 3000000,
			RuneAddress: "R1"},
		testdb.PoolActivate{Pool: "BNB.BNB"})

	blocks.NewBlock(t, "2021-01-01 12:00:00",
		testdb.AddLiquidity{Pool: "BNB.BNB", AssetAmount: 10, RuneAmount: 20, RuneAddress: "R2"},
		testdb.Withdraw{Pool: "BNB.BNB", EmitAsset: 1, EmitRune: 2, ImpLossProtection: 1,
			FromAddress: "R1"})

	// final depths are 1009 and 3029

	body := testdb.CallJSON(t,
		"http://localhost:8080/v2/pool/BNB.BNB/stats?period=24h")

	var result oapigen.PoolStatsResponse
	testdb.MustUnmarshal(t, body, &result)

	require.Equal(t, "30", result.AddAssetLiquidityVolume)
	require.Equal(t, "20", result.AddRuneLiquidityVolume)
	require.Equal(t, "50", result.AddLiquidityVolume)
	require.Equal(t, "1", result.AddLiquidityCount)
	require.Equal(t, "3", result.WithdrawAssetVolume)
	require.Equal(t, "2", result.WithdrawRuneVolume)
	require.Equal(t, "5", result.WithdrawVolume)
	require.Equal(t, "1", result.ImpermanentLossProtectionPaid)
	require.Equal(t, "1", result.WithdrawCount)
}

func TestPoolsPeriod(t *testing.T) {

	blocks := testdb.InitTestBlocks(t)

	blocks.NewBlock(t, "2010-01-01 00:00:00",
		testdb.AddLiquidity{Pool: "BNB.BNB", AssetAmount: 1000, RuneAmount: 2000},
	)

	// swap 25h ago
	blocks.NewBlock(t, "2021-01-01 12:00:00", testdb.Swap{
		Pool:               "BNB.BNB",
		EmitAsset:          "8 THOR.RUNE",
		Coin:               "0 BNB.BNB",
		LiquidityFeeInRune: 2,
		Slip:               1,
	})

	// swap 22h ago
	blocks.NewBlock(t, "2021-01-01 15:00:00", testdb.Swap{
		Pool:               "BNB.BNB",
		EmitAsset:          "28 THOR.RUNE",
		Coin:               "0 BNB.BNB",
		LiquidityFeeInRune: 2,
		Slip:               2,
	})

	blocks.NewBlock(t, "2021-01-02 13:00:00")

	var resultAll oapigen.PoolStatsResponse
	testdb.MustUnmarshal(t, testdb.CallJSON(t,
		"http://localhost:8080/v2/pool/BNB.BNB/stats"), &resultAll)
	require.Equal(t, "2", resultAll.SwapCount)

	var result24h oapigen.PoolStatsResponse
	testdb.MustUnmarshal(t, testdb.CallJSON(t,
		"http://localhost:8080/v2/pool/BNB.BNB/stats?period=24h"), &result24h)
	require.Equal(t, "1", result24h.SwapCount)
}

func TestPoolsStatsUniqueMemberCount(t *testing.T) {
	testdb.InitTest(t)

	timeseries.SetDepthsForTest([]timeseries.Depth{{
		Pool: "BNB.BNB", AssetDepth: 1000, RuneDepth: 2000,
	}})

	// 2 members
	testdb.InsertStakeEvent(t,
		testdb.FakeStake{Pool: "BNB.BNB", AssetAddress: "bnbaddr1", RuneAddress: "thoraddr1", StakeUnits: 2})
	testdb.InsertStakeEvent(t,
		testdb.FakeStake{Pool: "BNB.BNB", AssetAddress: "bnbaddr2", RuneAddress: "thoraddr2", StakeUnits: 5})

	// duplication
	testdb.InsertStakeEvent(t,
		testdb.FakeStake{Pool: "BNB.BNB", AssetAddress: "bnbaddr2", RuneAddress: "thoraddr2", StakeUnits: 5})

	// different pool
	testdb.InsertStakeEvent(t,
		testdb.FakeStake{Pool: "BTC.BTC", AssetAddress: "bnbaddr3", RuneAddress: "thoraddr3", StakeUnits: 5})

	db.RefreshAggregatesForTests()

	body := testdb.CallJSON(t,
		"http://localhost:8080/v2/pool/BNB.BNB/stats")

	var result oapigen.PoolStatsResponse
	testdb.MustUnmarshal(t, body, &result)

	require.Equal(t, "2", result.UniqueMemberCount)
}
