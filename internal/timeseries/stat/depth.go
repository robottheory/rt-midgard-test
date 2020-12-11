package stat

import (
	"context"
	"fmt"

	"gitlab.com/thorchain/midgard/internal/db"
)

type PoolDepthBucket struct {
	StartTime  db.Second
	EndTime    db.Second
	AssetDepth int64
	RuneDepth  int64
	AssetPrice float64
}

func AssetPrice(assetDepth, runeDepth int64) float64 {
	if assetDepth == 0 {
		return 0
	}
	return float64(runeDepth) / float64(assetDepth)
}

// Each bucket contains the latest depths before the timestamp.
func PoolDepthHistory(ctx context.Context, buckets db.Buckets, pool string) (ret []PoolDepthBucket, err error) {
	ret = make([]PoolDepthBucket, buckets.Count())

	// last rune and asset depths before the first bucket
	assetDepth, runeDepth, err := depthBefore(ctx, pool, buckets.Timestamps[0].ToNano())
	if err != nil {
		return nil, err
	}
	var q = `
		SELECT
			last(asset_e8, block_timestamp) as asset_e8,
			last(rune_e8, block_timestamp) as rune_e8,
			` + db.SelectTruncatedTimestamp("block_timestamp", "$4") + ` AS truncated
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
	var nextAsset, nextRune int64
	for i := 0; i < buckets.Count(); i++ {
		current := &ret[i]
		current.StartTime, current.EndTime = buckets.Bucket(i)

		if nextDBTimestamp < current.StartTime {
			if rows.Next() {
				var nextRuneP, nextAssetP *int64
				err := rows.Scan(&nextAssetP, &nextRuneP, &nextDBTimestamp)
				if err != nil {
					return nil, err
				}
				if nextRuneP == nil || nextAssetP == nil {
					// programming error
					return nil, fmt.Errorf("Internal error: empty rune or asset")
				}
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
			runeDepth = nextRune
			assetDepth = nextAsset
		}
		current.RuneDepth = runeDepth
		current.AssetDepth = assetDepth
		current.AssetPrice = AssetPrice(assetDepth, runeDepth)
	}
	return ret, nil
}

func depthBefore(ctx context.Context, pool string, time db.Nano) (firstAsset, firstRune int64, err error) {
	const firstValueQuery = `
		SELECT
			asset_e8,
			rune_e8
		FROM block_pool_depths
		WHERE pool = $1 AND block_timestamp < $2
		ORDER BY block_timestamp DESC
		LIMIT 1
	`
	rows, err := db.Query(ctx, firstValueQuery, pool, time)
	if err != nil {
		return
	}
	defer rows.Close()

	ok := rows.Next()
	if !ok {
		return 0, 0, nil
	}
	err = rows.Scan(&firstAsset, &firstRune)
	return
}
