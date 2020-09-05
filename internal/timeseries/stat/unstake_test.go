package stat

import (
	"testing"

	"github.com/pascaldekloe/sqltest"
)

func TestPoolAssetUnstakesLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).Query
	got, err := PoolAssetUnstakesLookup("BNB.MATIC-416", testWindow)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}

func TestPoolRuneUnstakesLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).Query
	got, err := PoolRuneUnstakesLookup("BNB.MATIC-416", testWindow)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}
