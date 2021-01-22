// End to end tests here are checking lookup functionality from Database to HTTP Api.
package timeseries_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

func TestActionsE2E(t *testing.T) {
	testdb.SetupTestDB(t)
	timeseries.SetLastTimeForTest(testdb.StrToSec("2020-09-30 23:00:00"))
	testdb.MustExec(t, "DELETE FROM stake_events")
	testdb.MustExec(t, "DELETE FROM unstake_events")
	testdb.MustExec(t, "DELETE FROM swap_events")
	testdb.MustExec(t, "DELETE FROM block_log")

	testdb.InsertBlockLog(t, 1, "2020-09-01 00:00:00")
	testdb.InsertBlockLog(t, 2, "2020-09-02 00:00:00")
	testdb.InsertBlockLog(t, 3, "2020-09-03 00:00:00")

	testdb.InsertSwapEvent(t, testdb.FakeSwap{FromAsset: "BNB.BNB", BlockTimestamp: "2020-09-03 00:00:00"})
	testdb.InsertStakeEvent(t, testdb.FakeStake{Pool: "BNB.TWT-123", BlockTimestamp: "2020-09-01 00:00:00", RuneAddress: "thoraddr1"})
	testdb.InsertUnstakeEvent(t, testdb.FakeUnstake{Asset: "BNB.TWT-123", BlockTimestamp: "2020-09-02 00:00:00"})

	// Basic request with no filters (should get all events ordered by height)
	body := testdb.CallV1(t, "http://localhost:8080/v2/actions?limit=50&offset=0")

	var v oapigen.ActionsResponse
	testdb.MustUnmarshal(t, body, &v)

	if v.Count != "3" {
		t.Fatal("Number of results changed.")
	}

	basicTx0 := v.Actions[0]
	basicTx1 := v.Actions[1]
	basicTx2 := v.Actions[2]

	if basicTx0.Type != "swap" || basicTx0.Height != "3" {
		t.Fatal("Results of results changed.")
	}
	if basicTx1.Type != "withdraw" || basicTx1.Height != "2" {
		t.Fatal("Results of results changed.")
	}
	if basicTx2.Type != "addLiquidity" || basicTx2.Height != "1" {
		t.Fatal("Results of results changed.")
	}

	// Filter by type request
	body = testdb.CallV1(t, "http://localhost:8080/v2/actions?limit=50&offset=0&type=swap")

	testdb.MustUnmarshal(t, body, &v)

	if v.Count != "1" {
		t.Fatal("Number of results changed.")
	}
	typeTx0 := v.Actions[0]

	if typeTx0.Type != "swap" {
		t.Fatal("Results of results changed.")
	}

	// Filter by asset request
	body = testdb.CallV1(t, "http://localhost:8080/v2/actions?limit=50&offset=0&asset=BNB.TWT-123")

	testdb.MustUnmarshal(t, body, &v)

	if v.Count != "2" {
		t.Fatal("Number of results changed.")
	}
	assetTx0 := v.Actions[0]
	assetTx1 := v.Actions[1]

	if assetTx0.Type != "withdraw" {
		t.Fatal("Results of results changed.")
	}
	if assetTx1.Type != "addLiquidity" {
		t.Fatal("Results of results changed.")
	}
}

func txResponseCount(t *testing.T, url string) string {
	body := testdb.CallV1(t, url)

	var v oapigen.ActionsResponse
	testdb.MustUnmarshal(t, body, &v)
	return v.Count
}

func TestDepositStakeByTxIds(t *testing.T) {
	testdb.SetupTestDB(t)
	timeseries.SetLastTimeForTest(testdb.StrToSec("2020-09-30 23:00:00"))
	testdb.MustExec(t, "DELETE FROM stake_events")
	testdb.MustExec(t, "DELETE FROM unstake_events")
	testdb.MustExec(t, "DELETE FROM swap_events")
	testdb.MustExec(t, "DELETE FROM block_log")

	testdb.InsertBlockLog(t, 1, "2020-09-01 00:00:00")

	testdb.InsertStakeEvent(t, testdb.FakeStake{
		Pool:           "BNB.TWT-123",
		BlockTimestamp: "2020-09-01 00:00:00",
		RuneAddress:    "thoraddr1",
		AssetTx:        "RUNETX1",
		RuneTx:         "ASSETTX1",
	})

	assert.Equal(t, "1", txResponseCount(t,
		"http://localhost:8080/v2/actions?limit=50&offset=0"))
	assert.Equal(t, "0", txResponseCount(t,
		"http://localhost:8080/v2/actions?txid=NOSUCHID&limit=50&offset=0"))
	assert.Equal(t, "1", txResponseCount(t,
		"http://localhost:8080/v2/actions?txid=ASSETTX1&limit=50&offset=0"))
	assert.Equal(t, "1", txResponseCount(t,
		"http://localhost:8080/v2/actions?txid=RUNETX1&limit=50&offset=0"))
}
