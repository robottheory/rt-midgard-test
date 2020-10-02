package stat

import (
	"context"
	"testing"
	"time"

	"github.com/pascaldekloe/sqltest"
)

func TestDepth(t *testing.T) {
	DBQuery = sqltest.NewTx(t).QueryContext
	got, err := PoolDepthBucketsLookup(context.Background(), "BNB.BNB", 24*time.Hour, Window{})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}
