package timeseries_test

import (
	"sort"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/stretchr/testify/require"

	"gitlab.com/thorchain/midgard/config"
	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/internal/graphql"
	"gitlab.com/thorchain/midgard/internal/graphql/generated"
	"gitlab.com/thorchain/midgard/internal/graphql/model"
	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/internal/util"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

func TestMembersE2E(t *testing.T) {
	blocks := testdb.InitTestBlocks(t)

	schema := generated.NewExecutableSchema(generated.Config{Resolvers: &graphql.Resolver{}})
	gqlClient := client.New(handler.NewDefaultServer(schema))

	// thoraddr1: stake symetrical then unstake all using rune address (should not appear)
	blocks.NewBlock(t, "2020-09-01 00:10:00",
		testdb.PoolActivate{Pool: "BNB.ASSET1"},
		testdb.PoolActivate{Pool: "BNB.ASSET2"},
		testdb.AddLiquidity{
			Pool:                   "BNB.ASSET1",
			AssetAddress:           "bnbaddr1",
			RuneAddress:            "thoraddr1",
			LiquidityProviderUnits: 2,
		},
	)

	blocks.NewBlock(t, "2020-09-01 00:20:00",
		testdb.Withdraw{
			Pool:                   "BNB.ASSET1",
			FromAddress:            "thoraddr1",
			LiquidityProviderUnits: 2,
		})

	// thoraddr2: stake two pools then remove all from one (should appear)
	blocks.NewBlock(t, "2020-09-01 00:30:00",
		testdb.AddLiquidity{
			Pool:                   "BNB.ASSET1",
			RuneAddress:            "thoraddr2",
			LiquidityProviderUnits: 1,
		},
		testdb.AddLiquidity{
			Pool:                   "BNB.ASSET2",
			RuneAddress:            "thoraddr2",
			LiquidityProviderUnits: 1,
		},
	)

	blocks.NewBlock(t, "2020-09-01 00:40:00",
		testdb.Withdraw{
			Pool:                   "BNB.ASSET1",
			FromAddress:            "thoraddr2",
			LiquidityProviderUnits: 1,
		})

	// bnbaddr3: stake asym with asset address (should appear)
	blocks.NewBlock(t, "2020-09-01 00:50:00",
		testdb.AddLiquidity{
			Pool:                   "BNB.ASSET1",
			AssetAddress:           "bnbaddr3",
			LiquidityProviderUnits: 1,
		})

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
	blocks := testdb.InitTestBlocks(t)

	blocks.NewBlock(t, "2020-09-01 00:10:00",
		testdb.PoolActivate{Pool: "BNB.BNB"},
		testdb.PoolActivate{Pool: "BNB.TOKEN1"},
		testdb.PoolActivate{Pool: "BTC.BTC"},
		testdb.AddLiquidity{
			Pool:                   "BNB.BNB",
			RuneAmount:             100,
			AssetAmount:            200,
			RuneAddress:            "thoraddr1",
			AssetAddress:           "bnbaddr1",
			LiquidityProviderUnits: 1,
		},
		testdb.AddLiquidity{
			Pool:                   "BNB.TOKEN1",
			RuneAmount:             700,
			AssetAmount:            800,
			RuneAddress:            "thoraddr3",
			AssetAddress:           "bnbaddr1",
			LiquidityProviderUnits: 4,
		},
		testdb.AddLiquidity{
			Pool:                   "BTC.BTC",
			RuneAddress:            "thoraddr1",
			AssetAddress:           "btcaddr1",
			LiquidityProviderUnits: 5,
		},
		testdb.AddLiquidity{
			Pool:                   "BTC.BTC",
			AssetAddress:           "btcaddr1",
			LiquidityProviderUnits: 6,
		},
	)

	blocks.NewBlock(t, "2020-09-01 00:10:10",
		testdb.AddLiquidity{
			Pool:                   "BNB.BNB",
			RuneAmount:             300,
			AssetAmount:            400,
			RuneAddress:            "thoraddr1",
			AssetAddress:           "bnbaddr1",
			LiquidityProviderUnits: 2,
		},
		testdb.AddLiquidity{
			Pool:                   "BNB.BNB",
			RuneAmount:             500,
			RuneAddress:            "thoraddr1",
			LiquidityProviderUnits: 3,
		},
	)

	blocks.NewBlock(t, "2020-09-01 00:15:00",
		testdb.Withdraw{
			Pool:                   "BNB.BNB",
			FromAddress:            "thoraddr1",
			LiquidityProviderUnits: 1,
			EmitRune:               200,
			EmitAsset:              400,
		},
		testdb.Withdraw{
			Pool:                   "BTC.BTC",
			FromAddress:            "thoraddr1",
			LiquidityProviderUnits: 5,
		},
	)

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
	require.Equal(t, util.IntStr(1+2+3-1), bnbPool.LiquidityUnits)
	require.Equal(t, util.IntStr(100+300+500), bnbPool.RuneAdded)
	require.Equal(t, util.IntStr(200+400), bnbPool.AssetAdded)
	require.Equal(t, "200", bnbPool.RuneWithdrawn)
	require.Equal(t, "400", bnbPool.AssetWithdrawn)
	require.Equal(t, util.IntStr(db.StrToSec("2020-09-01 00:10:00").ToI()), bnbPool.DateFirstAdded)
	require.Equal(t, util.IntStr(db.StrToSec("2020-09-01 00:10:10").ToI()), bnbPool.DateLastAdded)

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
	blocks := testdb.InitTestBlocks(t)

	blocks.NewBlock(t, "2020-09-01 00:10:00",
		testdb.PoolActivate{Pool: "BNB.BNB"},
		testdb.AddLiquidity{
			Pool: "BNB.BNB", LiquidityProviderUnits: 1,
			RuneAddress: "thoraddr1",
		})

	blocks.NewBlock(t, "2020-09-01 00:11:00",
		testdb.AddLiquidity{
			Pool: "BNB.BNB", LiquidityProviderUnits: 1,
			RuneAddress:  "thoraddr1",
			AssetAddress: "bnbaddr2",
		})

	var jsonApiResult oapigen.MemberDetailsResponse
	body := testdb.CallJSON(t, "http://localhost:8080/v2/member/thoraddr1")
	testdb.MustUnmarshal(t, body, &jsonApiResult)

	require.Equal(t, 1, len(jsonApiResult.Pools))
	bnbPool := jsonApiResult.Pools[0]
	require.Equal(t, "thoraddr1", bnbPool.RuneAddress)
	require.Equal(t, "bnbaddr2", bnbPool.AssetAddress)
}

func TestMemberPending(t *testing.T) {
	blocks := testdb.InitTestBlocks(t)

	blocks.NewBlock(t, "2020-09-01 00:10:00",
		testdb.PoolActivate{Pool: "BNB.BNB"},
		testdb.PendingLiquidity{
			Pool:        "BNB.BNB",
			RuneAddress: "thoraddr1",
			RuneAmount:  10,
			AssetAmount: 15,
			PendingType: testdb.PendingAdd,
		})

	var jsonApiResult oapigen.MemberDetailsResponse
	body := testdb.CallJSON(t, "http://localhost:8080/v2/member/thoraddr1")
	testdb.MustUnmarshal(t, body, &jsonApiResult)

	require.Equal(t, 1, len(jsonApiResult.Pools))
	bnbPool := jsonApiResult.Pools[0]
	require.Equal(t, "thoraddr1", bnbPool.RuneAddress)

	require.Equal(t, "10", bnbPool.RunePending)
	require.Equal(t, "15", bnbPool.AssetPending)
	require.Equal(t, "BNB.BNB", bnbPool.Pool)
}

func TestMemberPendingAlreadyAdded(t *testing.T) {
	blocks := testdb.InitTestBlocks(t)

	blocks.NewBlock(t, "2020-09-01 00:10:00",
		testdb.PoolActivate{Pool: "BNB.BNB"},
		testdb.PendingLiquidity{
			Pool:         "BNB.BNB",
			RuneAddress:  "thoraddr1",
			AssetAddress: "assetaddr1",
			RuneAmount:   10,
			PendingType:  testdb.PendingAdd,
		})
	blocks.NewBlock(t, "2020-09-01 00:20:00",
		testdb.AddLiquidity{
			Pool:                   "BNB.BNB",
			RuneAmount:             10,
			AssetAmount:            10,
			LiquidityProviderUnits: 1,
			RuneAddress:            "thoraddr1",
			AssetAddress:           "assetaddr1",
		})
	blocks.NewBlock(t, "2020-09-01 00:30:00",
		testdb.PendingLiquidity{
			Pool:        "BNB.BNB",
			RuneAddress: "thoraddr1",
			AssetAmount: 100,
			PendingType: testdb.PendingAdd,
		})

	{ // search by rune address
		var jsonApiResult oapigen.MemberDetailsResponse
		body := testdb.CallJSON(t, "http://localhost:8080/v2/member/thoraddr1")
		testdb.MustUnmarshal(t, body, &jsonApiResult)

		require.Equal(t, 1, len(jsonApiResult.Pools))
		bnbPool := jsonApiResult.Pools[0]
		require.Equal(t, "thoraddr1", bnbPool.RuneAddress)
		require.Equal(t, "assetaddr1", bnbPool.AssetAddress)
		require.Equal(t, "0", bnbPool.RunePending)
		require.Equal(t, "100", bnbPool.AssetPending)
	}

	{ // search by asset address
		var jsonApiResult oapigen.MemberDetailsResponse
		body := testdb.CallJSON(t, "http://localhost:8080/v2/member/assetaddr1")
		testdb.MustUnmarshal(t, body, &jsonApiResult)

		require.Equal(t, 1, len(jsonApiResult.Pools))
		bnbPool := jsonApiResult.Pools[0]
		require.Equal(t, "thoraddr1", bnbPool.RuneAddress)
		require.Equal(t, "assetaddr1", bnbPool.AssetAddress)
		require.Equal(t, "0", bnbPool.RunePending)
		require.Equal(t, "100", bnbPool.AssetPending)
	}
}

func TestMemberOnlyAsset(t *testing.T) {
	blocks := testdb.InitTestBlocks(t)

	blocks.NewBlock(t, "2020-01-01 00:00:00",
		testdb.PoolActivate{Pool: "BNB.BNB"},
		testdb.AddLiquidity{
			Pool:                   "BNB.BNB",
			RuneAmount:             10,
			AssetAmount:            10,
			LiquidityProviderUnits: 20,
			RuneAddress:            "thoraddr1",
			AssetAddress:           "assetaddr1",
		})
	blocks.NewBlock(t, "2020-01-01 00:00:01",
		testdb.AddLiquidity{
			Pool:                   "BNB.BNB",
			RuneAmount:             0,
			AssetAmount:            10,
			LiquidityProviderUnits: 10,
			AssetAddress:           "assetaddr2",
		})
	blocks.NewBlock(t, "2020-01-01 00:00:02",
		testdb.Withdraw{
			Pool: "BNB.BNB", LiquidityProviderUnits: 5, FromAddress: "assetaddr2", EmitAsset: 5,
		})

	{
		var jsonApiResult oapigen.MemberDetailsResponse
		body := testdb.CallJSON(t, "http://localhost:8080/v2/member/assetaddr2")
		testdb.MustUnmarshal(t, body, &jsonApiResult)

		require.Equal(t, 1, len(jsonApiResult.Pools))
		bnbPool := jsonApiResult.Pools[0]
		require.Equal(t, "", bnbPool.RuneAddress)
		require.Equal(t, "assetaddr2", bnbPool.AssetAddress)
		require.Equal(t, "10", bnbPool.AssetAdded)
		require.Equal(t, "5", bnbPool.AssetWithdrawn)
	}

	{
		var jsonApiResult oapigen.PoolStatsDetail
		body := testdb.CallJSON(t, "http://localhost:8080/v2/pool/BNB.BNB/stats")
		testdb.MustUnmarshal(t, body, &jsonApiResult)

		require.Equal(t, "2", jsonApiResult.AddLiquidityCount)

	}
}

func TestMemberPendingAlreadyWithdrawn(t *testing.T) {
	blocks := testdb.InitTestBlocks(t)

	blocks.NewBlock(t, "2020-09-01 00:00:00",
		testdb.PoolActivate{Pool: "BNB.BNB"},
		testdb.AddLiquidity{
			Pool:                   "BNB.BNB",
			RuneAmount:             1,
			AssetAmount:            1,
			LiquidityProviderUnits: 1,
			RuneAddress:            "thoraddr1",
		})
	blocks.NewBlock(t, "2020-09-01 00:10:00",
		testdb.PendingLiquidity{
			Pool:        "BNB.BNB",
			RuneAddress: "thoraddr1",
			RuneAmount:  10,
			PendingType: testdb.PendingAdd,
		})
	blocks.NewBlock(t, "2020-09-01 00:20:00",
		testdb.PendingLiquidity{
			Pool:        "BNB.BNB",
			RuneAmount:  10,
			RuneAddress: "thoraddr1",
			PendingType: testdb.PendingWithdraw,
		})

	var jsonApiResult oapigen.MemberDetailsResponse
	body := testdb.CallJSON(t, "http://localhost:8080/v2/member/thoraddr1")
	testdb.MustUnmarshal(t, body, &jsonApiResult)

	require.Equal(t, 1, len(jsonApiResult.Pools))
	bnbPool := jsonApiResult.Pools[0]
	require.Equal(t, "thoraddr1", bnbPool.RuneAddress)
	require.Equal(t, "0", bnbPool.RunePending)
}

func TestMemberAsymRune(t *testing.T) {
	blocks := testdb.InitTestBlocks(t)

	blocks.NewBlock(t, "2020-09-01 00:10:00",
		testdb.AddLiquidity{
			Pool: "BNB.BNB", LiquidityProviderUnits: 1, RuneAddress: "thoraddr1",
		},
		testdb.PoolActivate{Pool: "BNB.BNB"})

	var jsonApiResult oapigen.MemberDetailsResponse
	body := testdb.CallJSON(t, "http://localhost:8080/v2/member/thoraddr1")
	testdb.MustUnmarshal(t, body, &jsonApiResult)

	require.Equal(t, 1, len(jsonApiResult.Pools))
	bnbPool := jsonApiResult.Pools[0]
	require.Equal(t, "thoraddr1", bnbPool.RuneAddress)
	require.Equal(t, "", bnbPool.AssetAddress)
}

func TestMembersPoolFilter(t *testing.T) {
	blocks := testdb.InitTestBlocks(t)

	blocks.NewBlock(t, "2020-09-01 00:00:00",
		testdb.AddLiquidity{
			Pool: "P1", LiquidityProviderUnits: 1, RuneAddress: "thoraddr1",
		},
		testdb.PoolActivate{Pool: "P1"})

	blocks.NewBlock(t, "2020-09-01 00:00:01",
		testdb.AddLiquidity{
			Pool: "P2", LiquidityProviderUnits: 1, RuneAddress: "thoraddr2",
		},
		testdb.PoolActivate{Pool: "P2"})

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

func TestMemberSeparation(t *testing.T) {
	// There are two separate members : (thoraddr, bnbaddr) ; (null, bnbaddr)
	// This test checks that those are separated
	blocks := testdb.InitTestBlocks(t)

	blocks.NewBlock(t, "2020-09-01 00:00:01",
		testdb.PoolActivate{Pool: "BNB.BNB"},
		testdb.AddLiquidity{
			Pool: "BNB.BNB", LiquidityProviderUnits: 1,
			RuneAddress: "thoraddr", AssetAddress: "bnbaddr",
		},
	)
	blocks.NewBlock(t, "2020-09-01 00:00:02",
		testdb.AddLiquidity{
			Pool: "BNB.BNB", LiquidityProviderUnits: 2,
			AssetAddress: "bnbaddr",
		})

	{
		var jsonResult oapigen.MembersResponse

		body := testdb.CallJSON(t, "http://localhost:8080/v2/members")
		testdb.MustUnmarshal(t, body, &jsonResult)

		require.Equal(t, 2, len(jsonResult))
		sort.Strings(jsonResult)
		require.Equal(t, "bnbaddr", jsonResult[0])
		require.Equal(t, "thoraddr", jsonResult[1])
	}
	{
		var jsonApiResult oapigen.MemberDetailsResponse
		body := testdb.CallJSON(t, "http://localhost:8080/v2/member/thoraddr")
		testdb.MustUnmarshal(t, body, &jsonApiResult)

		require.Equal(t, 1, len(jsonApiResult.Pools))
		bnbPool := jsonApiResult.Pools[0]
		require.Equal(t, "1", bnbPool.LiquidityUnits)
		require.Equal(t, "thoraddr", bnbPool.RuneAddress)
		require.Equal(t, "bnbaddr", bnbPool.AssetAddress)
	}
	{
		var jsonApiResult oapigen.MemberDetailsResponse
		body := testdb.CallJSON(t, "http://localhost:8080/v2/member/bnbaddr")
		testdb.MustUnmarshal(t, body, &jsonApiResult)

		require.Equal(t, 2, len(jsonApiResult.Pools))

		p := jsonApiResult.Pools
		sort.Slice(p, func(i, j int) bool {
			return p[i].RuneAddress < p[j].RuneAdded
		})

		assetaddrMember := jsonApiResult.Pools[0]
		require.Equal(t, "2", assetaddrMember.LiquidityUnits)
		require.Equal(t, "", assetaddrMember.RuneAddress)
		require.Equal(t, "bnbaddr", assetaddrMember.AssetAddress)

		thoraddrMember := jsonApiResult.Pools[1]
		require.Equal(t, "1", thoraddrMember.LiquidityUnits)
		require.Equal(t, "thoraddr", thoraddrMember.RuneAddress)
		require.Equal(t, "bnbaddr", thoraddrMember.AssetAddress)
	}
}

func TestMemberRecreated(t *testing.T) {
	// * A member is created: (thoraddr, bnbaddr)
	// * Then 100% of the assets are removed
	// * A new member is added without asset address: (thoraddr, null)
	// * check that the new member doesn't have the old asset address associated
	// This test checks that those are separated
	blocks := testdb.InitTestBlocks(t)

	blocks.NewBlock(t, "2020-09-01 00:00:01",
		testdb.AddLiquidity{
			Pool: "BNB.BNB", LiquidityProviderUnits: 1,
			RuneAddress: "thoraddr", AssetAddress: "bnbaddr",
		},
		testdb.PoolActivate{Pool: "BNB.BNB"})
	{
		var jsonApiResult oapigen.MemberDetailsResponse
		body := testdb.CallJSON(t, "http://localhost:8080/v2/member/thoraddr")
		testdb.MustUnmarshal(t, body, &jsonApiResult)

		require.Equal(t, 1, len(jsonApiResult.Pools))
		bnbPool := jsonApiResult.Pools[0]
		require.Equal(t, "1", bnbPool.LiquidityUnits)
		require.Equal(t, "thoraddr", bnbPool.RuneAddress)
		require.Equal(t, "bnbaddr", bnbPool.AssetAddress)
	}

	blocks.NewBlock(t, "2020-09-01 00:00:02",
		testdb.Withdraw{
			Pool: "BNB.BNB", LiquidityProviderUnits: 1, FromAddress: "thoraddr",
		})

	testdb.JSONFailGeneral(t, "http://localhost:8080/v2/member/thoraddr") // not found

	blocks.NewBlock(t, "2020-09-01 00:00:03",
		testdb.AddLiquidity{
			Pool: "BNB.BNB", LiquidityProviderUnits: 1,
			RuneAddress: "thoraddr",
		})

	{
		var jsonApiResult oapigen.MemberDetailsResponse
		body := testdb.CallJSON(t, "http://localhost:8080/v2/member/thoraddr")
		testdb.MustUnmarshal(t, body, &jsonApiResult)

		require.Equal(t, 1, len(jsonApiResult.Pools))
		bnbPool := jsonApiResult.Pools[0]
		require.Equal(t, "1", bnbPool.LiquidityUnits)
		require.Equal(t, "thoraddr", bnbPool.RuneAddress)
		require.Equal(t, "", bnbPool.AssetAddress)
	}

	testdb.JSONFailGeneral(t, "http://localhost:8080/v2/member/bnbaddr") // not found
}

func TestFullMember(t *testing.T) {
	config.Global.UsdPools = []string{"BNB.BAT-07A"}
	blocks := testdb.InitTestBlocks(t)

	blocks.NewBlock(t, "2020-09-01 00:10:00",
		testdb.AddLiquidity{
			Pool:                   "BNB.BAT-07A",
			RuneAmount:             100,
			AssetAmount:            200,
			RuneAddress:            "thoraddr1",
			AssetAddress:           "bnbaddr1",
			LiquidityProviderUnits: 1,
		},
	)
	blocks.NewBlock(t, "2020-09-02 00:10:00",
		testdb.AddLiquidity{
			Pool:                   "TERRA.LUNA",
			RuneAmount:             700,
			AssetAmount:            800,
			RuneAddress:            "thoraddr2",
			AssetAddress:           "bnbaddr1",
			LiquidityProviderUnits: 4,
		},
	)
	timeseries.ResetLatestStateForTest()
	{
		var jsonApiResult []oapigen.FullMemberPool
		body := testdb.CallJSON(t, "http://localhost:8080/v2/full_member?address=thoraddr1,thoraddr2")
		testdb.MustUnmarshal(t, body, &jsonApiResult)
		require.Equal(t, []oapigen.FullMemberPool{{
			AssetAdded:     "200",
			AssetAddress:   "bnbaddr1",
			AssetPending:   "0",
			AssetWithdrawn: "0",
			DateFirstAdded: "1598919000",
			DateLastAdded:  "1598919000",
			Pool:           "BNB.BAT-07A",
			PoolUnits:      "0",
			RuneAdded:      "100",
			RuneAddress:    "thoraddr1",
			RunePending:    "0",
			RuneWithdrawn:  "0",
			SharedUnits:    "1",
		}, {
			AssetAdded:     "800",
			AssetAddress:   "bnbaddr1",
			AssetPending:   "0",
			AssetWithdrawn: "0",
			DateFirstAdded: "1599005400",
			DateLastAdded:  "1599005400",
			Pool:           "TERRA.LUNA",
			PoolUnits:      "0",
			RuneAdded:      "700",
			RuneAddress:    "thoraddr2",
			RunePending:    "0",
			RuneWithdrawn:  "0",
			SharedUnits:    "4",
		}}, jsonApiResult)
	}
	blocks.NewBlock(t, "2020-09-03 00:10:00",
		testdb.PoolActivate{
			Pool: "LTC.LTC",
		},
		testdb.PoolActivate{
			Pool: "BCH.BCH",
		},
		testdb.AddLiquidity{
			Pool:                   "LTC.LTC",
			RuneAmount:             100,
			AssetAmount:            200,
			RuneAddress:            "thoraddr3",
			AssetAddress:           "bnbaddr2",
			LiquidityProviderUnits: 1,
		},
		testdb.AddLiquidity{
			Pool:                   "BCH.BCH",
			RuneAmount:             100,
			AssetAmount:            200,
			RuneAddress:            "thoraddr4",
			AssetAddress:           "bnbaddr2",
			LiquidityProviderUnits: 2,
		},
	)
	{
		var jsonApiResult []oapigen.FullMemberPool
		body := testdb.CallJSON(t, "http://localhost:8080/v2/full_member?address=thoraddr3,thoraddr4")
		testdb.MustUnmarshal(t, body, &jsonApiResult)

		require.Equal(t, []oapigen.FullMemberPool{{
			AssetAdded:     "200",
			AssetAddress:   "bnbaddr2",
			AssetPending:   "0",
			AssetWithdrawn: "0",
			DateFirstAdded: "1599091800",
			DateLastAdded:  "1599091800",
			Pool:           "LTC.LTC",
			PoolUnits:      "1",
			RuneAdded:      "100",
			RuneAddress:    "thoraddr3",
			RunePending:    "0",
			RuneWithdrawn:  "0",
			SharedUnits:    "1",
		}, {
			AssetAdded:     "200",
			AssetAddress:   "bnbaddr2",
			AssetPending:   "0",
			AssetWithdrawn: "0",
			DateFirstAdded: "1599091800",
			DateLastAdded:  "1599091800",
			Pool:           "BCH.BCH",
			PoolUnits:      "2",
			RuneAdded:      "100",
			RuneAddress:    "thoraddr4",
			RunePending:    "0",
			RuneWithdrawn:  "0",
			SharedUnits:    "2",
		}}, jsonApiResult)
	}
}
