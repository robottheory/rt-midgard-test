package stat_test

import (
	"context"
	"testing"

	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/internal/timeseries/stat"
)

func TestStakesLookup(t *testing.T) {
	testdb.SetupTestDB(t)
	_, err := stat.StakesLookup(context.Background(), db.Window{})
	if err != nil {
		t.Fatal(err)
	}
	// TODO(acsaba): add a events to the database and check that we get at least one value.
}

func TestStakesAddrLookup(t *testing.T) {
	testdb.SetupTestDB(t)
	_, err := stat.StakesAddrLookup(
		context.Background(),
		"tbnb1uhkhl8ctdqal2rnx3n9k4hrf4yfqcz4wzuqc43",
		db.Window{})
	if err != nil {
		t.Fatal(err)
	}
	// TODO(acsaba): add a events to the database and check that we get at least one value.
}

func TestPoolStakesLookup(t *testing.T) {
	testdb.SetupTestDB(t)
	_, err := stat.PoolStakesLookup(
		context.Background(), "BNB.MATIC-416", db.Window{})
	if err != nil {
		t.Fatal(err)
	}
	// TODO(acsaba): add a events to the database and check that we get at least one value.
}

func TestPoolStakesAddrLookup(t *testing.T) {
	testdb.SetupTestDB(t)
	_, err := stat.PoolStakesAddrLookup(
		context.Background(), "BNB.MATIC-416",
		"tbnb1uhkhl8ctdqal2rnx3n9k4hrf4yfqcz4wzuqc43", db.Window{})
	if err != nil {
		t.Fatal(err)
	}
	// TODO(acsaba): add a events to the database and check that we get at least one value.
}
