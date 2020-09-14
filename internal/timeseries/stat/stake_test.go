package stat

import (
	"context"
	"testing"
	"time"

	"github.com/pascaldekloe/sqltest"
)

func TestStakesLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).QueryContext
	got, err := StakesLookup(context.Background(), Window{})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}

func TestStakesAddrLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).QueryContext
	got, err := StakesAddrLookup(context.Background(), "tbnb1uhkhl8ctdqal2rnx3n9k4hrf4yfqcz4wzuqc43", Window{})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}

func TestPoolStakesLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).QueryContext
	got, err := PoolStakesLookup(context.Background(), "BNB.MATIC-416", Window{})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}

func TestPoolStakesBucketsLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).QueryContext
	got, err := PoolStakesBucketsLookup(context.Background(), "BNB.MATIC-416", time.Hour, Window{Since: time.Now().Add(-24 * time.Hour), Until: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}

func TestPoolStakesAddrLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).QueryContext
	got, err := PoolStakesAddrLookup(context.Background(), "BNB.MATIC-416", "tbnb1uhkhl8ctdqal2rnx3n9k4hrf4yfqcz4wzuqc43", Window{})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}

func TestPoolStakesAddrBucketsLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).QueryContext
	got, err := PoolStakesAddrBucketsLookup(context.Background(), "BNB.MATIC-416", "tbnb1uhkhl8ctdqal2rnx3n9k4hrf4yfqcz4wzuqc43", time.Hour, Window{Since: time.Now().Add(-24 * time.Hour), Until: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}

func TestAllPoolStakesAddrLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).QueryContext
	got, err := AllPoolStakesAddrLookup(context.Background(), "tbnb1uhkhl8ctdqal2rnx3n9k4hrf4yfqcz4wzuqc43", Window{Since: time.Now().Add(-24 * time.Hour), Until: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}
