package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gitlab.com/thorchain/midgard/config"

	"github.com/julienschmidt/httprouter"
	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/graphql/model"
	"gitlab.com/thorchain/midgard/internal/util"
	"gitlab.com/thorchain/midgard/internal/util/miderr"

	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/internal/timeseries/stat"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

// Version 1 compatibility is a minimal effort attempt to provide smooth migration.

type Health struct {
	CatchingUp    bool  `json:"catching_up"`
	Database      bool  `json:"database"`
	ScannerHeight int64 `json:"scannerHeight,string"`
}

func jsonHealth(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	merr := util.CheckUrlEmpty(r.URL.Query())
	if merr != nil {
		merr.ReportHTTP(w)
		return
	}

	height, _, _ := timeseries.LastBlock()
	synced := db.FullyCaughtUp()

	respJSON(w, oapigen.HealthResponse{
		InSync:         synced,
		Database:       true,
		ScannerHeight:  util.IntStr(height + 1),
		LastQueried:    db.LastQueriedBlock.AsHeightTS(),
		LastFetched:    db.LastFetchedBlock.AsHeightTS(),
		LastCommitted:  db.LastCommittedBlock.AsHeightTS(),
		LastAggregated: db.LastAggregatedBlock.AsHeightTS(),
	})
}

func jsonEarningsHistory(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	f := func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		urlParams := r.URL.Query()
		buckets, merr := db.BucketsFromQuery(r.Context(), &urlParams)
		if merr != nil {
			merr.ReportHTTP(w)
			return
		}

		merr = util.CheckUrlEmpty(urlParams)
		if merr != nil {
			merr.ReportHTTP(w)
			return
		}

		var res oapigen.EarningsHistoryResponse
		res, err := stat.GetEarningsHistory(r.Context(), buckets)
		if err != nil {
			miderr.InternalErrE(err).ReportHTTP(w)
			return
		}
		if buckets.OneInterval() {
			res.Intervals = oapigen.EarningsHistoryIntervals{}
		}
		respJSON(w, res)
	}
	GlobalApiCacheStore.Get(GlobalApiCacheStore.LongTermLifetime, f, w, r, params)
}

func jsonLiquidityHistory(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	f := func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		urlParams := r.URL.Query()
		/*from:=util.ConsumeUrlParam(&urlParams, "from")
		to:=util.ConsumeUrlParam(&urlParams, "to")
		count:=util.ConsumeUrlParam(&urlParams,"count")
		interval:=util.ConsumeUrlParam(&urlParams,"interval")
		if from=="" && to=="" && interval=="day" && (count=="10" || count=="100") {
			if poolLiquidityChangesJob.response.buf.Len()>0{
				var res oapigen.LiquidityHistoryResponse
				err:=json.Unmarshal(poolLiquidityChangesJob.response.buf.Bytes(),&res)
				if err!=nil{
					res.Intervals
				}
			}
		}*/
		buckets, merr := db.BucketsFromQuery(r.Context(), &urlParams)
		if merr != nil {
			merr.ReportHTTP(w)
			return
		}

		pool := util.ConsumeUrlParam(&urlParams, "pool")
		if pool == "" {
			pool = "*"
		}
		merr = util.CheckUrlEmpty(urlParams)
		if merr != nil {
			merr.ReportHTTP(w)
			return
		}

		var res oapigen.LiquidityHistoryResponse
		res, err := stat.GetLiquidityHistory(r.Context(), buckets, pool)
		if err != nil {
			miderr.InternalErrE(err).ReportHTTP(w)
			return
		}
		if buckets.OneInterval() {
			res.Intervals = oapigen.LiquidityHistoryIntervals{}
		}
		respJSON(w, res)
	}
	GlobalApiCacheStore.Get(GlobalApiCacheStore.LongTermLifetime, f, w, r, params)
}

func jsonDepths(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	f := func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		pool := params[0].Value

		if !timeseries.PoolExists(pool) {
			miderr.BadRequestF("Unknown pool: %s", pool).ReportHTTP(w)
			return
		}

		urlParams := r.URL.Query()
		buckets, merr := db.BucketsFromQuery(r.Context(), &urlParams)
		if merr != nil {
			merr.ReportHTTP(w)
			return
		}

		merr = util.CheckUrlEmpty(urlParams)
		if merr != nil {
			merr.ReportHTTP(w)
			return
		}

		depths, err := stat.PoolDepthHistory(r.Context(), buckets, pool)
		if err != nil {
			miderr.InternalErrE(err).ReportHTTP(w)
			return
		}
		units, err := stat.PoolLiquidityUnitsHistory(r.Context(), buckets, pool)
		if err != nil {
			miderr.InternalErrE(err).ReportHTTP(w)
			return
		}
		if len(depths) != len(units) || depths[0].Window != units[0].Window {
			miderr.InternalErr("Buckets misalligned").ReportHTTP(w)
			return
		}
		var result oapigen.DepthHistoryResponse = toOapiDepthResponse(r.Context(), depths, units)
		respJSON(w, result)
	}
	GlobalApiCacheStore.Get(GlobalApiCacheStore.LongTermLifetime, f, w, r, params)
}

