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
	height, _, _, err := timeseries.LastBlock()
	m := map[string]interface{}{
		"database":      err == nil,
		"scannerHeight": height + 1,
		"catching_up":   !InSync(),
	}

	w.Header().Set("content-type", "application/json")
	json.NewEncoder(w).Encode(m)
}

func serveV1Nodes(w http.ResponseWriter, r *http.Request) {
	nodes, err := stat.NodeKeysLookup(time.Now())
	if err != nil {
		errorResp(w, r, err)
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

	w.Header().Set("content-type", "application/json")
	json.NewEncoder(w).Encode(array)
}

func serveV1Pools(w http.ResponseWriter, r *http.Request) {
	pool, err := stat.PoolsLookup()
	if err != nil {
		errorResp(w, r, err)
		return
	}

	w.Header().Set("content-type", "application/json")
	json.NewEncoder(w).Encode(pool)
}

func serveV1PoolsAsset(w http.ResponseWriter, r *http.Request) {
	asset := path.Base(r.URL.Path)

	assetE8DepthPerPool, runeE8DepthPerPool, timestamp := timeseries.AssetAndRuneDepths()
	assetDepth := assetE8DepthPerPool[asset]
	runeDepth := runeE8DepthPerPool[asset]

	window := stat.Window{time.Unix(0, 0), timestamp}

	status, err := stat.PoolStatusLookup(asset)
	if err != nil {
		errorResp(w, r, err)
		return
	}
	poolStakes, err := stat.PoolStakesLookup(asset, window)
	if err != nil {
		errorResp(w, r, err)
		return
	}
	buySwaps, err := stat.PoolSellSwapsLookup(asset, window)
	if err != nil {
		errorResp(w, r, err)
		return
	}
	sellSwaps, err := stat.PoolSellSwapsLookup(asset, window)
	if err != nil {
		errorResp(w, r, err)
		return
	}

	assetROI := new(big.Rat)
	runeROI := new(big.Rat)
	if poolStakes.AssetE8Total != 0 {
		assetROI.SetFrac64(assetDepth-poolStakes.AssetE8Total, poolStakes.AssetE8Total)
	}
	if poolStakes.RuneE8Total != 0 {
		runeROI.SetFrac64(runeDepth-poolStakes.RuneE8Total, poolStakes.RuneE8Total)
	}

	m := map[string]interface{}{
		"status":           status,
		"asset":            asset,
		"assetDepth":       assetDepth,
		"assetROI":         floatRat(assetROI),
		"assetStakedTotal": poolStakes.AssetE8Total,
		"buyAssetCount":    buySwaps.AssetE8Total,
		"buyFeeAverage":    float64(buySwaps.LiqFeeE8Total) / float64(buySwaps.TxCount),
		"buyFeesTotal":     buySwaps.LiqFeeE8Total,
		"buySlipAverage":   float64(buySwaps.TradeSlipBPTotal) / float64(buySwaps.TxCount),
		"buyTxAverage":     float64(buySwaps.AssetE8Total) / float64(buySwaps.TxCount),
		"buyVolume":        buySwaps.AssetE8Total,
		"poolDepth":        2 * runeDepth,
		"poolFeeAverage":   float64(buySwaps.LiqFeeE8Total+sellSwaps.LiqFeeE8Total) / float64(buySwaps.TxCount+sellSwaps.TxCount),
		"poolFeesTotal":    buySwaps.LiqFeeE8Total + sellSwaps.LiqFeeE8Total,
		"poolROI":          floatRat(new(big.Rat).Mul(big.NewRat(1, 2), new(big.Rat).Add(assetROI, runeROI))),
		"poolSlipAverage":  float64(buySwaps.TradeSlipBPTotal+sellSwaps.TradeSlipBPTotal) / float64(buySwaps.TxCount+sellSwaps.TxCount),

		"runeDepth":       runeDepth,
		"runeROI":         floatRat(runeROI),
		"runeStakedTotal": poolStakes.RuneE8Total,
		"sellFeeAverage":  float64(sellSwaps.LiqFeeE8Total) / float64(sellSwaps.TxCount),
		"sellFeesTotal":   sellSwaps.LiqFeeE8Total,
		"stakeTxCount":    poolStakes.TxCount,
	}
	/* TODO:
	PoolROI12        float64
	PoolStakedTotal  uint64
	PoolTxAverage    float64
	PoolUnits        uint64
	PoolVolume       uint64
	PoolVolume24hr   uint64
	Price            float64
	SellAssetCount   uint64
	SellSlipAverage  float64
	SellTxAverage    float64
	SellVolume       uint64
	StakersCount     uint64
	StakingTxCount   uint64
	SwappersCount    uint64
	SwappingTxCount  uint64
	WithdrawTxCount  uint64
	*/

	w.Header().Set("content-type", "application/json")
	json.NewEncoder(w).Encode(m)
}

func serveV1Stakers(w http.ResponseWriter, r *http.Request) {
	addrStakes, err := stat.AllAddrStakesLookup(time.Now())
	if err != nil {
		errorResp(w, r, err)
		return
	}

	array := make([]string, len(addrStakes))
	for i, stakes := range addrStakes {
		array[i] = stakes.Addr
	}

	w.Header().Set("content-type", "application/json")
	json.NewEncoder(w).Encode(array)
}

func errorResp(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("HTTP %q %q: %s", r.Method, r.URL.Path, err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func floatRat(r *big.Rat) float64 {
	f, _ := r.Float64()
	return f
}
