package timeseries_test

import (
	"bytes"
	"testing"
	"time"

	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/internal/fetch/chain"
	"gitlab.com/thorchain/midgard/internal/timeseries"
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
	block := chain.Block{
		Height:  height,
		Time:    timestamp,
		Hash:    hash,
		Results: &coretypes.ResultBlockResults{},
	}
	if err := timeseries.ProcessBlock(block, true); err != nil {
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
