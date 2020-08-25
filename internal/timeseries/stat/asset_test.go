package stat

import (
	"testing"

	_ "github.com/lib/pq"

	"github.com/pascaldekloe/sqltest"
)

func TestPoolFeesLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).Query
	got, err := PoolFeesLookup("BNB.BNB", Window{})
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
