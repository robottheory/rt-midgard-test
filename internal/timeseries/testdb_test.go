// Package name is timeseries_test in order to depend on internal/api without a dependency loop.
package timeseries_test

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"gitlab.com/thorchain/midgard/internal/api"
	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/internal/timeseries/stat"
)

var testDBQuery func(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
var testDBExec func(query string, args ...interface{}) (sql.Result, error)

func init() {
	// TODO(acsaba): Should have special handling for short mode?
	testDbPort := getEnvVariable("DB_PORT", "5433")
	testHost := getEnvVariable("DB_HOST", "localhost")

	db, err := sql.Open("pgx", fmt.Sprintf("user=midgard dbname=midgard sslmode=disable password=password host=%s port=%s", testHost, testDbPort))
	if err != nil {
		log.Fatal("Failed to connect to PostgreSQL. Did you `docker-compose up -d pg`? (", err, ")")
	}

	testDBQuery = db.QueryContext
	testDBExec = db.Exec
}

func setupTestDB(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
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
	pool           string
	blockTimestamp int64
	assetTx        string
	runeTx         string
}

func insertStakeEvent(t *testing.T, fake fakeStake) {
	const insertq = (`INSERT INTO stake_events ` +
		`(pool, asset_tx, asset_chain, asset_E8, rune_tx, rune_addr, rune_E8, stake_units, block_timestamp) ` +
		`VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`)

	mustExec(t, insertq, fake.pool, fake.assetTx, "chain", 1, fake.runeTx, "rune_addr", 2, 3, fake.blockTimestamp)
}

type fakeUnstake struct {
	asset          string
	blockTimestamp int64
}

func insertUnstakeEvent(t *testing.T, fake fakeUnstake) {
	const insertq = (`INSERT INTO unstake_events ` +
		`(tx, chain, from_addr, to_addr, asset, asset_E8, memo, pool, stake_units, basis_points, asymmetry, block_timestamp) ` +
		`VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`)

	mustExec(t, insertq, "tx", "chain", "from_addr", "to_addr", fake.asset, 1, "memo", "pool", 2, 3, 4, fake.blockTimestamp)
}

type fakeSwap struct {
	fromAsset      string
	blockTimestamp int64
}

func insertSwapEvent(t *testing.T, fake fakeSwap) {
	const insertq = (`INSERT INTO swap_events ` +
		`(tx, chain, from_addr, to_addr, from_asset, from_E8, memo, pool, to_E8_min, trade_slip_BP,
		liq_fee_E8, liq_fee_in_rune_E8, block_timestamp) ` +
		`VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`)

	mustExec(t, insertq, "tx", "chain", "from_addr", "to_addr", fake.fromAsset, 1, "memo", "pool", 1, 2, 3, 4, fake.blockTimestamp)
}

func insertBlockLog(t *testing.T, height, timestamp int64) {
	const insertq = (`INSERT INTO block_log ` +
		`(height, timestamp, hash) ` +
		`VALUES ($1, $2, $3)`)

	mustExec(t, insertq, height, timestamp, fmt.Sprintf("%d-%d", height, timestamp))
}

func getEnvVariable(key, def string) string {
	value := os.Getenv(key)

	if value == "" {
		value = def
	}

	return value
}
