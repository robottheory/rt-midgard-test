package stat_test

import (
	"context"
	"testing"

	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/internal/timeseries/stat"
)

func TestAssetUnstakesLookup(t *testing.T) {
	testdb.SetupTestDB(t)
	_, err := stat.UnstakesLookup(context.Background(), testWindow)
	if err != nil {
		t.Fatal(err)
	}
	// TODO(acsaba): add a events to the database and check that we get at least one value.
}

func TestPoolAssetUnstakesLookup(t *testing.T) {
	testdb.SetupTestDB(t)
	_, err := stat.PoolUnstakesLookup(context.Background(), "BNB.DOS-120", testWindow)
	if err != nil {
		t.Fatal(err)
	}
	// TODO(acsaba): add a events to the database and check that we get at least one value.
}
