package stat

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"gitlab.com/thorchain/midgard/internal/graphql/model"
)

type PoolDepth struct {
	Time  time.Time
	Rune  int64
	Asset int64
	Price float64
}

// Each bucket contains the latest depths before the timestamp.
func PoolDepthBucketsLookup(ctx context.Context, pool string, interval model.Interval, w Window) ([]*model.PoolHistoryBucket, error) {
	ret := []*model.PoolHistoryBucket{}

	w, err := calcBounds(w, interval)
	if err != nil {
		return nil, err
	}

	timestamps, err := generateBuckets(ctx, interval, w)
	if err != nil {
		return nil, err
	}
	if 0 == len(timestamps) {
		return ret, nil
	}

	// last rune and asset depths before the first bucket
	prevRune, prevAsset, err := depthBefore(ctx, pool, timestamps[0]*1000000000)
	if err != nil {
		return nil, err
	}

	const q = `
		SELECT
			last(rune_e8, block_timestamp) as rune_e8,
			last(asset_e8, block_timestamp) as asset_e8,
			date_trunc($4, to_timestamp(block_timestamp/1000000000)) as truncated
		FROM block_pool_depths
		WHERE  pool = $1 AND $2 <= block_timestamp AND block_timestamp < $3
		GROUP BY truncated
		ORDER BY truncated ASC
	`

	rows, err := DBQuery(ctx, q, pool, w.From.UnixNano(), w.Until.UnixNano(), interval)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nextTimestamp int64
	var nextRune, nextAsset int64
	for _, bucketTime := range timestamps {
		var price float64
		if prevAsset != 0 {
			price, _ = big.NewRat(prevRune, prevAsset).Float64()
		}
		bucket := model.PoolHistoryBucket{
			Time:  bucketTime,
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
				nextTimestamp = first.Unix()
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

func depthBefore(ctx context.Context, pool string, time int64) (firstRune, firstAsset int64, err error) {
	const firstValueQuery = `
		SELECT
			rune_e8,
			asset_e8
		FROM block_pool_depths
		WHERE pool = $1 AND block_timestamp < $2
		ORDER BY block_timestamp DESC
		LIMIT 1
	`
	rows, err := DBQuery(ctx, firstValueQuery, pool, time)
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

// Returns all the buckets for the window, so other queries don't have to care about gapfill functionality.
func generateBuckets(ctx context.Context, interval model.Interval, w Window) ([]int64, error) {
	// We use an SQL query to use the date_trunc of sql.
	// It's not important which table we select we just need a timestamp type and we use WHERE 1=0
	// in order not to actually select any data.
	// We could consider writing an sql function instead or programming dategeneration in go.

	gapfill, err := getGapfillFromLimit(interval)
	if err != nil {
		return nil, err
	}

	q := fmt.Sprintf(`
		WITH gapfill AS (
			SELECT
				time_bucket_gapfill(%s, block_timestamp, $1::BIGINT, $2::BIGINT) as bucket
			FROM block_pool_depths
			WHERE 1=0
			GROUP BY bucket)
		SELECT
			date_trunc($3, to_timestamp(bucket/1000000000)) as truncated
		FROM gapfill
		GROUP BY truncated
		ORDER BY truncated ASC
	`, gapfill)

	// TODO(acsaba): change the gapfill parameter to seconds, and pass seconds here too.
	rows, err := DBQuery(ctx, q, w.From.UnixNano(), w.Until.UnixNano()-1000000000, interval)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ret := []int64{}
	for rows.Next() {
		var timestamp time.Time
		err := rows.Scan(&timestamp)
		if err != nil {
			return nil, err
		}
		ret = append(ret, timestamp.Unix())
	}
	return ret, nil
}
