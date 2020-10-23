// End to end tests here are checkning lookup funcionality from Database to HTTP Api.

package timeseries_test

import (
	"encoding/json"
	"gitlab.com/thorchain/midgard/internal/timeseries/testdb"
	"reflect"
	"sort"
	"testing"

	"gitlab.com/thorchain/midgard/internal/timeseries"
)

func TestPoolsE2E(t *testing.T) {
	testdb.SetupTestDB(t)
	timeseries.SetLastTrackForTest(1, testdb.ToTime("2020-09-30 23:00:00"), "hash0")
	testdb.MustExec(t, "DELETE FROM stake_events")

	testdb.InsertStakeEvent(t, testdb.FakeStake{Pool: "BNB.BNB"})
	testdb.InsertStakeEvent(t, testdb.FakeStake{Pool: "POOL2"})
	testdb.InsertStakeEvent(t, testdb.FakeStake{Pool: "POOL3"})

	body := testdb.CallV1(t, "http://localhost:8080/v1/pools")

	var v []string
	json.Unmarshal(body, &v)
	sort.Strings(v)
	expected := []string{"BNB.BNB", "POOL2", "POOL3"}
	if !reflect.DeepEqual(v, expected) {
		t.Fatalf("/v1/pools returned unexpected results (actual: %v, expected: %v", v, expected)
	}
}
