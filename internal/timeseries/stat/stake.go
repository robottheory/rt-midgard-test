package stat

import "time"

// Stakes are statistics without asset classification.
type Stakes struct {
	TxCount         int64
	RuneE8Total     int64
	StakeUnitsTotal int64
	First, Last     time.Time
}

// PoolStakes are statistics for a specific asset.
type PoolStakes struct {
	TxCount         int64
	AssetE8Total    int64
	RuneE8Total     int64
	StakeUnitsTotal int64
	First, Last     time.Time
}

func StakesLookup(w Window) (*Stakes, error) {
	const q = `SELECT COALESCE(COUNT(*), 0), COALESCE(SUM(rune_e8), 0), COALESCE(SUM(stake_units), 0), COALESCE(MIN(block_timestamp), 0), COALESCE(MAX(block_timestamp), 0)
FROM stake_events
WHERE block_timestamp >= $1 AND block_timestamp < $2`
	rows, err := DBQuery(q, w.Since.UnixNano(), w.Until.UnixNano())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var r Stakes
	if rows.Next() {
		var first, last int64
		if err := rows.Scan(&r.TxCount, &r.StakeUnitsTotal, &r.RuneE8Total, &first, &last); err != nil {
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

func PoolStakesLookup(asset string, w Window) (*PoolStakes, error) {
	const q = `SELECT COALESCE(COUNT(*), 0), COALESCE(SUM(asset_e8), 0), COALESCE(SUM(rune_e8), 0), COALESCE(SUM(stake_units), 0), COALESCE(MIN(block_timestamp), 0), COALESCE(MAX(block_timestamp), 0)
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

	const q = `SELECT COALESCE(COUNT(*), 0), COALESCE(SUM(asset_e8), 0), COALESCE(SUM(rune_e8), 0), COALESCE(SUM(stake_units), 0), COALESCE(MIN(block_timestamp), 0), COALESCE(MAX(block_timestamp), 0)
FROM stake_events
WHERE pool = $1 AND block_timestamp >= $2 AND block_timestamp < $3
GROUP BY time_bucket($4, block_timestamp)
ORDER BY time_bucket($4, block_timestamp)
	`
	return appendPoolStakes(a, q, asset, w.Since.UnixNano(), w.Until.UnixNano(), bucketSize.Nanoseconds())
}

func PoolStakesAddrLookup(asset, addr string, w Window) (*PoolStakes, error) {
	const q = `SELECT COALESCE(COUNT(*), 0), COALESCE(SUM(asset_e8), 0), COALESCE(SUM(rune_e8), 0), COALESCE(SUM(stake_units), 0), COALESCE(MIN(block_timestamp), 0), COALESCE(MAX(block_timestamp), 0)
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

	const q = `SELECT COALESCE(COUNT(*), 0), COALESCE(SUM(asset_e8), 0), COALESCE(SUM(rune_e8), 0), COALESCE(SUM(stake_units), 0), COALESCE(MIN(block_timestamp), 0), COALESCE(MAX(block_timestamp), 0)
FROM stake_events
WHERE rune_addr = $1 AND pool = $2 AND block_timestamp >= $3 AND block_timestamp < $4
GROUP BY time_bucket($5, block_timestamp)
ORDER BY time_bucket($5, block_timestamp)
	`
	return appendPoolStakes(a, q, addr, asset, w.Since.UnixNano(), w.Until.UnixNano(), bucketSize.Nanoseconds())
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
		err := rows.Scan(&r.TxCount, &r.AssetE8Total, &r.RuneE8Total, &r.StakeUnitsTotal, &first, &last)
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
