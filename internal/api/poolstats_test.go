package api_test

import (
	"testing"

	"gitlab.com/thorchain/midgard/internal/api"

	"github.com/stretchr/testify/require"
	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

func deleteStatsTables(t *testing.T) {
	testdb.MustExec(t, "DELETE FROM swap_events")
	testdb.MustExec(t, "DELETE FROM block_pool_depths")
	testdb.MustExec(t, "DELETE FROM stake_events")
	testdb.MustExec(t, "DELETE FROM unstake_events")
}

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
	testdb.SetupTestDB(t)
	deleteStatsTables(t)

	timeseries.SetLastTimeForTest(testdb.StrToSec("2021-01-01 23:00:00"))
	timeseries.SetDepthsForTest([]timeseries.Depth{{
		Pool: "BNB.BNB", AssetDepth: 1000, RuneDepth: 2000,
	}})

	testdb.InsertBlockPoolDepth(t, "BNB.BNB", 100, 300, 0, "2021-01-01 12:00:00")

	testdb.InsertStakeEvent(t, testdb.FakeStake{
		Pool:         "BNB.BNB",
		AssetAddress: "bnbaddr1", RuneAddress: "thoraddr1", StakeUnits: 10,
		AssetE8:        10,
		RuneE8:         20,
		BlockTimestamp: "2021-01-01 12:00:00",
	})
	testdb.InsertUnstakeEvent(t, testdb.FakeUnstake{
		Pool:     "BNB.BNB",
		FromAddr: "thoraddr1", StakeUnits: 1,
		EmitAssetE8:         1,
		EmitRuneE8:          2,
		ImpLossProtectionE8: 1,
		BlockTimestamp:      "2021-01-01 12:00:00",
	})
	api.GlobalApiCacheStore.Flush()
	body := testdb.CallJSON(t,
		"http://localhost:8080/v2/pool/BNB.BNB/stats")

	var result oapigen.PoolStatsResponse
	testdb.MustUnmarshal(t, body, &result)

	require.Equal(t, "30", result.AddAssetLiquidityVolume)
	require.Equal(t, "20", result.AddRuneLiquidityVolume)
	require.Equal(t, "50", result.AddLiquidityVolume)
	require.Equal(t, "1", result.AddLiquidityCount)
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

	api.GlobalApiCacheStore.Flush()
	var resultAll oapigen.PoolStatsResponse
	testdb.MustUnmarshal(t, testdb.CallJSON(t,
		"http://localhost:8080/v2/pool/BNB.BNB/stats"), &resultAll)
	require.Equal(t, "2", resultAll.SwapCount)

	var result24h oapigen.PoolStatsResponse
	testdb.MustUnmarshal(t, testdb.CallJSON(t,
		"http://localhost:8080/v2/pool/BNB.BNB/stats?period=24h"), &result24h)
	require.Equal(t, "1", result24h.SwapCount)
}

func fetchBNBSwapperCount(t *testing.T, period string) string {
	body := testdb.CallJSON(t,
		"http://localhost:8080/v2/pool/BNB.BNB/stats?period="+period)

	var result oapigen.PoolStatsResponse
	testdb.MustUnmarshal(t, body, &result)

	return result.UniqueSwapperCount
}

func TestPoolsStatsUniqueSwapperCount(t *testing.T) {
	blocks := testdb.InitTestBlocks(t)

	blocks.NewBlock(t, "2010-01-01 00:00:00",
		testdb.AddLiquidity{Pool: "BNB.BNB", AssetAmount: 1000, RuneAmount: 2000},
	)

	api.GlobalApiCacheStore.Flush()
	require.Equal(t, "0", fetchBNBSwapperCount(t, "24h"))

	blocks.NewBlock(t, "2021-01-09 12:00:00", testdb.Swap{
		Pool:        "BNB.BNB",
		FromAddress: "ADDR_A",
		EmitAsset:   "8 THOR.RUNE",
		Coin:        "0 BNB.BNB",
	})
	api.GlobalApiCacheStore.Flush()
	require.Equal(t, "1", fetchBNBSwapperCount(t, "24h"))

	// same member
	blocks.NewBlock(t, "2021-01-09 13:00:00", testdb.Swap{
		Pool:        "BNB.BNB",
		FromAddress: "ADDR_A",
		EmitAsset:   "8 THOR.RUNE",
		Coin:        "0 BNB.BNB",
	})
	require.Equal(t, "1", fetchBNBSwapperCount(t, "24h"))

	// different pool
	blocks.NewBlock(t, "2021-01-09 13:00:00", testdb.Swap{
		Pool:        "BTC.BTC",
		FromAddress: "ADDR_B",
		EmitAsset:   "8 THOR.RUNE",
		Coin:        "0 BTC.BTC",
	})
	require.Equal(t, "1", fetchBNBSwapperCount(t, "24h"))

	// 2nd member in same pool
	blocks.NewBlock(t, "2021-01-09 13:00:00", testdb.Swap{
		Pool:        "BNB.BNB",
		FromAddress: "ADDR_B",
		EmitAsset:   "8 THOR.RUNE",
		Coin:        "0 BTC.BTC",
	})
	api.GlobalApiCacheStore.Flush()
	require.Equal(t, "2", fetchBNBSwapperCount(t, "24h"))

	blocks.NewBlock(t, "2021-01-10 00:00:00")
	// shorter period
	require.Equal(t, "0", fetchBNBSwapperCount(t, "1h"))
}

func TestPoolsStatsUniqueMemberCount(t *testing.T) {
	testdb.SetupTestDB(t)
	deleteStatsTables(t)

	timeseries.SetLastTimeForTest(testdb.StrToSec("2020-12-20 23:00:00"))
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

	body := testdb.CallJSON(t,
		"http://localhost:8080/v2/pool/BNB.BNB/stats")

	var result oapigen.PoolStatsResponse
	testdb.MustUnmarshal(t, body, &result)

	require.Equal(t, "2", result.UniqueMemberCount)
}
