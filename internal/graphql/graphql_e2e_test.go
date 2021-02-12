package graphql_test

import (
	"fmt"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/stretchr/testify/require"
	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/internal/graphql"
	"gitlab.com/thorchain/midgard/internal/graphql/generated"
	"gitlab.com/thorchain/midgard/internal/graphql/model"
)

func TestStakeHistoryE2E(t *testing.T) {
	testdb.SetupTestDB(t)
	schema := generated.NewExecutableSchema(generated.Config{Resolvers: &graphql.Resolver{}})
	gqlClient := client.New(handler.NewDefaultServer(schema))
	testdb.MustExec(t, "DELETE FROM stake_events")

	// first one is skipped
	testdb.InsertStakeEvent(t, testdb.FakeStake{Pool: "BNB.TWT-123", BlockTimestamp: "2020-09-02 08:00:00", RuneE8: 200, AssetE8: 1000, StakeUnits: 50})
	testdb.InsertStakeEvent(t, testdb.FakeStake{Pool: "BNB.TWT-123", BlockTimestamp: "2020-09-03 10:00:00", RuneE8: 1000, AssetE8: 5000, StakeUnits: 200})
	testdb.InsertStakeEvent(t, testdb.FakeStake{Pool: "BNB.TWT-123", BlockTimestamp: "2020-09-05 11:00:00", RuneE8: 3000, AssetE8: 2000, StakeUnits: 100})
	testdb.InsertStakeEvent(t, testdb.FakeStake{Pool: "BNB.TWT-123", BlockTimestamp: "2020-09-05 12:00:00", RuneE8: 1500, AssetE8: 4000, StakeUnits: 300})

	from := testdb.StrToSec("2020-09-03 00:00:01")
	until := testdb.StrToSec("2020-09-06 00:00:00")

	queryString := fmt.Sprintf(`{
		stakeHistory(pool: "BNB.TWT-123", from: %d, until: %d, interval: DAY) {
		  	meta {
				count
				first
				last
				runeVolume
				assetVolume
				units
			}
			intervals {
				time
				count
				runeVolume
				assetVolume
				units
		  }
		}
	}`, from, until)

	type Result struct {
		StakeHistory model.PoolStakeHistory
	}
	var actual Result
	gqlClient.MustPost(queryString, &actual)

	require.Equal(t, testdb.StrToSec("2020-09-05 00:00:00").ToI(), actual.StakeHistory.Meta.Last)
	require.Equal(t, int64(11000), actual.StakeHistory.Meta.AssetVolume)
	require.Equal(t, int64(600), actual.StakeHistory.Meta.Units)

	require.Equal(t, int64(1), actual.StakeHistory.Intervals[0].Count)
	require.Equal(t, int64(5000), actual.StakeHistory.Intervals[0].AssetVolume)
	require.Equal(t, int64(1000), actual.StakeHistory.Intervals[0].RuneVolume)
	require.Equal(t, int64(200), actual.StakeHistory.Intervals[0].Units)

	// gapfill
	require.Equal(t, testdb.StrToSec("2020-09-04 00:00:00").ToI(), actual.StakeHistory.Intervals[1].Time)
	require.Equal(t, int64(0), actual.StakeHistory.Intervals[1].Count)
	require.Equal(t, int64(0), actual.StakeHistory.Intervals[1].RuneVolume)

	require.Equal(t, int64(2), actual.StakeHistory.Intervals[2].Count)
	require.Equal(t, testdb.StrToSec("2020-09-05 00:00:00").ToI(), actual.StakeHistory.Intervals[2].Time)
	require.Equal(t, int64(6000), actual.StakeHistory.Intervals[2].AssetVolume)
	require.Equal(t, int64(4500), actual.StakeHistory.Intervals[2].RuneVolume)
	require.Equal(t, int64(400), actual.StakeHistory.Intervals[2].Units)
}
