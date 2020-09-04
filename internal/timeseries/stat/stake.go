package stat

import (
	"strings"
	"time"
)

// Stakes are statistics without asset classification.
type Stakes struct {
	TxCount     int64
	UnitsTotal  int64
	RuneE8Total int64
	First, Last time.Time
}

// PoolStakes are statistics for a specific asset.
type PoolStakes struct {
	TxCount      int64
	UnitsTotal   int64
	RuneE8Total  int64
	AssetE8Total int64
	First, Last  time.Time
}

// AddrStakes are statistics for a specific address.
type AddrStakes struct {
	Addr        string
	TxCount     int64
	UnitsTotal  int64
	RuneE8Total int64
	First, Last time.Time
}

func StakesLookup(w Window) (Stakes, error) {
	const q = `SELECT COALESCE(COUNT(*), 0), COALESCE(SUM(stake_units), 0), COALESCE(SUM(rune_e8), 0), COALESCE(MIN(block_timestamp), 0), COALESCE(MAX(block_timestamp), 0)
FROM stake_events
WHERE block_timestamp >= $1 AND block_timestamp < $2`

	return queryStakes(q, w.Since.UnixNano(), w.Until.UnixNano())
}

func PoolStakesBucketLookup(pool string, bucketSize time.Duration, w Window) ([]PoolStakes, error) {
	const q = `SELECT COALESCE(COUNT(*), 0), COALESCE(SUM(stake_units), 0), COALESCE(SUM(rune_e8), 0), COALESCE(SUM(asset_e8), 0), COALESCE(MIN(block_timestamp), 0), COALESCE(MAX(block_timestamp), 0)
	FROM stake_events
	WHERE rune_addr = $1 AND pool = $2 AND block_timestamp >= $3 AND block_timestamp < $4
	GROUP BY time_bucket($4, block_timestamp)
	ORDER BY time_bucket($4, block_timestamp)
	`

	rows, err := DBQuery(q, pool, w.Since.UnixNano(), w.Until.UnixNano(), bucketSize)
	if err != nil {
		return []PoolStakes{}, err
	}
	defer rows.Close()

	var pools []PoolStakes
	for rows.Next() {
		var r PoolStakes
		var first, last int64
		if err := rows.Scan(&r.TxCount, &r.UnitsTotal, &r.RuneE8Total, &r.AssetE8Total, &first, &last); err != nil {
			return pools, err
		}
		if first != 0 {
			r.First = time.Unix(0, first)
		}
		if last != 0 {
			r.Last = time.Unix(0, last)
		}
		pools = append(pools, r)
	}
	return pools, rows.Err()
}

func StakesAddrLookup(addr string, w Window) (Stakes, error) {
	const q = `SELECT COALESCE(COUNT(*), 0), COALESCE(SUM(stake_units), 0), COALESCE(SUM(rune_e8), 0), COALESCE(MIN(block_timestamp), 0), COALESCE(MAX(block_timestamp), 0)
FROM stake_events
WHERE rune_addr = $1 AND block_timestamp >= $2 AND block_timestamp < $3`

	return queryStakes(q, addr, w.Since.UnixNano(), w.Until.UnixNano())
}

func PoolStakesLookup(pool string, w Window) (PoolStakes, error) {
	const q = `SELECT COALESCE(COUNT(*), 0), COALESCE(SUM(stake_units), 0), COALESCE(SUM(rune_e8), 0), COALESCE(SUM(asset_e8), 0), COALESCE(MIN(block_timestamp), 0), COALESCE(MAX(block_timestamp), 0)
FROM stake_events
WHERE pool = $1 AND block_timestamp >= $2 AND block_timestamp < $3`

	return queryPoolStakes(q, pool, w.Since.UnixNano(), w.Until.UnixNano())
}

func PoolStakesAddrLookup(addr, pool string, w Window) (PoolStakes, error) {
	const q = `SELECT COALESCE(COUNT(*), 0), COALESCE(SUM(stake_units), 0), COALESCE(SUM(rune_e8), 0), COALESCE(SUM(asset_e8), 0), COALESCE(MIN(block_timestamp), 0), COALESCE(MAX(block_timestamp), 0)
FROM stake_events
WHERE rune_addr = $1 AND pool = $2 AND block_timestamp >= $3 AND block_timestamp < $4`

	return queryPoolStakes(q, addr, pool, w.Since.UnixNano(), w.Until.UnixNano())
}

func AllAddrStakesLookup(t time.Time) ([]AddrStakes, error) {
	const q = `SELECT rune_addr, COALESCE(COUNT(*), 0), COALESCE(SUM(stake_units), 0), COALESCE(SUM(rune_e8), 0), COALESCE(MIN(block_timestamp), 0), COALESCE(MAX(block_timestamp), 0)
FROM stake_events
WHERE block_timestamp < $1
GROUP BY rune_addr`

	rows, err := DBQuery(q, t.UnixNano())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var a []AddrStakes
	for rows.Next() {
		var r AddrStakes
		var first, last int64
		if err := rows.Scan(&r.Addr, &r.TxCount, &r.UnitsTotal, &r.RuneE8Total, &first, &last); err != nil {
			return a, err
		}
		if first != 0 {
			r.First = time.Unix(0, first)
		}
		if last != 0 {
			r.Last = time.Unix(0, last)
		}
		r.Addr = strings.TrimRight(r.Addr, " ")

		a = append(a, r)
	}
	return a, rows.Err()
}

func queryStakes(q string, args ...interface{}) (Stakes, error) {
	rows, err := DBQuery(q, args...)
	if err != nil {
		return Stakes{}, err
	}
	defer rows.Close()

	if !rows.Next() {
		return Stakes{}, rows.Err()
	}

	var r Stakes
	var first, last int64
	if err := rows.Scan(&r.TxCount, &r.UnitsTotal, &r.RuneE8Total, &first, &last); err != nil {
		return Stakes{}, err
	}
	if first != 0 {
		r.First = time.Unix(0, first)
	}
	if last != 0 {
		r.Last = time.Unix(0, last)
	}
	return r, rows.Err()
}

func queryPoolStakes(q string, args ...interface{}) (PoolStakes, error) {
	rows, err := DBQuery(q, args...)
	if err != nil {
		return PoolStakes{}, err
	}
	defer rows.Close()

	if !rows.Next() {
		return PoolStakes{}, rows.Err()
	}

	var r PoolStakes
	var first, last int64
	if err := rows.Scan(&r.TxCount, &r.UnitsTotal, &r.RuneE8Total, &r.AssetE8Total, &first, &last); err != nil {
		return PoolStakes{}, err
	}
	if first != 0 {
		r.First = time.Unix(0, first)
	}
	if last != 0 {
		r.Last = time.Unix(0, last)
	}
	return r, rows.Err()
}
