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
	runeDepth int64
}

func statsForPool(ctx context.Context, pool string) (
	ret oapigen.PoolStatsResponse, extra extraStats, merr miderr.Err) {

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
	ret = oapigen.PoolStatsResponse{
		Asset:      pool,
		AssetDepth: intStr(assetDepth),
		RuneDepth:  intStr(runeDepth),
		PoolAPY:    floatStr(poolAPY),
		AssetPrice: floatStr(stat.AssetPrice(assetDepth, runeDepth)),
		Status:     status,
		Units:      intStr(poolUnits),
		Volume24h:  intStr(dailyVolume),
	}
	extra.runeDepth = runeDepth

	buckets := db.AllHistoryBuckets()
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

	ret.SwappingTxCount = intStr(swapHistory.TotalCount)
	ret.PoolTxAverage = ratioStr(swapHistory.TotalVolume, swapHistory.TotalCount)
	ret.TotalFees = intStr(swapHistory.TotalFees)

	ret.ToRuneVolume = intStr(swapHistory.ToRuneVolume)
	ret.ToAssetVolume = intStr(swapHistory.ToAssetVolume)
	ret.PoolVolume = intStr(swapHistory.ToRuneVolume + swapHistory.ToAssetVolume)
	ret.SellTxAverage = ratioStr(swapHistory.ToRuneVolume, swapHistory.ToRuneCount)
	ret.BuyTxAverage = ratioStr(swapHistory.ToAssetVolume, swapHistory.ToAssetCount)
	ret.AverageSlip = ratioStr(swapHistory.TotalSlip, swapHistory.TotalCount)
	ret.PoolFeeAverage = ratioStr(swapHistory.TotalFees, swapHistory.TotalCount)
	ret.ToRuneCount = intStr(swapHistory.ToRuneCount)
	ret.ToAssetCount = intStr(swapHistory.ToAssetCount)
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
		SwappingTxCount: stats.SwappingTxCount,
		PoolSlipAverage: stats.AverageSlip,
		PoolTxAverage:   stats.PoolTxAverage,
		PoolFeesTotal:   stats.TotalFees,
		PoolDepth:       intStr(2 * extra.runeDepth),
		SellVolume:      stats.ToRuneVolume,
		BuyVolume:       stats.ToAssetVolume,
		PoolVolume:      stats.PoolVolume,
		SellTxAverage:   stats.SellTxAverage,
		BuyTxAverage:    stats.BuyTxAverage,
		PoolFeeAverage:  stats.PoolFeeAverage,
		SellAssetCount:  stats.ToRuneCount,
		BuyAssetCount:   stats.ToAssetCount,
	}

	respJSON(w, result)
}
