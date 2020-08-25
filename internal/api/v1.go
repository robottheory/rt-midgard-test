package api

import (
	"encoding/json"
	"log"
	"net/http"
	"path"

	"gitlab.com/thorchain/midgard/internal/timeseries/stat"
)

func servePools(w http.ResponseWriter, r *http.Request) {
	pool, err := stat.PoolsLookup()
	if err != nil {
		errorResp(w, r, err)
		return
	}
	json.NewEncoder(w).Encode(pool)
}

func servePoolsAsset(w http.ResponseWriter, r *http.Request) {
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
	json.NewEncoder(w).Encode(m)
}

func errorResp(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("HTTP %q %q: %s", r.Method, r.URL.Path, err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
