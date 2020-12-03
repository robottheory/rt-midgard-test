package stat

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/graphql/model"
)

type PoolDepth struct {
	Time  time.Time
	Rune  int64
	Asset int64
	Price float64
}

// Each bucket contains the latest depths before the timestamp.
func PoolDepthBucketsLookup(ctx context.Context, pool string, interval Interval, w Window) ([]*model.PoolHistoryBucket, error) {
	ret := []*model.PoolHistoryBucket{}

	timestamps, w, err := generateBuckets(ctx, interval, w)
	if err != nil {
		return nil, err
	}
	if 0 == len(timestamps) {
		return ret, nil
	}

	// last rune and asset depths before the first bucket
	prevRune, prevAsset, err := depthBefore(ctx, pool, timestamps[0].ToNano())
	if err != nil {
		return nil, err
	}

	const q = `
		SELECT
			last(rune_e8, block_timestamp) as rune_e8,
			last(asset_e8, block_timestamp) as asset_e8,
			date_trunc($4, to_timestamp(block_timestamp/1000000000/300*300)) as truncated
		FROM block_pool_depths
		WHERE  pool = $1 AND $2 <= block_timestamp AND block_timestamp < $3
		GROUP BY truncated
		ORDER BY truncated ASC
	`

	rows, err := db.Query(ctx, q, pool, w.From.UnixNano(), w.Until.UnixNano(), dbIntervalName[interval])
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nextTimestamp Second
	var nextRune, nextAsset int64
	for _, bucketTime := range timestamps {
		var price float64
		if prevAsset != 0 {
			price, _ = big.NewRat(prevRune, prevAsset).Float64()
		}
		bucket := model.PoolHistoryBucket{
			Time:  bucketTime.ToI(),
			Rune:  prevRune,
			Asset: prevAsset,
			Price: price,
		}
		ret = append(ret, &bucket)

		// We read values after we created this bucket because
		// the values found here are the depths for the next bucket.
		if nextTimestamp < bucketTime {
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
				nextTimestamp = TimeToSecond(first)
				nextRune = *nextRuneP
				nextAsset = *nextAssetP
			} else {
				// There were no more depths, all following buckets will
				// repeat the previous values
				nextTimestamp = timestamps[len(timestamps)-1] + 1
			}
			if nextTimestamp < bucketTime {
				// Should never happen, gapfill buckets were incomplete.
				return nil, fmt.Errorf("Internal error, buckets misalligned.")
			}
		}
		if nextTimestamp == bucketTime {
			prevRune = nextRune
			prevAsset = nextAsset
		}
	}
	return ret, nil
}

func depthBefore(ctx context.Context, pool string, time Nano) (firstRune, firstAsset int64, err error) {
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
		return 0, 0, fmt.Errorf("No depth found for asset %v before %v", pool, time)
	}
	err = rows.Scan(&firstRune, &firstAsset)
	if err != nil {
		return 0, 0, err
	}
	return firstRune, firstAsset, nil
}
