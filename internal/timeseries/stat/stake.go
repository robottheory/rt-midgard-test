package stat

import "time"

func StakeAddrsLookup() (addrs []string, err error) {
	rows, err := DBQuery(`SELECT rune_addr FROM stake_events GROUP BY rune_addr`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	addrs = make([]string, 0, 1024)
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return addrs, err
		}
		addrs = append(addrs, s)
	}
	return addrs, rows.Err()
}

// Stakes are generic stake statistics.
type Stakes struct {
	TxCount         int64
	RuneAddrCount   int64 // Number of unique staker addresses involved.
	RuneE8Total     int64
	StakeUnitsTotal int64
	First, Last     time.Time
}

func StakesLookup(w Window) (*Stakes, error) {
	const q = `SELECT COALESCE(COUNT(*), 0), COALESCE(COUNT(DISTINCT(rune_addr))), COALESCE(SUM(rune_e8), 0), COALESCE(SUM(stake_units), 0), COALESCE(MIN(block_timestamp), 0), COALESCE(MAX(block_timestamp), 0)
FROM stake_events
WHERE block_timestamp >= $1 AND block_timestamp < $2`

	return queryStakes(q, w.Since.UnixNano(), w.Until.UnixNano())
}

func StakesAddrLookup(addr string, w Window) (*Stakes, error) {
	const q = `SELECT COALESCE(COUNT(*), 0), COALESCE(COUNT(DISTINCT(rune_addr))), COALESCE(SUM(rune_e8), 0), COALESCE(SUM(stake_units), 0), COALESCE(MIN(block_timestamp), 0), COALESCE(MAX(block_timestamp), 0)
FROM stake_events
WHERE rune_addr = $1 AND block_timestamp >= $2 AND block_timestamp < $3`

	return queryStakes(q, addr, w.Since.UnixNano(), w.Until.UnixNano())
}

func queryStakes(q string, args ...interface{}) (*Stakes, error) {
	rows, err := DBQuery(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var r Stakes
	if rows.Next() {
		var first, last int64
		err := rows.Scan(&r.TxCount, &r.RuneAddrCount, &r.StakeUnitsTotal, &r.RuneE8Total, &first, &last)
		if err != nil {
			return nil, err
		}
		if first != 0 {
			r.First = time.Unix(0, first)
		}
		if last != 0 {
			r.Last = time.Unix(0, last)
		}
	}
	return &r, rows.Err()
}

// PoolStakes are statistics for a specific asset.
type PoolStakes struct {
	Asset           string
	TxCount         int64
	AssetE8Total    int64
	RuneE8Total     int64
	StakeUnitsTotal int64
	First, Last     time.Time
}

func PoolStakesLookup(asset string, w Window) (*PoolStakes, error) {
	const q = `SELECT $1, COALESCE(COUNT(*), 0), COALESCE(SUM(asset_e8), 0), COALESCE(SUM(rune_e8), 0), COALESCE(SUM(stake_units), 0), COALESCE(MIN(block_timestamp), 0), COALESCE(MAX(block_timestamp), 0)
FROM stake_events
WHERE pool = $1 AND block_timestamp >= $2 AND block_timestamp < $3`

	var a [1]PoolStakes
	_, err := appendPoolStakes(a[:0], q, asset, w.Since.UnixNano(), w.Until.UnixNano())
	return &a[0], err
}

func PoolStakesBucketsLookup(asset string, bucketSize time.Duration, w Window) ([]PoolStakes, error) {
	n, err := bucketsFor(bucketSize, w)
	if err != nil {
		return nil, err
	}
	a := make([]PoolStakes, 0, n)

	const q = `SELECT $1, COALESCE(COUNT(*), 0), COALESCE(SUM(asset_e8), 0), COALESCE(SUM(rune_e8), 0), COALESCE(SUM(stake_units), 0), COALESCE(MIN(block_timestamp), 0), COALESCE(MAX(block_timestamp), 0)
FROM stake_events
WHERE pool = $1 AND block_timestamp >= $2 AND block_timestamp < $3
GROUP BY time_bucket($4, block_timestamp)
ORDER BY time_bucket($4, block_timestamp)
	`
	return appendPoolStakes(a, q, asset, w.Since.UnixNano(), w.Until.UnixNano(), bucketSize.Nanoseconds())
}

func PoolStakesAddrLookup(asset, addr string, w Window) (*PoolStakes, error) {
	const q = `SELECT $2, COALESCE(COUNT(*), 0), COALESCE(SUM(asset_e8), 0), COALESCE(SUM(rune_e8), 0), COALESCE(SUM(stake_units), 0), COALESCE(MIN(block_timestamp), 0), COALESCE(MAX(block_timestamp), 0)
FROM stake_events
WHERE rune_addr = $1 AND pool = $2 AND block_timestamp >= $3 AND block_timestamp < $4`

	var a [1]PoolStakes
	_, err := appendPoolStakes(a[:0], q, addr, asset, w.Since.UnixNano(), w.Until.UnixNano())
	return &a[0], err
}

func PoolStakesAddrBucketsLookup(asset, addr string, bucketSize time.Duration, w Window) ([]PoolStakes, error) {
	n, err := bucketsFor(bucketSize, w)
	if err != nil {
		return nil, err
	}
	a := make([]PoolStakes, 0, n)

	const q = `SELECT $2, COALESCE(COUNT(*), 0), COALESCE(SUM(asset_e8), 0), COALESCE(SUM(rune_e8), 0), COALESCE(SUM(stake_units), 0), COALESCE(MIN(block_timestamp), 0), COALESCE(MAX(block_timestamp), 0)
FROM stake_events
WHERE rune_addr = $1 AND pool = $2 AND block_timestamp >= $3 AND block_timestamp < $4
GROUP BY time_bucket($5, block_timestamp)
ORDER BY time_bucket($5, block_timestamp)
	`
	return appendPoolStakes(a, q, addr, asset, w.Since.UnixNano(), w.Until.UnixNano(), bucketSize.Nanoseconds())
}

func AllPoolStakesAddrLookup(addr string, w Window) ([]PoolStakes, error) {
	const q = `SELECT pool, COALESCE(COUNT(*), 0), COALESCE(SUM(asset_e8), 0), COALESCE(SUM(rune_e8), 0), COALESCE(SUM(stake_units), 0), COALESCE(MIN(block_timestamp), 0), COALESCE(MAX(block_timestamp), 0)
FROM stake_events
WHERE rune_addr = $1 AND block_timestamp >= $2 AND block_timestamp < $3
GROUP BY pool`

	return appendPoolStakes(nil, q, addr, w.Since.UnixNano(), w.Until.UnixNano())
}

func appendPoolStakes(a []PoolStakes, q string, args ...interface{}) ([]PoolStakes, error) {
	rows, err := DBQuery(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var r PoolStakes
		var first, last int64
		err := rows.Scan(&r.Asset, &r.TxCount, &r.AssetE8Total, &r.RuneE8Total, &r.StakeUnitsTotal, &first, &last)
		if err != nil {
			return a, err
		}
		if first != 0 {
			r.First = time.Unix(0, first)
		}
		if last != 0 {
			r.Last = time.Unix(0, last)
		}
		a = append(a, r)
	}
	return a, rows.Err()
}
