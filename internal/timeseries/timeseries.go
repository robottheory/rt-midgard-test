package timeseries

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"
)

// DBQuery is the SQL client.
var DBQuery func(query string, args ...interface{}) (*sql.Rows, error)

// Chain Position Tracking
var (
	blockMutex     sync.Mutex
	blockHeight    int64
	blockTimestamp time.Time
	blockHash      []byte
)

// CommitBlock marks the given height as done.
func CommitBlock(height int64, timestamp time.Time, hash []byte) error {
	const q = "INSERT INTO block_log (height, timestamp, hash) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING"
	result, err := DBExec(q, height, timestamp.UnixNano(), hash)
	if err != nil {
		return fmt.Errorf("persist block height %d: %w", height, err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("persist block height %d result: %w", height, err)
	}
	if n == 0 {
		log.Printf("block height %d already committed", height)
	}

	blockMutex.Lock()
	defer blockMutex.Unlock()
	blockHeight = height
	blockTimestamp = timestamp
	blockHash = make([]byte, len(hash))
	copy(blockHash, hash)

	return nil
}

// LastBlock gets the highest commit.
func LastBlock() (height int64, timestamp time.Time, hash []byte, err error) {
	blockMutex.Lock()
	defer blockMutex.Unlock()
	if blockHash != nil {
		// use in-memory state which is defined by either the most
		// recent CommitBlock, or by the restore routine blow.
		return blockHeight, blockTimestamp, blockHash, nil
	}

	// read last persisted CommitBlock into in-memory state
	const q = "SELECT height, timestamp, hash FROM block_log ORDER BY height DESC LIMIT 1"
	rows, err := DBQuery(q)
	if err != nil {
		return 0, time.Time{}, nil, fmt.Errorf("last block lookup: %w", err)
	}
	defer rows.Close()
	if rows.Next() {
		var ns int64
		rows.Scan(&blockHeight, &ns, &blockHash)
		blockTimestamp = time.Unix(0, ns)
	}
	return blockHeight, blockTimestamp, blockHash, rows.Err()
}
