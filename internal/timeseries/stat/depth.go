package stat

import (
	"context"
	"database/sql"

	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/util/miderr"
)

type PoolDepthBucket struct {
	Window     db.Window
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

// - Queries database
// - Calls scanNext for each row. scanNext should store the values outside.
// - Calls saveBucket for each bucket signaling if the stored value is for the current bucket.
func queryBucketedGeneral(
	ctx context.Context, buckets db.Buckets,
	scanNext func(*sql.Rows) (db.Second, error),
	saveBucket func(idx int, bucketWindow db.Window, nextIsCurrent bool),
	q string, qargs ...interface{}) error {

	rows, err := db.Query(ctx, q, qargs...)
	if err != nil {
		return err
	}
	defer rows.Close()

	var nextDBTimestamp db.Second

	for i := 0; i < buckets.Count(); i++ {
		bucketWindow := buckets.BucketWindow(i)

		if nextDBTimestamp < bucketWindow.From {
			if rows.Next() {
				nextDBTimestamp, err = scanNext(rows)
				if err != nil {
					return err
				}
			} else {
				// There were no more depths, all following buckets will
				// repeat the previous values
				nextDBTimestamp = buckets.End() + 1
			}
			if nextDBTimestamp < bucketWindow.From {
				// Should never happen, gapfill buckets were incomplete.
				return miderr.InternalErr("Internal error, buckets misalligned.")
			}
		}
		saveBucket(i, bucketWindow, nextDBTimestamp == bucketWindow.From)
	}

	return nil
}

// Each bucket contains the latest depths before the timestamp.
func PoolDepthHistory(ctx context.Context, buckets db.Buckets, pool string) (
	ret []PoolDepthBucket, err error) {

	// last rune and asset depths before the first bucket
	lastAssetDepth, lastRuneDepth, err := depthBefore(ctx, pool, buckets.Timestamps[0].ToNano())
	if err != nil {
		return nil, err
	}

	var q = `
		SELECT
			last(asset_e8, block_timestamp) as asset_e8,
			last(rune_e8, block_timestamp) as rune_e8,
			` + db.SelectTruncatedTimestamp("block_timestamp", buckets) + ` AS truncated
		FROM block_pool_depths
		WHERE pool = $1 AND $2 <= block_timestamp AND block_timestamp < $3
		GROUP BY truncated
		ORDER BY truncated ASC
	`
	qargs := []interface{}{pool, buckets.Start().ToNano(), buckets.End().ToNano()}

	ret = make([]PoolDepthBucket, buckets.Count())
	var next PoolDepthBucket

	err = queryBucketedGeneral(
		ctx, buckets,
		func(rows *sql.Rows) (nextTimestamp db.Second, err error) {
			// read a single row
			var nextRuneP, nextAssetP *int64
			err = rows.Scan(&nextAssetP, &nextRuneP, &nextTimestamp)
			if err != nil {
				return 0, err
			}
			// TODO(acsaba): fields are not null this can be deleted.
			if nextRuneP == nil || nextAssetP == nil {
				// programming error
				return 0, miderr.InternalErr("Internal error: empty rune or asset")
			}
			next.RuneDepth = *nextRuneP
			next.AssetDepth = *nextAssetP
			return
		},
		func(idx int, bucketWindow db.Window, nextIsCurrent bool) {
			// Save data for bucket
			if nextIsCurrent {
				lastAssetDepth = next.AssetDepth
				lastRuneDepth = next.RuneDepth
			}
			ret[idx].Window = bucketWindow
			ret[idx].AssetDepth = lastAssetDepth
			ret[idx].RuneDepth = lastRuneDepth
			ret[idx].AssetPrice = AssetPrice(lastAssetDepth, lastRuneDepth)
		},
		q, qargs...)

	if err != nil {
		return nil, err
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
