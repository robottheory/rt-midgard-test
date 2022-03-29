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
		LastThorNode:   db.LastThorNodeBlock.AsHeightTS(),
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

		beforeDepth, depths, err := stat.PoolDepthHistory(r.Context(), buckets, pool)
		if err != nil {
			miderr.InternalErrE(err).ReportHTTP(w)
			return
		}
		beforeUnit, units, err := stat.PoolLiquidityUnitsHistory(r.Context(), buckets, pool)
		if err != nil {
			miderr.InternalErrE(err).ReportHTTP(w)
			return
		}
		if len(depths) != len(units) || depths[0].Window != units[0].Window {
			miderr.InternalErr("Buckets misalligned").ReportHTTP(w)
			return
		}
		var result oapigen.DepthHistoryResponse = toOapiDepthResponse(r.Context(), beforeDepth, depths, beforeUnit, units)
		respJSON(w, result)
	}
	GlobalApiCacheStore.Get(GlobalApiCacheStore.LongTermLifetime, f, w, r, params)
}

func toOapiDepthResponse(
	ctx context.Context,
	beforeDepth timeseries.PoolDepths,
	depths []stat.PoolDepthBucket,
	beforeUnit int64,
	units []stat.UnitsBucket) (
	result oapigen.DepthHistoryResponse) {
	result.Intervals = make(oapigen.DepthHistoryIntervals, 0, len(depths))
	for i, bucket := range depths {
		liquidityUnits := units[i].Units
		synthUnits := timeseries.GetSinglePoolSynthUnits(ctx, bucket.Depths.AssetDepth, bucket.Depths.SynthDepth, liquidityUnits)
		poolUnits := liquidityUnits + synthUnits
		assetDepth := bucket.Depths.AssetDepth
		runeDepth := bucket.Depths.RuneDepth
		liqUnitValIndex := luvi(bucket.Depths.AssetDepth, bucket.Depths.RuneDepth, liquidityUnits)
		result.Intervals = append(result.Intervals, oapigen.DepthHistoryItem{
			StartTime:      util.IntStr(bucket.Window.From.ToI()),
			EndTime:        util.IntStr(bucket.Window.Until.ToI()),
			AssetDepth:     util.IntStr(assetDepth),
			RuneDepth:      util.IntStr(runeDepth),
			AssetPrice:     floatStr(bucket.Depths.AssetPrice()),
			AssetPriceUSD:  floatStr(bucket.AssetPriceUSD),
			LiquidityUnits: util.IntStr(liquidityUnits),
			SynthUnits:     util.IntStr(synthUnits),
			SynthSupply:    util.IntStr(bucket.Depths.SynthDepth),
			Units:          util.IntStr(poolUnits),
			Luvi:           floatStr(liqUnitValIndex),
		})
	}
	result.Meta.StartTime = util.IntStr(depths[0].Window.From.ToI())
	result.Meta.EndTime = util.IntStr(depths[len(depths)-1].Window.Until.ToI())
	result.Meta.PriceShiftLoss = floatStr(priceShiftLoss(beforeDepth, depths[len(depths)-1].Depths))
	result.Meta.LuviIncrease = floatStr(luviIncrease(beforeDepth, depths[len(depths)-1].Depths, beforeUnit, units[len(units)-1].Units))
	result.Meta.StartAssetDepth = util.IntStr(beforeDepth.AssetDepth)
	result.Meta.StartRuneDepth = util.IntStr(beforeDepth.RuneDepth)
	result.Meta.StartLiquidityUnits = util.IntStr(beforeUnit)
	result.Meta.EndAssetDepth = util.IntStr(depths[len(depths)-1].Depths.AssetDepth)
	result.Meta.EndRuneDepth = util.IntStr(depths[len(depths)-1].Depths.RuneDepth)
	result.Meta.EndLiquidityUnits = util.IntStr(units[len(units)-1].Units)
	return
}

func luvi(assetE8 int64, runeE8 int64, liquidityUnits int64) float64 {
	if liquidityUnits <= 0 {
		return math.NaN()
	}
	return math.Sqrt(float64(assetE8)*float64(runeE8)) / float64(liquidityUnits)
}

func luviIncrease(beforeDepth timeseries.PoolDepths, lastDepth timeseries.PoolDepths, beforeUnit int64, lastLiqUnit int64) float64 {
	// LUVI_Increase = LUVI1 / LUVI0
	liqUnitValIndex0 := luvi(beforeDepth.AssetDepth, beforeDepth.RuneDepth, beforeUnit)
	liqUnitValIndex1 := luvi(lastDepth.AssetDepth, lastDepth.RuneDepth, lastLiqUnit)
	return liqUnitValIndex1 / liqUnitValIndex0
}

