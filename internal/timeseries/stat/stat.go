// Package stat provides status information about the blockchain readings.
package stat

import (
	"database/sql"
	"fmt"
	"time"
)

// DBQuery is the data source connection.
var DBQuery func(query string, args ...interface{}) (*sql.Rows, error)

// Window specifies the applicable time period.
type Window struct {
	Since time.Time // lower bound [inclusive]
	Until time.Time // upper bound [exclusive]
}

// Bucket Nature
const (
	// BucketLimit is the maximum amount of buckets allowed per request.
	BucketLimit = 101

	// BucketResolution is the minimim quantity in time.
	BucketResolution = 5 * time.Minute
)

func bucketsFor(size time.Duration, w Window) (n int64, err error) {
	if size < BucketResolution {
		return 0, fmt.Errorf("bucket size %s smaller than resolution %s", size, BucketResolution)
	}
	if size%BucketResolution != 0 {
		return 0, fmt.Errorf("bucket size %s not a multiple of %s", size, BucketResolution)
	}
	first := w.Since.UnixNano() / int64(size)
	last := w.Until.UnixNano() / int64(size)
	n = last - first + 1
	if n > BucketLimit {
		return 0, fmt.Errorf("bucket amount %d exceeds limit of %d", n, BucketLimit)
	}
	return n, nil
}

// PoolsLookup returs the (asset) identifiers.
func PoolsLookup() ([]string, error) {
	const q = `SELECT pool FROM stake_events GROUP BY pool`

	rows, err := DBQuery(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pools []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return pools, err
		}
		pools = append(pools, s)
	}
	return pools, rows.Err()
}
