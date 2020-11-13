package api

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"path"
	"sort"
	"strconv"
	"time"

	"gitlab.com/thorchain/midgard/chain/notinchain"
	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/internal/timeseries/stat"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

// Version 1 compatibility is a minimal effort attempt to provide smooth migration.

// InSync returns whether the entire blockchain is processed.
var InSync func() bool

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

func jsonVolume(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	from, err := convertStringToTime(query.Get("from"))
	if err != nil {
		http.Error(w, fmt.Errorf("Invalid query parameter: from (%v)", err).Error(), http.StatusBadRequest)
		return
	}
	to, err := convertStringToTime(query.Get("to"))
	if err != nil {
		http.Error(w, fmt.Errorf("Invalid query parameter: to (%v)", err).Error(), http.StatusBadRequest)
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

func jsonNetwork(w http.ResponseWriter, r *http.Request) {
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
	weeklyBlockRewards := yearlyBlockRewards / weeksInYear

	weeklyTotalIncome := weeklyBlockRewards + float64(weeklyLiquidityFeesRune)
	weeklyBondIncome := weeklyTotalIncome * (1 - poolShareFactor)
	weeklyPoolIncome := weeklyTotalIncome * poolShareFactor

	var bondingAPY float64
	if bondMetrics.TotalActiveBond > 0 {
		weeklyBondingRate := weeklyBondIncome / float64(bondMetrics.TotalActiveBond)
		bondingAPY = calculateAPY(weeklyBondingRate, weeksInYear)
	}

	var liquidityAPY float64
	if runeDepth > 0 {
		poolDepthInRune := float64(2 * runeDepth)
		weeklyPoolRate := weeklyPoolIncome / poolDepthInRune
		liquidityAPY = calculateAPY(weeklyPoolRate, weeksInYear)
	}

	// BUILD RESPONSE
	respJSON(w, oapigen.Network{
		ActiveBonds:     intArrayStrs(activeBonds),
		ActiveNodeCount: intStr(int64(len(activeNodes))),
		BlockRewards: oapigen.BlockRewards{
			BlockReward: intStr(blockRewards.BlockReward),
			BondReward:  intStr(blockRewards.BondReward),
			PoolReward:  intStr(blockRewards.PoolReward),
		},
		// TODO(acsaba): create bondmetrics right away with this type.
		BondMetrics: oapigen.BondMetrics{
			TotalActiveBond:    intStr(bondMetrics.TotalActiveBond),
			AverageActiveBond:  intStr(bondMetrics.AverageActiveBond),
			MedianActiveBond:   intStr(bondMetrics.MedianActiveBond),
			MinimumActiveBond:  intStr(bondMetrics.MinimumActiveBond),
			MaximumActiveBond:  intStr(bondMetrics.MaximumActiveBond),
			TotalStandbyBond:   intStr(bondMetrics.TotalStandbyBond),
			AverageStandbyBond: intStr(bondMetrics.AverageStandbyBond),
			MedianStandbyBond:  intStr(bondMetrics.MedianStandbyBond),
			MinimumStandbyBond: intStr(bondMetrics.MinimumStandbyBond),
			MaximumStandbyBond: intStr(bondMetrics.MaximumStandbyBond),
		},
		BondingAPY:              floatStr(bondingAPY),
		LiquidityAPY:            floatStr(liquidityAPY),
		NextChurnHeight:         intStr(nextChurnHeight),
		PoolActivationCountdown: intStr(newPoolCycle - currentHeight%newPoolCycle),
		PoolShareFactor:         floatStr(poolShareFactor),
		StandbyBonds:            intArrayStrs(standbyBonds),
		StandbyNodeCount:        intStr(int64(len(standbyNodes))),
		TotalReserve:            intStr(vaultData.TotalReserve),
		TotalPooledRune:         intStr(runeDepth),
	})
}

type sortedBonds []int64

func (b sortedBonds) Len() int           { return len(b) }
func (b sortedBonds) Less(i, j int) bool { return b[i] < b[j] }
func (b sortedBonds) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }

type bondMetricsInts struct {
	TotalActiveBond   int64
	MinimumActiveBond int64
	MaximumActiveBond int64
	AverageActiveBond int64
	MedianActiveBond  int64

	TotalStandbyBond   int64
	MinimumStandbyBond int64
	MaximumStandbyBond int64
	AverageStandbyBond int64
	MedianStandbyBond  int64
}

func activeAndStandbyBondMetrics(active, standby sortedBonds) *bondMetricsInts {
	var metrics bondMetricsInts
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

type blockRewardsInts struct {
	BlockReward int64
	BondReward  int64
	PoolReward  int64
}

func calculateBlockRewards(emissionCurve int64, blocksPerYear int64, totalReserve int64, poolShareFactor float64) *blockRewardsInts {

	blockReward := float64(totalReserve) / float64(emissionCurve*blocksPerYear)
	bondReward := (1 - poolShareFactor) * blockReward
	poolReward := blockReward - bondReward

	rewards := blockRewardsInts{int64(blockReward), int64(bondReward), int64(poolReward)}
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

func jsonNodes(w http.ResponseWriter, r *http.Request) {
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

	array := make([]oapigen.NodeKey, 0, len(m))
	for _, e := range m {
		array = append(array, oapigen.NodeKey{
			Secp256k1: e.Secp,
			Ed25519:   e.Ed,
		})
	}
	respJSON(w, array)
}

func jsonPools(w http.ResponseWriter, r *http.Request) {
	// NOTE: this DateCreated field relates to the first time a stake event is seen
	//	for a pool. This technically is not the true creation date for every pool since
	//	some (native chain asset pools) are created during the Genesis block.
	//	Not sure if that distinction is worth being made or not.
	//	If it does we should also query date pool was Enabled and do a min between both dates
	pools, err := timeseries.PoolsWithDateCreated(r.Context())
	if err != nil {
		respError(w, r, err)
		return
	}
	assetE8DepthPerPool, runeE8DepthPerPool, _ := timeseries.AssetAndRuneDepths()

	poolsResponse := make(oapigen.PoolsResponse, len(pools))
	for i, pool := range pools {
		assetDepth := assetE8DepthPerPool[pool.Asset]
		runeDepth := runeE8DepthPerPool[pool.Asset]
		m := oapigen.PoolSummary{
			Asset:       pool.Asset,
			AssetDepth:  intStr(assetDepth),
			DateCreated: intStr(pool.DateCreated),
			RuneDepth:   intStr(runeDepth),
		}
		if assetDepth != 0 {
			m.Price = floatStr(float64(runeDepth) / float64(assetDepth))
		}
		poolsResponse[i] = m
	}

	respJSON(w, poolsResponse)
}

const weeksInYear = 365. / 7

func jsonPoolDetails(w http.ResponseWriter, r *http.Request) {
	pool := path.Base(r.URL.Path)

	if pool == "detail" {
		// TODO(acsaba):
		//	- Delete this endpoint.
		//	- Submit a PR to Heimdall to use /v2/pools
		//		to get midgard data instead when deleting
		jsonPools(w, r)
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

	// TODO(acsaba): make this calculation the same as priceRune form /v2/asset
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

	poolWeeklyRewards, err := timeseries.PoolTotalIncome(r.Context(), pool, timestamp.Add(-1*time.Hour*24*7), timestamp)
	if err != nil {
		respError(w, r, err)
		return
	}

	// NOTE(elfedy): By definition a pool has the same amount of asset
	// and rune because assetPrice = RuneDepth / AssetDepth
	// hence total assets meassured in RUNE = 2 * RuneDepth
	poolRate := float64(poolWeeklyRewards) / (2 * float64(runeDepthE8))
	poolAPY := calculateAPY(poolRate, weeksInYear)

	poolData := oapigen.PoolDetailResponse{
		Volume24h:  intStr(dailyVolume),
		Asset:      pool,
		AssetDepth: intStr(assetDepthE8),
		PoolAPY:    floatStr(poolAPY),
		Price:      floatStr(price),
		RuneDepth:  intStr(runeDepthE8),
		Status:     status,
		Units:      intStr(poolUnits),
	}

	respJSON(w, poolData)
}

// returns string array
func jsonMembers(w http.ResponseWriter, r *http.Request) {
	addrs, err := timeseries.StakeAddrs(r.Context(), time.Time{})
	if err != nil {
		respError(w, r, err)
		return
	}
	result := oapigen.MembersResponse(addrs)
	respJSON(w, result)
}

func jsonMemberDetails(w http.ResponseWriter, r *http.Request) {
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
	respJSON(w, oapigen.MemberDetailsResponse{
		StakeArray:  assets,
		TotalStaked: intStr(runeE8Total)},
	)
}

func jsonStats(w http.ResponseWriter, r *http.Request) {
	_, runeE8DepthPerPool, timestamp := timeseries.AssetAndRuneDepths()
	window := stat.Window{From: time.Unix(0, 0), Until: timestamp}

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

	respJSON(w, oapigen.StatsResponse{
		DailyActiveUsers:   intStr(dailySwapsFromRune.RuneAddrCount + dailySwapsToRune.RuneAddrCount),
		DailyTx:            intStr(dailySwapsFromRune.TxCount + dailySwapsToRune.TxCount),
		MonthlyActiveUsers: intStr(monthlySwapsFromRune.RuneAddrCount + monthlySwapsToRune.RuneAddrCount),
		MonthlyTx:          intStr(monthlySwapsFromRune.TxCount + monthlySwapsToRune.TxCount),
		TotalAssetBuys:     intStr(swapsFromRune.TxCount),
		TotalAssetSells:    intStr(swapsToRune.TxCount),
		TotalDepth:         intStr(runeDepth),
		TotalUsers:         intStr(swapsFromRune.RuneAddrCount + swapsToRune.RuneAddrCount),
		TotalStakeTx:       intStr(stakes.TxCount + unstakes.TxCount),
		TotalStaked:        intStr(stakes.RuneE8Total - unstakes.RuneE8Total),
		TotalTx:            intStr(swapsFromRune.TxCount + swapsToRune.TxCount + stakes.TxCount + unstakes.TxCount),
		TotalVolume:        intStr(swapsFromRune.RuneE8Total + swapsToRune.RuneE8Total),
		TotalWithdrawTx:    intStr(unstakes.RuneE8Total),
	})
	/* TODO(pascaldekloe)
	   "poolCount":"20",
	   "totalEarned":"1827445688454",
	   "totalVolume24hr":"37756279870656",
	*/
}

func jsonTx(w http.ResponseWriter, r *http.Request) {
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

func jsonSwagger(w http.ResponseWriter, r *http.Request) {
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

func convertStringToTime(input string) (time.Time, error) {
	i, err := strconv.ParseInt(input, 10, 64)
	if err != nil {
		return time.Time{}, err
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

func floatStr(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
