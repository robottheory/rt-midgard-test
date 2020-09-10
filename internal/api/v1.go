package api

import (
	"encoding/json"
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

// InSync returns whether the entire blockchain is processed.
var InSync func() bool

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
	pool, err := stat.PoolsLookup()
	if err != nil {
		respError(w, r, err)
		return
	}
	respJSON(w, pool)
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

	assets := strings.SplitN(r.URL.Query().Get("asset"), ",", 10)
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
	status, err := stat.PoolStatusLookup(asset)
	if err != nil {
		return nil, err
	}
	stakeAddrs, err := stat.StakeAddrsLookup()
	if err != nil {
		return nil, err
	}
	poolStakes, err := stat.PoolStakesLookup(asset, window)
	if err != nil {
		return nil, err
	}
	buySwaps, err := stat.PoolBuySwapsLookup(asset, window)
	if err != nil {
		return nil, err
	}
	sellSwaps, err := stat.PoolSellSwapsLookup(asset, window)
	if err != nil {
		return nil, err
	}
	assetUnstakes, err := stat.PoolAssetUnstakesLookup(asset, window)
	if err != nil {
		return nil, err
	}
	runeUnstakes, err := stat.PoolRuneUnstakesLookup(asset, window)
	if err != nil {
		return nil, err
	}

	assetDepth := assetE8DepthPerPool[asset]
	runeDepth := runeE8DepthPerPool[asset]

	m := map[string]interface{}{
		"asset":            asset,
		"assetDepth":       intStr(assetDepth),
		"assetStakedTotal": intStr(poolStakes.AssetE8Total),
		"buyAssetCount":    intStr(buySwaps.TxCount),
		"buyFeesTotal":     intStr(buySwaps.LiqFeeE8Total),
		"poolDepth":        intStr(2 * runeDepth),
		"poolFeesTotal":    intStr(buySwaps.LiqFeeE8Total + sellSwaps.LiqFeeE8Total),
		"poolUnits":        intStr(poolStakes.StakeUnitsTotal),
		"runeDepth":        intStr(runeDepth),
		"runeStakedTotal":  intStr(poolStakes.RuneE8Total),
		"sellAssetCount":   intStr(sellSwaps.TxCount),
		"sellFeesTotal":    intStr(sellSwaps.LiqFeeE8Total),
		"stakeTxCount":     intStr(poolStakes.TxCount),
		"stakersCount":     strconv.Itoa(len(stakeAddrs)),
		"stakingTxCount":   intStr(poolStakes.TxCount + assetUnstakes.TxCount + runeUnstakes.TxCount),
		"status":           status,
		"swappingTxCount":  intStr(buySwaps.TxCount + sellSwaps.TxCount),
		"withdrawTxCount":  intStr(assetUnstakes.TxCount + runeUnstakes.TxCount),
	}

	if runeDepth != 0 {
		priceInRune := big.NewRat(assetDepth, runeDepth)
		m["price"] = ratFloat(priceInRune)

		poolStakedTotal := big.NewRat(poolStakes.AssetE8Total, 1)
		poolStakedTotal.Mul(poolStakedTotal, priceInRune)
		poolStakedTotal.Add(poolStakedTotal, big.NewRat(poolStakes.RuneE8Total, 1))
		m["poolStakedTotal"] = ratIntStr(poolStakedTotal)

		buyVolume := big.NewRat(buySwaps.AssetE8Total, 1)
		buyVolume.Mul(buyVolume, priceInRune)
		m["buyVolume"] = ratIntStr(buyVolume)

		sellVolume := big.NewRat(sellSwaps.AssetE8Total, 1)
		sellVolume.Mul(sellVolume, priceInRune)
		m["sellVolume"] = ratIntStr(sellVolume)

		poolVolume := big.NewRat(buySwaps.AssetE8Total+sellSwaps.AssetE8Total, 1)
		poolVolume.Mul(poolVolume, priceInRune)
		m["poolVolume"] = ratIntStr(poolVolume)

		if n := buySwaps.TxCount; n != 0 {
			r := big.NewRat(n, 1)
			r.Quo(buyVolume, r)
			m["buyTxAverage"] = ratFloat(r)
		}
		if n := sellSwaps.TxCount; n != 0 {
			r := big.NewRat(n, 1)
			r.Quo(sellVolume, r)
			m["sellTxAverage"] = ratFloat(r)
		}
		if n := buySwaps.TxCount + sellSwaps.TxCount; n != 0 {
			r := big.NewRat(n, 1)
			r.Quo(poolVolume, r)
			m["poolTxAverage"] = ratFloat(r)
		}
	}

	if poolStakes.AssetE8Total != 0 {
		m["assetROI"] = ratFloat(big.NewRat(assetDepth-poolStakes.AssetE8Total, poolStakes.AssetE8Total))
	}
	if poolStakes.RuneE8Total != 0 {
		m["runeROI"] = ratFloat(big.NewRat(runeDepth-poolStakes.RuneE8Total, poolStakes.RuneE8Total))
	}
	if poolStakes.AssetE8Total != 0 && poolStakes.RuneE8Total != 0 {
		// pool asset ROI
		r := big.NewRat(assetDepth-poolStakes.AssetE8Total, poolStakes.AssetE8Total)
		// add RUNE ROI
		r.Add(r, big.NewRat(runeDepth-poolStakes.RuneE8Total, poolStakes.RuneE8Total))
		// average seems weird, perhaps incorret?
		r.Mul(r, big.NewRat(1, 2))
		m["poolROI"] = ratFloat(r)
	}

	if n := buySwaps.TxCount; n != 0 {
		m["buyFeeAverage"] = ratFloat(big.NewRat(buySwaps.LiqFeeE8Total, n))
	}
	if n := sellSwaps.TxCount; n != 0 {
		m["sellFeeAverage"] = ratFloat(big.NewRat(sellSwaps.LiqFeeE8Total, n))
	}
	if n := buySwaps.TxCount + sellSwaps.TxCount; n != 0 {
		m["poolFeeAverage"] = ratFloat(big.NewRat(buySwaps.LiqFeeE8Total+sellSwaps.LiqFeeE8Total, n))
	}

	if n := buySwaps.TxCount; n != 0 {
		r := big.NewRat(buySwaps.TradeSlipBPTotal, n)
		r.Quo(r, big.NewRat(10000, 1))
		m["buySlipAverage"] = ratFloat(r)
	}
	if n := sellSwaps.TxCount; n != 0 {
		r := big.NewRat(sellSwaps.TradeSlipBPTotal, n)
		r.Quo(r, big.NewRat(10000, 1))
		m["sellSlipAverage"] = ratFloat(r)
	}
	if n := buySwaps.TxCount + sellSwaps.TxCount; n != 0 {
		r := big.NewRat(buySwaps.TradeSlipBPTotal+sellSwaps.TradeSlipBPTotal, n)
		r.Quo(r, big.NewRat(10000, 1))
		m["poolSlipAverage"] = ratFloat(r)
	}

	/* TODO:
	PoolROI12        float64
	PoolVolume24hr   uint64
	SwappersCount    uint64
	*/

	return m, nil
}

func serveV1Stakers(w http.ResponseWriter, r *http.Request) {
	addrs, err := stat.StakeAddrsLookup()
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
func ratFloat(r *big.Rat) float64 {
	f, _ := r.Float64()
	return f
}
