package stat

import (
	"testing"

	"github.com/pascaldekloe/sqltest"
)

func TestPoolSwapsLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).Query
	got, err := PoolSwapsLookup("BNB.MATIC-416", Window{})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}
