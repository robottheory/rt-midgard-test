package api

import (
	"encoding/json"
	"log"
	"net/http"

	"gitlab.com/thorchain/midgard/internal/timeseries/stat"
)

func servePools(w http.ResponseWriter, r *http.Request) {
	pool, err := stat.PoolsLookup()
	if err != nil {
		log.Print(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(pool)
}
