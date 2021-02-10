package stat_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/internal/timeseries/stat"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

func TestUsdPrices(t *testing.T) {
	testdb.SetupTestDB(t)
	timeseries.SetLastTimeForTest(testdb.StrToSec("2020-12-20 23:00:00"))
	timeseries.SetDepthsForTest([]timeseries.Depth{
		{Pool: "BNB.BNB", AssetDepth: 2000, RuneDepth: 1000},
		{Pool: "USDA", AssetDepth: 100, RuneDepth: 300},
		{Pool: "USDB", AssetDepth: 1000, RuneDepth: 5000},
	})

	stat.SetUsdPoolWhitelistForTest([]string{"USDA", "USDB"})

	{
		body := testdb.CallV1(t,
			"http://localhost:8080/v2/stats")

		var result oapigen.StatsData
		testdb.MustUnmarshal(t, body, &result)
		assert.Equal(t, "5", result.RunePriceUSD)
	}

	{
		body := testdb.CallV1(t,
			"http://localhost:8080/v2/pool/BNB.BNB/stats")

		var result oapigen.PoolStatsDetail
		testdb.MustUnmarshal(t, body, &result)
		assert.Equal(t, "10", result.AssetPriceUSD)
	}

	// TODO(acsaba): fix prices in pool list
	// {
	// 	body := testdb.CallV1(t,
	// 		"http://localhost:8080/v2/pool/BNB.BNB")

	// 	var result oapigen.PoolDetail
	// 	testdb.MustUnmarshal(t, body, &result)
	// 	assert.Equal(t, "10", result.AssetPriceUSD)
	// }
}
