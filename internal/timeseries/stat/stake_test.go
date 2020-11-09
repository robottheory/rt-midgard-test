package stat_test

import (
	"context"
	"gitlab.com/thorchain/midgard/internal/graphql/model"
	"testing"
	"time"

	"gitlab.com/thorchain/midgard/internal/timeseries/stat"
	"gitlab.com/thorchain/midgard/internal/timeseries/testdb"
)

func TestStakesLookup(t *testing.T) {
	testdb.SetupTestDB(t)
	_, err := stat.StakesLookup(context.Background(), stat.Window{})
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
		stat.Window{})
	if err != nil {
		t.Fatal(err)
	}
	// TODO(acsaba): add a events to the database and check that we get at least one value.
}

func TestPoolStakesLookup(t *testing.T) {
	testdb.SetupTestDB(t)
	_, err := stat.PoolStakesLookup(
		context.Background(), "BNB.MATIC-416", stat.Window{})
	if err != nil {
		t.Fatal(err)
	}
	// TODO(acsaba): add a events to the database and check that we get at least one value.
}

func TestPoolStakesBucketsLookup(t *testing.T) {
	testdb.SetupTestDB(t)
	_, err := stat.GetPoolStakes(
		context.Background(), "BNB.MATIC-416",
		stat.Window{From: time.Now().Add(-24 * time.Hour), Until: time.Now()}, model.IntervalHour)
	if err != nil {
		t.Fatal(err)
	}
	// TODO(acsaba): add a events to the database and check that we get at least one value.
}

func TestPoolStakesAddrLookup(t *testing.T) {
	testdb.SetupTestDB(t)
	_, err := stat.PoolStakesAddrLookup(
		context.Background(), "BNB.MATIC-416",
		"tbnb1uhkhl8ctdqal2rnx3n9k4hrf4yfqcz4wzuqc43", stat.Window{})
	if err != nil {
		t.Fatal(err)
	}
	// TODO(acsaba): add a events to the database and check that we get at least one value.
}

func TestPoolStakesAddrBucketsLookup(t *testing.T) {
	testdb.SetupTestDB(t)
	_, err := stat.PoolStakesAddrBucketsLookup(
		context.Background(),
		"BNB.MATIC-416",
		"tbnb1uhkhl8ctdqal2rnx3n9k4hrf4yfqcz4wzuqc43",
		time.Hour,
		stat.Window{From: time.Now().Add(-24 * time.Hour), Until: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	// TODO(acsaba): add a events to the database and check that we get at least one value.
}

func TestAllPoolStakesAddrLookup(t *testing.T) {
	testdb.SetupTestDB(t)
	_, err := stat.AllPoolStakesAddrLookup(
		context.Background(), "tbnb1uhkhl8ctdqal2rnx3n9k4hrf4yfqcz4wzuqc43",
		stat.Window{From: time.Now().Add(-24 * time.Hour), Until: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	// TODO(acsaba): add a events to the database and check that we get at least one value.
}
