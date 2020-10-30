package testdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v4/stdlib"

	"gitlab.com/thorchain/midgard/internal/api"
	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/internal/timeseries/stat"
)

var testDBQuery func(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
var testDBExec func(query string, args ...interface{}) (sql.Result, error)

func init() {
	testDbPort := getEnvVariable("DB_PORT", "5433")
	testHost := getEnvVariable("DB_HOST", "localhost")

	db, err := sql.Open("pgx", fmt.Sprintf("user=midgard dbname=midgard sslmode=disable password=password host=%s port=%s", testHost, testDbPort))
	if err != nil {
		log.Fatal("Failed to connect to PostgreSQL. Did you `docker-compose up -d pg`? (", err, ")")
	}

	testDBQuery = db.QueryContext
	testDBExec = db.Exec
}

func SetupTestDB(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
	stat.DBQuery = testDBQuery
	timeseries.DBExec = testDBExec
	timeseries.DBQuery = testDBQuery
}

func MustUnmarshal(t *testing.T, data []byte, v interface{}) {
	err := json.Unmarshal(data, v)
	if err != nil {
		t.FailNow()
	}
}

func ToTime(s string) time.Time {
	const format = "2006-01-02 15:04:05"
	t, err := time.Parse(format, s)
	if err != nil {
		log.Panicf("Bad date format %v", err)
	}
	return t
}

func SecToString(t int64) string {
	return time.Unix(t, 0).UTC().Format("2006-01-02 15:04:05")
}

// Execute a query on the database.
func MustExec(t *testing.T, query string, args ...interface{}) {
	_, err := timeseries.DBExec(query, args...)
	if err != nil {
		t.Fatal("db query failed. Did you `docker-compose up -d pg`? ", err, "query: ", query, "args: ", args)
	}
}

// Make an HTTP call to the /v1 api, return the body which can be parsed as a JSON.
func CallV1(t *testing.T, url string) (body []byte) {
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

type FakeStake struct {
	Pool           string
	BlockTimestamp string
	AssetE8        int64
	RuneE8         int64
	StakeUnits     int64
}

func InsertStakeEvent(t *testing.T, fake FakeStake) {
	const insertq = `INSERT INTO stake_events ` +
		`(pool, asset_tx, asset_chain, asset_E8, rune_tx, rune_addr, rune_E8, stake_units, block_timestamp) ` +
		`VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	timestamp := getTimestamp(fake.BlockTimestamp)
	MustExec(t, insertq, fake.Pool, "stakeTx", "chain", fake.AssetE8, "stakeTx", "rune_addr", fake.RuneE8, fake.StakeUnits, timestamp.UnixNano())
}

type FakeUnstake struct {
	Asset          string
	BlockTimestamp string
}

func InsertUnstakeEvent(t *testing.T, fake FakeUnstake) {
	const insertq = `INSERT INTO unstake_events ` +
		`(tx, chain, from_addr, to_addr, asset, asset_E8, memo, pool, stake_units, basis_points, asymmetry, block_timestamp) ` +
		`VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	timestamp := getTimestamp(fake.BlockTimestamp)
	MustExec(t, insertq, "tx", "chain", "from_addr", "to_addr", fake.Asset, 1, "memo", "pool", 2, 3, 4, timestamp.UnixNano())
}

type FakeSwap struct {
	FromAsset      string
	FromE8         int64
	Pool           string
	BlockTimestamp string
}

func InsertSwapEvent(t *testing.T, fake FakeSwap) {
	const insertq = `INSERT INTO swap_events ` +
		`(tx, chain, from_addr, to_addr, from_asset, from_E8, memo, pool, to_E8_min, trade_slip_BP,
			liq_fee_E8, liq_fee_in_rune_E8, block_timestamp) ` +
		`VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

	timestamp := getTimestamp(fake.BlockTimestamp)
	MustExec(t, insertq, "tx", "chain", "from_addr", "to_addr", fake.FromAsset, fake.FromE8, "memo", fake.Pool, 1, 2, 3, 4, timestamp.UnixNano())
}

func InsertBlockLog(t *testing.T, height int64, fakeTimestamp string) {
	const insertq = `INSERT INTO block_log ` +
		`(height, timestamp, hash) ` +
		`VALUES ($1, $2, $3)`

	timestamp := getTimestamp(fakeTimestamp)
	MustExec(t, insertq, height, timestamp.UnixNano(), fmt.Sprintf("%d-%d", height, timestamp.UnixNano()))
}

func InsertBlockPoolDepth(t *testing.T, pool string, assetE8, runeE8 int64, blockTimestamp string) {
	const insertq = `INSERT INTO block_pool_depths ` +
		`(pool, asset_e8, rune_e8, block_timestamp) ` +
		`VALUES ($1, $2, $3, $4)`

	timestamp := getTimestamp(blockTimestamp)
	MustExec(t, insertq, pool, assetE8, runeE8, timestamp.UnixNano())
}

func getEnvVariable(key, def string) string {
	value := os.Getenv(key)

	if value == "" {
		value = def
	}

	return value
}

func getTimestamp(fakeTimestamp string) time.Time {
	var timestamp time.Time

	if fakeTimestamp == "" {
		timestamp = ToTime("2000-01-01 00:00:00")
	} else {
		timestamp = ToTime(fakeTimestamp)
	}

	return timestamp
}
