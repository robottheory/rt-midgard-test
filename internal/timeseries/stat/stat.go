// Package stat provides status information about the blockchain readings.
package stat

import (
	"database/sql"
	"time"
)

// DBQuery is the data source connection.
var DBQuery func(query string, args ...interface{}) (*sql.Rows, error)

// Window specifies a time period.
// The zero value matches all time.
type Window struct {
	Start time.Time // optional lower bound [inclusive]
	End   time.Time // optional upper bound [exclusive]
}

// Since returns a new period as of t.
func Since(t time.Time) Window {
	return Window{Start: t}
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
