// End to end tests here are checkning lookup funcionality from Database to HTTP Api.

package timeseries_test

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"

	"gitlab.com/thorchain/midgard/internal/timeseries"
)

func TestPoolsE2E(t *testing.T) {
	setupTestDB(t)
	timeseries.SetLastTrackForTest(1, toTime("2020-09-30 23:00:00"), "hash0")
	mustExec(t, "DELETE FROM stake_events")

	insertStakeEvent(t, fakeStake{pool: "BNB.BNB"})
	insertStakeEvent(t, fakeStake{pool: "POOL2"})
	insertStakeEvent(t, fakeStake{pool: "POOL3"})

	body := callV1(t, "http://localhost:8080/v1/pools")

	var v []string
	json.Unmarshal(body, &v)
	sort.Strings(v)
	expected := []string{"BNB.BNB", "POOL2", "POOL3"}
	if !reflect.DeepEqual(v, expected) {
		t.Fatalf("/v1/pools returned unexpected results (actual: %v, expected: %v", v, expected)
	}
}
