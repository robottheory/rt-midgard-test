package stat

import (
	"testing"
	"time"

	"github.com/pascaldekloe/sqltest"
)

func TestNodeKeysLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).Query
	got, err := NodeKeysLookup(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}
