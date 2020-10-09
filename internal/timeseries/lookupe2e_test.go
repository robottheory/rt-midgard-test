// End to end tests here are checkning lookup funcionality from Database to HTTP Api.

// Package name is timeseries_test in order to depend on internal/api without a dependency loop.
package timeseries_test

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http/httptest"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/pascaldekloe/sqltest"
	"gitlab.com/thorchain/midgard/internal/api"
	"gitlab.com/thorchain/midgard/internal/timeseries"
)

func init() {
	sqltest.Setup("pgx", "user=midgard password=password host=localhost port=5432 sslmode=disable dbname=midgard")
}

func setupDb(t *testing.T) {
	tx := sqltest.NewTx(t)
	timeseries.DBExec = tx.Exec
	timeseries.DBQuery = tx.QueryContext
}

func toTime(s string) time.Time {
	const format = "2006-01-02 15:04:05"
	t, err := time.Parse(format, s)
	if err != nil {
		log.Panicf("Bad date format %v", err)
	}
	return t
}

// Execute a query on the database.
func mustExec(t *testing.T, query string, args ...interface{}) {
	_, err := timeseries.DBExec(query, args...)
	if err != nil {
		t.Fatal("db query failed:", err)
	}
}

// Make an HTTP call to the /v1 api, return the body which can be parsed as a JSON.
func callV1(t *testing.T, url string) (body []byte) {
	api.InitHandler("", []string{})
	req := httptest.NewRequest("GET", url, nil)
	w := httptest.NewRecorder()
	api.Handler.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()
	if res.Status != "200 OK" {
		t.Fatal("Bad response status:", res.Status)
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal("error reading body:", res.Body)
	}

	return body
}

func insertStakeEvent(t *testing.T, poolName string) {
	const insertq = (`INSERT INTO stake_events ` +
		`(pool, asset_tx, asset_chain, asset_E8, rune_tx, rune_addr, rune_E8, stake_units, block_timestamp) ` +
		`VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`)

	mustExec(t, insertq, poolName, "asset_tx", "chain", 1, "rune_tx", "rune_addr", 2, 3, 4)
}

func TestPoolsE2E(t *testing.T) {
	timeseries.SetLastTrackForTest(1, toTime("2020-09-30 23:00:00"), "hash0")
	setupDb(t)
	mustExec(t, "DELETE FROM stake_events")

	insertStakeEvent(t, "BNB.BNB")
	insertStakeEvent(t, "POOL2")
	insertStakeEvent(t, "POOL3")

	body := callV1(t, "http://localhost:8080/v1/pools")

	var v []string
	json.Unmarshal(body, &v)
	sort.Strings(v)
	expected := []string{"BNB.BNB", "POOL2", "POOL3"}
	if !reflect.DeepEqual(v, expected) {
		t.Fatalf("/v1/pools returned unexpected results (actual: %v, expected: %v", v, expected)
	}
}
