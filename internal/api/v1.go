package api

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math"
	"math/big"
	"net/http"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"gitlab.com/thorchain/midgard/chain/notinchain"
	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/internal/timeseries/stat"
)

// Version 1 compatibility is a minimal effort attempt to provide smooth migration.

// InSync returns whether the entire blockchain is processed.
var InSync func() bool

func serveV1Assets(w http.ResponseWriter, r *http.Request) {
	assets, err := assetParam(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	assetE8DepthPerPool, runeE8DepthPerPool, timestamp := timeseries.AssetAndRuneDepths()
	window := stat.Window{time.Unix(0, 0), timestamp}

	array := make([]interface{}, len(assets))
	for i, asset := range assets {
		stakes, err := stat.PoolStakesLookup(r.Context(), asset, window)
		if err != nil {
			respError(w, r, err)
			return
		}
		m := map[string]interface{}{
			"asset":       asset,
			"dateCreated": stakes.First.Unix(),
		}
		if assetDepth := assetE8DepthPerPool[asset]; assetDepth != 0 {
			m["priceRune"] = strconv.FormatFloat(float64(runeE8DepthPerPool[asset])/float64(assetDepth), 'f', -1, 64)
		}
		array[i] = m
	}

	respJSON(w, array)
}

func serveV1Health(w http.ResponseWriter, r *http.Request) {
	height, _, _ := timeseries.LastBlock()
	respJSON(w, map[string]interface{}{
		"database":      true,
		"scannerHeight": height + 1,
		"catching_up":   !InSync(),
	})
}

func serveV1TotalVolume(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	from, err := convertStringToTime(query.Get("from"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	to, err := convertStringToTime(query.Get("to"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	interval := query.Get("interval")
	if interval == "" {
		http.Error(w, "'interval' parameter is required", http.StatusBadRequest)
		return
	}
	pool := query.Get("pool")
	if pool == "" {
		pool = "*"
	}

	res, err := stat.TotalVolumeChanges(r.Context(), interval, pool, from, to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respJSON(w, res)
}

func serveV1Network(w http.ResponseWriter, r *http.Request) {
	// GET DATA
	// in memory lookups
	_, runeE8DepthPerPool, _ := timeseries.AssetAndRuneDepths()
	var runeDepth int64
	for _, depth := range runeE8DepthPerPool {
		runeDepth += depth
	}
	currentHeight, _, _ := timeseries.LastBlock()

	// db lookups
	lastChurnHeight, err := timeseries.LastChurnHeight(r.Context())
	if err != nil {
		respError(w, r, err)
		return
	}

	// Thorchain constants
	emissionCurve, err := timeseries.GetLastConstantValue(r.Context(), "EmissionCurve")
	if err != nil {
		respError(w, r, err)
		return
	}
	blocksPerYear, err := timeseries.GetLastConstantValue(r.Context(), "BlocksPerYear")
	if err != nil {
		respError(w, r, err)
		return
	}
	rotatePerBlockHeight, err := timeseries.GetLastConstantValue(r.Context(), "RotatePerBlockHeight")
	if err != nil {
		respError(w, r, err)
		return
	}
	rotateRetryBlocks, err := timeseries.GetLastConstantValue(r.Context(), "RotateRetryBlocks")
	if err != nil {
		respError(w, r, err)
		return
	}
	newPoolCycle, err := timeseries.GetLastConstantValue(r.Context(), "NewPoolCycle")
	if err != nil {
		respError(w, r, err)
		return
	}

	// Thornode queries
	nodes, err := notinchain.NodeAccountsLookup()
	if err != nil {
		respError(w, r, err)
		return
	}
	vaultData, err := notinchain.VaultDataLookup()
	if err != nil {
		respError(w, r, err)
		return
	}

	// PROCESS DATA
	activeNodes := make(map[string]struct{})
	standbyNodes := make(map[string]struct{})
	var activeBonds, standbyBonds sortedBonds
	for _, node := range nodes {
		switch node.Status {
		case "active":
			activeNodes[node.NodeAddr] = struct{}{}
			activeBonds = append(activeBonds, node.Bond)
		case "standby":
			standbyNodes[node.NodeAddr] = struct{}{}
			standbyBonds = append(standbyBonds, node.Bond)
		}
	}
	sort.Sort(activeBonds)
	sort.Sort(standbyBonds)

	bondMetrics := activeAndStandbyBondMetrics(activeBonds, standbyBonds)

	var poolShareFactor float64
	if bondMetrics.TotalActiveBond > runeDepth {
		poolShareFactor = float64(bondMetrics.TotalActiveBond-runeDepth) / float64(bondMetrics.TotalActiveBond+runeDepth)
	}

	blockRewards := calculateBlockRewards(emissionCurve, blocksPerYear, vaultData.TotalReserve, poolShareFactor)

	nextChurnHeight := calculateNextChurnHeight(currentHeight, lastChurnHeight, rotatePerBlockHeight, rotateRetryBlocks)

	yearlyBondRewards := float64(blockRewards.BondReward * blocksPerYear)
	monthlyBondRewards := yearlyBondRewards / 12
	yearlyPoolRewards := float64(blockRewards.PoolReward * blocksPerYear)
	monthlyPoolRewards := yearlyPoolRewards / 12

	bondingROI := yearlyBondRewards / float64(bondMetrics.TotalActiveBond)
	bondingAPY := math.Pow(1+monthlyBondRewards/float64(bondMetrics.TotalActiveBond), 12) - 1

	// TODO(elfedy): Maybe pool depth should be 2*runeDepth to
	// account for pooled assets.
	// Also check if we need to only count enabled pools
	poolDepthInRune := float64(runeDepth)
	poolROI := yearlyPoolRewards / poolDepthInRune
	poolAPY := math.Pow(1+monthlyPoolRewards/poolDepthInRune, 12) - 1

	// BUILD RESPONSE
	respJSON(w, map[string]interface{}{
		"activeBonds":             intArrayStrs([]int64(activeBonds)),
		"activeNodeCount":         len(activeNodes),
		"blockRewards":            *blockRewards,
		"bondMetrics":             *bondMetrics,
		"bondingROI":              floatStr(bondingROI),
		"bondingAPY":              floatStr(bondingAPY),
		"poolROI":                 floatStr(poolROI),
		"poolAPY":                 floatStr(poolAPY),
		"nextChurnHeight":         intStr(nextChurnHeight),
		"poolActivationCountdown": newPoolCycle - currentHeight%newPoolCycle,
		"poolShareFactor":         floatStr(poolShareFactor),
		"standbyBonds":            intArrayStrs([]int64(standbyBonds)),
		"standbyNodeCount":        strconv.Itoa(len(standbyNodes)),
		"totalReserve":            intStr(vaultData.TotalReserve),
		"totalPooled":             intStr(runeDepth),
	})
}

type sortedBonds []int64

func (b sortedBonds) Len() int           { return len(b) }
func (b sortedBonds) Less(i, j int) bool { return b[i] < b[j] }
func (b sortedBonds) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }

type BondMetrics struct {
	TotalActiveBond   int64 `json:"totalActiveBond,string"`
	MinimumActiveBond int64 `json:"minimumActiveBond,string"`
	MaximumActiveBond int64 `json:"maximumActiveBond,string"`
	AverageActiveBond int64 `json:"averageActiveBond,string"`
	MedianActiveBond  int64 `json:"medianActiveBond,string"`

	TotalStandbyBond   int64 `json:"totalStandbyBond,string"`
	MinimumStandbyBond int64 `json:"minimumStandbyBond,string"`
	MaximumStandbyBond int64 `json:"maximumStandbyBond,string"`
	AverageStandbyBond int64 `json:"averageStandbyBond,string"`
	MedianStandbyBond  int64 `json:"medianStandbyBond,string"`
}

func activeAndStandbyBondMetrics(active, standby sortedBonds) *BondMetrics {
	var metrics BondMetrics
	if len(active) != 0 {
		var total int64
		for _, n := range active {
			total += n
		}
		metrics.TotalActiveBond = total
		metrics.MinimumActiveBond = active[0]
		metrics.MaximumActiveBond = active[len(active)-1]
		metrics.AverageActiveBond = total / int64(len(active))
		metrics.MedianActiveBond = active[len(active)/2]
	}
	if len(standby) != 0 {
		var total int64
		for _, n := range standby {
			total += n
		}
		metrics.TotalStandbyBond = total
		metrics.MinimumStandbyBond = standby[0]
		metrics.MaximumStandbyBond = standby[len(standby)-1]
		metrics.AverageStandbyBond = total / int64(len(standby))
		metrics.MedianStandbyBond = standby[len(standby)/2]
	}
	return &metrics
}

type BlockRewards struct {
	BlockReward int64 `json:"blockReward,string"`
	BondReward  int64 `json:"bondReward,string"`
	PoolReward  int64 `json:"poolReward,string"`
}

func calculateBlockRewards(emissionCurve int64, blocksPerYear int64, totalReserve int64, poolShareFactor float64) *BlockRewards {

	blockReward := float64(totalReserve) / float64(emissionCurve*blocksPerYear)
	bondReward := (1 - poolShareFactor) * blockReward
	poolReward := blockReward - bondReward

	rewards := BlockRewards{int64(blockReward), int64(bondReward), int64(poolReward)}
	return &rewards
}

func calculateNextChurnHeight(currentHeight int64, lastChurnHeight int64, rotatePerBlockHeight int64, rotateRetryBlocks int64) int64 {
	var next int64
	if currentHeight-lastChurnHeight <= rotatePerBlockHeight {
		next = lastChurnHeight + rotatePerBlockHeight
	} else {
		next = currentHeight + ((currentHeight - lastChurnHeight + rotatePerBlockHeight) % rotateRetryBlocks)
	}
	return next
}

func serveV1Nodes(w http.ResponseWriter, r *http.Request) {
	secpAddrs, edAddrs, err := timeseries.NodesSecpAndEd(r.Context(), time.Now())
	if err != nil {
		respError(w, r, err)
		return
	}

	m := make(map[string]struct {
		Secp string `json:"secp256k1"`
		Ed   string `json:"ed25519"`
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

	array := make([]interface{}, 0, len(m))
	for _, e := range m {
		array = append(array, e)
	}
	respJSON(w, array)
}

func serveV1Pools(w http.ResponseWriter, r *http.Request) {
	pools, err := timeseries.Pools(r.Context(), time.Time{})
	if err != nil {
		respError(w, r, err)
		return
	}
	respJSON(w, pools)
}

func serveV1Pool(w http.ResponseWriter, r *http.Request) {
	pool := path.Base(r.URL.Path)
	if pool == "detail" {
		serveV1PoolsDetail(w, r)
		return
	}

	assetE8DepthPerPool, runeE8DepthPerPool, timestamp := timeseries.AssetAndRuneDepths()

	assetDepthE8, assetOk := assetE8DepthPerPool[pool]
	runeDepthE8, runeOk := runeE8DepthPerPool[pool]

	// Return not found if there's no track of the pool
	if !assetOk && !runeOk {
		http.NotFound(w, r)
		return
	}

	status, err := timeseries.PoolStatus(r.Context(), pool, timestamp)
	if err != nil {
		respError(w, r, err)
		return
	}

	var price float64
	if assetDepthE8 > 0 {
		price = float64(runeDepthE8) / float64(assetDepthE8)
	}

	dailyVolume, err := stat.PoolTotalVolume(r.Context(), pool, timestamp.Add(-24*time.Hour), timestamp)
	if err != nil {
		respError(w, r, err)
		return
	}

	poolUnits, err := timeseries.PoolUnits(r.Context(), pool)
	if err != nil {
		respError(w, r, err)
		return
	}

	poolWeeklyRewards, err := timeseries.PoolTotalRewards(r.Context(), pool, timestamp.Add(-1*time.Hour*24*7), timestamp)
	if err != nil {
		respError(w, r, err)
		return
	}

	// NOTE(elfedy): By definition a pool has the same amount of asset
	// and rune because assetPrice = RuneDepth / AssetDepth
	// hence total assets meassured in RUNE = 2 * RuneDepth
	poolROI := float64(poolWeeklyRewards) * 52 / (2 * float64(runeDepthE8))

	poolData := map[string]string{
		"24hVolume":  intStr(dailyVolume),
		"asset":      pool,
		"assetDepth": intStr(assetDepthE8),
		"poolROI":    floatStr(poolROI),
		"price":      floatStr(price),
		"runeDepth":  intStr(runeDepthE8),
		"status":     status,
		"units":      intStr(poolUnits),
	}

	respJSON(w, poolData)
}

// compatibility layer
// TODO(elfedy): this response is left for now as it is used by smoke tests
// but we are not fully supporting the endpoint so we should submit a PR
// to heimdall using v1/pools/:asset as a source for pool depths and delete this
func serveV1PoolsDetail(w http.ResponseWriter, r *http.Request) {
	assetE8DepthPerPool, runeE8DepthPerPool, timestamp := timeseries.AssetAndRuneDepths()
	window := stat.Window{time.Unix(0, 0), timestamp}

	assets, err := assetParam(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	array := make([]interface{}, len(assets))
	for i, asset := range assets {
		m, err := poolsAsset(r.Context(), asset, assetE8DepthPerPool, runeE8DepthPerPool, window)
		if err != nil {
			respError(w, r, err)
			return
		}
		array[i] = m
	}

	respJSON(w, array)
}

// TODO(elfedy): This function is only used by serveV1PoolDetails now, which should be removed
// in the near future. Remove this as well when appropriate
func poolsAsset(ctx context.Context, asset string, assetE8DepthPerPool, runeE8DepthPerPool map[string]int64, window stat.Window) (map[string]interface{}, error) {
	status, err := timeseries.PoolStatus(ctx, asset, window.Until)
	if err != nil {
		return nil, err
	}
	stakeAddrs, err := timeseries.StakeAddrs(ctx, window.Until)
	if err != nil {
		return nil, err
	}
	stakes, err := stat.PoolStakesLookup(ctx, asset, window)
	if err != nil {
		return nil, err
	}
	unstakes, err := stat.PoolUnstakesLookup(ctx, asset, window)
	if err != nil {
		return nil, err
	}
	swapsFromRune, err := stat.PoolSwapsFromRuneLookup(ctx, asset, window)
	if err != nil {
		return nil, err
	}
	swapsToRune, err := stat.PoolSwapsToRuneLookup(ctx, asset, window)
	if err != nil {
		return nil, err
	}

	assetDepth := assetE8DepthPerPool[asset]
	runeDepth := runeE8DepthPerPool[asset]

	m := map[string]interface{}{
		"asset":            asset,
		"assetDepth":       intStr(assetDepth),
		"assetStakedTotal": intStr(stakes.AssetE8Total),
		"buyAssetCount":    intStr(swapsFromRune.TxCount),
		"buyFeesTotal":     intStr(swapsFromRune.LiqFeeE8Total),
		"poolDepth":        intStr(2 * runeDepth),
		"poolFeesTotal":    intStr(swapsFromRune.LiqFeeE8Total + swapsToRune.LiqFeeE8Total),
		"poolUnits":        intStr(stakes.StakeUnitsTotal - unstakes.StakeUnitsTotal),
		"runeDepth":        intStr(runeDepth),
		"runeStakedTotal":  intStr(stakes.RuneE8Total - unstakes.RuneE8Total),
		"sellAssetCount":   intStr(swapsToRune.TxCount),
		"sellFeesTotal":    intStr(swapsToRune.LiqFeeE8Total),
		"stakeTxCount":     intStr(stakes.TxCount),
		"stakersCount":     strconv.Itoa(len(stakeAddrs)),
		"stakingTxCount":   intStr(stakes.TxCount + unstakes.TxCount),
		"status":           status,
		"swappingTxCount":  intStr(swapsFromRune.TxCount + swapsToRune.TxCount),
		"withdrawTxCount":  intStr(unstakes.TxCount),
	}

	if assetDepth != 0 {
		priceInRune := big.NewRat(runeDepth, assetDepth)
		m["price"] = ratFloatStr(priceInRune)

		poolStakedTotal := big.NewRat(stakes.AssetE8Total-unstakes.AssetE8Total, 1)
		poolStakedTotal.Mul(poolStakedTotal, priceInRune)
		poolStakedTotal.Add(poolStakedTotal, big.NewRat(stakes.RuneE8Total-unstakes.RuneE8Total, 1))
		m["poolStakedTotal"] = ratIntStr(poolStakedTotal)

		buyVolume := big.NewRat(swapsFromRune.AssetE8Total, 1)
		buyVolume.Mul(buyVolume, priceInRune)
		m["buyVolume"] = ratIntStr(buyVolume)

		sellVolume := big.NewRat(swapsToRune.AssetE8Total, 1)
		sellVolume.Mul(sellVolume, priceInRune)
		m["sellVolume"] = ratIntStr(sellVolume)

		poolVolume := big.NewRat(swapsFromRune.AssetE8Total+swapsToRune.AssetE8Total, 1)
		poolVolume.Mul(poolVolume, priceInRune)
		m["poolVolume"] = ratIntStr(poolVolume)

		if n := swapsFromRune.TxCount; n != 0 {
			r := big.NewRat(n, 1)
			r.Quo(buyVolume, r)
			m["buyTxAverage"] = ratFloatStr(r)
		}
		if n := swapsToRune.TxCount; n != 0 {
			r := big.NewRat(n, 1)
			r.Quo(sellVolume, r)
			m["sellTxAverage"] = ratFloatStr(r)
		}
		if n := swapsFromRune.TxCount + swapsToRune.TxCount; n != 0 {
			r := big.NewRat(n, 1)
			r.Quo(poolVolume, r)
			m["poolTxAverage"] = ratFloatStr(r)
		}
	}

	var assetROI, runeROI *big.Rat
	if staked := stakes.AssetE8Total - unstakes.AssetE8Total; staked != 0 {
		assetROI = big.NewRat(assetDepth-staked, staked)
		m["assetROI"] = ratFloatStr(assetROI)
	}
	if staked := stakes.RuneE8Total - unstakes.RuneE8Total; staked != 0 {
		runeROI = big.NewRat(runeDepth-staked, staked)
		m["runeROI"] = ratFloatStr(runeROI)
	}
	if assetROI != nil || runeROI != nil {
		// why an average?
		avg := new(big.Rat)
		avg.Add(assetROI, runeROI)
		avg.Mul(avg, big.NewRat(1, 2))
		m["poolROI"] = ratFloatStr(avg)
	}

	if n := swapsFromRune.TxCount; n != 0 {
		m["buyFeeAverage"] = ratFloatStr(big.NewRat(swapsFromRune.LiqFeeE8Total, n))
	}
	if n := swapsToRune.TxCount; n != 0 {
		m["sellFeeAverage"] = ratFloatStr(big.NewRat(swapsToRune.LiqFeeE8Total, n))
	}
	if n := swapsFromRune.TxCount + swapsToRune.TxCount; n != 0 {
		m["poolFeeAverage"] = ratFloatStr(big.NewRat(swapsFromRune.LiqFeeE8Total+swapsToRune.LiqFeeE8Total, n))
	}

	if n := swapsFromRune.TxCount; n != 0 {
		r := big.NewRat(swapsFromRune.TradeSlipBPTotal, n)
		r.Quo(r, big.NewRat(10000, 1))
		m["buySlipAverage"] = ratFloatStr(r)
	}
	if n := swapsToRune.TxCount; n != 0 {
		r := big.NewRat(swapsToRune.TradeSlipBPTotal, n)
		r.Quo(r, big.NewRat(10000, 1))
		m["sellSlipAverage"] = ratFloatStr(r)
	}
	if n := swapsFromRune.TxCount + swapsToRune.TxCount; n != 0 {
		r := big.NewRat(swapsFromRune.TradeSlipBPTotal+swapsToRune.TradeSlipBPTotal, n)
		r.Quo(r, big.NewRat(10000, 1))
		m["poolSlipAverage"] = ratFloatStr(r)
	}

	/* TODO:
	PoolROI12        float64
	PoolVolume24hr   uint64
	SwappersCount    uint64
	*/

	return m, nil
}

func serveV1Stakers(w http.ResponseWriter, r *http.Request) {
	addrs, err := timeseries.StakeAddrs(r.Context(), time.Time{})
	if err != nil {
		respError(w, r, err)
		return
	}
	respJSON(w, addrs)
}

func serveV1StakersAddr(w http.ResponseWriter, r *http.Request) {
	addr := path.Base(r.URL.Path)
	pools, err := stat.AllPoolStakesAddrLookup(r.Context(), addr, stat.Window{Until: time.Now()})
	if err != nil {
		respError(w, r, err)
		return
	}

	var runeE8Total int64
	assets := make([]string, len(pools))
	for i := range pools {
		assets[i] = pools[i].Asset
		runeE8Total += pools[i].RuneE8Total
	}

	// TODO(pascaldekloe): unstakes

	respJSON(w, map[string]interface{}{
		// TODO(pascaldekloe)
		//“totalEarned” : “123123123”,
		//“totalROI” : “0.20”
		"stakeArray":  assets,
		"totalStaked": intStr(runeE8Total),
	})
}

func serveV1Stats(w http.ResponseWriter, r *http.Request) {
	_, runeE8DepthPerPool, timestamp := timeseries.AssetAndRuneDepths()
	window := stat.Window{time.Unix(0, 0), timestamp}

	stakes, err := stat.StakesLookup(r.Context(), window)
	if err != nil {
		respError(w, r, err)
		return
	}
	unstakes, err := stat.UnstakesLookup(r.Context(), window)
	if err != nil {
		respError(w, r, err)
		return
	}
	swapsFromRune, err := stat.SwapsFromRuneLookup(r.Context(), window)
	if err != nil {
		respError(w, r, err)
		return
	}
	swapsToRune, err := stat.SwapsToRuneLookup(r.Context(), window)
	if err != nil {
		respError(w, r, err)
		return
	}
	dailySwapsFromRune, err := stat.SwapsFromRuneLookup(r.Context(), stat.Window{From: timestamp.Add(-24 * time.Hour), Until: timestamp})
	if err != nil {
		respError(w, r, err)
		return
	}
	dailySwapsToRune, err := stat.SwapsToRuneLookup(r.Context(), stat.Window{From: timestamp.Add(-24 * time.Hour), Until: timestamp})
	if err != nil {
		respError(w, r, err)
		return
	}
	monthlySwapsFromRune, err := stat.SwapsFromRuneLookup(r.Context(), stat.Window{From: timestamp.Add(-30 * 24 * time.Hour), Until: timestamp})
	if err != nil {
		respError(w, r, err)
		return
	}
	monthlySwapsToRune, err := stat.SwapsToRuneLookup(r.Context(), stat.Window{From: timestamp.Add(-30 * 24 * time.Hour), Until: timestamp})
	if err != nil {
		respError(w, r, err)
		return
	}

	var runeDepth int64
	for _, depth := range runeE8DepthPerPool {
		runeDepth += depth
	}

	respJSON(w, map[string]interface{}{
		"dailyActiveUsers":   intStr(dailySwapsFromRune.RuneAddrCount + dailySwapsToRune.RuneAddrCount),
		"dailyTx":            intStr(dailySwapsFromRune.TxCount + dailySwapsToRune.TxCount),
		"monthlyActiveUsers": intStr(monthlySwapsFromRune.RuneAddrCount + monthlySwapsToRune.RuneAddrCount),
		"monthlyTx":          intStr(monthlySwapsFromRune.TxCount + monthlySwapsToRune.TxCount),
		"totalAssetBuys":     intStr(swapsFromRune.TxCount),
		"totalAssetSells":    intStr(swapsToRune.TxCount),
		"totalDepth":         intStr(runeDepth),
		"totalUsers":         intStr(swapsFromRune.RuneAddrCount + swapsToRune.RuneAddrCount),
		"totalStakeTx":       intStr(stakes.TxCount + unstakes.TxCount),
		"totalStaked":        intStr(stakes.RuneE8Total - unstakes.RuneE8Total),
		"totalTx":            intStr(swapsFromRune.TxCount + swapsToRune.TxCount + stakes.TxCount + unstakes.TxCount),
		"totalVolume":        intStr(swapsFromRune.RuneE8Total + swapsToRune.RuneE8Total),
		"totalWithdrawTx":    intStr(unstakes.RuneE8Total),
	})
	/* TODO(pascaldekloe)
	   "poolCount":"20",
	   "totalEarned":"1827445688454",
	   "totalVolume24hr":"37756279870656",
	*/
}

func serveV1Tx(w http.ResponseWriter, r *http.Request) {
	// Parse params
	urlParams := r.URL.Query()
	lookupParamKeys := []string{"limit", "offset", "type", "address", "txid", "asset"}
	lookupParams := make(map[string]string)
	for _, key := range lookupParamKeys {
		val := ""
		if urlParams[key] != nil {
			val = urlParams[key][0]
		}
		lookupParams[key] = val
	}

	// Get results
	txs, err := timeseries.TxList(r.Context(), time.Time{}, lookupParams)

	// Send response
	if err != nil {
		respError(w, r, err)
		return
	}
	respJSON(w, txs)
}

const assetListMax = 10

func assetParam(r *http.Request) ([]string, error) {
	list := strings.Join(r.URL.Query()["asset"], ",")
	if list == "" {
		return nil, errors.New("asset query parameter required")
	}
	assets := strings.SplitN(list, ",", assetListMax+1)
	if len(assets) > assetListMax {
		return nil, errors.New("too many entries in asset query parameter")
	}
	return assets, nil
}

func convertStringToTime(input string) (time.Time, error) {
	i, err := strconv.ParseInt(input, 10, 64)
	if err != nil {
		return time.Time{}, errors.New("invalid input")
	}
	return time.Unix(i, 0), nil
}

func respJSON(w http.ResponseWriter, body interface{}) {
	w.Header().Set("Content-Type", "application/json")

	e := json.NewEncoder(w)
	e.SetIndent("", "\t")
	// Error discarded
	_ = e.Encode(body)
}

func respError(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("HTTP %q %q: %s", r.Method, r.URL.Path, err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

// IntStr returns the value as a decimal string.
// JSON numbers are double-precision floating-points.
// We don't want any unexpected rounding due to the 57-bit limit.
func intStr(v int64) string {
	return strconv.FormatInt(v, 10)
}

func intArrayStrs(a []int64) []string {
	b := make([]string, len(a))
	for i, v := range a {
		b[i] = intStr(v)
	}
	return b
}

// floatStr returns float as string. Used in api responses.
// 123.45 -> "123.45"
func floatStr(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// RatIntStr returs the (rounded) integer value as a decimal string.
// We don't want any unexpected rounding due to the 57-bit limit.
func ratIntStr(v *big.Rat) string {
	return new(big.Int).Div(v.Num(), v.Denom()).String()
}

// RatFloat transforms the rational value, possibly with loss of precision.
func ratFloatStr(r *big.Rat) string {
	f, _ := r.Float64()
	return strconv.FormatFloat(f, 'f', -1, 64)
}
