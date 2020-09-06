package stat

import (
	"testing"

	"github.com/pascaldekloe/sqltest"
)

func TestPoolAddsLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).Query
	got, err := PoolAddsLookup("BNB.MATIC-416", Window{})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}

func TestPoolErratasLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).Query
	got, err := PoolErratasLookup("BNB.BNB", Window{})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}

func TestPoolFeesLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).Query
	got, err := PoolFeesLookup("BNB.BNB", Window{})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}

func TestPoolGasLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).Query
	got, err := PoolGasLookup("BNB.MATIC-416", Window{})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}

func TestPoolSlashesLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).Query
	got, err := PoolSlashesLookup("BNB.MATIC-416", Window{})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}
