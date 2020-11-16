// End to end tests here are checkning lookup funcionality from Database to HTTP Api.

package timeseries_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/thorchain/midgard/internal/timeseries/testdb"

	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

func callPools(t *testing.T, url string) map[string]oapigen.PoolSummary {
	body := testdb.CallV1(t, url)

	var response oapigen.PoolsResponse
	testdb.MustUnmarshal(t, body, &response)
	sortedResp := map[string]oapigen.PoolSummary{}

	for _, poolSummary := range response {
		sortedResp[poolSummary.Asset] = poolSummary
	}
	return sortedResp
}

func TestPoolsE2E(t *testing.T) {
	testdb.SetupTestDB(t)
	timeseries.SetLastTimeForTest(testdb.ToTime("2020-09-30 23:00:00"))
	testdb.MustExec(t, "DELETE FROM stake_events")
	testdb.MustExec(t, "DELETE FROM pool_events")

	testdb.InsertStakeEvent(t, testdb.FakeStake{Pool: "BNB.BNB", BlockTimestamp: "2020-01-01 00:00:00"})
	testdb.InsertStakeEvent(t, testdb.FakeStake{Pool: "POOL2"})
	testdb.InsertStakeEvent(t, testdb.FakeStake{Pool: "POOL3"})

	testdb.InsertPoolEvents(t, "BNB.BNB", "Enabled")
	testdb.InsertPoolEvents(t, "POOL2", "Enabled")
	testdb.InsertPoolEvents(t, "POOL3", "Bootstrap")

	timeseries.SetDepthsForTest("POOL2", 2, 1)
	sortedResp := callPools(t, "http://localhost:8080/v2/pools")

	assert.Equal(t, len(sortedResp), 3)
	assert.Equal(t, sortedResp["BNB.BNB"].DateCreated, testdb.ToUnixNanoStr("2020-01-01 00:00:00"))
	assert.Equal(t, sortedResp["POOL2"].AssetDepth, "2")
	assert.Equal(t, sortedResp["POOL2"].RuneDepth, "1")
	assert.Equal(t, sortedResp["POOL2"].Price, "0.5")
	_, has_pool3 := sortedResp["POOL3"]
	assert.Equal(t, has_pool3, true) // Without filter we have the Bootstrap pool

	// check filtering
	sortedResp = callPools(t, "http://localhost:8080/v2/pools?status=enabled")
	assert.Equal(t, len(sortedResp), 2)
	_, has_pool3 = sortedResp["POOL3"]
	assert.Equal(t, has_pool3, false)

	// Check bad requests fail.
	testdb.CallV1Fail(t, "http://localhost:8080/v2/pools?status=enabled&status=bootstrap")
	testdb.CallV1Fail(t, "http://localhost:8080/v2/pools?status=badname")
}
