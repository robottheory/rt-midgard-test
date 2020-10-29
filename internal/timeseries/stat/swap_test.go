package stat_test

import (
	"context"
	"testing"
	"time"

	"gitlab.com/thorchain/midgard/internal/timeseries/stat"
	"gitlab.com/thorchain/midgard/internal/timeseries/testdb"
)

var testWindow = stat.Window{
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

func TestPoolSwapsFromRuneLookup(t *testing.T) {
	testdb.SetupTestDB(t)
	_, err := stat.PoolSwapsFromRuneLookup(context.Background(), "BNB.MATIC-416", testWindow)
	if err != nil {
		t.Fatal(err)
	}
	// TODO(acsaba): add a events to the database and check that we get at least one value.
}

func TestPoolSwapsToRuneLookup(t *testing.T) {
	testdb.SetupTestDB(t)
	_, err := stat.PoolSwapsToRuneLookup(context.Background(), "BNB.MATIC-416", testWindow)
	if err != nil {
		t.Fatal(err)
	}
	// TODO(acsaba): add a events to the database and check that we get at least one value.
}

func TestPoolSwapsFromRuneBucketsLookup(t *testing.T) {
	testdb.SetupTestDB(t)
	_, err := stat.PoolSwapsFromRuneBucketsLookup(context.Background(), "BNB.MATIC-416", 24*time.Hour, testWindow)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPoolSwapsToRuneBucketsLookup(t *testing.T) {
	testdb.SetupTestDB(t)
	got, err := stat.PoolSwapsToRuneBucketsLookup(context.Background(), "BNB.MATIC-416", 24*time.Hour, testWindow)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %d buckets", len(got))
}
