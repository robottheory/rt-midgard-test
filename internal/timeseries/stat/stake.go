package stat

import (
	"context"
	"errors"
	"time"

	"gitlab.com/thorchain/midgard/internal/db"
)

// TODO(elfedy): This file should be renamed to deposit.go once the terminology of all
// functions is updated

type AddressPoolDeposits struct {
	AssetE8Total   int64
	RuneE8Total    int64
	UnitsTotal     int64
	DateFirstAdded int64
	DateLastAdded  int64
}

// AddressPoolDepositsLookup aggregates deposits by pool for a given address
func AddressPoolDepositsLookup(ctx context.Context, address string) (map[string]AddressPoolDeposits, error) {
	q := `SELECT pool, COALESCE(SUM(asset_E8), 0), COALESCE(SUM(rune_E8), 0), COALESCE(SUM(stake_units), 0), COALESCE(MIN(block_timestamp), 0), COALESCE(MAX(block_timestamp), 0)
	FROM stake_events
	WHERE rune_addr = $1
	GROUP BY pool`

	rows, err := db.Query(ctx, q, address)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]AddressPoolDeposits)
	for rows.Next() {
		var pool string
		var deposits AddressPoolDeposits
		err := rows.Scan(&pool, &deposits.AssetE8Total, &deposits.RuneE8Total, &deposits.UnitsTotal, &deposits.DateFirstAdded, &deposits.DateLastAdded)
		if err != nil {
			return nil, err
		}

		result[pool] = deposits
	}
	return result, nil
}

// Stakes are generic stake statistics.
type Stakes struct {
	TxCount         int64
	RuneAddrCount   int64 // Number of unique staker addresses involved.
	RuneE8Total     int64
	StakeUnitsTotal int64
	First, Last     time.Time
}

func StakesLookup(ctx context.Context, w Window) (*Stakes, error) {
	const q = `SELECT COALESCE(COUNT(*), 0), COALESCE(COUNT(DISTINCT(rune_addr))), COALESCE(SUM(rune_e8), 0), COALESCE(SUM(stake_units), 0), COALESCE(MIN(block_timestamp), 0), COALESCE(MAX(block_timestamp), 0)
FROM stake_events
WHERE block_timestamp >= $1 AND block_timestamp < $2`

	return queryStakes(ctx, q, w.From.UnixNano(), w.Until.UnixNano())
}

func StakesAddrLookup(ctx context.Context, addr string, w Window) (*Stakes, error) {
	const q = `SELECT COALESCE(COUNT(*), 0), COALESCE(COUNT(DISTINCT(rune_addr))), COALESCE(SUM(rune_e8), 0), COALESCE(SUM(stake_units), 0), COALESCE(MIN(block_timestamp), 0), COALESCE(MAX(block_timestamp), 0)
FROM stake_events
WHERE rune_addr = $1 AND block_timestamp >= $2 AND block_timestamp < $3`

	return queryStakes(ctx, q, addr, w.From.UnixNano(), w.Until.UnixNano())
}

func queryStakes(ctx context.Context, q string, args ...interface{}) (*Stakes, error) {
	rows, err := db.Query(ctx, q, args...)
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
	Time            time.Time
	First, Last     time.Time
}

func PoolStakesLookup(ctx context.Context, asset string, w Window) (*PoolStakes, error) {
	const q = `SELECT $1, COALESCE(COUNT(*), 0), COALESCE(SUM(asset_e8), 0), COALESCE(SUM(rune_e8), 0), COALESCE(SUM(stake_units), 0), COALESCE(MIN(block_timestamp), 0), COALESCE(MAX(block_timestamp), 0)
FROM stake_events
WHERE pool = $1 AND block_timestamp >= $2 AND block_timestamp < $3`

	var a [1]PoolStakes
	_, err := appendPoolStakes(ctx, a[:0], q, asset, w.From.UnixNano(), w.Until.UnixNano())
	return &a[0], err
}

// Returns gapfilled PoolStakes for given pool, window and interval
func GetPoolStakes(ctx context.Context, pool string, window Window, interval Interval) ([]PoolStakes, error) {
	timestamps, window, err := generateBuckets(ctx, interval, window)
	if err != nil {
		return nil, err
	}
	if 0 == len(timestamps) {
		return nil, errors.New("no buckets were generated for given timeframe")
	}

	stakesArr, err := getPoolStakesSparse(ctx, pool, interval, window)
	if err != nil {
		return nil, err
	}

	result := mergeStakesGapfill(pool, timestamps, stakesArr)

	return result, nil
}

