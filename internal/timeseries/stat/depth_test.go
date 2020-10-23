package stat_test

import (
	"context"
	"testing"
	"time"

	"gitlab.com/thorchain/midgard/internal/timeseries/stat"
	"gitlab.com/thorchain/midgard/internal/timeseries/testdb"
)

func TestDepth(t *testing.T) {
	testdb.SetupTestDB(t)
	_, err := stat.PoolDepthBucketsLookup(context.Background(), "BNB.BNB", 24*time.Hour, stat.Window{})
	if err != nil {
		t.Fatal(err)
	}
	// TODO(acsaba): add a events to the database and check that we get at least one value.
}