func toOapiDepthResponse(
	ctx context.Context,
	depths []stat.PoolDepthBucket,
	units []stat.UnitsBucket) (
	result oapigen.DepthHistoryResponse) {
	result.Intervals = make(oapigen.DepthHistoryIntervals, 0, len(depths))
	for i, bucket := range depths {
		liquidityUnits := units[i].Units
		synthUnits := timeseries.GetSinglePoolSynthUnits(ctx, bucket.Depths.AssetDepth, bucket.Depths.SynthDepth, liquidityUnits)
		poolUnits := liquidityUnits + synthUnits
		result.Intervals = append(result.Intervals, oapigen.DepthHistoryItem{
			StartTime:      util.IntStr(bucket.Window.From.ToI()),
			EndTime:        util.IntStr(bucket.Window.Until.ToI()),
			AssetDepth:     util.IntStr(bucket.Depths.AssetDepth),
			RuneDepth:      util.IntStr(bucket.Depths.RuneDepth),
			AssetPrice:     floatStr(bucket.Depths.AssetPrice()),
			AssetPriceUSD:  floatStr(bucket.AssetPriceUSD),
			LiquidityUnits: util.IntStr(liquidityUnits),
			SynthUnits:     util.IntStr(synthUnits),
			SynthSupply:    util.IntStr(bucket.Depths.SynthDepth),
			Units:          util.IntStr(poolUnits),
		})
	}
	result.Meta.StartTime = util.IntStr(depths[0].Window.From.ToI())
	result.Meta.EndTime = util.IntStr(depths[len(depths)-1].Window.Until.ToI())
	return
}

func toOapiOhlcvResponse(
	depths []stat.OHLCVBucket) (
	result oapigen.OHLCVHistoryResponse) {
	result.Intervals = make(oapigen.OHLCVHistoryIntervals, 0, len(depths))
	for _, bucket := range depths {
		minDate := bucket.Depths.MinDate
		maxDate := bucket.Depths.MaxDate
		if bucket.Depths.MinDate < bucket.Window.From.ToI() {
			minDate = bucket.Window.From.ToI()
		}
		if bucket.Depths.MinDate > bucket.Window.Until.ToI() {
			minDate = bucket.Window.Until.ToI()
		}
		if bucket.Depths.MaxDate < bucket.Window.From.ToI() {
			maxDate = bucket.Window.From.ToI()
		}
		if bucket.Depths.MaxDate > bucket.Window.Until.ToI() {
			maxDate = bucket.Window.Until.ToI()
		}
		result.Intervals = append(result.Intervals, oapigen.OHLCVHistoryItem{
			ClosePrice: floatStr(bucket.Depths.LastPrice),
			CloseTime:  util.IntStr(bucket.Window.Until.ToI()),
			HighPrice:  floatStr(bucket.Depths.MaxPrice),
			HighTime:   util.IntStr(maxDate),
			Liquidity:  util.IntStr(bucket.Depths.Liquidity),
			LowPrice:   floatStr(bucket.Depths.MinPrice),
			LowTime:    util.IntStr(minDate),
			OpenPrice:  floatStr(bucket.Depths.FirstPrice),
			OpenTime:   util.IntStr(bucket.Window.From.ToI()),
			Volume:     util.IntStr(bucket.Depths.Volume),
		})
	}
	result.Meta.StartTime = util.IntStr(depths[0].Window.From.ToI())
	result.Meta.EndTime = util.IntStr(depths[len(depths)-1].Window.Until.ToI())
	return
}

func jsonSwapHistory(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	f := func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		urlParams := r.URL.Query()

		buckets, merr := db.BucketsFromQuery(r.Context(), &urlParams)
		if merr != nil {
			merr.ReportHTTP(w)
			return
		}

		var pool *string
		poolParam := util.ConsumeUrlParam(&urlParams, "pool")
		if poolParam != "" {
			pool = &poolParam
		}

		merr = util.CheckUrlEmpty(urlParams)
		if merr != nil {
			merr.ReportHTTP(w)
			return
		}

		mergedPoolSwaps, err := stat.GetPoolSwaps(r.Context(), pool, buckets)
		if err != nil {
			miderr.InternalErr(err.Error()).ReportHTTP(w)
			return
		}
		var result oapigen.SwapHistoryResponse = createVolumeIntervals(mergedPoolSwaps)
		if buckets.OneInterval() {
			result.Intervals = oapigen.SwapHistoryIntervals{}
		}
		respJSON(w, result)
	}
	GlobalApiCacheStore.Get(GlobalApiCacheStore.LongTermLifetime, f, w, r, params)
}

func toSwapHistoryItem(bucket stat.SwapBucket) oapigen.SwapHistoryItem {
	return oapigen.SwapHistoryItem{
		StartTime:              util.IntStr(bucket.StartTime.ToI()),
		EndTime:                util.IntStr(bucket.EndTime.ToI()),
		ToAssetVolume:          util.IntStr(bucket.RuneToAssetVolume),
		ToRuneVolume:           util.IntStr(bucket.AssetToRuneVolume),
		SynthMintVolume:        util.IntStr(bucket.RuneToSynthVolume),
		SynthRedeemVolume:      util.IntStr(bucket.SynthToRuneVolume),
		TotalVolume:            util.IntStr(bucket.TotalVolume),
		ToAssetCount:           util.IntStr(bucket.RuneToAssetCount),
		ToRuneCount:            util.IntStr(bucket.AssetToRuneCount),
		SynthMintCount:         util.IntStr(bucket.RuneToSynthCount),
		SynthRedeemCount:       util.IntStr(bucket.SynthToRuneCount),
		TotalCount:             util.IntStr(bucket.TotalCount),
		ToAssetFees:            util.IntStr(bucket.RuneToAssetFees),
		ToRuneFees:             util.IntStr(bucket.AssetToRuneFees),
		SynthMintFees:          util.IntStr(bucket.RuneToSynthFees),
		SynthRedeemFees:        util.IntStr(bucket.SynthToRuneFees),
		TotalFees:              util.IntStr(bucket.TotalFees),
		ToAssetAverageSlip:     ratioStr(bucket.RuneToAssetSlip, bucket.RuneToAssetCount),
		ToRuneAverageSlip:      ratioStr(bucket.AssetToRuneSlip, bucket.AssetToRuneCount),
		SynthMintAverageSlip:   ratioStr(bucket.RuneToSynthSlip, bucket.RuneToSynthCount),
		SynthRedeemAverageSlip: ratioStr(bucket.SynthToRuneSlip, bucket.SynthToRuneCount),
		AverageSlip:            ratioStr(bucket.TotalSlip, bucket.TotalCount),
		RunePriceUSD:           floatStr(bucket.RunePriceUSD),
	}
}