func priceShiftLoss(beforeDepth timeseries.PoolDepths, lastDepth timeseries.PoolDepths) float64 {
	// Price0 = R0 / A0 (rune depth at time 0, asset depth at time 0)
	// Price1 = R1 / A1 (rune depth at time 1, asset depth at time 1)
	// PriceShift = Price1 / Price0
	// PriceShiftLoss = 2*sqrt(PriceShift) / (1 + PriceShift)
	price0 := float64(beforeDepth.RuneDepth) / float64(beforeDepth.AssetDepth)
	price1 := float64(lastDepth.RuneDepth) / float64(lastDepth.AssetDepth)
	ratio := price1 / price0
	return 2 * math.Sqrt(ratio) / (1 + ratio)
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

func jsonTsSwapHistory(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
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

		mergedPoolSwaps, err := stat.GetPoolTsSwaps(r.Context(), pool, buckets)
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

func jsonNodes(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	secpAddrs, edAddrs, err := timeseries.NodesSecpAndEd(r.Context(), time.Now())
	if err != nil {
		respError(w, err)
		return
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
	respJSON(w, array)
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

	addr := ps[0].Value

	var pools timeseries.MemberPools
	var err error
	for _, addr := range []string{addr, strings.ToLower(addr)} {
		pools, err = timeseries.GetMemberPools(r.Context(), addr)
		if err != nil {
			respError(w, err)
			return
		}
		if len(pools) > 0 {
			break
		}
	}
	if len(pools) == 0 {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	respJSON(w, oapigen.MemberDetailsResponse{
		Pools: pools.ToOapigen(),
	})
}

func jsonFullMemberDetails(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	urlParams := r.URL.Query()
	addrs := util.ConsumeUrlParam(&urlParams, "address")
	address := strings.Split(addrs, ",")
	var allPools []oapigen.FullMemberPool
	for _, addr := range address {
		var pools timeseries.MemberPools
		var err error
		for _, addr := range []string{addr, strings.ToLower(addr)} {
			exists := false
			for _, oldAddr := range allPools {
				if strings.ToLower(addr) == strings.ToLower(oldAddr.RuneAddress) || strings.ToLower(addr) == strings.ToLower(oldAddr.AssetAddress) {
					exists = true
					break
				}
			}
			if exists {
				continue
			}
			pools, err = timeseries.GetFullMemberPools(r.Context(), addr)
			if err != nil {
				respError(w, err)
				return
			}
			if len(pools) > 0 {
				break
			}
		}
		if pools != nil {
			for _, memberPool := range pools {
				aggregates, err := getPoolAggregates(r.Context(), []string{memberPool.Pool})
				if err != nil {
					miderr.InternalErrE(err).ReportHTTP(w)
					return
				}
				if _, ok := aggregates.liquidityUnits[memberPool.Pool]; !ok {
					miderr.InternalErrE(err).ReportHTTP(w)
					return
				}
				allPools = append(allPools, oapigen.FullMemberPool{
					Pool:           memberPool.Pool,
					RuneAddress:    memberPool.RuneAddress,
					AssetAddress:   memberPool.AssetAddress,
					PoolUnits:      util.IntStr(aggregates.liquidityUnits[memberPool.Pool]),
					SharedUnits:    util.IntStr(memberPool.LiquidityUnits),
					RuneAdded:      util.IntStr(memberPool.RuneAdded),
					AssetAdded:     util.IntStr(memberPool.AssetAdded),
					RuneWithdrawn:  util.IntStr(memberPool.RuneWithdrawn),
					AssetWithdrawn: util.IntStr(memberPool.AssetWithdrawn),
					RunePending:    util.IntStr(memberPool.RunePending),
					AssetPending:   util.IntStr(memberPool.AssetPending),
					DateFirstAdded: util.IntStr(memberPool.DateFirstAdded),
					DateLastAdded:  util.IntStr(memberPool.DateLastAdded),
				})
			}
		}
	}
	respJSON(w, allPools)
}

func jsonLPDetails(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	addr := ps[0].Value

	urlParams := r.URL.Query()
	poolsStr := util.ConsumeUrlParam(&urlParams, "pools")
	if poolsStr == "" {
		http.Error(w, "Invalid pools", http.StatusBadRequest)
		return
	}
	pools := strings.Split(poolsStr, ",")
	allPools, err := timeseries.GetFullMemberPools(r.Context(), addr)
	if err != nil {
		respError(w, err)
		return
	}
	var selectedPools []timeseries.MemberPool
	for _, memberPool := range allPools {
		for _, pool := range pools {
			if memberPool.Pool == pool {
				selectedPools = append(selectedPools, memberPool)
			}
		}
	}
	if len(selectedPools) == 0 {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	aggregates, err := getPoolAggregates(r.Context(), pools)
	if err != nil {
		miderr.InternalErrE(err).ReportHTTP(w)
		return
	}
	var lpDetails []oapigen.LPDetail
	for _, memberPool := range selectedPools {
		var lpDetail []timeseries.LPDetail
		lpDetail, err = timeseries.GetLpDetail(r.Context(), memberPool.RuneAddress, memberPool.AssetAddress, memberPool.Pool)
		if err != nil {
			respError(w, err)
			return
		}
		for i := 0; i < len(lpDetail); i++ {
			for j := i + 1; j < len(lpDetail); j++ {
				if lpDetail[i].Date > lpDetail[j].Date {
					temp := lpDetail[i]
					lpDetail[i] = lpDetail[j]
					lpDetail[j] = temp
				}
			}
		}
		for i := 0; i < len(lpDetail); i++ {
			start := lpDetail[i].Date
			end := int64(-1)
			if i+1 < len(lpDetail) {
				end = lpDetail[i+1].Date
			}
			res, err := stat.GetPoolSwapsFee(r.Context(), memberPool.Pool, start, end)
			if err != nil {
				miderr.InternalErrE(err).ReportHTTP(w)
				return
			}
			rewards, err := stat.GetPoolRewards(r.Context(), memberPool.Pool, start, end)
			if err != nil {
				miderr.InternalErrE(err).ReportHTTP(w)
				return
			}
			lpDetail[i].RuneLiquidityFee = res.RuneAmount
			lpDetail[i].AssetLiquidityFee = res.AssetAmount
			lpDetail[i].BlockRewards = rewards
		}
		assetPrice := float64(aggregates.runeE8DepthPerPool[memberPool.Pool]) / float64(aggregates.assetE8DepthPerPool[memberPool.Pool])
		runePrice := stat.RunePriceUSD()

		units := int64(0)
		stakeDetail := make([]oapigen.StakeDetail, 0)
		withdrawDetail := make([]oapigen.StakeDetail, 0)
		stakedAsset := int64(0)
		stakedRune := int64(0)
		stakedUsd := int64(0)
		runeFees := float64(0)
		assetFees := float64(0)
		usdFees := float64(0)
		rewards := float64(0)
		totalUsd := float64(0)
		for _, lp := range lpDetail {
			units += lp.LiquidityUnits
			totalUsd = float64(units) * float64(aggregates.assetE8DepthPerPool[memberPool.Pool]) * assetPrice * runePrice
			totalUsd += float64(units) * float64(aggregates.runeE8DepthPerPool[memberPool.Pool]) * runePrice
			if totalUsd <= 1 || units == 0 {
				units = int64(0)
				stakeDetail = make([]oapigen.StakeDetail, 0)
				withdrawDetail = make([]oapigen.StakeDetail, 0)
				stakedAsset = int64(0)
				stakedRune = int64(0)
				stakedUsd = int64(0)
				runeFees = float64(0)
				assetFees = float64(0)
				usdFees = float64(0)
				rewards = float64(0)
				totalUsd = float64(0)
				continue
			}
			runeFee := float64(lp.RuneLiquidityFee) * float64(units) / float64(lp.PoolUnit)
			assetFee := float64(lp.AssetLiquidityFee) * float64(units) / float64(lp.PoolUnit)
			usdFee := runeFee*lp.RunePriceUsd + assetFee*lp.AssetPriceUsd
			runeFees += runeFee
			assetFees += assetFee
			usdFees += usdFee
			rewards += float64(lp.BlockRewards) * float64(units) / float64(aggregates.liquidityUnits[memberPool.Pool])
			if lp.AssetAdded > 0 || lp.RuneAdded > 0 {
				stakeDetail = append(stakeDetail, oapigen.StakeDetail{
					AssetAmount:   util.IntStr(lp.AssetAdded),
					RuneAmount:    util.IntStr(lp.RuneAdded),
					AssetPriceUsd: floatStr(lp.AssetPriceUsd),
					RunePriceUsd:  floatStr(lp.RunePriceUsd),
					AssetPrice:    floatStr(lp.AssetPriceUsd / lp.RunePriceUsd),
					Date:          util.IntStr(lp.Date),
					Height:        util.IntStr(lp.Height),
					AssetDepth:    util.IntStr(lp.AssetDepth),
					RuneDepth:     util.IntStr(lp.RuneDepth),
					PoolUnits:     util.IntStr(lp.PoolUnit),
					SharedUnits:   util.IntStr(lp.LiquidityUnits),
				})
			} else {
				withdrawDetail = append(withdrawDetail, oapigen.StakeDetail{
					AssetAmount:   util.IntStr(lp.AssetWithdrawn),
					RuneAmount:    util.IntStr(lp.RuneWithdrawn),
					AssetPriceUsd: floatStr(lp.AssetPriceUsd),
					RunePriceUsd:  floatStr(lp.RunePriceUsd),
					AssetPrice:    floatStr(lp.AssetPriceUsd / lp.RunePriceUsd),
					Date:          util.IntStr(lp.Date),
					Height:        util.IntStr(lp.Height),
					AssetDepth:    util.IntStr(lp.AssetDepth),
					RuneDepth:     util.IntStr(lp.RuneDepth),
					PoolUnits:     util.IntStr(lp.PoolUnit),
					SharedUnits:   util.IntStr(lp.LiquidityUnits),
					BasisPoint:    floatStr(10000 * float64(-1*lp.LiquidityUnits) / float64(units-lp.LiquidityUnits)),
				})
			}
			stakedAsset = stakedAsset + lp.AssetAdded - lp.AssetWithdrawn
			stakedRune = stakedRune + lp.RuneAdded - lp.RuneWithdrawn
			stakedUsd = stakedUsd + int64(float64(lp.AssetAdded-lp.AssetWithdrawn)*lp.AssetPriceUsd)
			stakedUsd = stakedUsd + int64(float64(lp.RuneAdded-lp.RuneWithdrawn)*lp.RunePriceUsd)
		}

		lpDetails = append(lpDetails, oapigen.LPDetail{
			SharedUnits:    util.IntStr(memberPool.LiquidityUnits),
			StakeDetail:    stakeDetail,
			WithdrawDetail: withdrawDetail,
			AssetEarned:    floatStr(assetFees),
			RuneEarned:     floatStr(runeFees),
			UsdEarned:      floatStr(usdFees),
			Rewards:        floatStr(rewards),
			Pool:           memberPool.Pool,
			RuneAddress:    memberPool.RuneAddress,
			AssetAddress:   memberPool.AssetAddress,
		})
	}
	assetE8DepthPerPool, runeE8DepthPerPool, _, _ := timeseries.AllDepths()
	liquidityUnits, err := stat.CurrentPoolsLiquidityUnits(ctx, pools)
	if err != nil {
		miderr.InternalErrE(err).ReportHTTP(w)
		return
	}
	for i, lp := range lpDetails {
		assetPrice := float64(runeE8DepthPerPool[lp.Pool]) / float64(assetE8DepthPerPool[lp.Pool])
		runePrice := stat.RunePriceUSD()
		lpDetails[i].AssetDepth = util.IntStr(assetE8DepthPerPool[lp.Pool])
		lpDetails[i].RuneDepth = util.IntStr(runeE8DepthPerPool[lp.Pool])
		lpDetails[i].AssetPriceUsd = floatStr(assetPrice * runePrice)
		lpDetails[i].AssetPrice = floatStr(assetPrice)
		lpDetails[i].RunePriceUsd = floatStr(runePrice)
		state := timeseries.Latest.GetState()
		poolInfo := state.PoolInfo(lp.Pool)
		synthUnits := timeseries.GetSinglePoolSynthUnits(ctx, poolInfo.AssetDepth, poolInfo.SynthDepth, liquidityUnits[lp.Pool])
		lpDetails[i].PoolUnits = util.IntStr(liquidityUnits[lp.Pool] + synthUnits)
	}
	respJSON(w, lpDetails)
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

	caseSensitiveAddr := ps[0].Value

	var names []string
	for _, addr := range []string{caseSensitiveAddr, strings.ToLower(caseSensitiveAddr)} {
		var err error
		names, err = timeseries.GetTHORNamesByAddress(r.Context(), &addr)
		if err != nil {
			respError(w, err)
			return
		}
		if 0 < len(names) {
			break
		}
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

// TODO(muninn): measure which part of this funcion is slow
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
			Address:    util.ConsumeUrlParam(&urlParams, "address"),
			TXId:       util.ConsumeUrlParam(&urlParams, "txid"),
			Asset:      util.ConsumeUrlParam(&urlParams, "asset"),
		}
		merr := util.CheckUrlEmpty(urlParams)
		if merr != nil {
			merr.ReportHTTP(w)
			return
		}

		// Get results
		actions, err := timeseries.GetActions(r.Context(), time.Time{}, params)
		// Send response
		if err != nil {
			respError(w, err)
			return
		}

		// check for lowercase address
		if len(actions.Actions) == 0 {
			params.Address = strings.ToLower(params.Address)
			actions, err = timeseries.GetActions(r.Context(), time.Time{}, params)
			if err != nil {
				respError(w, err)
				return
			}
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
