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
	"strconv"
	"sync"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v4/stdlib"

	"gitlab.com/thorchain/midgard/internal/api"
	"gitlab.com/thorchain/midgard/internal/db"
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
	db.Exec = testDBExec
	db.Query = testDBQuery
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

func ToUnixNanoStr(s string) string {
	return strconv.Itoa(int(ToTime(s).UnixNano()))
}

func SecToString(t int64) string {
	return time.Unix(t, 0).UTC().Format("2006-01-02 15:04:05")
}

// Execute a query on the database.
func MustExec(t *testing.T, query string, args ...interface{}) {
	_, err := db.Exec(query, args...)
	if err != nil {
		t.Fatal("db query failed. Did you `docker-compose up -d pg`? ", err, "query: ", query, "args: ", args)
	}
}

var apiOnce sync.Once

func initApi() {
	apiOnce.Do(func() {
		api.InitHandler("", []string{})
	})
}

// Make an HTTP call to the /v1 api, return the body which can be parsed as a JSON.
func CallV1(t *testing.T, url string) (body []byte) {
	initApi()
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

func CallV1Fail(t *testing.T, url string) {
	initApi()
	req := httptest.NewRequest("GET", url, nil)
	w := httptest.NewRecorder()
	api.Handler.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()
	if res.Status == "200 OK" {
		t.Fatal("Expected to fail, but didn't:", url)
	}

}

type FakeStake struct {
	Pool           string
	BlockTimestamp string
	AssetE8        int64
	RuneE8         int64
	StakeUnits     int64
	RuneAddress    string
}

func InsertStakeEvent(t *testing.T, fake FakeStake) {
	const insertq = `INSERT INTO stake_events ` +
		`(pool, asset_tx, asset_chain, asset_addr, asset_E8, rune_tx, rune_addr, rune_E8, stake_units, block_timestamp) ` +
		`VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	timestamp := getTimestamp(fake.BlockTimestamp)
	MustExec(t, insertq, fake.Pool, "stakeTx", "chain", "assetAddr", fake.AssetE8, "stakeTx", fake.RuneAddress, fake.RuneE8, fake.StakeUnits, timestamp.UnixNano())
}

type FakeUnstake struct {
	Asset          string
	BlockTimestamp string
	StakeUnits     int64
	Pool           string
	EmitAssetE8    int64
	EmitRuneE8     int64
}

func InsertUnstakeEvent(t *testing.T, fake FakeUnstake) {
	const insertq = `INSERT INTO unstake_events ` +
		`(tx, chain, from_addr, to_addr, asset, asset_E8, emit_asset_E8, emit_rune_E8, memo, pool, stake_units, basis_points, asymmetry, block_timestamp) ` +
		`VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

	timestamp := getTimestamp(fake.BlockTimestamp)
	MustExec(t, insertq, "tx", "chain", "from_addr", "to_addr", fake.Asset, 1, fake.EmitAssetE8, fake.EmitRuneE8, "memo", fake.Pool, fake.StakeUnits, 3, 4, timestamp.UnixNano())
}

type FakeSwap struct {
	Pool           string
	FromAsset      string
	FromE8         int64
	ToE8           int64
	LiqFeeInRuneE8 int64
	TradeSlipBP    int64
	BlockTimestamp string
}

func InsertSwapEvent(t *testing.T, fake FakeSwap) {
	const insertq = `INSERT INTO swap_events ` +
		`(tx, chain, from_addr, to_addr, from_asset, from_E8, to_E8, memo, pool, to_E8_min, trade_slip_BP,
			liq_fee_E8, liq_fee_in_rune_E8, block_timestamp) ` +
		`VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

	timestamp := getTimestamp(fake.BlockTimestamp)
	MustExec(t, insertq,
		"tx", "chain", "from_addr", "to_addr", fake.FromAsset, fake.FromE8, fake.ToE8,
		"memo", fake.Pool, 2, fake.TradeSlipBP, 4, fake.LiqFeeInRuneE8, timestamp.UnixNano())
}

func InsertRewardsEvent(t *testing.T, bondE8 int64, fakeTimestamp string) {
	const insertq = `INSERT INTO rewards_events ` +
		`(bond_e8, block_timestamp) ` +
		`VALUES ($1, $2)`

	timestamp := getTimestamp(fakeTimestamp)
	MustExec(t, insertq, bondE8, timestamp.UnixNano())
}

func InsertRewardsEventEntry(t *testing.T, bondE8 int64, pool, fakeTimestamp string) {
	const insertq = `INSERT INTO rewards_event_entries ` +
		`(rune_e8, block_timestamp, pool) ` +
		`VALUES ($1, $2, $3)`

	timestamp := getTimestamp(fakeTimestamp)
	MustExec(t, insertq, bondE8, timestamp.UnixNano(), pool)
}

func InsertBlockLog(t *testing.T, height int64, fakeTimestamp string) {
	const insertq = `INSERT INTO block_log ` +
		`(height, timestamp, hash) ` +
		`VALUES ($1, $2, $3)`

	timestamp := getTimestamp(fakeTimestamp)
	MustExec(t, insertq, height, timestamp.UnixNano(), fmt.Sprintf("%d-%d", height, timestamp.UnixNano()))
}

func InsertPoolEvents(t *testing.T, pool, status string) {
	const insertq = `INSERT INTO  pool_events` +
		`(asset, status, block_timestamp) ` +
		`VALUES ($1, $2, 1)`

	MustExec(t, insertq, pool, status)
}

func InsertBlockPoolDepth(t *testing.T, pool string, assetE8, runeE8 int64, blockTimestamp string) {
	const insertq = `INSERT INTO block_pool_depths ` +
		`(pool, asset_e8, rune_e8, block_timestamp) ` +
		`VALUES ($1, $2, $3, $4)`

	timestamp := getTimestamp(blockTimestamp)
	MustExec(t, insertq, pool, assetE8, runeE8, timestamp.UnixNano())
}

func InsertUpdateNodeAccountStatusEvent(t *testing.T, former, current, blockTimestamp string) {
	const insertq = `INSERT INTO update_node_account_status_events ` +
		`(node_addr, former, current, block_timestamp) ` +
		`VALUES ($1, $2, $3, $4)`

	timestamp := getTimestamp(blockTimestamp)
	MustExec(t, insertq, "node_addr", former, current, timestamp.UnixNano())
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

func InsertActiveVaultEvent(t *testing.T, address string, blockTimestamp string) {
	const insertq = `INSERT INTO active_vault_events ` +
		`(add_asgard_addr, block_timestamp) ` +
		`VALUES ($1, $2)`

	timestamp := getTimestamp(blockTimestamp)
	MustExec(t, insertq, address, timestamp.UnixNano())
}

type FakeThornodeConstants struct {
	EmissionCurve      int64
	BlocksPerYear      int64
	ChurnInterval      int64
	ChurnRetryInterval int64
	PoolCycle          int64
}

func SetThornodeConstants(t *testing.T, constants *FakeThornodeConstants, timestamp string) {
	insertMimirEvent(t, "EmissionCurve", constants.EmissionCurve, timestamp)
	insertMimirEvent(t, "BlocksPerYear", constants.BlocksPerYear, timestamp)
	insertMimirEvent(t, "ChurnInterval", constants.ChurnInterval, timestamp)
	insertMimirEvent(t, "ChurnRetryInterval", constants.ChurnRetryInterval, timestamp)
	insertMimirEvent(t, "PoolCycle", constants.PoolCycle, timestamp)
}

func insertMimirEvent(t *testing.T, key string, value int64, blockTimestamp string) {
	const insertq = `INSERT INTO set_mimir_events ` +
		`(key, value, block_timestamp) ` +
		`VALUES ($1, $2, $3)`

	timestamp := getTimestamp(blockTimestamp)
	MustExec(t, insertq, key, strconv.FormatInt(value, 10), timestamp.UnixNano())
}