func createVolumeIntervals(buckets []stat.SwapBucket) (result oapigen.SwapHistoryResponse) {
	metaBucket := stat.SwapBucket{}

	for _, bucket := range buckets {
		metaBucket.AddBucket(bucket)

		result.Intervals = append(result.Intervals, toSwapHistoryItem(bucket))
	}

	result.Meta = toSwapHistoryItem(metaBucket)
	result.Meta.StartTime = result.Intervals[0].StartTime
	result.Meta.EndTime = result.Intervals[len(result.Intervals)-1].EndTime
	result.Meta.RunePriceUSD = result.Intervals[len(result.Intervals)-1].RunePriceUSD
	return
}

// TODO(huginn): remove when bonds are fixed
var ShowBonds bool = false

func jsonTVLHistory(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	f := func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		urlParams := r.URL.Query()

		buckets, merr := db.BucketsFromQuery(r.Context(), &urlParams)
		if merr != nil {
			merr.ReportHTTP(w)
			return
		}
		merr = util.CheckUrlEmpty(urlParams)
		if merr != nil {
			merr.ReportHTTP(w)
			return
		}

		depths, err := stat.TVLDepthHistory(r.Context(), buckets)
		if err != nil {
			miderr.InternalErrE(err).ReportHTTP(w)
			return
		}
		bonds, err := stat.BondsHistory(r.Context(), buckets)
		if err != nil {
			miderr.InternalErrE(err).ReportHTTP(w)
			return
		}
		if len(depths) != len(bonds) || depths[0].Window != bonds[0].Window {
			miderr.InternalErr("Buckets misalligned").ReportHTTP(w)
			return
		}
		var result oapigen.TVLHistoryResponse = toTVLHistoryResponse(depths, bonds)
		respJSON(w, result)
	}
	GlobalApiCacheStore.Get(GlobalApiCacheStore.LongTermLifetime, f, w, r, params)
}

func toTVLHistoryResponse(depths []stat.TVLDepthBucket, bonds []stat.BondBucket) (result oapigen.TVLHistoryResponse) {
	showBonds := func(value string) *string {
		if !ShowBonds {
			return nil
		}
		return &value
	}

	result.Intervals = make(oapigen.TVLHistoryIntervals, 0, len(depths))
	for i, bucket := range depths {
		pools := 2 * bucket.TotalPoolDepth
		bonds := bonds[i].Bonds
		result.Intervals = append(result.Intervals, oapigen.TVLHistoryItem{
			StartTime:        util.IntStr(bucket.Window.From.ToI()),
			EndTime:          util.IntStr(bucket.Window.Until.ToI()),
			TotalValuePooled: util.IntStr(pools),
			TotalValueBonded: showBonds(util.IntStr(bonds)),
			TotalValueLocked: showBonds(util.IntStr(pools + bonds)),
			RunePriceUSD:     floatStr(bucket.RunePriceUSD),
		})
	}
	result.Meta = result.Intervals[len(depths)-1]
	result.Meta.StartTime = result.Intervals[0].StartTime
	return
}

type Network struct {
	ActiveBonds     []string `json:"activeBonds,string"`
	ActiveNodeCount int      `json:"activeNodeCount,string"`
	BlockRewards    struct {
		BlockReward int64 `json:"blockReward,string"`
		BondReward  int64 `json:"bondReward,string"`
		PoolReward  int64 `json:"poolReward,string"`
	} `json:"blockRewards"`
	BondMetrics struct {
		TotalActiveBond    int64 `json:"totalActiveBond,string"`
		AverageActiveBond  int64 `json:"averageActiveBond,string"`
		MedianActiveBond   int64 `json:"medianActiveBond,string"`
		MinimumActiveBond  int64 `json:"minimumActiveBond,string"`
		MaximumActiveBond  int64 `json:"maximumActiveBond,string"`
		TotalStandbyBond   int64 `json:"totalStandbyBond,string"`
		MinimumStandbyBond int64 `json:"minimumStandbyBond,string"`
		MaximumStandbyBond int64 `json:"maximumStandbyBond,string"`
		AverageStandbyBond int64 `json:"averageStandbyBond,string"`
		MedianStandbyBond  int64 `json:"medianStandbyBond,string"`
	} `json:"bondMetrics"`
	StandbyBonds            []string `json:"standbyBonds,string"`
	StandbyNodeCount        int      `json:"standbyNodeCount,string"`
	TotalPooledRune         int64    `json:"totalPooledRune,string"`
	TotalReserve            int64    `json:"totalReserve,string"`
	NextChurnHeight         int64    `json:"nextChurnHeight,string"`
	PoolActivationCountdown int64    `json:"poolActivationCountdown,string"`
	PoolShareFactor         float64  `json:"poolShareFactor,string"`
	BondingAPY              float64  `json:"bondingAPY,string"`
	LiquidityAPY            float64  `json:"liquidityAPY,string"`
}

