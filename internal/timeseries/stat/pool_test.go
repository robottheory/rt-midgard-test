package stat

import (
	"testing"

	"github.com/pascaldekloe/sqltest"
)

func TestPoolStatusLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).Query
	got, err := PoolStatusLookup("BNB.MATIC-416")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}
