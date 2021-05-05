// End to end tests here are checkning lookup funcionality from Database to HTTP Api.

package timeseries_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/stretchr/testify/require"
	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/internal/graphql"
	"gitlab.com/thorchain/midgard/internal/graphql/generated"
	"gitlab.com/thorchain/midgard/internal/graphql/model"

	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

func callPools(t *testing.T, url string) map[string]oapigen.PoolDetail {
	body := testdb.CallJSON(t, url)

	var response oapigen.PoolsResponse
	testdb.MustUnmarshal(t, body, &response)
	sortedResp := map[string]oapigen.PoolDetail{}

	for _, poolDetail := range response {
		sortedResp[poolDetail.Asset] = poolDetail
	}
	return sortedResp
}

func callPoolsGraphql() map[string]model.Pool {
	schema := generated.NewExecutableSchema(generated.Config{Resolvers: &graphql.Resolver{}})
	gqlClient := client.New(handler.NewDefaultServer(schema))

	queryString := `{
	  pools(limit: 10) {
	  asset
	  depth {
		assetDepth
		runeDepth
	  }
	  poolAPY
	  price
	  status
	  units
      volume24h
	  }
	}`

	type Response struct {
		Pools []*model.Pool
	}
	var response Response
	gqlClient.MustPost(queryString, &response)

	sortedResp := map[string]model.Pool{}

	for _, poolDetail := range response.Pools {
		sortedResp[poolDetail.Asset] = *poolDetail
	}
	return sortedResp
}

func callPoolGraphqlFail(t *testing.T, gqlClient *client.Client, pool string) {
	queryString := fmt.Sprintf(`{
		pool(asset: "%s") {
		  asset
		  depth {
			assetDepth
			runeDepth
		  }
		  poolAPY
		  price
		  status
		  units
		  volume24h
		}
	}`, pool)

	type GraphqlResult struct {
		Pool model.Pool
	}
	var graphqlResult GraphqlResult

	err := gqlClient.Post(queryString, &graphqlResult)
	if err == nil {
		t.Fatal("Query was expected to fail, but didn't:", queryString)
	}
}

func TestPoolsE2E(t *testing.T) {
	testdb.InitTest(t)

	testdb.InsertStakeEvent(t, testdb.FakeStake{Pool: "BNB.BNB", BlockTimestamp: "2020-01-01 00:00:00"})
	testdb.InsertStakeEvent(t, testdb.FakeStake{Pool: "POOL2"})
	testdb.InsertStakeEvent(t, testdb.FakeStake{Pool: "POOL3"})

	testdb.InsertPoolEvents(t, "BNB.BNB", "Available")
	testdb.InsertPoolEvents(t, "POOL2", "Available")
	testdb.InsertPoolEvents(t, "POOL3", "Staged")

	depths := []timeseries.Depth{
		{"BNB.BNB", 2, 1},
		{"POOL2", 2, 1},
		{"POOL3", 2, 1},
	}
	timeseries.SetDepthsForTest(depths)

	sortedResp := callPools(t, "http://localhost:8080/v2/pools")
	sortedRespGraphql := callPoolsGraphql()

	require.Equal(t, len(sortedResp), 3)
	require.Equal(t, len(sortedRespGraphql), 3)
	require.Equal(t, sortedResp["POOL2"].AssetDepth, "2")
	require.Equal(t, sortedRespGraphql["POOL2"].Depth.AssetDepth, int64(2))
	require.Equal(t, sortedResp["POOL2"].RuneDepth, "1")
	require.Equal(t, sortedRespGraphql["POOL2"].Depth.RuneDepth, int64(1))
	require.Equal(t, sortedResp["POOL2"].AssetPrice, "0.5")
	require.Equal(t, sortedRespGraphql["POOL2"].Price, 0.5)
	_, has_pool3 := sortedResp["POOL3"]
	require.Equal(t, has_pool3, true) // Without filter we have the Staged pool
	_, has_pool3_graphql := sortedRespGraphql["POOL3"]
	require.Equal(t, has_pool3_graphql, true)

	// check filtering
	sortedResp = callPools(t, "http://localhost:8080/v2/pools?status=available")
	require.Equal(t, len(sortedResp), 2)
	_, has_pool3 = sortedResp["POOL3"]
	require.Equal(t, has_pool3, false)

	// Check bad requests fail.
	testdb.JSONFailGeneral(t, "http://localhost:8080/v2/pools?status=available&status=staged")
	testdb.JSONFailGeneral(t, "http://localhost:8080/v2/pools?status=badname")
}