func jsonNetwork(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	f := func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		merr := util.CheckUrlEmpty(r.URL.Query())
		if merr != nil {
			merr.ReportHTTP(w)
			return
		}

		network, err := timeseries.GetNetworkData(r.Context())
		if err != nil {
			respError(w, err)
			return
		}

		respJSON(w, convertNetwork(network))
	}
	GlobalApiCacheStore.Get(GlobalApiCacheStore.MidTermLifetime, f, w, r, params)
}

func convertNetwork(network model.Network) oapigen.Network {
	return oapigen.Network{
		ActiveBonds:     intArrayStrs(network.ActiveBonds),
		ActiveNodeCount: util.IntStr(network.ActiveNodeCount),
		BlockRewards: oapigen.BlockRewards{
			BlockReward: util.IntStr(network.BlockRewards.BlockReward),
			BondReward:  util.IntStr(network.BlockRewards.BondReward),
			PoolReward:  util.IntStr(network.BlockRewards.PoolReward),
		},
		// TODO(acsaba): create bondmetrics right away with this type.
		BondMetrics: oapigen.BondMetrics{
			TotalActiveBond:    util.IntStr(network.BondMetrics.Active.TotalBond),
			AverageActiveBond:  util.IntStr(network.BondMetrics.Active.AverageBond),
			MedianActiveBond:   util.IntStr(network.BondMetrics.Active.MedianBond),
			MinimumActiveBond:  util.IntStr(network.BondMetrics.Active.MinimumBond),
			MaximumActiveBond:  util.IntStr(network.BondMetrics.Active.MaximumBond),
			TotalStandbyBond:   util.IntStr(network.BondMetrics.Standby.TotalBond),
			AverageStandbyBond: util.IntStr(network.BondMetrics.Standby.AverageBond),
			MedianStandbyBond:  util.IntStr(network.BondMetrics.Standby.MedianBond),
			MinimumStandbyBond: util.IntStr(network.BondMetrics.Standby.MinimumBond),
			MaximumStandbyBond: util.IntStr(network.BondMetrics.Standby.MaximumBond),
		},
		BondingAPY:              floatStr(network.BondingApy),
		LiquidityAPY:            floatStr(network.LiquidityApy),
		NextChurnHeight:         util.IntStr(network.NextChurnHeight),
		PoolActivationCountdown: util.IntStr(network.PoolActivationCountdown),
		PoolShareFactor:         floatStr(network.PoolShareFactor),
		StandbyBonds:            intArrayStrs(network.StandbyBonds),
		StandbyNodeCount:        util.IntStr(network.StandbyNodeCount),
		TotalReserve:            util.IntStr(network.TotalReserve),
		TotalPooledRune:         util.IntStr(network.TotalPooledRune),
	}
}

type Node struct {
	Secp256K1 string `json:"secp256k1"`
	Ed25519   string `json:"ed25519"`
}

func calculateJsonNodes(ctx context.Context, w io.Writer) error {
	secpAddrs, edAddrs, err := timeseries.NodesSecpAndEd(ctx, time.Now())
	if err != nil {
		return err
	}

	m := make(map[string]struct {
		Secp string
		Ed   string
	}, len(secpAddrs))
	for key, addr := range secpAddrs {
		e := m[addr]
		e.Secp = key
		m[addr] = e
	}
	for key, addr := range edAddrs {
		e := m[addr]
		e.Ed = key
		m[addr] = e
	}

	array := make([]oapigen.Node, 0, len(m))
	for key, e := range m {
		array = append(array, oapigen.Node{
			Secp256k1:   e.Secp,
			Ed25519:     e.Ed,
			NodeAddress: key,
		})
	}
	writeJSON(w, array)
	return nil
}

func cachedJsonNodes() httprouter.Handle {
	cachedHandler := CreateAndRegisterCache(calculateJsonNodes, "nodes")
	return cachedHandler.ServeHTTP
}

// Filters out Suspended pools.
// If there is a status url parameter then returns pools with that status only.
func poolsWithRequestedStatus(ctx context.Context, urlParams *url.Values, statusMap map[string]string) ([]string, error) {
	pools, err := timeseries.PoolsWithDeposit(ctx)
	if err != nil {
		return nil, err
	}
	requestedStatus := util.ConsumeUrlParam(urlParams, "status")
	if requestedStatus != "" {
		const errormsg = "Max one status parameter, accepted values: available, staged, suspended"
		requestedStatus = strings.ToLower(requestedStatus)
		// Allowed statuses in
		// https://gitlab.com/thorchain/thornode/-/blob/master/x/thorchain/types/type_pool.go
		if requestedStatus != "available" && requestedStatus != "staged" && requestedStatus != "suspended" {
			return nil, fmt.Errorf(errormsg)
		}
	}
	ret := []string{}
	for _, pool := range pools {
		poolStatus := poolStatusFromMap(pool, statusMap)
		if poolStatus != "suspended" && (requestedStatus == "" || poolStatus == requestedStatus) {
			ret = append(ret, pool)
		}
	}
	return ret, nil
}

type poolAggregates struct {
	dailyVolumes        map[string]int64
	liquidityUnits      map[string]int64
	synthSupply         map[string]int64
	poolAPYs            map[string]float64
	assetE8DepthPerPool map[string]int64
	runeE8DepthPerPool  map[string]int64
}

