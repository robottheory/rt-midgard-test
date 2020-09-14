package stat

import (
	"context"
	"testing"

	"github.com/pascaldekloe/sqltest"
)

func TestPoolAddsLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).QueryContext
	got, err := PoolAddsLookup(context.Background(), "BNB.MATIC-416", Window{})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}

func TestPoolErratasLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).QueryContext
	got, err := PoolErratasLookup(context.Background(), "BNB.BNB", Window{})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}

func TestPoolFeesLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).QueryContext
	got, err := PoolFeesLookup(context.Background(), "BNB.BNB", Window{})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}

func TestPoolGasLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).QueryContext
	got, err := PoolGasLookup(context.Background(), "BNB.MATIC-416", Window{})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}

func TestPoolSlashesLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).QueryContext
	got, err := PoolSlashesLookup(context.Background(), "BNB.MATIC-416", Window{})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}
