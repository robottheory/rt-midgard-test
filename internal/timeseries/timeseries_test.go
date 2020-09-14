package timeseries

import (
	"bytes"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v4/stdlib"

	"github.com/pascaldekloe/sqltest"
)

func init() {
	sqltest.Setup("pgx", "user=midgard password=password host=localhost port=5432 sslmode=disable dbname=midgard")
}

func mustSetup(t *testing.T) {
	tx := sqltest.NewTx(t)
	DBExec = tx.Exec
	DBQuery = tx.QueryContext
	_, _, _, err := Setup()
	if err != nil {
		t.Fatal("package setup:", err)
	}
}

func TestLastBlockNone(t *testing.T) {
	mustSetup(t)

	_, err := DBExec("DELETE FROM block_log")
	if err != nil {
		t.Fatal("clear block log:", err)
	}
	height, timestamp, hash, err := Setup()
	if err != nil {
		t.Fatal("setup error (with empty block_log):", err)
	}
	if height != 0 || !timestamp.IsZero() || hash != nil {
		t.Errorf("got [%d, %s, %x], want [0, epoch, nil]", height, timestamp, hash)
	}
}

func TestCommitBlock(t *testing.T) {
	mustSetup(t)

	// high height should exceed whatever is in store
	const height = 1 << 60
	timestamp := time.Now()
	hash := []byte{4, 2}
	if err := CommitBlock(height, timestamp, hash); err != nil {
		t.Fatal("commit error:", err)
	}

	// test from in-memory lookup
	gotHeight, gotTimestamp, gotHash := LastBlock()
	if gotHeight != height || !gotTimestamp.Equal(timestamp) || !bytes.Equal(gotHash, hash) {
		t.Errorf("got [%d, %s, %q], want [%d, %s, %q]", gotHeight, gotTimestamp, gotHash, height, timestamp, hash)
	}

	// test database state restore
	if gotHeight, gotTimestamp, gotHash, err := Setup(); err != nil {
		t.Fatal("setup error:", err)
	} else if gotHeight != height || !gotTimestamp.Equal(timestamp) || !bytes.Equal(gotHash, hash) {
		t.Errorf("cold start got [%d, %s, %q], want [%d, %s, %q]", gotHeight, gotTimestamp, gotHash, height, timestamp, hash)
	}
}
