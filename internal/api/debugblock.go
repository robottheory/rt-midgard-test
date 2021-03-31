package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/util/miderr"
)

var DebugFetchResults func(ctx context.Context, height int64) (*coretypes.ResultBlockResults, error)

func debugBlock(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	idStr := ps[0].Value
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		fmt.Fprintf(w, "Provide an integer height or timestamp (%s): %v ", idStr, err)
		return
	}

	height, timestamp, err := TimestampAndHeight(r.Context(), id)
	if err != nil {
		fmt.Fprintf(w, "Height and timestamp lookup error: %v", err)
		return
	}
	fmt.Fprintf(w, "Height: %d ; Timestamp: %d\n", height, timestamp)

	var results *coretypes.ResultBlockResults
	results, err = DebugFetchResults(r.Context(), height)
	if err != nil {
		fmt.Fprint(w, "Failed to fetch block: ", err)
	}

	buf, _ := json.Marshal(results)
	var any interface{}
	err = json.Unmarshal(buf, &any)
	if err != nil {
		fmt.Fprint(w, "Failed to convert block to interface{}: ", err)
	}

	unwrapBase64Fields(any)
	e := json.NewEncoder(w)
	e.SetIndent("", "\t")

	// Error discarded
	_ = e.Encode(any)
}

func TimestampAndHeight(ctx context.Context, id int64) (
	height int64, timestamp db.Nano, err error) {
	q := `
		SELECT height, timestamp
		FROM block_log
		WHERE height=$1 OR timestamp=$1
	`
	rows, err := db.Query(ctx, q, id)
	if err != nil {
		return
	}
	defer rows.Close()

	if !rows.Next() {
		err = miderr.BadRequestF("No such height or timestamp: %d", id)
		return
	}
	err = rows.Scan(&height, &timestamp)
	return
}

var fieldsToUnwrap = map[string]bool{"key": true, "value": true}

func unwrapBase64Fields(any interface{}) {
	msgMap, ok := any.(map[string]interface{})
	if ok {
		for k, v := range msgMap {
			if fieldsToUnwrap[k] {
				s, err := base64.StdEncoding.DecodeString(v.(string))
				if err == nil {
					msgMap[k] = string(s)
				} else {
					msgMap[k] = "ERROR during base64 decoding"
				}
			} else {
				unwrapBase64Fields(v)
			}
		}
	}
	msgSlice, ok := any.([]interface{})
	if ok {
		for i := range msgSlice {
			unwrapBase64Fields(msgSlice[i])
		}
	}
}
