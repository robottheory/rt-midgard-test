package stat

import (
	"testing"

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

func TestAddrStakesLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).Query
	got, err := AddrStakesLookup("tbnb1uhkhl8ctdqal2rnx3n9k4hrf4yfqcz4wzuqc43", Window{})
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

func TestAddrPoolStakesLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).Query
	got, err := AddrPoolStakesLookup("tbnb1uhkhl8ctdqal2rnx3n9k4hrf4yfqcz4wzuqc43", "BNB.MATIC-416", Window{})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}
