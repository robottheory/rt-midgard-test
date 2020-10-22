// End to end tests here are checkning lookup funcionality from Database to HTTP Api.
package timeseries_test

import (
	"encoding/json"
	"testing"

	"gitlab.com/thorchain/midgard/internal/timeseries"
)

func TestTxListE2E(t *testing.T) {
	setupTestDB(t)
	timeseries.SetLastTrackForTest(1, toTime("2020-09-30 23:00:00"), "hash0")
	mustExec(t, "DELETE FROM stake_events")
	mustExec(t, "DELETE FROM unstake_events")
	mustExec(t, "DELETE FROM swap_events")
	mustExec(t, "DELETE FROM block_log")

	insertBlockLog(t, 1, 100)
	insertBlockLog(t, 2, 200)
	insertBlockLog(t, 3, 300)

	insertSwapEvent(t, fakeSwap{fromAsset: "BNB.BNB", blockTimestamp: 300})
	insertStakeEvent(t, fakeStake{pool: "BNB.TWT-123", blockTimestamp: 100, assetTx: "stake_tx", runeTx: "stake_tx"})
	insertUnstakeEvent(t, fakeUnstake{asset: "BNB.TWT-123", blockTimestamp: 200})

	// Basic request with no filters (should get all events ordered by height)
	body := callV1(t, "http://localhost:8080/v1/tx?limit=50&offset=0")

	var v timeseries.TxTransactions
	json.Unmarshal(body, &v)

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
	body = callV1(t, "http://localhost:8080/v1/tx?limit=50&offset=0&type=swap")

	json.Unmarshal(body, &v)

	if v.Count != 1 {
		t.Fatal("Number of results changed.")
	}
	typeTx0 := v.Txs[0]

	if typeTx0.EventType != "swap" {
		t.Fatal("Results of results changed.")
	}

	// Filter by asset request
	body = callV1(t, "http://localhost:8080/v1/tx?limit=50&offset=0&asset=BNB.TWT-123")

	json.Unmarshal(body, &v)

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
