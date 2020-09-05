package api

import (
	"encoding/json"
	"log"
	"math/big"
	"net/http"
	"path"
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

	buyTxAverage := big.NewRat(buySwaps.TxCount, 1)
	buyTxAverage.Quo(buyVolume, buyTxAverage)
	sellTxAverage := big.NewRat(sellSwaps.TxCount, 1)
	sellTxAverage.Quo(sellVolume, sellTxAverage)
	poolTxAverage := big.NewRat(buySwaps.TxCount+sellSwaps.TxCount, 1)
	poolTxAverage.Quo(poolVolume, poolTxAverage)

	respJSON(w, map[string]interface{}{
		"asset":            asset,
		"assetDepth":       assetDepth,
		"assetROI":         floatRat(assetROI),
		"assetStakedTotal": poolStakes.AssetE8Total,
		"buyAssetCount":    buySwaps.AssetE8Total,
		"buyFeeAverage":    float64(buySwaps.LiqFeeE8Total) / float64(buySwaps.TxCount),
		"buyFeesTotal":     buySwaps.LiqFeeE8Total,
		"buySlipAverage":   float64(buySwaps.TradeSlipBPTotal) / float64(buySwaps.TxCount),
		"buyTxAverage":     floatRat(buyTxAverage),
		"buyVolume":        intRat(buyVolume),
		"poolDepth":        2 * runeDepth,
		"poolFeeAverage":   float64(buySwaps.LiqFeeE8Total+sellSwaps.LiqFeeE8Total) / float64(buySwaps.TxCount+sellSwaps.TxCount),
		"poolFeesTotal":    buySwaps.LiqFeeE8Total + sellSwaps.LiqFeeE8Total,
		"poolROI":          floatRat(new(big.Rat).Mul(big.NewRat(1, 2), new(big.Rat).Add(assetROI, runeROI))),
		"poolSlipAverage":  float64(buySwaps.TradeSlipBPTotal+sellSwaps.TradeSlipBPTotal) / float64(buySwaps.TxCount+sellSwaps.TxCount),
		"poolStakedTotal":  intRat(poolStakedTotal),
		"poolTxAverage":    floatRat(poolTxAverage),
		"poolUnits":        poolStakes.StakeUnitsTotal,
		"poolVolume":       intRat(poolVolume),
		"price":            floatRat(priceInRune),
		"runeDepth":        runeDepth,
		"runeROI":          floatRat(runeROI),
		"runeStakedTotal":  poolStakes.RuneE8Total,
		"sellAssetCount":   sellSwaps.AssetE8Total,
		"sellFeeAverage":   float64(sellSwaps.LiqFeeE8Total) / float64(sellSwaps.TxCount),
		"sellFeesTotal":    sellSwaps.LiqFeeE8Total,
		"sellSlipAverage":  float64(sellSwaps.TradeSlipBPTotal) / float64(sellSwaps.TxCount),
		"sellTxAverage":    floatRat(sellTxAverage),
		"sellVolume":       intRat(sellVolume),
		"stakeTxCount":     poolStakes.TxCount,
		"stakingTxCount":   poolStakes.TxCount + assetUnstakes.TxCount + runeUnstakes.TxCount,
		"status":           status,
		"swappingTxCount":  buySwaps.TxCount + sellSwaps.TxCount,
		"withdrawTxCount":  assetUnstakes.TxCount + runeUnstakes.TxCount,
	})

	/* TODO:
	PoolROI12        float64
	PoolVolume24hr   uint64
	StakersCount     uint64
	SwappersCount    uint64
	*/
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

func intRat(r *big.Rat) int64 {
	return new(big.Int).Div(r.Num(), r.Denom()).Int64()
}

func floatRat(r *big.Rat) float64 {
	f, _ := r.Float64()
	return f
}
