// Package name is timeseries_test in order to depend on internal/api without a dependency loop.
package timeseries_test

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"net/http/httptest"
	"testing"
	"time"

	"gitlab.com/thorchain/midgard/event"
	"gitlab.com/thorchain/midgard/internal/api"
	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/internal/timeseries/stat"
)

var testDBQuery func(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
var testDBExec func(query string, args ...interface{}) (sql.Result, error)

func init() {
	// TODO(acsaba): Should have special handling for short mode?
	testDbPort := 5433
	db, err := sql.Open("pgx", fmt.Sprintf("user=midgard dbname=midgard sslmode=disable password=password host=localhost port=%d", testDbPort))
	if err != nil {
		log.Fatal("Failed to connect to PostgreSQL. Did you `docker-compose up -d pg`? (", err, ")")
	}

	testDBQuery = db.QueryContext
	testDBExec = db.Exec
}

func setupTestDB() {
	stat.DBQuery = testDBQuery
	timeseries.DBExec = testDBExec
	timeseries.DBQuery = testDBQuery
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
		t.Fatal("db query failed. Did you `docker-compose up -d pg`? ", err, "query: ", query, "args: ", args)
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
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal("error reading body:", res.Body)
	}

	if res.Status != "200 OK" {
		t.Fatal("Bad response status:", res.Status, ". Body: ", string(body))
	}

	return body
}

type fakeStake struct {
	assetChain     string
	blockTimestamp int64
}

func insertStakeEvent(t *testing.T, fake fakeStake) {
	if fake.assetChain == "" {
		fake.assetChain = "BNB.BNB"
	}

	const insertq = (`INSERT INTO stake_events ` +
		`(pool, asset_tx, asset_chain, asset_E8, rune_tx, rune_addr, rune_E8, stake_units, block_timestamp) ` +
		`VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`)

	mustExec(t, insertq, fake.assetChain, "swap_tx", "chain", 1, "swap_tx", "rune_addr", 2, 3, fake.blockTimestamp)
}

type fakeUnstake struct {
	pool           string
	blockTimestamp int64
}

func insertUnstakeEvent(t *testing.T, fake fakeUnstake) {
	if fake.pool == "" {
		fake.pool = "BNB.BNB"
	}

	const insertq = (`INSERT INTO unstake_events ` +
		`(tx, chain, from_addr, to_addr, asset, asset_E8, memo, pool, stake_units, basis_points, asymmetry, block_timestamp) ` +
		`VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`)

	mustExec(t, insertq, "unstake_tx", "chain", "unstaker_addr", "vault_addr", fake.pool, 1, "WITHDRAW:"+fake.pool+":1000", fake.pool, 123, 1000, 0, fake.blockTimestamp)
}

type fakeSwap struct {
	fromAsset      string
	blockTimestamp int64
}

func insertSwapEvent(t *testing.T, fake fakeSwap) {
	defaultAsset := "BNB.BNB"

	if fake.fromAsset == "" {
		fake.fromAsset = defaultAsset
	}

	toAsset := event.RuneAsset()
	pool := defaultAsset
	if event.IsRune([]byte(fake.fromAsset)) {
		toAsset = defaultAsset
	} else {
		pool = fake.fromAsset
	}

	const insertq = (`INSERT INTO swap_events ` +
		`(tx, chain, from_addr, to_addr, from_asset, from_E8, memo, pool, to_E8_min, trade_slip_BP,
		liq_fee_E8, liq_fee_in_rune_E8, block_timestamp) ` +
		`VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`)

	mustExec(t, insertq, "swap_tx", "chain", "swapper_addr", "vault_addr", fake.fromAsset, 10000000, "SWAP:"+toAsset, pool, 0, 10000, 12345, 678910, fake.blockTimestamp)
}

func insertBlockLog(t *testing.T, height, timestamp int64) {
	const insertq = (`INSERT INTO block_log ` +
		`(height, timestamp, hash) ` +
		`VALUES ($1, $2, $3)`)

	mustExec(t, insertq, height, timestamp, fmt.Sprintf("%d-%d", height, timestamp))
}
