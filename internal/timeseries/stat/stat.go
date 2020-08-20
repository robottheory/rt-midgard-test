// Package stat provides status information about the blockchain readings.
package stat

import (
	"database/sql"
	"time"
)

// DBQuery is used to read the data.
var DBQuery func(query string, args ...interface{}) (*sql.Rows, error)

// Window specifies a time period.
// The zero value matches all time.
type Window struct {
	From  time.Time // optional lower limit [inclusive]
	Until time.Time // optional upper limit [exclusive]
}

// Since returns a new period as of t.
func Since(t time.Time) Window {
	return Window{From: t}
}

func (w *Window) normalize() {
	if w.Until.IsZero() {
		w.Until = time.Now()
	}
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

type PoolStakes struct {
	TxCount         int64
	StakeUnitsTotal int64
	AssetE8Total    int64
	RuneE8Total     int64
}

func PoolStakesLookup(pool string, w Window) (PoolStakes, error) {
	w.normalize()

	const q = `SELECT COUNT(*), SUM(stake_units), SUM(rune_e8), SUM(asset_e8)
FROM stake_events
WHERE pool = $1 AND block_timestamp >= $2 AND block_timestamp < $3`

	rows, err := DBQuery(q, pool, w.From, w.Until)
	if err != nil {
		return PoolStakes{}, err
	}
	defer rows.Close()

	if !rows.Next() {
		return PoolStakes{}, rows.Err()
	}

	var r PoolStakes
	if err := rows.Scan(&r.TxCount, &r.StakeUnitsTotal, &r.RuneE8Total, &r.AssetE8Total); err != nil {
		return PoolStakes{}, err
	}
	return r, rows.Err()
}
