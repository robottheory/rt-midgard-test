// End to end tests here are checkning lookup funcionality from Database to HTTP Api.

package timeseries_test

import (
	"reflect"
	"sort"
	"testing"

	"gitlab.com/thorchain/midgard/internal/timeseries/testdb"

	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

func TestPoolsE2E(t *testing.T) {
	testdb.SetupTestDB(t)
	timeseries.SetLastTrackForTest(1, testdb.ToTime("2020-09-30 23:00:00"), "hash0")
	testdb.MustExec(t, "DELETE FROM stake_events")

	testdb.InsertStakeEvent(t, testdb.FakeStake{Pool: "BNB.BNB"})
	testdb.InsertStakeEvent(t, testdb.FakeStake{Pool: "POOL2"})
	testdb.InsertStakeEvent(t, testdb.FakeStake{Pool: "POOL3"})

	body := testdb.CallV1(t, "http://localhost:8080/v2/pools")

	// TODO(acsaba): test other fields too
	var response oapigen.PoolsResponse
	testdb.MustUnmarshal(t, body, &response)
	var v []string

	for _, poolSummary := range response {
		v = append(v, poolSummary.Asset)
	}

	sort.Strings(v)
	expected := []string{"BNB.BNB", "POOL2", "POOL3"}
	if !reflect.DeepEqual(v, expected) {
		t.Fatalf("/v2/pools returned unexpected results (actual: %v, expected: %v", v, expected)
	}
}
