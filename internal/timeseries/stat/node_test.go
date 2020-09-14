package stat

import (
	"context"
	"testing"
	"time"

	"github.com/pascaldekloe/sqltest"
)

func TestNodeKeysLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).QueryContext
	got, err := NodeKeysLookup(context.Background(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}
