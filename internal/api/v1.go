package api

import (
	"encoding/json"
	"log"
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

	poolStakes, err := stat.PoolStakesLookup(asset, stat.Window{})
	if err != nil {
		errorResp(w, r, err)
		return
	}
	poolFees, err := stat.PoolFeesLookup(asset, stat.Window{})
	if err != nil {
		errorResp(w, r, err)
		return
	}

	m := map[string]interface{}{
		"asset":          asset,
		"assetDepth":     poolStakes.AssetE8Total,
		"runeDepth":      poolStakes.RuneE8Total,
		"poolDepth":      poolStakes.RuneE8Total * 2,
		"poolFeeAverage": int64(poolFees.AssetE8Avg), // wrong
		"stakeTxCount":   poolStakes.TxCount,
	}

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
