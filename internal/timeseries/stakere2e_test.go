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

func TestStakerE2E(t *testing.T) {
	testdb.SetupTestDB(t)
	schema := generated.NewExecutableSchema(generated.Config{Resolvers: &graphql.Resolver{}})
	gqlClient := client.New(handler.NewDefaultServer(schema))

	testdb.MustExec(t, "DELETE FROM stake_events")

	testdb.InsertStakeEvent(t, testdb.FakeStake{
		Pool:           "BNB.BNB",
		BlockTimestamp: "2020-09-01 00:10:00",
		RuneE8:         100,
		RuneAddress:    "address1",
	})

	body := testdb.CallV1(t, "http://localhost:8080/v2/members/address1")
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

func TestStakersE2E(t *testing.T) {

	testdb.SetupTestDB(t)
	schema := generated.NewExecutableSchema(generated.Config{Resolvers: &graphql.Resolver{}})
	gqlClient := client.New(handler.NewDefaultServer(schema))

	testdb.MustExec(t, "DELETE FROM stake_events")

	testdb.InsertStakeEvent(t, testdb.FakeStake{
		Pool:           "BNB.BNB",
		BlockTimestamp: "2020-09-01 00:10:00",
		RuneAddress:    "address1",
	})

	body := testdb.CallV1(t, "http://localhost:8080/v2/members")
	var jsonApiResult oapigen.MembersResponse
	testdb.MustUnmarshal(t, body, &jsonApiResult)

	// poolsArray and TotalStaked fields are not implemented in the the stakers route
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
	assert.Equal(t, "address1", graphqlResult.Stakers[0].Address)
	assert.Equal(t, "address1", jsonApiResult[0])
}
