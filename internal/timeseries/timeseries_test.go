package timeseries_test

import (
	"bytes"
	"testing"
	"time"

	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/internal/timeseries/testdb"
)

func TestLastBlockNone(t *testing.T) {
	testdb.SetupTestDB(t)
	testdb.MustExec(t, "DELETE FROM block_log")

	height, timestamp, hash, err := timeseries.Setup()
	if err != nil {
		t.Fatal("setup error (with empty block_log):", err)
	}
	if height != 0 || !timestamp.IsZero() || hash != nil {
		t.Errorf("got [%d, %s, %x], want [0, epoch, nil]", height, timestamp, hash)
	}
}

func TestCommitBlock(t *testing.T) {
	testdb.SetupTestDB(t)
	testdb.MustExec(t, "DELETE FROM block_log")

	// high height should exceed whatever is in store
	const height = 1 << 60
	timestamp := time.Now()
	hash := []byte{4, 2}
	if err := timeseries.CommitBlock(height, timestamp, hash); err != nil {
		t.Fatal("commit error:", err)
	}

	// test from in-memory lookup
	gotHeight, gotTimestamp, gotHash := timeseries.LastBlock()
	if gotHeight != height || !gotTimestamp.Equal(timestamp) || !bytes.Equal(gotHash, hash) {
		t.Errorf("got [%d, %s, %q], want [%d, %s, %q]", gotHeight, gotTimestamp, gotHash, height, timestamp, hash)
	}

	// test database state restore
	if gotHeight, gotTimestamp, gotHash, err := timeseries.Setup(); err != nil {
		t.Fatal("setup error:", err)
	} else if gotHeight != height || !gotTimestamp.Equal(timestamp) || !bytes.Equal(gotHash, hash) {
		t.Errorf("cold start got [%d, %s, %q], want [%d, %s, %q]", gotHeight, gotTimestamp, gotHash, height, timestamp, hash)
	}
}