func TestPoolE2E(t *testing.T) {
	testdb.InitTest(t)
	schema := generated.NewExecutableSchema(generated.Config{Resolvers: &graphql.Resolver{}})
	gqlClient := client.New(handler.NewDefaultServer(schema))
	timeseries.SetLastTimeForTest(testdb.StrToSec("2020-09-01 23:00:00"))
	timeseries.SetDepthsForTest([]timeseries.Depth{{"BNB.TWT-123", 30000000000000, 2240582804123679}})

	testdb.InsertPoolEvents(t, "BNB.TWT-123", "Enabled")
	testdb.InsertBlockPoolDepth(t, "BNB.TWT-123", 4, 5, "2020-09-01 00:00:00")

	testdb.InsertSwapEvent(t, testdb.FakeSwap{
		Pool: "BNB.TWT-123", FromAsset: "BNB.RUNE",
		FromE8: 30000, LiqFeeInRuneE8: 2,
		BlockTimestamp: "2020-09-01 00:00:00",
	})
	testdb.InsertSwapEvent(t, testdb.FakeSwap{
		Pool: "BNB.TWT-123", FromAsset: "BNB.TWT-123",
		FromE8: 20000, ToE8: 25000 - 2, LiqFeeInRuneE8: 2,
		BlockTimestamp: "2020-09-01 13:00:00",
	})

	testdb.InsertStakeEvent(t, testdb.FakeStake{Pool: "BNB.TWT-123", BlockTimestamp: "2020-09-01 00:00:00", StakeUnits: 80})
	testdb.InsertUnstakeEvent(t, testdb.FakeUnstake{Pool: "BNB.TWT-123", Asset: "BNB.TWT-123", BlockTimestamp: "2020-09-01 00:00:00", StakeUnits: 30})

	queryString := `{
		pool(asset: "BNB.TWT-123") {
			asset
		  depth {
			assetDepth
			runeDepth
		  }
		  poolAPY
		  price
		  status
		  units
		  volume24h
		}
	}`

	type GraphqlResult struct {
		Pool model.Pool
	}
	var graphqlResult GraphqlResult
	gqlClient.MustPost(queryString, &graphqlResult)

	body := testdb.CallJSON(t, "http://localhost:8080/v2/pool/BNB.TWT-123")
	var jsonApiResponse oapigen.PoolResponse
	testdb.MustUnmarshal(t, body, &jsonApiResponse)

	require.Equal(t, "BNB.TWT-123", jsonApiResponse.Asset)
	require.Equal(t, "BNB.TWT-123", graphqlResult.Pool.Asset)
	// stake - unstake -> 80 - 30 = 50
	require.Equal(t, "50", jsonApiResponse.Units)
	require.Equal(t, int64(50), graphqlResult.Pool.Units)
	// runeDepth / assetDepth
	require.Equal(t, "74.6860934707893", jsonApiResponse.AssetPrice)
	require.Equal(t, 74.6860934707893, graphqlResult.Pool.Price)
	require.Greater(t, mustParseFloat(t, jsonApiResponse.PoolAPY), 0.)
	require.Greater(t, graphqlResult.Pool.PoolApy, 0.)

	// 30000 + 5/4 * 20000
	require.Equal(t, "55000", jsonApiResponse.Volume24h)
	require.Equal(t, int64(55000), graphqlResult.Pool.Volume24h)
	require.Equal(t, "enabled", jsonApiResponse.Status)
	require.Equal(t, "enabled", graphqlResult.Pool.Status)

	// Tests for not existing pools
	testdb.JSONFailGeneral(t, "http://localhost:8080/v2/pools/BNB.BNB")
	callPoolGraphqlFail(t, gqlClient, "BNB.BNB")
}

func mustParseFloat(t *testing.T, s string) float64 {
	ret, err := strconv.ParseFloat(s, 64)
	if err != nil {
		require.Fail(t, "Couldn't parse result float: ", s)
	}
	return ret
}

func TestGenesisNodeGoesOut(t *testing.T) {
	testdb.InitTest(t)
	testdb.InsertUpdateNodeAccountStatusEvent(t,
		testdb.FakeNodeStatus{NodeAddr: "node1", Former: "Standby", Current: "Active"},
		"2020-09-02 12:00:00")
	testdb.InsertUpdateNodeAccountStatusEvent(t,
		testdb.FakeNodeStatus{NodeAddr: "node2", Former: "Standby", Current: "Active"},
		"2020-09-02 12:00:00")
	testdb.InsertUpdateNodeAccountStatusEvent(t,
		testdb.FakeNodeStatus{NodeAddr: "genesisNode", Former: "Active", Current: "Standby"},
		"2020-09-03 12:00:00")

	n, err := timeseries.ActiveNodeCount(context.Background(),
		testdb.StrToSec("2020-09-10 12:00:00").ToNano())
	require.NoError(t, err)
	require.Equal(t, int64(2), n)
}