func getPoolAggregates(ctx context.Context, pools []string) (*poolAggregates, error) {
	assetE8DepthPerPool, runeE8DepthPerPool, synthE8DepthPerPool, _ := timeseries.AllDepths()
	now := db.NowSecond()

	var dailyVolumes map[string]int64
	if poolVol24job != nil && poolVol24job.response.buf.Len() > 0 {
		err := json.Unmarshal(poolVol24job.response.buf.Bytes(), &dailyVolumes)
		if err != nil {
			return nil, err
		}
	} else {
		window24h := db.Window{From: now - 24*60*60, Until: now}
		var err error
		dailyVolumes, err = stat.PoolsTotalVolume(ctx, pools, window24h)
		if err != nil {
			return nil, err
		}
	}

	liquidityUnits, err := stat.CurrentPoolsLiquidityUnits(ctx, pools)
	if err != nil {
		return nil, err
	}

	var poolAPYs map[string]float64
	if poolApyJob != nil && poolVol24job.response.buf.Len() > 0 {
		err = json.Unmarshal(poolApyJob.response.buf.Bytes(), &poolAPYs)
	} else {
		week := db.Window{From: now - 7*24*60*60, Until: now}
		poolAPYs, err = timeseries.GetPoolAPY(ctx, runeE8DepthPerPool, pools, week)
	}

	aggregates := poolAggregates{
		dailyVolumes:        dailyVolumes,
		liquidityUnits:      liquidityUnits,
		synthSupply:         synthE8DepthPerPool,
		poolAPYs:            poolAPYs,
		assetE8DepthPerPool: assetE8DepthPerPool,
		runeE8DepthPerPool:  runeE8DepthPerPool,
	}

	return &aggregates, nil
}

func poolStatusFromMap(pool string, statusMap map[string]string) string {
	status, ok := statusMap[pool]
	if !ok {
		status = timeseries.DefaultPoolStatus
	}
	return status
}

func buildPoolDetail(
	ctx context.Context, pool, status string, aggregates poolAggregates, runePriceUsd float64) oapigen.PoolDetail {
	assetDepth := aggregates.assetE8DepthPerPool[pool]
	runeDepth := aggregates.runeE8DepthPerPool[pool]
	synthSupply := aggregates.synthSupply[pool]
	dailyVolume := aggregates.dailyVolumes[pool]
	liquidityUnits := aggregates.liquidityUnits[pool]
	synthUnits := timeseries.GetSinglePoolSynthUnits(ctx, assetDepth, synthSupply, liquidityUnits)
	poolUnits := liquidityUnits + synthUnits
	poolAPY := aggregates.poolAPYs[pool]
	price := timeseries.AssetPrice(assetDepth, runeDepth)
	priceUSD := price * runePriceUsd

	return oapigen.PoolDetail{
		Asset:          pool,
		AssetDepth:     util.IntStr(assetDepth),
		RuneDepth:      util.IntStr(runeDepth),
		PoolAPY:        floatStr(poolAPY),
		AssetPrice:     floatStr(price),
		AssetPriceUSD:  floatStr(priceUSD),
		Status:         status,
		Units:          util.IntStr(poolUnits),
		LiquidityUnits: util.IntStr(liquidityUnits),
		SynthUnits:     util.IntStr(synthUnits),
		SynthSupply:    util.IntStr(synthSupply),
		Volume24h:      util.IntStr(dailyVolume),
	}
}

func jsonPools(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	f := func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		urlParams := r.URL.Query()

		_, lastTime, _ := timeseries.LastBlock()
		statusMap, err := timeseries.GetPoolsStatuses(r.Context(), db.Nano(lastTime.UnixNano()))
		if err != nil {
			respError(w, err)
			return
		}
		pools, err := poolsWithRequestedStatus(r.Context(), &urlParams, statusMap)
		if err != nil {
			respError(w, err)
			return
		}

		merr := util.CheckUrlEmpty(urlParams)
		if merr != nil {
			merr.ReportHTTP(w)
			return
		}

		aggregates, err := getPoolAggregates(r.Context(), pools)
		if err != nil {
			respError(w, err)
			return
		}

		runePriceUsd := stat.RunePriceUSD()

		poolsResponse := oapigen.PoolsResponse{}
		for _, pool := range pools {
			runeDepth := aggregates.runeE8DepthPerPool[pool]
			assetDepth := aggregates.assetE8DepthPerPool[pool]
			if 0 < runeDepth && 0 < assetDepth {
				status := poolStatusFromMap(pool, statusMap)
				poolsResponse = append(poolsResponse, buildPoolDetail(r.Context(), pool, status, *aggregates, runePriceUsd))
			}
		}

		respJSON(w, poolsResponse)
	}
	GlobalApiCacheStore.Get(GlobalApiCacheStore.ShortTermLifetime, f, w, r, params)
}

func jsonohlcv(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	f := func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		pool := ps[0].Value

		if !timeseries.PoolExists(pool) {
			miderr.BadRequestF("Unknown pool: %s", pool).ReportHTTP(w)
			return
		}

		urlParams := r.URL.Query()
		buckets, merr := db.BucketsFromQuery(r.Context(), &urlParams)
		if merr != nil {
			merr.ReportHTTP(w)
			return
		}

		merr = util.CheckUrlEmpty(urlParams)
		if merr != nil {
			merr.ReportHTTP(w)
			return
		}

		depths, err := stat.PoolOHLCVHistory(r.Context(), buckets, pool)
		if err != nil {
			miderr.InternalErrE(err).ReportHTTP(w)
			return
		}
		swaps, err := stat.GetPoolSwaps(r.Context(), &pool, buckets)
		if err != nil {
			miderr.InternalErrE(err).ReportHTTP(w)
			return
		}
		for i := 0; i < buckets.Count(); i++ {
			depths[i].Depths.Volume = swaps[i].TotalVolume
		}
		var result oapigen.OHLCVHistoryResponse = toOapiOhlcvResponse(depths)
		respJSON(w, result)
	}
	GlobalApiCacheStore.Get(GlobalApiCacheStore.LongTermLifetime, f, w, r, ps)
}

