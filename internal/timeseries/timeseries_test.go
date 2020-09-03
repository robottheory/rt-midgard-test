package timeseries

import (
	"testing"
	"time"

	_ "github.com/lib/pq"

	"github.com/pascaldekloe/sqltest"
)

func init() {
	sqltest.Setup("postgres", "user=midgard password=password host=localhost port=5432 sslmode=disable dbname=midgard")
}

func TestLastBlockNone(t *testing.T) {
	tx := sqltest.NewTx(t)
	DBExec = tx.Exec
	DBQuery = tx.Query

	_, err := DBExec("DELETE FROM block_log")
	if err != nil {
		t.Fatal("clear block log:", err)
	}

	height, timestamp, hash, err := LastBlock()
	if err != nil {
		t.Fatal("lookup error:", err)
	}
	if height != 0 || !timestamp.IsZero() || hash != nil {
		t.Errorf("got [%d, %s, %q], want [0, epoch, nil]", height, timestamp, hash)
	}
}

func TestCommitBlock(t *testing.T) {
	tx := sqltest.NewTx(t)
	DBExec = tx.Exec
	DBQuery = tx.Query

	// high height should exceed whatever is in store
	const height = 1 << 60
	timestamp := time.Now()
	const hash = "0xdeadbeef"

	if err := CommitBlock(height, timestamp, []byte(hash)); err != nil {
		t.Fatal("commit error:", err)
	}
	gotHeight, gotTimestamp, gotHash, err := LastBlock()
	if err != nil {
		t.Fatal("lookup error:", err)
	}
	if gotHeight != height || !gotTimestamp.Equal(timestamp) || string(gotHash) != hash {
		t.Errorf("got [%d, %s, %q], want [%d, %s, %q]", gotHeight, gotTimestamp, gotHash, height, timestamp, hash)
	}
}
