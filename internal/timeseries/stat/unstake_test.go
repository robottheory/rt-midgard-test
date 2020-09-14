package stat

import (
	"context"
	"testing"

	"github.com/pascaldekloe/sqltest"
)

func TestAssetUnstakesLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).QueryContext
	got, err := UnstakesLookup(context.Background(), testWindow)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}

func TestPoolAssetUnstakesLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).QueryContext
	got, err := PoolUnstakesLookup(context.Background(), "BNB.DOS-120", testWindow)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}
