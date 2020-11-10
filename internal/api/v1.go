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
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

// Version 1 compatibility is a minimal effort attempt to provide smooth migration.

// InSync returns whether the entire blockchain is processed.
var InSync func() bool

type Assets struct {
	Asset       string  `json:"asset"`
	DateCreated int64   `json:"dateCreated,string"`
	PriceRune   float64 `json:"priceRune,string,omitempty"`
}

func serveV1Assets(w http.ResponseWriter, r *http.Request) {
	assets, err := assetParam(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	assetE8DepthPerPool, runeE8DepthPerPool, timestamp := timeseries.AssetAndRuneDepths()
	window := stat.Window{time.Unix(0, 0), timestamp}

	array := make([]Assets, len(assets))
	for i, asset := range assets {
		stakes, err := stat.PoolStakesLookup(r.Context(), asset, window)
		if err != nil {
			respError(w, r, err)
			return
		}
		m := Assets{
			Asset:       asset,
			DateCreated: stakes.First.Unix(),
		}
		if assetDepth := assetE8DepthPerPool[asset]; assetDepth != 0 {
			m.PriceRune = float64(runeE8DepthPerPool[asset]) / float64(assetDepth)
		}
		array[i] = m
	}

	respJSON(w, array)
}

type Health struct {
	CatchingUp    bool  `json:"catching_up"`
	Database      bool  `json:"database"`
	ScannerHeight int64 `json:"scannerHeight,string"`
}

func jsonHealth(w http.ResponseWriter, r *http.Request) {
	height, _, _ := timeseries.LastBlock()
	synced := InSync()
	respJSON(w, oapigen.HealthResponse{
		InSync:        synced,
		Database:      true,
		ScannerHeight: intStr(height + 1),
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
		MinimumActiveBond  int64 `json:"minimumActiveBond,string"`
		MaximumActiveBond  int64 `json:"maximumActiveBond,string"`
		AverageActiveBond  int64 `json:"averageActiveBond,string"`
		MedianActiveBond   int64 `json:"medianActiveBond,string"`
		TotalStandbyBond   int64 `json:"totalStandbyBond,string"`
		MinimumStandbyBond int64 `json:"minimumStandbyBond,string"`
		MaximumStandbyBond int64 `json:"maximumStandbyBond,string"`
		AverageStandbyBond int64 `json:"averageStandbyBond,string"`
		MedianStandbyBond  int64 `json:"medianStandbyBond,string"`
	} `json:"bondMetrics"`
	BondingAPY              float64  `json:"bondingAPY,string"`
	NextChurnHeight         int64    `json:"nextChurnHeight,string"`
	LiquidityAPY            float64  `json:"liquidityAPY,string"`
	PoolActivationCountdown int64    `json:"poolActivationCountdown,string"`
	PoolShareFactor         float64  `json:"poolShareFactor,string"`
	StandbyBonds            []string `json:"standbyBonds,string"`
	StandbyNodeCount        int      `json:"standbyNodeCount,string"`
	TotalPooledRune         int64    `json:"totalPooledRune,string"`
	TotalReserve            int64    `json:"totalReserve,string"`
}

func serveV1Network(w http.ResponseWriter, r *http.Request) {
	// GET DATA
	// in memory lookups
	_, runeE8DepthPerPool, timestamp := timeseries.AssetAndRuneDepths()
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

	weeklyLiquidityFeesRune, err := timeseries.TotalLiquidityFeesRune(r.Context(), timestamp.Add(-1*time.Hour*24*7), timestamp)
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

	// Calculate pool/node weekly income and extrapolate to get liquidity/bonding APY
	yearlyBlockRewards := float64(blockRewards.BlockReward * blocksPerYear)
	weeklyBlockRewards := yearlyBlockRewards / 52

	weeklyTotalIncome := weeklyBlockRewards + float64(weeklyLiquidityFeesRune)
	weeklyBondIncome := weeklyTotalIncome * (1 - poolShareFactor)
	weeklyPoolIncome := weeklyTotalIncome * poolShareFactor

	var bondingAPY float64
	if bondMetrics.TotalActiveBond > 0 {
		weeklyBondingRate := weeklyBondIncome / float64(bondMetrics.TotalActiveBond)
		bondingAPY = calculateAPY(weeklyBondingRate, 52)
	}

	var liquidityAPY float64
	if runeDepth > 0 {
		poolDepthInRune := float64(2 * runeDepth)
		weeklyPoolRate := weeklyPoolIncome / poolDepthInRune
		liquidityAPY = calculateAPY(weeklyPoolRate, 52)
	}

	// BUILD RESPONSE
	respJSON(w, Network{
		ActiveBonds:             intArrayStrs(activeBonds),
		ActiveNodeCount:         len(activeNodes),
		BlockRewards:            *blockRewards,
		BondMetrics:             *bondMetrics,
		BondingAPY:              bondingAPY,
		LiquidityAPY:            liquidityAPY,
		NextChurnHeight:         nextChurnHeight,
		PoolActivationCountdown: newPoolCycle - currentHeight%newPoolCycle,
		PoolShareFactor:         poolShareFactor,
		StandbyBonds:            intArrayStrs(standbyBonds),
		StandbyNodeCount:        len(standbyNodes),
		TotalReserve:            vaultData.TotalReserve,
		TotalPooledRune:         runeDepth,
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

type Node struct {
	Secp256K1 string `json:"secp256k1"`
	Ed25519   string `json:"ed25519"`
}

func serveV1Nodes(w http.ResponseWriter, r *http.Request) {
	secpAddrs, edAddrs, err := timeseries.NodesSecpAndEd(r.Context(), time.Now())
	if err != nil {
		respError(w, r, err)
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

	array := make([]Node, 0, len(m))
	for _, e := range m {
		array = append(array, Node{
			Secp256K1: e.Secp,
			Ed25519:   e.Ed,
		})
	}
	respJSON(w, array)
}

// returns string array
func jsonPools(w http.ResponseWriter, r *http.Request) {
	pools, err := timeseries.Pools(r.Context(), time.Time{})
	if err != nil {
		respError(w, r, err)
		return
	}
	respJSON(w, oapigen.PoolsResponse(pools))
}

type Pool struct {
	Two4HVolume int64   `json:"24hVolume,string"`
	Asset       string  `json:"asset"`
	AssetDepth  int64   `json:"assetDepth,string"`
	PoolAPY     float64 `json:"poolAPY,string"`
	Price       float64 `json:"price,string"`
	RuneDepth   int64   `json:"runeDepth,string"`
	Status      string  `json:"status"`
	Units       int64   `json:"units,string"`
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
	poolRate := float64(poolWeeklyRewards) / (2 * float64(runeDepthE8))
	poolAPY := calculateAPY(poolRate, 52)

	poolData := Pool{
		Two4HVolume: dailyVolume,
		Asset:       pool,
		AssetDepth:  assetDepthE8,
		PoolAPY:     poolAPY,
		Price:       price,
		RuneDepth:   runeDepthE8,
		Status:      status,
		Units:       poolUnits,
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

// returns string array
func serveV1Stakers(w http.ResponseWriter, r *http.Request) {
	addrs, err := timeseries.StakeAddrs(r.Context(), time.Time{})
	if err != nil {
		respError(w, r, err)
		return
	}
	respJSON(w, addrs)
}

type StakersAddr struct {
	StakeArray  []string `json:"stakeArray"`
	TotalStaked int64    `json:"totalStaked,string"`
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
	respJSON(w, StakersAddr{
		StakeArray:  assets,
		TotalStaked: runeE8Total},
	)
}

type Stats struct {
	DailyActiveUsers   int64 `json:"dailyActiveUsers,string"`
	DailyTx            int64 `json:"dailyTx,string"`
	MonthlyActiveUsers int64 `json:"monthlyActiveUsers,string"`
	MonthlyTx          int64 `json:"monthlyTx,string"`
	TotalAssetBuys     int64 `json:"totalAssetBuys,string"`
	TotalAssetSells    int64 `json:"totalAssetSells,string"`
	TotalDepth         int64 `json:"totalDepth,string"`
	TotalStakeTx       int64 `json:"totalStakeTx,string"`
	TotalStaked        int64 `json:"totalStaked,string"`
	TotalTx            int64 `json:"totalTx,string"`
	TotalUsers         int64 `json:"totalUsers,string"`
	TotalVolume        int64 `json:"totalVolume,string"`
	TotalWithdrawTx    int64 `json:"totalWithdrawTx,string"`
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

	respJSON(w, Stats{
		DailyActiveUsers:   dailySwapsFromRune.RuneAddrCount + dailySwapsToRune.RuneAddrCount,
		DailyTx:            dailySwapsFromRune.TxCount + dailySwapsToRune.TxCount,
		MonthlyActiveUsers: monthlySwapsFromRune.RuneAddrCount + monthlySwapsToRune.RuneAddrCount,
		MonthlyTx:          monthlySwapsFromRune.TxCount + monthlySwapsToRune.TxCount,
		TotalAssetBuys:     swapsFromRune.TxCount,
		TotalAssetSells:    swapsToRune.TxCount,
		TotalDepth:         runeDepth,
		TotalUsers:         swapsFromRune.RuneAddrCount + swapsToRune.RuneAddrCount,
		TotalStakeTx:       stakes.TxCount + unstakes.TxCount,
		TotalStaked:        stakes.RuneE8Total - unstakes.RuneE8Total,
		TotalTx:            swapsFromRune.TxCount + swapsToRune.TxCount + stakes.TxCount + unstakes.TxCount,
		TotalVolume:        swapsFromRune.RuneE8Total + swapsToRune.RuneE8Total,
		TotalWithdrawTx:    unstakes.RuneE8Total,
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

func serveV1SwaggerJSON(w http.ResponseWriter, r *http.Request) {
	swagger, err := oapigen.GetSwagger()
	if err != nil {
		respError(w, r, err)
		return
	}
	respJSON(w, swagger)
}

func calculateAPY(periodicRate float64, periodsPerYear float64) float64 {
	return math.Pow(1+periodicRate, periodsPerYear) - 1
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
