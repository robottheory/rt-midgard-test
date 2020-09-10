package stat

import (
	"testing"
	"time"

	"github.com/pascaldekloe/sqltest"
)

func TestStakesLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).Query
	got, err := StakesLookup(Window{})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}

func TestPoolStakesLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).Query
	got, err := PoolStakesLookup("BNB.MATIC-416", Window{})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}

func TestPoolStakesBucketsLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).Query
	got, err := PoolStakesBucketsLookup("BNB.MATIC-416", time.Hour, Window{Since: time.Now().Add(-24 * time.Hour), Until: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}

func TestPoolStakesAddrLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).Query
	got, err := PoolStakesAddrLookup("BNB.MATIC-416", "tbnb1uhkhl8ctdqal2rnx3n9k4hrf4yfqcz4wzuqc43", Window{})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}

func TestPoolStakesAddrBucketsLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).Query
	got, err := PoolStakesAddrBucketsLookup("BNB.MATIC-416", "tbnb1uhkhl8ctdqal2rnx3n9k4hrf4yfqcz4wzuqc43", time.Hour, Window{Since: time.Now().Add(-24 * time.Hour), Until: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}
