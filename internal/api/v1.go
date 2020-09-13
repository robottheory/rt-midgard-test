package api

import (
	"encoding/json"
	"errors"
	"log"
	"math/big"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

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
		stakes, err := stat.PoolStakesLookup(asset, window)
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

func serveV1Nodes(w http.ResponseWriter, r *http.Request) {
	nodes, err := stat.NodeKeysLookup(time.Now())
	if err != nil {
		respError(w, r, err)
		return
	}

	array := make([]struct {
		S string `json:"secp256k1"`
		E string `json:"ed25519"`
	}, len(nodes))
	for i, n := range nodes {
		array[i].S = n.Secp256k1
		array[i].E = n.Ed25519
	}
	respJSON(w, array)
}

func serveV1Pools(w http.ResponseWriter, r *http.Request) {
	pools, err := timeseries.Pools(time.Time{})
	if err != nil {
		respError(w, r, err)
		return
	}
	respJSON(w, pools)
}

func serveV1PoolsAsset(w http.ResponseWriter, r *http.Request) {
	asset := path.Base(r.URL.Path)
	if asset == "detail" {
		serveV1PoolsDetail(w, r)
		return
	}

	assetE8DepthPerPool, runeE8DepthPerPool, timestamp := timeseries.AssetAndRuneDepths()
	window := stat.Window{time.Unix(0, 0), timestamp}

	m, err := poolsAsset(asset, assetE8DepthPerPool, runeE8DepthPerPool, window)
	if err != nil {
		respError(w, r, err)
		return
	}

	respJSON(w, m)
}

// compatibility layer
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
		m, err := poolsAsset(asset, assetE8DepthPerPool, runeE8DepthPerPool, window)
		if err != nil {
			respError(w, r, err)
			return
		}
		array[i] = m
	}

	respJSON(w, array)
}

func poolsAsset(asset string, assetE8DepthPerPool, runeE8DepthPerPool map[string]int64, window stat.Window) (map[string]interface{}, error) {
	status, err := timeseries.PoolStatus(asset, window.Until)
	if err != nil {
		return nil, err
	}
	stakeAddrs, err := timeseries.StakeAddrs(window.Until)
	if err != nil {
		return nil, err
	}
	stakes, err := stat.PoolStakesLookup(asset, window)
	if err != nil {
		return nil, err
	}
	unstakes, err := stat.PoolUnstakesLookup(asset, window)
	if err != nil {
		return nil, err
	}
	swapsFromRune, err := stat.PoolSwapsFromRuneLookup(asset, window)
	if err != nil {
		return nil, err
	}
	swapsToRune, err := stat.PoolSwapsToRuneLookup(asset, window)
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
	addrs, err := timeseries.StakeAddrs(time.Time{})
	if err != nil {
		respError(w, r, err)
		return
	}
	respJSON(w, addrs)
}

func serveV1StakersAddr(w http.ResponseWriter, r *http.Request) {
	addr := path.Base(r.URL.Path)
	pools, err := stat.AllPoolStakesAddrLookup(addr, stat.Window{Until: time.Now()})
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

	stakes, err := stat.StakesLookup(window)
	if err != nil {
		respError(w, r, err)
		return
	}
	unstakes, err := stat.UnstakesLookup(window)
	if err != nil {
		respError(w, r, err)
		return
	}
	swapsFromRune, err := stat.SwapsFromRuneLookup(window)
	if err != nil {
		respError(w, r, err)
		return
	}
	swapsToRune, err := stat.SwapsToRuneLookup(window)
	if err != nil {
		respError(w, r, err)
		return
	}
	dailySwapsFromRune, err := stat.SwapsFromRuneLookup(stat.Window{Since: timestamp.Add(-24 * time.Hour), Until: timestamp})
	if err != nil {
		respError(w, r, err)
		return
	}
	dailySwapsToRune, err := stat.SwapsToRuneLookup(stat.Window{Since: timestamp.Add(-24 * time.Hour), Until: timestamp})
	if err != nil {
		respError(w, r, err)
		return
	}
	monthlySwapsFromRune, err := stat.SwapsFromRuneLookup(stat.Window{Since: timestamp.Add(-30 * 24 * time.Hour), Until: timestamp})
	if err != nil {
		respError(w, r, err)
		return
	}
	monthlySwapsToRune, err := stat.SwapsToRuneLookup(stat.Window{Since: timestamp.Add(-30 * 24 * time.Hour), Until: timestamp})
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

func respJSON(w http.ResponseWriter, body interface{}) {
	w.Header().Set("Content-Type", "application/json")

	e := json.NewEncoder(w)
	e.SetIndent("", "\t")
	e.Encode(body)
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
