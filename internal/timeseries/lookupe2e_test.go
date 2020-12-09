// End to end tests here are checkning lookup funcionality from Database to HTTP Api.

package timeseries_test

import (
	"fmt"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/internal/graphql"
	"gitlab.com/thorchain/midgard/internal/graphql/generated"
	"gitlab.com/thorchain/midgard/internal/graphql/model"

	"github.com/stretchr/testify/assert"

	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

func callPools(t *testing.T, url string) map[string]oapigen.PoolDetail {
	body := testdb.CallV1(t, url)

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

	depths := []timeseries.Depth{
		{"BNB.BNB", 2, 1},
		{"POOL2", 2, 1},
		{"POOL3", 2, 1},
	}
	timeseries.SetDepthsForTest(depths)

	sortedResp := callPools(t, "http://localhost:8080/v2/pools")
	sortedRespGraphql := callPoolsGraphql()

	assert.Equal(t, len(sortedResp), 3)
	assert.Equal(t, len(sortedRespGraphql), 3)
	assert.Equal(t, sortedResp["POOL2"].AssetDepth, "2")
	assert.Equal(t, sortedRespGraphql["POOL2"].Depth.AssetDepth, int64(2))
	assert.Equal(t, sortedResp["POOL2"].RuneDepth, "1")
	assert.Equal(t, sortedRespGraphql["POOL2"].Depth.RuneDepth, int64(1))
	assert.Equal(t, sortedResp["POOL2"].AssetPrice, "0.5")
	assert.Equal(t, sortedRespGraphql["POOL2"].Price, 0.5)
	_, has_pool3 := sortedResp["POOL3"]
	assert.Equal(t, has_pool3, true) // Without filter we have the Bootstrap pool
	_, has_pool3_graphql := sortedRespGraphql["POOL3"]
	assert.Equal(t, has_pool3_graphql, true)

	// check filtering
	sortedResp = callPools(t, "http://localhost:8080/v2/pools?status=enabled")
	assert.Equal(t, len(sortedResp), 2)
	_, has_pool3 = sortedResp["POOL3"]
	assert.Equal(t, has_pool3, false)

	// Check bad requests fail.
	testdb.CallV1Fail(t, "http://localhost:8080/v2/pools?status=enabled&status=bootstrap")
	testdb.CallV1Fail(t, "http://localhost:8080/v2/pools?status=badname")
}

func TestPoolE2E(t *testing.T) {
	testdb.SetupTestDB(t)
	schema := generated.NewExecutableSchema(generated.Config{Resolvers: &graphql.Resolver{}})
	gqlClient := client.New(handler.NewDefaultServer(schema))
	timeseries.SetLastTimeForTest(testdb.ToTime("2020-09-01 23:00:00"))
	timeseries.SetDepthsForTest([]timeseries.Depth{{"BNB.TWT-123", 30000000000000, 2240582804123679}})

	testdb.MustExec(t, "DELETE FROM stake_events")
	testdb.MustExec(t, "DELETE FROM unstake_events")
	testdb.MustExec(t, "DELETE FROM swap_events")
	testdb.MustExec(t, "DELETE FROM block_pool_depths")
	testdb.MustExec(t, "DELETE FROM pool_events")

	testdb.InsertPoolEvents(t, "BNB.TWT-123", "Enabled")
	testdb.InsertBlockPoolDepth(t, "BNB.TWT-123", 4, 5, "2020-09-01 00:00:00")
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.TWT-123", FromAsset: "BNB.RUNE", FromE8: 30000, LiqFeeInRuneE8: 39087201297999, BlockTimestamp: "2020-09-01 00:00:00"})
	testdb.InsertSwapEvent(t, testdb.FakeSwap{Pool: "BNB.TWT-123", FromAsset: "BNB.TWT-123", FromE8: 20000, LiqFeeInRuneE8: 39087201297999, BlockTimestamp: "2020-09-01 13:00:00"})
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

	body := testdb.CallV1(t, "http://localhost:8080/v2/pool/BNB.TWT-123")
	var jsonApiResponse oapigen.PoolResponse
	testdb.MustUnmarshal(t, body, &jsonApiResponse)

	assert.Equal(t, "BNB.TWT-123", jsonApiResponse.Asset)
	assert.Equal(t, "BNB.TWT-123", graphqlResult.Pool.Asset)
	// stake - unstake -> 80 - 30 = 50
	assert.Equal(t, "50", jsonApiResponse.Units)
	assert.Equal(t, int64(50), graphqlResult.Pool.Units)
	// runeDepth / assetDepth
	assert.Equal(t, "74.6860934707893", jsonApiResponse.AssetPrice)
	assert.Equal(t, 74.6860934707893, graphqlResult.Pool.Price)
	assert.Equal(t, "1.4579401225658155", jsonApiResponse.PoolAPY)
	assert.Equal(t, 1.4579401225658155, graphqlResult.Pool.PoolApy)
	// 30000 + 5/4 * 20000
	assert.Equal(t, "55000", jsonApiResponse.Volume24h)
	assert.Equal(t, int64(55000), graphqlResult.Pool.Volume24h)
	assert.Equal(t, "enabled", jsonApiResponse.Status)
	assert.Equal(t, "enabled", graphqlResult.Pool.Status)

	// Tests for not existing pools
	testdb.CallV1Fail(t, "http://localhost:8080/v2/pools/BNB.BNB")
	callPoolGraphqlFail(t, gqlClient, "BNB.BNB")
}
