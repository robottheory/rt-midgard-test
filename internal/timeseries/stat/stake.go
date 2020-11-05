package stat

import (
	"context"
	"errors"
	"fmt"
	"gitlab.com/thorchain/midgard/internal/graphql/model"
	"time"
)

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
	rows, err := DBQuery(ctx, q, args...)
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
func GetPoolStakes(ctx context.Context, pool string, window Window, interval model.Interval) ([]PoolStakes, error) {
	timestamps, err := generateBuckets(ctx, interval, window)
	if err != nil {
		return nil, err
	}
	if 0 == len(timestamps) {
		return nil, errors.New("no buckets were generated for given timeframe")
	}

	result := make([]PoolStakes, len(timestamps))

	stakesArr, err := PoolStakesBucketsLookup(ctx, pool, interval, window)
	if err != nil {
		return nil, err
	}

	if len(stakesArr) == 0 {
		for i, ts := range timestamps {
			ps := PoolStakes{
				Asset: pool,
				Time:  time.Unix(ts, 0),
			}
			result[i] = ps
		}
		return result, nil
	}

	stakesArrCounter := 0
	for i, ts := range timestamps {
		saResult := stakesArr[stakesArrCounter]
		saTimestamp := saResult.Time

		ps := PoolStakes{
			Time:  time.Unix(ts, 0),
			Asset: pool,
		}

		if saTimestamp.Unix() == ts {
			ps = PoolStakes{
				Asset:           saResult.Asset,
				TxCount:         saResult.TxCount,
				AssetE8Total:    saResult.AssetE8Total,
				RuneE8Total:     saResult.RuneE8Total,
				StakeUnitsTotal: saResult.StakeUnitsTotal,
				Time:            saResult.Time,
			}
			if stakesArrCounter < len(stakesArr)-1 {
				stakesArrCounter++
			}
		}
		result[i] = ps
	}

	return result, nil
}

func PoolStakesBucketsLookup(ctx context.Context, pool string, interval model.Interval, w Window) ([]PoolStakes, error) {
	bucket, err := getBucketFromInterval(interval)
	if err != nil {
		return nil, err
	}

	q := fmt.Sprintf(
		`
		SELECT
		  $1,	
		  COALESCE(COUNT(*), 0) as count,
		  COALESCE(SUM(asset_e8), 0) as asset_E8,
		  COALESCE(SUM(rune_e8), 0) as rune_E8,
		  COALESCE(SUM(stake_units), 0) as stake_units,
		  time_bucket('%s', date_trunc($4, to_timestamp(block_timestamp/1000000000))) AS bucket
		FROM stake_events
		  WHERE pool = $1 AND block_timestamp >= $2 AND block_timestamp < $3
		  GROUP BY bucket
		  ORDER BY bucket ASC`,
		bucket)

	return appendPoolStakesBuckets(ctx, []PoolStakes{}, q, pool, w.From.UnixNano(), w.Until.UnixNano(), interval)
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
	rows, err := DBQuery(ctx, q, args...)
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
	rows, err := DBQuery(ctx, q, args...)
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
