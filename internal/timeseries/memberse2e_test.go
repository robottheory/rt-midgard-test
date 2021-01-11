package timeseries_test

import (
	"fmt"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/stretchr/testify/assert"

	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/internal/graphql"
	"gitlab.com/thorchain/midgard/internal/graphql/generated"
	"gitlab.com/thorchain/midgard/internal/graphql/model"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

func TestMembersE2E(t *testing.T) {
	testdb.SetupTestDB(t)
	schema := generated.NewExecutableSchema(generated.Config{Resolvers: &graphql.Resolver{}})
	gqlClient := client.New(handler.NewDefaultServer(schema))

	testdb.MustExec(t, "DELETE FROM stake_events")
	testdb.MustExec(t, "DELETE FROM unstake_events")

	// thoraddr1: stake symetrical then unstake all using both addresses (should not appear)
	testdb.InsertStakeEvent(t,
		testdb.FakeStake{Pool: "BNB.ASSET1", AssetAddress: "bnbaddr1", RuneAddress: "thoraddr1", StakeUnits: 2})
	testdb.InsertUnstakeEvent(t,
		testdb.FakeUnstake{Pool: "BNB.ASSET1", FromAddr: "thoraddr1", StakeUnits: 1})
	testdb.InsertUnstakeEvent(t,
		testdb.FakeUnstake{Pool: "BNB.ASSET1", FromAddr: "bnbaddr1", StakeUnits: 1})

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

	body := testdb.CallV1(t, "http://localhost:8080/v2/members")

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
	assert.Equal(t, len(jsonApiResult), len(graphqlResult.Stakers))

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

	if !thor2There {
		t.Fatal("thoraddr2 should be part of the response")
	}
	if !bnb3There {
		t.Fatal("bnbaddr3 should be part of the response")
	}
}

func TestMemberE2E(t *testing.T) {
	testdb.SetupTestDB(t)
	schema := generated.NewExecutableSchema(generated.Config{Resolvers: &graphql.Resolver{}})
	gqlClient := client.New(handler.NewDefaultServer(schema))

	testdb.MustExec(t, "DELETE FROM stake_events")

	testdb.InsertStakeEvent(t, testdb.FakeStake{
		Pool:           "BNB.BNB",
		BlockTimestamp: "2020-09-01 00:10:00",
		RuneE8:         100,
		RuneAddress:    "address1",
		StakeUnits:     1,
	})

	body := testdb.CallV1(t, "http://localhost:8080/v2/member/address1")
	var jsonApiResult oapigen.MemberDetailsResponse
	testdb.MustUnmarshal(t, body, &jsonApiResult)

	// poolsArray and TotalStaked fields are not implemented in the the stakers route
	queryString := fmt.Sprintf(`{
	  staker(address: "%s") {
		address
    	poolsArray
    	totalStaked
	  }
	}`, "address1")

	type Result struct {
		Staker model.Staker
	}
	var graphqlResult Result
	gqlClient.MustPost(queryString, &graphqlResult)

	assert.Equal(t, "address1", graphqlResult.Staker.Address)
	assert.Equal(t, len(jsonApiResult.Pools), len(graphqlResult.Staker.PoolsArray))
	assert.Equal(t, "BNB.BNB", jsonApiResult.Pools[0].Pool)
	assert.Equal(t, "BNB.BNB", *graphqlResult.Staker.PoolsArray[0])
	assert.Equal(t, "100", jsonApiResult.Pools[0].RuneAdded)
	assert.Equal(t, int64(100), graphqlResult.Staker.TotalStaked)
}
