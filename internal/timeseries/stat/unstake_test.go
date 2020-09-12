package stat

import (
	"testing"

	"github.com/pascaldekloe/sqltest"
)

func TestAssetUnstakesLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).Query
	got, err := UnstakesLookup(testWindow)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}

func TestPoolAssetUnstakesLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).Query
	got, err := PoolUnstakesLookup("BNB.DOS-120", testWindow)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}
