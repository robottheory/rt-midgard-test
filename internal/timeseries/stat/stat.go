// Package stat provides status information about the blockchain readings.
package stat

import (
	"database/sql"
	"time"
)

// DBQuery is the data source connection.
var DBQuery func(query string, args ...interface{}) (*sql.Rows, error)

// Window specifies the applicable time period.
type Window struct {
	Since time.Time // lower bound [inclusive]
	Until time.Time // upper bound [exclusive]
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
