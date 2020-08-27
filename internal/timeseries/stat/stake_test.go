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

func TestStakesAddrLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).Query
	got, err := StakesAddrLookup("tbnb1uhkhl8ctdqal2rnx3n9k4hrf4yfqcz4wzuqc43", Window{})
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

func TestPoolStakesAddrLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).Query
	got, err := PoolStakesAddrLookup("tbnb1uhkhl8ctdqal2rnx3n9k4hrf4yfqcz4wzuqc43", "BNB.MATIC-416", Window{})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}

func TestAllAddrStakesLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).Query
	got, err := AllAddrStakesLookup(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}
