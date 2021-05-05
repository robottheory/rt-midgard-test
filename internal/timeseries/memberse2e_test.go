package timeseries_test

import (
	"sort"
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

func TestMembersE2E(t *testing.T) {
	testdb.SetupTestDB(t)
	schema := generated.NewExecutableSchema(generated.Config{Resolvers: &graphql.Resolver{}})
	gqlClient := client.New(handler.NewDefaultServer(schema))

	testdb.MustExec(t, "DELETE FROM stake_events")
	testdb.MustExec(t, "DELETE FROM unstake_events")

	// thoraddr1: stake symetrical then unstake all using rune address (should not appear)
	testdb.InsertStakeEvent(t,
		testdb.FakeStake{Pool: "BNB.ASSET1", AssetAddress: "bnbaddr1", RuneAddress: "thoraddr1", StakeUnits: 2})
	testdb.InsertUnstakeEvent(t,
		testdb.FakeUnstake{Pool: "BNB.ASSET1", FromAddr: "thoraddr1", StakeUnits: 2})

	// thoraddr2: stake two pools then remove all from one (should appear)
	testdb.InsertStakeEvent(t,
		testdb.FakeStake{Pool: "BNB.ASSET1", RuneAddress: "thoraddr2", StakeUnits: 1})
	testdb.InsertStakeEvent(t,
		testdb.FakeStake{Pool: "BNB.ASSET2", RuneAddress: "thoraddr2", StakeUnits: 1})
	testdb.InsertUnstakeEvent(t,
		testdb.FakeUnstake{Pool: "BNB.ASSET1", FromAddr: "thoraddr2", StakeUnits: 1})

	// bnbaddr3: stake asym with asset address (should appear)
	testdb.InsertStakeEvent(t,
		testdb.FakeStake{Pool: "BNB.ASSET1", AssetAddress: "bnbaddr3", StakeUnits: 1})

	body := testdb.CallJSON(t, "http://localhost:8080/v2/members")

	var jsonApiResult oapigen.MembersResponse
	testdb.MustUnmarshal(t, body, &jsonApiResult)

	queryString := `{
	  stakers {
		address
	  }
	}`
	type Result struct {
		Stakers []model.Staker
	}
	var graphqlResult Result
	gqlClient.MustPost(queryString, &graphqlResult)
	require.Equal(t, len(jsonApiResult), len(graphqlResult.Stakers))

	thor2There := false
	bnb3There := false

	for _, addr := range jsonApiResult {
		switch addr {
		case "thoraddr1", "bnbaddr1", "bnbaddr2":
			t.Fatal(addr + " should not be part of the response")
		case "thoraddr2":
			thor2There = true
		case "bnbaddr3":
			bnb3There = true
		}
	}

	require.True(t, thor2There)
	require.True(t, bnb3There)
}

func TestMemberE2E(t *testing.T) {
	testdb.SetupTestDB(t)

	testdb.MustExec(t, "DELETE FROM stake_events")
	testdb.MustExec(t, "DELETE FROM unstake_events")

	testdb.InsertStakeEvent(t, testdb.FakeStake{
		Pool:           "BNB.BNB",
		BlockTimestamp: "2020-09-01 00:10:00",
		RuneE8:         100,
		AssetE8:        200,
		RuneAddress:    "thoraddr1",
		AssetAddress:   "bnbaddr1",
		StakeUnits:     1,
	})
	testdb.InsertStakeEvent(t, testdb.FakeStake{
		Pool:           "BNB.BNB",
		BlockTimestamp: "2020-09-01 00:10:10",
		RuneE8:         300,
		AssetE8:        400,
		RuneAddress:    "thoraddr1",
		AssetAddress:   "bnbaddr1",
		StakeUnits:     2,
	})
	testdb.InsertStakeEvent(t, testdb.FakeStake{
		Pool:           "BNB.BNB",
		BlockTimestamp: "2020-09-01 00:10:10",
		RuneE8:         500,
		RuneAddress:    "thoraddr1",
		StakeUnits:     3,
	})
	testdb.InsertStakeEvent(t, testdb.FakeStake{
		Pool:           "BNB.TOKEN1",
		BlockTimestamp: "2020-09-01 00:10:00",
		RuneE8:         700,
		AssetE8:        800,
		RuneAddress:    "thoraddr3",
		AssetAddress:   "bnbaddr1",
		StakeUnits:     4,
	})
	testdb.InsertStakeEvent(t, testdb.FakeStake{
		Pool:           "BTC.BTC",
		BlockTimestamp: "2020-09-01 00:10:00",
		RuneAddress:    "thoraddr1",
		AssetAddress:   "btcaddr1",
		StakeUnits:     5,
	})
	testdb.InsertStakeEvent(t, testdb.FakeStake{
		Pool:           "BTC.BTC",
		BlockTimestamp: "2020-09-01 00:10:00",
		AssetAddress:   "btcaddr1",
		StakeUnits:     6,
	})

	testdb.InsertUnstakeEvent(t, testdb.FakeUnstake{
		Pool:           "BNB.BNB",
		BlockTimestamp: "2020-09-01 00:15:00",
		FromAddr:       "thoraddr1",
		StakeUnits:     1,
		EmitRuneE8:     200,
		EmitAssetE8:    400,
	})
	testdb.InsertUnstakeEvent(t, testdb.FakeUnstake{
		Pool:           "BTC.BTC",
		BlockTimestamp: "2020-09-01 00:15:00",
		FromAddr:       "thoraddr1",
		StakeUnits:     5,
	})

	var jsonApiResult oapigen.MemberDetailsResponse
	// thoraddr1
	//	- BNB.BNB pool
	//	- BTC.BTC should not show as it has 0 Liquidity units
	body := testdb.CallJSON(t, "http://localhost:8080/v2/member/thoraddr1")
	testdb.MustUnmarshal(t, body, &jsonApiResult)

	require.Equal(t, 1, len(jsonApiResult.Pools))
	bnbPool := jsonApiResult.Pools[0]
	require.Equal(t, "BNB.BNB", bnbPool.Pool)
	require.Equal(t, "thoraddr1", bnbPool.RuneAddress)
	require.Equal(t, "bnbaddr1", bnbPool.AssetAddress)
	require.Equal(t, intStr(1+2+3-1), bnbPool.LiquidityUnits)
	require.Equal(t, intStr(100+300+500), bnbPool.RuneAdded)
	require.Equal(t, intStr(200+400), bnbPool.AssetAdded)
	require.Equal(t, "200", bnbPool.RuneWithdrawn)
	require.Equal(t, "400", bnbPool.AssetWithdrawn)
	require.Equal(t, intStr(testdb.StrToSec("2020-09-01 00:10:00").ToI()), bnbPool.DateFirstAdded)
	require.Equal(t, intStr(testdb.StrToSec("2020-09-01 00:10:10").ToI()), bnbPool.DateLastAdded)

	// bnbaddr1
	// - BNB.BNB
	// - BNB.TOKEN1
	body = testdb.CallJSON(t, "http://localhost:8080/v2/member/bnbaddr1")
	testdb.MustUnmarshal(t, body, &jsonApiResult)
	require.Equal(t, 2, len(jsonApiResult.Pools))
	bnbPools := jsonApiResult.Pools
	tokenIsThere := false
	bnbIsThere := false
	for _, pool := range bnbPools {
		switch pool.Pool {
		case "BNB.TOKEN1":
			tokenIsThere = true
		case "BNB.BNB":
			bnbIsThere = true
		}
	}
	require.True(t, tokenIsThere)
	require.True(t, bnbIsThere)

	// btcaddr1
	// - Asym BTC.BTC only (the sym one has 0 liquidity units)
	body = testdb.CallJSON(t, "http://localhost:8080/v2/member/btcaddr1")
	testdb.MustUnmarshal(t, body, &jsonApiResult)
	require.Equal(t, 1, len(jsonApiResult.Pools))
	btcPool := jsonApiResult.Pools[0]
	require.Equal(t, "BTC.BTC", btcPool.Pool)
	require.Equal(t, "", btcPool.RuneAddress)
}

func TestMemberPicksFirstAssetAddress(t *testing.T) {
	testdb.SetupTestDB(t)

	testdb.MustExec(t, "DELETE FROM stake_events")
	testdb.MustExec(t, "DELETE FROM unstake_events")

	testdb.InsertStakeEvent(t, testdb.FakeStake{
		Pool:           "BNB.BNB",
		BlockTimestamp: "2020-09-01 00:10:00",
		RuneAddress:    "thoraddr1",
		StakeUnits:     1,
	})
	testdb.InsertStakeEvent(t, testdb.FakeStake{
		Pool:           "BNB.BNB",
		BlockTimestamp: "2020-09-01 00:11:00",
		RuneAddress:    "thoraddr1",
		AssetAddress:   "bnbaddr2",
		StakeUnits:     1,
	})

	var jsonApiResult oapigen.MemberDetailsResponse
	body := testdb.CallJSON(t, "http://localhost:8080/v2/member/thoraddr1")
	testdb.MustUnmarshal(t, body, &jsonApiResult)

	require.Equal(t, 1, len(jsonApiResult.Pools))
	bnbPool := jsonApiResult.Pools[0]
	require.Equal(t, "thoraddr1", bnbPool.RuneAddress)
	require.Equal(t, "bnbaddr2", bnbPool.AssetAddress)
}

func TestMemberAsymRune(t *testing.T) {
	testdb.SetupTestDB(t)

	testdb.MustExec(t, "DELETE FROM stake_events")
	testdb.MustExec(t, "DELETE FROM unstake_events")

	testdb.InsertStakeEvent(t, testdb.FakeStake{
		Pool:           "BNB.BNB",
		BlockTimestamp: "2020-09-01 00:10:00",
		RuneAddress:    "thoraddr1",
		StakeUnits:     1,
	})

	var jsonApiResult oapigen.MemberDetailsResponse
	body := testdb.CallJSON(t, "http://localhost:8080/v2/member/thoraddr1")
	testdb.MustUnmarshal(t, body, &jsonApiResult)

	require.Equal(t, 1, len(jsonApiResult.Pools))
	bnbPool := jsonApiResult.Pools[0]
	require.Equal(t, "thoraddr1", bnbPool.RuneAddress)
	require.Equal(t, "", bnbPool.AssetAddress)
}

func TestMembersPoolFilter(t *testing.T) {
	testdb.SetupTestDB(t)
	timeseries.SetDepthsForTest([]timeseries.Depth{
		{Pool: "P1", AssetDepth: 1, RuneDepth: 1},
	})

	testdb.MustExec(t, "DELETE FROM stake_events")
	testdb.MustExec(t, "DELETE FROM unstake_events")

	testdb.InsertStakeEvent(t, testdb.FakeStake{
		Pool:        "P1",
		RuneAddress: "thoraddr1",
		StakeUnits:  1,
	})
	testdb.InsertStakeEvent(t, testdb.FakeStake{
		Pool:        "P2",
		RuneAddress: "thoraddr2",
		StakeUnits:  1,
	})

	{
		body := testdb.CallJSON(t, "http://localhost:8080/v2/members")

		var jsonApiResult oapigen.MembersResponse
		testdb.MustUnmarshal(t, body, &jsonApiResult)

		sort.Strings(jsonApiResult)
		require.Equal(t, []string{"thoraddr1", "thoraddr2"}, []string(jsonApiResult))
	}
	{
		body := testdb.CallJSON(t, "http://localhost:8080/v2/members?pool=P1")

		var jsonApiResult oapigen.MembersResponse
		testdb.MustUnmarshal(t, body, &jsonApiResult)

		sort.Strings(jsonApiResult)
		require.Equal(t, []string{"thoraddr1"}, []string(jsonApiResult))
	}
}

func intStr(v int64) string {
	return strconv.FormatInt(v, 10)
}
