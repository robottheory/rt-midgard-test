package api

import (
	"encoding/json"
	"log"
	"math/big"
	"net/http"
	"path"
	"strconv"
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

	assetE8DepthPerPool, runeE8DepthPerPool, timestamp := timeseries.AssetAndRuneDepths()
	window := stat.Window{time.Unix(0, 0), timestamp}

	status, err := stat.PoolStatusLookup(asset)
	if err != nil {
		respError(w, r, err)
		return
	}
	poolStakes, err := stat.PoolStakesLookup(asset, window)
	if err != nil {
		respError(w, r, err)
		return
	}
	buySwaps, err := stat.PoolSellSwapsLookup(asset, window)
	if err != nil {
		respError(w, r, err)
		return
	}
	sellSwaps, err := stat.PoolSellSwapsLookup(asset, window)
	if err != nil {
		respError(w, r, err)
		return
	}
	assetUnstakes, err := stat.PoolAssetUnstakesLookup(asset, window)
	if err != nil {
		respError(w, r, err)
		return
	}
	runeUnstakes, err := stat.PoolRuneUnstakesLookup(asset, window)
	if err != nil {
		respError(w, r, err)
		return
	}

	assetDepth := assetE8DepthPerPool[asset]
	runeDepth := runeE8DepthPerPool[asset]
	priceInRune := big.NewRat(assetDepth, runeDepth)

	assetROI := new(big.Rat)
	runeROI := new(big.Rat)
	if poolStakes.AssetE8Total != 0 {
		assetROI.SetFrac64(assetDepth-poolStakes.AssetE8Total, poolStakes.AssetE8Total)
	}
	if poolStakes.RuneE8Total != 0 {
		runeROI.SetFrac64(runeDepth-poolStakes.RuneE8Total, poolStakes.RuneE8Total)
	}
	poolStakedTotal := big.NewRat(poolStakes.AssetE8Total, 1)
	poolStakedTotal.Mul(poolStakedTotal, priceInRune)
	poolStakedTotal.Add(poolStakedTotal, big.NewRat(poolStakes.RuneE8Total, 1))

	buyVolume := big.NewRat(buySwaps.AssetE8Total, 1)
	buyVolume.Mul(buyVolume, priceInRune)
	sellVolume := big.NewRat(sellSwaps.AssetE8Total, 1)
	sellVolume.Mul(sellVolume, priceInRune)
	poolVolume := big.NewRat(buySwaps.AssetE8Total+sellSwaps.AssetE8Total, 1)
	poolVolume.Mul(poolVolume, priceInRune)

	m := map[string]interface{}{
		"asset":            asset,
		"assetDepth":       intStr(assetDepth),
		"assetROI":         ratFloat(assetROI),
		"assetStakedTotal": intStr(poolStakes.AssetE8Total),
		"buyAssetCount":    intStr(buySwaps.TxCount),
		"buyFeesTotal":     intStr(buySwaps.LiqFeeE8Total),
		"buyVolume":        ratIntStr(buyVolume),
		"poolDepth":        intStr(2 * runeDepth),
		"poolFeesTotal":    intStr(buySwaps.LiqFeeE8Total + sellSwaps.LiqFeeE8Total),
		"poolROI":          ratFloat(new(big.Rat).Mul(big.NewRat(1, 2), new(big.Rat).Add(assetROI, runeROI))),
		"poolStakedTotal":  ratIntStr(poolStakedTotal),
		"poolUnits":        intStr(poolStakes.StakeUnitsTotal),
		"poolVolume":       ratIntStr(poolVolume),
		"price":            ratFloat(priceInRune),
		"runeDepth":        intStr(runeDepth),
		"runeROI":          ratFloat(runeROI),
		"runeStakedTotal":  intStr(poolStakes.RuneE8Total),
		"sellAssetCount":   intStr(sellSwaps.TxCount),
		"sellFeesTotal":    intStr(sellSwaps.LiqFeeE8Total),
		"sellVolume":       ratIntStr(sellVolume),
		"stakeTxCount":     intStr(poolStakes.TxCount),
		"stakingTxCount":   intStr(poolStakes.TxCount + assetUnstakes.TxCount + runeUnstakes.TxCount),
		"status":           status,
		"swappingTxCount":  intStr(buySwaps.TxCount + sellSwaps.TxCount),
		"withdrawTxCount":  intStr(assetUnstakes.TxCount + runeUnstakes.TxCount),
	}

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
	StakersCount     uint64
	SwappersCount    uint64
	*/

	respJSON(w, m)
}

func serveV1Stakers(w http.ResponseWriter, r *http.Request) {
	addrStakes, err := stat.AllAddrStakesLookup(time.Now())
	if err != nil {
		respError(w, r, err)
		return
	}

	array := make([]string, len(addrStakes))
	for i, stakes := range addrStakes {
		array[i] = stakes.Addr
	}
	respJSON(w, array)
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