func jsonPool(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	f := func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		merr := util.CheckUrlEmpty(r.URL.Query())
		if merr != nil {
			merr.ReportHTTP(w)
			return
		}

		pool := ps[0].Value

		if !timeseries.PoolExistsNow(pool) {
			miderr.BadRequestF("Unknown pool: %s", pool).ReportHTTP(w)
			return
		}

		status, err := timeseries.PoolStatus(r.Context(), pool)
		if err != nil {
			miderr.InternalErrE(err).ReportHTTP(w)
			return
		}

		aggregates, err := getPoolAggregates(r.Context(), []string{pool})
		if err != nil {
			miderr.InternalErrE(err).ReportHTTP(w)
			return
		}

		runePriceUsd := stat.RunePriceUSD()

		var poolResponse oapigen.PoolResponse
		poolResponse = oapigen.PoolResponse(
			buildPoolDetail(r.Context(), pool, status, *aggregates, runePriceUsd))
		respJSON(w, poolResponse)
	}
	GlobalApiCacheStore.Get(GlobalApiCacheStore.ShortTermLifetime, f, w, r, ps)
}

// returns string array
func jsonMembers(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	urlParams := r.URL.Query()

	var pool *string
	poolParam := util.ConsumeUrlParam(&urlParams, "pool")
	if poolParam != "" {
		pool = &poolParam
		if !timeseries.PoolExists(*pool) {
			miderr.BadRequestF("Unknown pool: %s", *pool).ReportHTTP(w)
			return
		}

	}
	merr := util.CheckUrlEmpty(urlParams)
	if merr != nil {
		merr.ReportHTTP(w)
		return
	}

	addrs, err := timeseries.GetMemberAddrs(r.Context(), pool)
	if err != nil {
		respError(w, err)
		return
	}
	result := oapigen.MembersResponse(addrs)
	respJSON(w, result)
}

