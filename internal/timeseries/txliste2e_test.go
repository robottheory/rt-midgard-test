// End to end tests here are checking lookup functionality from Database to HTTP Api.
package timeseries_test

import (
	"testing"

	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/internal/timeseries"
)

func TestTxListE2E(t *testing.T) {
	testdb.SetupTestDB(t)
	timeseries.SetLastTimeForTest(testdb.ToTime("2020-09-30 23:00:00"))
	testdb.MustExec(t, "DELETE FROM stake_events")
	testdb.MustExec(t, "DELETE FROM unstake_events")
	testdb.MustExec(t, "DELETE FROM swap_events")
	testdb.MustExec(t, "DELETE FROM block_log")

	testdb.InsertBlockLog(t, 1, "2020-09-01 00:00:00")
	testdb.InsertBlockLog(t, 2, "2020-09-02 00:00:00")
	testdb.InsertBlockLog(t, 3, "2020-09-03 00:00:00")

	testdb.InsertSwapEvent(t, testdb.FakeSwap{FromAsset: "BNB.BNB", BlockTimestamp: "2020-09-03 00:00:00"})
	testdb.InsertStakeEvent(t, testdb.FakeStake{Pool: "BNB.TWT-123", BlockTimestamp: "2020-09-01 00:00:00"})
	testdb.InsertUnstakeEvent(t, testdb.FakeUnstake{Asset: "BNB.TWT-123", BlockTimestamp: "2020-09-02 00:00:00"})

	// Basic request with no filters (should get all events ordered by height)
	body := testdb.CallV1(t, "http://localhost:8080/v2/tx?limit=50&offset=0")

	var v timeseries.TxTransactions
	testdb.MustUnmarshal(t, body, &v)

	if v.Count != 3 {
		t.Fatal("Number of results changed.")
	}

	basicTx0 := v.Txs[0]
	basicTx1 := v.Txs[1]
	basicTx2 := v.Txs[2]

	if basicTx0.EventType != "swap" || basicTx0.Height != 3 {
		t.Fatal("Results of results changed.")
	}
	if basicTx1.EventType != "unstake" || basicTx1.Height != 2 {
		t.Fatal("Results of results changed.")
	}
	if basicTx2.EventType != "stake" || basicTx2.Height != 1 {
		t.Fatal("Results of results changed.")
	}

	// Filter by type request
	body = testdb.CallV1(t, "http://localhost:8080/v2/tx?limit=50&offset=0&type=swap")

	testdb.MustUnmarshal(t, body, &v)

	if v.Count != 1 {
		t.Fatal("Number of results changed.")
	}
	typeTx0 := v.Txs[0]

	if typeTx0.EventType != "swap" {
		t.Fatal("Results of results changed.")
	}

	// Filter by asset request
	body = testdb.CallV1(t, "http://localhost:8080/v2/tx?limit=50&offset=0&asset=BNB.TWT-123")

	testdb.MustUnmarshal(t, body, &v)

	if v.Count != 2 {
		t.Fatal("Number of results changed.")
	}
	assetTx0 := v.Txs[0]
	assetTx1 := v.Txs[1]

	if assetTx0.EventType != "unstake" {
		t.Fatal("Results of results changed.")
	}
	if assetTx1.EventType != "stake" {
		t.Fatal("Results of results changed.")
	}
}
