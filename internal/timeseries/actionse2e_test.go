// End to end tests here are checking lookup functionality from Database to HTTP Api.
package timeseries_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

func TestActionsE2E(t *testing.T) {
	testdb.InitTest(t)

	testdb.InsertBlockLog(t, 1, "2020-09-01 00:00:00")
	testdb.InsertBlockLog(t, 2, "2020-09-02 00:00:00")
	testdb.InsertBlockLog(t, 3, "2020-09-03 00:00:00")

	testdb.InsertSwapEvent(t, testdb.FakeSwap{FromAsset: "BNB.BNB", BlockTimestamp: "2020-09-03 00:00:00"})
	testdb.InsertStakeEvent(t, testdb.FakeStake{Pool: "BNB.TWT-123", BlockTimestamp: "2020-09-01 00:00:00", RuneAddress: "thoraddr1"})
	testdb.InsertUnstakeEvent(t, testdb.FakeUnstake{Asset: "BNB.TWT-123", BlockTimestamp: "2020-09-02 00:00:00"})

	// Basic request with no filters (should get all events ordered by height)
	body := testdb.CallJSON(t, "http://localhost:8080/v2/actions?limit=50&offset=0")

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
	body = testdb.CallJSON(t, "http://localhost:8080/v2/actions?limit=50&offset=0&type=swap")

	testdb.MustUnmarshal(t, body, &v)

	if v.Count != "1" {
		t.Fatal("Number of results changed.")
	}
	typeTx0 := v.Actions[0]

	if typeTx0.Type != "swap" {
		t.Fatal("Results of results changed.")
	}

	// Filter by asset request
	body = testdb.CallJSON(t, "http://localhost:8080/v2/actions?limit=50&offset=0&asset=BNB.TWT-123")

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
	body := testdb.CallJSON(t, url)

	var v oapigen.ActionsResponse
	testdb.MustUnmarshal(t, body, &v)
	return v.Count
}

func TestDepositStakeByTxIds(t *testing.T) {
	testdb.InitTest(t)

	testdb.InsertBlockLog(t, 1, "2020-09-01 00:00:00")

	testdb.InsertStakeEvent(t, testdb.FakeStake{
		Pool:           "BNB.TWT-123",
		BlockTimestamp: "2020-09-01 00:00:00",
		RuneAddress:    "thoraddr1",
		AssetTx:        "RUNETX1",
		RuneTx:         "ASSETTX1",
	})

	require.Equal(t, "1", txResponseCount(t,
		"http://localhost:8080/v2/actions?limit=50&offset=0"))
	require.Equal(t, "0", txResponseCount(t,
		"http://localhost:8080/v2/actions?txid=NOSUCHID&limit=50&offset=0"))
	require.Equal(t, "1", txResponseCount(t,
		"http://localhost:8080/v2/actions?txid=ASSETTX1&limit=50&offset=0"))
	require.Equal(t, "1", txResponseCount(t,
		"http://localhost:8080/v2/actions?txid=RUNETX1&limit=50&offset=0"))
}

func TestDoubleSwap(t *testing.T) {
	testdb.InitTest(t)

	testdb.InsertBlockLog(t, 1, "2020-09-03 00:00:00")

	testdb.InsertSwapEvent(t, testdb.FakeSwap{
		Tx:             "double",
		FromAsset:      "BNB.BNB",
		Pool:           "BNB.BNB",
		SwapSlipBP:     100,
		LiqFeeInRuneE8: 10000,
		BlockTimestamp: "2020-09-03 00:00:00",
	})
	testdb.InsertSwapEvent(t, testdb.FakeSwap{
		Tx:             "double",
		FromAsset:      "THOR.RUNE",
		Pool:           "BTC.BTC",
		ToE8Min:        50000,
		SwapSlipBP:     200,
		LiqFeeInRuneE8: 20000,
		BlockTimestamp: "2020-09-03 00:00:00",
	})

	body := testdb.CallJSON(t, "http://localhost:8080/v2/actions?limit=50&offset=0&type=swap")

	var v oapigen.ActionsResponse
	testdb.MustUnmarshal(t, body, &v)

	doubleSwap := v.Actions[0]
	metadata := doubleSwap.Metadata.Swap
	require.Equal(t, metadata.SwapSlip, "298") // 100+200-(100*200)/10000
	require.Equal(t, metadata.LiquidityFee, "30000")
	require.Equal(t, metadata.SwapTarget, "50000")
}

func checkFilter(t *testing.T, urlPostfix string, expectedResultsPool []string) {
	body := testdb.CallJSON(t,
		"http://localhost:8080/v2/actions?limit=50&offset=0"+urlPostfix)
	var v oapigen.ActionsResponse
	testdb.MustUnmarshal(t, body, &v)

	require.Equal(t, strconv.Itoa(len(expectedResultsPool)), v.Count)
	for i, pool := range expectedResultsPool {
		require.Equal(t, []string{pool}, v.Actions[i].Pools)
	}
}

func TestAdderessFilter(t *testing.T) {

	testdb.InitTest(t)

	testdb.InsertBlockLog(t, 1, "2020-09-01 00:00:00")
	testdb.InsertBlockLog(t, 2, "2020-09-02 00:00:00")
	testdb.InsertBlockLog(t, 3, "2020-09-03 00:00:00")

	testdb.InsertStakeEvent(t, testdb.FakeStake{
		Pool: "POOL1.A", BlockTimestamp: "2020-09-01 00:00:00", RuneAddress: "thoraddr1"})

	testdb.InsertSwapEvent(t, testdb.FakeSwap{
		Pool:      "POOL2.A",
		FromAsset: "POOL2.A", BlockTimestamp: "2020-09-02 00:00:00",
		FromAddr: "thoraddr2",
		ToAddr:   "thoraddr3",
	})

	testdb.InsertUnstakeEvent(t, testdb.FakeUnstake{
		Pool:  "POOL3.A",
		Asset: "POOL3.A", BlockTimestamp: "2020-09-03 00:00:00",
		ToAddr: "thoraddr4",
	})

	checkFilter(t, "", []string{"POOL3.A", "POOL2.A", "POOL1.A"})
	checkFilter(t, "&address=thoraddr1", []string{"POOL1.A"})
	checkFilter(t, "&address=thoraddr2", []string{"POOL2.A"})
	checkFilter(t, "&address=thoraddr4", []string{"POOL3.A"})

	checkFilter(t, "&address=thoraddr1,thoraddr4", []string{"POOL3.A", "POOL1.A"})
}