func jsonMemberDetails(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	merr := util.CheckUrlEmpty(r.URL.Query())
	if merr != nil {
		merr.ReportHTTP(w)
		return
	}

	addr := strings.ToLower(ps[0].Value)

	pools, err := timeseries.GetMemberPools(r.Context(), addr)
	if err != nil {
		respError(w, err)
		return
	}
	if len(pools) == 0 {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	respJSON(w, oapigen.MemberDetailsResponse{
		Pools: pools.ToOapigen(),
	})
}

func jsonTHORName(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	merr := util.CheckUrlEmpty(r.URL.Query())
	if merr != nil {
		merr.ReportHTTP(w)
		return
	}

	name := ps[0].Value

	n, err := timeseries.GetTHORName(r.Context(), &name)
	if err != nil {
		respError(w, err)
		return
	}
	if n.Owner == "" {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	entries := make([]oapigen.THORNameEntry, len(n.Entries))
	for i, e := range n.Entries {
		entries[i] = oapigen.THORNameEntry{
			Chain:   e.Chain,
			Address: e.Address,
		}
	}

	respJSON(w, oapigen.THORNameDetailsResponse{
		Owner:   n.Owner,
		Expire:  util.IntStr(n.Expire),
		Entries: entries,
	})
}

func jsonTHORNameAddress(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	merr := util.CheckUrlEmpty(r.URL.Query())
	if merr != nil {
		merr.ReportHTTP(w)
		return
	}

	addr := strings.ToLower(ps[0].Value)

	names, err := timeseries.GetTHORNamesByAddress(r.Context(), &addr)
	if err != nil {
		respError(w, err)
		return
	}
	if len(names) == 0 {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	respJSON(w, oapigen.ReverseTHORNameResponse(
		names,
	))
}

func calculateAPY(periodicRate float64, periodsPerYear float64) float64 {
	if 1 < periodsPerYear {
		return math.Pow(1+periodicRate, periodsPerYear) - 1
	}
	return periodicRate * periodsPerYear
}

func calculatePoolAPY(ctx context.Context, w io.Writer) error {
	pools, err := timeseries.PoolsWithDeposit(ctx)
	if err != nil {
		return err
	}
	now := db.NowSecond()
	window := db.Window{From: now - 7*24*60*60, Until: now}
	fromNano := window.From.ToNano()
	toNano := window.Until.ToNano()

	income, err := timeseries.PoolsTotalIncome(ctx, pools, fromNano, toNano)
	if err != nil {
		return miderr.InternalErrE(err)
	}

	periodsPerYear := float64(365*24*60*60) / float64(window.Until-window.From)
	if timeseries.Latest.GetState().Timestamp == 0 {
		return errors.New("Last block is not ready yet")
	}
	_, runeDepths, _, _ := timeseries.AllDepths()
	ret := map[string]float64{}
	for _, pool := range pools {
		runeDepth := runeDepths[pool]
		if 0 < runeDepth {
			poolRate := float64(income[pool]) / (2 * float64(runeDepth))

			ret[pool] = calculateAPY(poolRate, periodsPerYear)
		}
	}
	bt, err := json.Marshal(ret)
	if err != nil {
		return err
	}
	_, err = w.Write(bt)
	return err
}

func calculatePoolVolume(ctx context.Context, w io.Writer) error {
	if timeseries.Latest.GetState().Timestamp == 0 {
		return errors.New("Last block is not ready yet")
	}
	pools, err := timeseries.PoolsWithDeposit(ctx)
	if err != nil {
		return err
	}
	now := db.NowSecond()
	window24h := db.Window{From: now - 24*60*60, Until: now}
	dailyVolumes, err := stat.PoolsTotalVolume(ctx, pools, window24h)
	if err != nil {
		return err
	}
	bt, err := json.Marshal(dailyVolumes)
	if err != nil {
		return err
	}
	_, err = w.Write(bt)
	return err
}

func calculatePoolLiquidityChanges(ctx context.Context, w io.Writer) error {
	if timeseries.Latest.GetState().Timestamp == 0 {
		return errors.New("Last block is not ready yet")
	}
	var urls url.Values
	urls = map[string][]string{
		"interval": {"day"},
		"count":    {"100"},
	}
	buckets, merr := db.BucketsFromQuery(ctx, &urls)
	if merr != nil {
		return merr
	}
	pool := "*"
	var res oapigen.LiquidityHistoryResponse
	res, err := stat.GetLiquidityHistory(ctx, buckets, pool)
	if err != nil {
		return err
	}
	if buckets.OneInterval() {
		res.Intervals = oapigen.LiquidityHistoryIntervals{}
	}
	bt, err := json.Marshal(res)
	if err == nil {
		_, err = w.Write(bt)
	}
	return err
}

/*func calculateOHLCV(ctx context.Context, w io.Writer) error {
	pools, err := timeseries.PoolsWithDeposit(ctx)
	if err != nil {
		return err
	}
	go func(ctx context.Context) {
		for _, pool := range pools {
			params := httprouter.Params{
				{
					"pool",
					pool,
				},
			}
			ct, _ := context.WithTimeout(context.Background(), time.Minute*5)
			req, err := http.NewRequestWithContext(ct, "GET", "http://127.0.0.1:8080/v2/history/ohlcv/"+pool+"?interval=hour&count="+strconv.Itoa(ohlcvCount), nil)
			if err != nil {
				log.Error().Interface("error", err).Str("path", req.URL.Path).Msg("panic ohlcv cron job")
			}
			writer := httptest.NewRecorder()
			jsonohlcv(writer, req, params)

			req, err = http.NewRequestWithContext(ct, "GET", "http://127.0.0.1:8080/v2/history/ohlcv/"+pool+"?interval=day&count="+strconv.Itoa(ohlcvCount), nil)
			if err != nil {
				log.Error().Interface("error", err).Str("path", req.URL.Path).Msg("panic ohlcv cron job")
			}
			writer = httptest.NewRecorder()
			jsonohlcv(writer, req, params)

			req, err = http.NewRequestWithContext(ct, "GET", "http://127.0.0.1:8080/v2/history/ohlcv/"+pool+"?interval=month&count="+strconv.Itoa(ohlcvCount), nil)
			if err != nil {
				log.Error().Interface("error", err).Str("path", req.URL.Path).Msg("panic ohlcv cron job")
			}
			writer = httptest.NewRecorder()
			jsonohlcv(writer, req, params)
		}
	}(ctx)
	return err
}*/

func calculateJsonStats(ctx context.Context, w io.Writer) error {
	state := timeseries.Latest.GetState()
	now := db.NowSecond()
	window := db.Window{From: 0, Until: now}

	stakes, err := stat.StakesLookup(ctx, window)
	if err != nil {
		return err
	}
	unstakes, err := stat.UnstakesLookup(ctx, window)
	if err != nil {
		return err
	}
	swapsFromRune, err := stat.SwapsFromRuneLookup(ctx, window)
	if err != nil {
		return err
	}
	swapsToRune, err := stat.SwapsToRuneLookup(ctx, window)
	if err != nil {
		return err
	}

	window24h := db.Window{From: now - 24*60*60, Until: now}
	window30d := db.Window{From: now - 30*24*60*60, Until: now}

	dailySwapsFromRune, err := stat.SwapsFromRuneLookup(ctx, window24h)
	if err != nil {
		return err
	}
	dailySwapsToRune, err := stat.SwapsToRuneLookup(ctx, window24h)
	if err != nil {
		return err
	}
	monthlySwapsFromRune, err := stat.SwapsFromRuneLookup(ctx, window30d)
	if err != nil {
		return err
	}
	monthlySwapsToRune, err := stat.SwapsToRuneLookup(ctx, window30d)
	if err != nil {
		return err
	}

	var runeDepth int64
	for _, poolInfo := range state.Pools {
		runeDepth += poolInfo.RuneDepth
	}

	switchedRune, err := stat.SwitchedRune(ctx)
	if err != nil {
		return err
	}

	runePrice := stat.RunePriceUSD()

	// TODO(acsaba): validate/correct calculations:
	//   - UniqueSwapperCount is it correct to do fromRune+toRune with multichain? (Now overlap?)
	//   - Swap count with doubleswaps are counted twice?
	//   - Predecessor to AddLiquidityVolume was totalStaked, which was stakes-withdraws.
	//       Is the new one ok?
	//   - AddLiquidityVolume looks only on rune, doesn't work with assymetric.
	//   - consider adding 24h 30d and total for everything.
	writeJSON(w, oapigen.StatsResponse{
		RuneDepth:                     util.IntStr(runeDepth),
		SwitchedRune:                  util.IntStr(switchedRune),
		RunePriceUSD:                  floatStr(runePrice),
		SwapVolume:                    util.IntStr(swapsFromRune.RuneE8Total + swapsToRune.RuneE8Total),
		SwapCount24h:                  util.IntStr(dailySwapsFromRune.TxCount + dailySwapsToRune.TxCount),
		SwapCount30d:                  util.IntStr(monthlySwapsFromRune.TxCount + monthlySwapsToRune.TxCount),
		SwapCount:                     util.IntStr(swapsFromRune.TxCount + swapsToRune.TxCount),
		ToAssetCount:                  util.IntStr(swapsFromRune.TxCount),
		ToRuneCount:                   util.IntStr(swapsToRune.TxCount),
		DailyActiveUsers:              util.IntStr(dailySwapsFromRune.RuneAddrCount + dailySwapsToRune.RuneAddrCount),
		MonthlyActiveUsers:            util.IntStr(monthlySwapsFromRune.RuneAddrCount + monthlySwapsToRune.RuneAddrCount),
		UniqueSwapperCount:            util.IntStr(swapsFromRune.RuneAddrCount + swapsToRune.RuneAddrCount),
		AddLiquidityVolume:            util.IntStr(stakes.TotalVolume),
		WithdrawVolume:                util.IntStr(unstakes.TotalVolume),
		ImpermanentLossProtectionPaid: util.IntStr(unstakes.ImpermanentLossProtection),
		AddLiquidityCount:             util.IntStr(stakes.Count),
		WithdrawCount:                 util.IntStr(unstakes.Count),
	})
	/* TODO(pascaldekloe)
	   "poolCount":"20",
	   "totalEarned":"1827445688454",
	   "totalVolume24hr":"37756279870656",
	*/
	return nil
}

func cachedJsonStats() httprouter.Handle {
	cachedHandler := CreateAndRegisterCache(calculateJsonStats, "stats")
	return cachedHandler.ServeHTTP
}

var (
	poolVol24job            *cache
	poolApyJob              *cache
	poolLiquidityChangesJob *cache
	// poolOHLCVJob            *cache
)

func init() {
	poolVol24job = CreateAndRegisterCache(calculatePoolVolume, "volume24")
	poolApyJob = CreateAndRegisterCache(calculatePoolAPY, "poolApy")
	poolLiquidityChangesJob = CreateAndRegisterCache(calculatePoolLiquidityChanges, "poolLiqduityChanges")
	// poolOHLCVJob = CreateAndRegisterCache(calculateOHLCV, "poolOHLCV")
}

func jsonActions(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	f := func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		urlParams := r.URL.Query()
		params := timeseries.ActionsParams{
			Limit:      util.ConsumeUrlParam(&urlParams, "limit"),
			Offset:     util.ConsumeUrlParam(&urlParams, "offset"),
			ActionType: util.ConsumeUrlParam(&urlParams, "type"),
			Address:    strings.ToLower(util.ConsumeUrlParam(&urlParams, "address")),
			TXId:       util.ConsumeUrlParam(&urlParams, "txid"),
			Asset:      util.ConsumeUrlParam(&urlParams, "asset"),
		}
		merr := util.CheckUrlEmpty(urlParams)
		if merr != nil {
			merr.ReportHTTP(w)
			return
		}

		// normalize address to lowercase if chain is not case sensitive
		chain := strings.Split(params.Asset, ".")[0]
		// Get results
		actions, err := timeseries.GetActions(r.Context(), time.Time{}, params)
		if !config.Global.CaseSensitiveChains[chain] {
			params.Address = strings.ToLower(params.Address)
		}

		// Send response
		if err != nil {
			respError(w, err)
			return
		}
		respJSON(w, actions)
	}
	urlParams := r.URL.Query()
	actionParams := timeseries.ActionsParams{
		Limit:      util.ConsumeUrlParam(&urlParams, "limit"),
		Offset:     util.ConsumeUrlParam(&urlParams, "offset"),
		ActionType: util.ConsumeUrlParam(&urlParams, "type"),
		Address:    util.ConsumeUrlParam(&urlParams, "address"),
		TXId:       util.ConsumeUrlParam(&urlParams, "txid"),
		Asset:      util.ConsumeUrlParam(&urlParams, "asset"),
	}
	if actionParams.TXId == "" && actionParams.Address == "" {
		GlobalApiCacheStore.Get(GlobalApiCacheStore.MidTermLifetime, f, w, r, params)
	} else {
		GlobalApiCacheStore.Get(GlobalApiCacheStore.IgnoreCache, f, w, r, params)
	}
}

func jsonSwagger(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	swagger, err := oapigen.GetSwagger()
	if err != nil {
		respError(w, err)
		return
	}
	respJSON(w, swagger)
}

func writeJSON(w io.Writer, body interface{}) {
	e := json.NewEncoder(w)
	e.SetIndent("", "\t")
	// Error discarded
	_ = e.Encode(body)
}

func respJSON(w http.ResponseWriter, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, body)
}

func respError(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func intArrayStrs(a []int64) []string {
	b := make([]string, len(a))
	for i, v := range a {
		b[i] = util.IntStr(v)
	}
	return b
}

func ratioStr(a, b int64) string {
	if b == 0 {
		return "0"
	} else {
		return strconv.FormatFloat(float64(a)/float64(b), 'f', -1, 64)
	}
}

func floatStr(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
