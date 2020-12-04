package stat_test

import (
	"context"
	"testing"
	"time"

	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/internal/timeseries/stat"
)

var testWindow = db.Window{
	From:  time.Date(2020, 8, 1, 0, 0, 0, 0, time.UTC),
	Until: time.Date(2020, 9, 1, 0, 0, 0, 0, time.UTC)}

func TestSwapsFromRuneLookup(t *testing.T) {
	testdb.SetupTestDB(t)
	_, err := stat.SwapsFromRuneLookup(context.Background(), testWindow)
	if err != nil {
		t.Fatal(err)
	}
	// TODO(acsaba): add a events to the database and check that we get at least one value.
}

func TestSwapsToRuneLookup(t *testing.T) {
	testdb.SetupTestDB(t)
	_, err := stat.SwapsToRuneLookup(context.Background(), testWindow)
	if err != nil {
		t.Fatal(err)
	}
	// TODO(acsaba): add a events to the database and check that we get at least one value.
}