func getPoolStakesSparse(ctx context.Context, pool string, interval Interval, w Window) ([]PoolStakes, error) {
	q := `
	SELECT
		$1,
		COALESCE(COUNT(*), 0) as count,
		COALESCE(SUM(asset_e8), 0) as asset_E8,
		COALESCE(SUM(rune_e8), 0) as rune_E8,
		COALESCE(SUM(stake_units), 0) as stake_units,
		date_trunc($4, to_timestamp(block_timestamp/1000000000/300*300)) AS truncated
	FROM stake_events
	WHERE pool = $1 AND block_timestamp >= $2 AND block_timestamp < $3
	GROUP BY truncated
	ORDER BY truncated ASC`

	return appendPoolStakesBuckets(ctx, []PoolStakes{}, q, pool, w.From.UnixNano(), w.Until.UnixNano(), dbIntervalName[interval])
}

func mergeStakesGapfill(pool string, timestamps []db.Second, stakesArr []PoolStakes) []PoolStakes {
	stakesArrCounter := 0
	result := make([]PoolStakes, len(timestamps))

	for i, ts := range timestamps {
		if len(stakesArr) != 0 && db.TimeToSecond(stakesArr[stakesArrCounter].Time) == ts {
			result[i] = stakesArr[stakesArrCounter]
			if stakesArrCounter < len(stakesArr)-1 {
				stakesArrCounter++
			}
		} else {
			result[i] = PoolStakes{
				Time:  ts.ToTime(),
				Asset: pool,
			}
		}
	}
	return result
}

func PoolStakesAddrLookup(ctx context.Context, asset, addr string, w Window) (*PoolStakes, error) {
	const q = `SELECT $2, COALESCE(COUNT(*), 0), COALESCE(SUM(asset_e8), 0), COALESCE(SUM(rune_e8), 0), COALESCE(SUM(stake_units), 0), COALESCE(MIN(block_timestamp), 0), COALESCE(MAX(block_timestamp), 0)
FROM stake_events
WHERE rune_addr = $1 AND pool = $2 AND block_timestamp >= $3 AND block_timestamp < $4`

	var a [1]PoolStakes
	_, err := appendPoolStakes(ctx, a[:0], q, addr, asset, w.From.UnixNano(), w.Until.UnixNano())
	return &a[0], err
}

func PoolStakesAddrBucketsLookup(ctx context.Context, asset, addr string, bucketSize time.Duration, w Window) ([]PoolStakes, error) {
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
	return appendPoolStakes(ctx, a, q, addr, asset, w.From.UnixNano(), w.Until.UnixNano(), bucketSize.Nanoseconds())
}

func AllPoolStakesAddrLookup(ctx context.Context, addr string, w Window) ([]PoolStakes, error) {
	const q = `SELECT pool, COALESCE(COUNT(*), 0), COALESCE(SUM(asset_e8), 0), COALESCE(SUM(rune_e8), 0), COALESCE(SUM(stake_units), 0), COALESCE(MIN(block_timestamp), 0), COALESCE(MAX(block_timestamp), 0)
FROM stake_events
WHERE rune_addr = $1 AND block_timestamp >= $2 AND block_timestamp < $3
GROUP BY pool`

	return appendPoolStakes(ctx, nil, q, addr, w.From.UnixNano(), w.Until.UnixNano())
}

func appendPoolStakes(ctx context.Context, a []PoolStakes, q string, args ...interface{}) ([]PoolStakes, error) {
	rows, err := db.Query(ctx, q, args...)
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

func appendPoolStakesBuckets(ctx context.Context, a []PoolStakes, q string, args ...interface{}) ([]PoolStakes, error) {
	rows, err := db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var r PoolStakes
		err := rows.Scan(&r.Asset, &r.TxCount, &r.AssetE8Total, &r.RuneE8Total, &r.StakeUnitsTotal, &r.Time)
		if err != nil {
			return a, err
		}
		a = append(a, r)
	}
	return a, rows.Err()
}
