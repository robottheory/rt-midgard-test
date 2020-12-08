package stat

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"gitlab.com/thorchain/midgard/internal/db"
)

type PoolDepthBucket struct {
	StartTime  db.Second
	EndTime    db.Second
	AssetDepth int64
	RuneDepth  int64
	AssetPrice float64
}

// Each bucket contains the latest depths before the timestamp.
// TODO(acsaba): change logic so it uses the value from the end of the interval( not the beginning)
// TODO(acsaba): add unit test for v1 api.
func PoolDepthHistory(ctx context.Context, buckets db.Buckets, pool string) (ret []PoolDepthBucket, err error) {
	ret = make([]PoolDepthBucket, buckets.Count())

	// last rune and asset depths before the first bucket
	prevRune, prevAsset, err := depthBefore(ctx, pool, buckets.Timestamps[0].ToNano())
	if err != nil {
		return nil, err
	}

	const q = `
		SELECT
			last(rune_e8, block_timestamp) as rune_e8,
			last(asset_e8, block_timestamp) as asset_e8,
			date_trunc($4, to_timestamp(block_timestamp/1000000000/300*300)) as truncated
		FROM block_pool_depths
		WHERE pool = $1 AND $2 <= block_timestamp AND block_timestamp < $3
		GROUP BY truncated
		ORDER BY truncated ASC
	`

	rows, err := db.Query(ctx, q, pool,
		buckets.Start().ToNano(), buckets.End().ToNano(), db.DBIntervalName[buckets.Interval])
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nextDBTimestamp db.Second
	var nextRune, nextAsset int64
	for i := 0; i < buckets.Count(); i++ {
		current := &ret[i]
		current.StartTime, current.EndTime = buckets.Bucket(i)
		current.RuneDepth = prevRune
		current.AssetDepth = prevAsset
		if prevAsset != 0 {
			// TODONOW SOLVE
			current.AssetPrice, _ = big.NewRat(prevRune, prevAsset).Float64()
		}

		// We read values after we created this bucket because
		// the values found here are the depths for the next bucket.
		if nextDBTimestamp < current.StartTime {
			if rows.Next() {
				var first time.Time
				var nextRuneP, nextAssetP *int64
				err := rows.Scan(&nextRuneP, &nextAssetP, &first)
				if err != nil {
					return nil, err
				}
				if nextRuneP == nil || nextAssetP == nil {
					// programming error
					return nil, fmt.Errorf("Internal error: empty rune or asset")
				}
				// TODO(acsaba): check if this is correct (UTC)?
				// TODO(acsaba): check if all results should be UTC?
				nextDBTimestamp = db.TimeToSecond(first)
				nextRune = *nextRuneP
				nextAsset = *nextAssetP
			} else {
				// There were no more depths, all following buckets will
				// repeat the previous values
				nextDBTimestamp = buckets.End() + 1
			}
			if nextDBTimestamp < current.StartTime {
				// Should never happen, gapfill buckets were incomplete.
				return nil, fmt.Errorf("Internal error, buckets misalligned.")
			}
		}
		if nextDBTimestamp == current.StartTime {
			prevRune = nextRune
			prevAsset = nextAsset
		}
	}
	return ret, nil
}

func depthBefore(ctx context.Context, pool string, time db.Nano) (firstRune, firstAsset int64, err error) {
	const firstValueQuery = `
		SELECT
			rune_e8,
			asset_e8
		FROM block_pool_depths
		WHERE pool = $1 AND block_timestamp < $2
		ORDER BY block_timestamp DESC
		LIMIT 1
	`
	rows, err := db.Query(ctx, firstValueQuery, pool, time)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	ok := rows.Next()
	if !ok {
		return 0, 0, nil
	}
	err = rows.Scan(&firstRune, &firstAsset)
	if err != nil {
		return 0, 0, err
	}
	return firstRune, firstAsset, nil
}
