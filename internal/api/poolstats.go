package api

import (
	"context"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/internal/timeseries/stat"
	"gitlab.com/thorchain/midgard/internal/util/miderr"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

// statsForPools is used both for stats and stats/legacy.
// The creates PoolStatsResponse, but to help the creation of PoolLegacyResponse it also
// puts int values into this struct. This way we don't need to convert back and forth between
// strings and ints.
type extraStats struct {
	runeDepth     int64
	toAssetCount  int64
	toRuneCount   int64
	swapCount     int64
	toAssetVolume int64
	toRuneVolume  int64
	totalVolume   int64
	totalFees     int64
}

func setAggregatesStats(
	ctx context.Context, pool string,
	ret *oapigen.PoolStatsResponse, extra *extraStats) (merr miderr.Err) {

	assetE8DepthPerPool, runeE8DepthPerPool, timestamp := timeseries.AssetAndRuneDepths()
	_, assetOk := assetE8DepthPerPool[pool]
	_, runeOk := runeE8DepthPerPool[pool]

	// TODO(acsaba): check that pool exists.
	// Return not found if there's no track of the pool
	if !assetOk && !runeOk {
		merr = miderr.BadRequestF("Unknown pool: %s", pool)
		return
	}

	status, err := timeseries.PoolStatus(ctx, pool, timestamp)
	if err != nil {
		merr = miderr.InternalErrE(err)
		return
	}

	aggregates, err := getPoolAggregates(ctx, []string{pool})
	if err != nil {
		merr = miderr.InternalErrE(err)
		return
	}

	assetDepth := aggregates.assetE8DepthPerPool[pool]
	runeDepth := aggregates.runeE8DepthPerPool[pool]
	dailyVolume := aggregates.dailyVolumes[pool]
	poolUnits := aggregates.poolUnits[pool]
	rewards := aggregates.poolWeeklyRewards[pool]
	poolAPY := timeseries.GetPoolAPY(runeDepth, rewards)

	ret.Asset = pool
	ret.AssetDepth = intStr(assetDepth)
	ret.RuneDepth = intStr(runeDepth)
	ret.PoolAPY = floatStr(poolAPY)
	ret.AssetPrice = floatStr(stat.AssetPrice(assetDepth, runeDepth))
	ret.Status = status
	ret.Units = intStr(poolUnits)
	ret.Volume24h = intStr(dailyVolume)

	extra.runeDepth = runeDepth
	return
}

func setSwapStats(
	ctx context.Context, pool string, buckets db.Buckets,
	ret *oapigen.PoolStatsResponse, extra *extraStats) (merr miderr.Err) {

	allSwaps, err := stat.GetPoolSwaps(ctx, pool, buckets)
	if err != nil {
		merr = miderr.InternalErrE(err)
		return
	}
	if len(allSwaps) != 1 {
		merr = miderr.InternalErr("Internal error: wrong time interval.")
		return
	}
	var swapHistory stat.SwapBucket = allSwaps[0]

	ret.ToRuneVolume = intStr(swapHistory.ToRuneVolume)
	ret.ToAssetVolume = intStr(swapHistory.ToAssetVolume)
	ret.SwapVolume = intStr(swapHistory.TotalVolume)

	ret.ToRuneCount = intStr(swapHistory.ToRuneCount)
	ret.ToAssetCount = intStr(swapHistory.ToAssetCount)
	ret.SwapCount = intStr(swapHistory.TotalCount)

	ret.AverageSlip = ratioStr(swapHistory.TotalSlip, swapHistory.TotalCount)
	ret.TotalFees = intStr(swapHistory.TotalFees)

	extra.toAssetCount = swapHistory.ToAssetCount
	extra.toRuneCount = swapHistory.ToRuneCount
	extra.swapCount = swapHistory.TotalCount
	extra.toAssetVolume = swapHistory.ToAssetVolume
	extra.toRuneVolume = swapHistory.ToRuneVolume
	extra.totalVolume = swapHistory.TotalVolume
	extra.totalFees = swapHistory.TotalFees
	return
}

func setLiquidityStats(
	ctx context.Context, pool string, buckets db.Buckets,
	ret *oapigen.PoolStatsResponse, extra *extraStats) (merr miderr.Err) {

	var allLiquidity oapigen.LiquidityHistoryResponse
	allLiquidity, err := stat.GetLiquidityHistory(ctx, buckets, pool)
	if err != nil {
		merr = miderr.InternalErrE(err)
		return
	}
	ret.AddLiquidityVolume = allLiquidity.Meta.AddLiquidityVolume
	ret.AddLiquidityCount = allLiquidity.Meta.AddLiquidityCount
	ret.WithdrawVolume = allLiquidity.Meta.WithdrawVolume
	ret.WithdrawCount = allLiquidity.Meta.WithdrawCount
	return
}

func statsForPool(ctx context.Context, pool string) (
	ret oapigen.PoolStatsResponse, extra extraStats, merr miderr.Err) {

	setAggregatesStats(ctx, pool, &ret, &extra)

	buckets := db.AllHistoryBuckets()
	setSwapStats(ctx, pool, buckets, &ret, &extra)
	setLiquidityStats(ctx, pool, buckets, &ret, &extra)
	return
}

func jsonPoolStats(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	pool := ps[0].Value
	result, _, merr := statsForPool(r.Context(), pool)
	if merr != nil {
		merr.ReportHTTP(w)
	}

	respJSON(w, result)
}
func jsonPoolStatsLegacy(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	pool := ps[0].Value
	stats, extra, merr := statsForPool(r.Context(), pool)
	if merr != nil {
		merr.ReportHTTP(w)
	}

	result := oapigen.PoolLegacyResponse{
		Asset:           stats.Asset,
		Volume24h:       stats.Volume24h,
		AssetDepth:      stats.AssetDepth,
		RuneDepth:       stats.RuneDepth,
		Price:           stats.AssetPrice,
		PoolAPY:         stats.PoolAPY,
		Status:          stats.Status,
		PoolUnits:       stats.Units,
		SwappingTxCount: stats.SwapCount,
		PoolSlipAverage: stats.AverageSlip,
		PoolTxAverage:   ratioStr(extra.totalVolume, extra.swapCount),
		PoolFeesTotal:   stats.TotalFees,
		PoolDepth:       intStr(2 * extra.runeDepth),
		SellVolume:      stats.ToRuneVolume,
		BuyVolume:       stats.ToAssetVolume,
		PoolVolume:      stats.SwapVolume,
		SellTxAverage:   ratioStr(extra.toRuneVolume, extra.toRuneCount),
		BuyTxAverage:    ratioStr(extra.toAssetVolume, extra.toAssetCount),
		PoolFeeAverage:  ratioStr(extra.totalFees, extra.swapCount),
		SellAssetCount:  stats.ToRuneCount,
		BuyAssetCount:   stats.ToAssetCount,
	}

	respJSON(w, result)
}
