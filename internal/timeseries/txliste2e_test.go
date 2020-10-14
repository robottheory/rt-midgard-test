// End to end tests here are checkning lookup funcionality from Database to HTTP Api.
package timeseries_test

import (
	"encoding/json"
	"testing"

	"gitlab.com/thorchain/midgard/internal/timeseries"
)

func TestTxListE2E(t *testing.T) {
	setupTestDB()
	timeseries.SetLastTrackForTest(1, toTime("2020-09-30 23:00:00"), "hash0")
	mustExec(t, "DELETE FROM stake_events")
	mustExec(t, "DELETE FROM block_log")

	insertBlockLog(t, 1, 100)
	insertBlockLog(t, 2, 200)
	insertBlockLog(t, 3, 300)

	insertStakeEvent(t, fakeStake{assetChain: "BNB.BNB", blockTimestamp: 100})

	body := callV1(t, "http://localhost:8080/v1/tx?limit=50&offset=0")

	var v timeseries.TxTransactions
	json.Unmarshal(body, &v)
	// TODO(elfedy): this seems wrong, it returns the values twice:
	//     check the logic and update the test if needed.
	if v.Count != 2 {
		t.Fatal("Number of reults changed.")
	}
	tx0 := v.Txs[0]
	tx1 := v.Txs[1]
	if tx0.EventType != "stake" || tx1.EventType != "stake" || tx0.Height != 1 || tx1.Height != 1 {
		t.Fatal("Results of reults changed.")
	}
}
