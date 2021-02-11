package stat

import (
	"context"
	"database/sql"

	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/timeseries"
)

type PoolDepthBucket struct {
	Window        db.Window
	Depths        timeseries.DepthPair
	AssetPriceUSD float64
}

// - Queries database, possibly multiple rows per window.
// - Calls scanNext for each row. scanNext should store the values outside.
// - Calls applyLastScanned to save the scanned value for the current bucket.
// - Calls saveBucket for each bucket.
func queryBucketedGeneral(
	ctx context.Context, buckets db.Buckets,
	scan func(*sql.Rows) (db.Second, error),
	applyLastScanned func(),
	saveBucket func(idx int, bucketWindow db.Window),
	q string, qargs ...interface{}) error {

	rows, err := db.Query(ctx, q, qargs...)
	if err != nil {
		return err
	}
	defer rows.Close()

	var nextDBTimestamp db.Second

	for i := 0; i < buckets.Count(); i++ {
		bucketWindow := buckets.BucketWindow(i)

		if nextDBTimestamp == bucketWindow.From {
			applyLastScanned()
		}

		for nextDBTimestamp <= bucketWindow.From {
			if rows.Next() {
				nextDBTimestamp, err = scan(rows)
				if err != nil {
					return err
				}
				if nextDBTimestamp == bucketWindow.From {
					applyLastScanned()
				}
			} else {
				// There were no more depths, all following buckets will
				// repeat the previous values
				nextDBTimestamp = buckets.End() + 1
			}
		}
		saveBucket(i, bucketWindow)
	}

	return nil
}

func addUsdPools(pool string) []string {
	allPools := make([]string, 0, len(usdPoolWhitelist)+1)
	allPools = append(allPools, pool)
	allPools = append(allPools, usdPoolWhitelist...)
	return allPools
}

// Each bucket contains the latest depths before the timestamp.
func PoolDepthHistory(ctx context.Context, buckets db.Buckets, pool string) (
	ret []PoolDepthBucket, err error) {

	allPools := addUsdPools(pool)

	// last rune and asset depths before the first bucket
	poolDepths, err := depthBefore(ctx, allPools, buckets.Timestamps[0].ToNano())

	if err != nil {
		return nil, err
	}

	var q = `
		SELECT
			pool,
			last(asset_e8, block_timestamp) as asset_e8,
			last(rune_e8, block_timestamp) as rune_e8,
			` + db.SelectTruncatedTimestamp("block_timestamp", buckets) + ` AS truncated
		FROM block_pool_depths
		WHERE pool = ANY($1) AND $2 <= block_timestamp AND block_timestamp < $3
		GROUP BY truncated, pool
		ORDER BY truncated ASC
	`
	qargs := []interface{}{allPools, buckets.Start().ToNano(), buckets.End().ToNano()}

	ret = make([]PoolDepthBucket, buckets.Count())

	var next struct {
		pool   string
		depths timeseries.DepthPair
	}

	readNext := func(rows *sql.Rows) (nextTimestamp db.Second, err error) {
		err = rows.Scan(&next.pool, &next.depths.AssetDepth, &next.depths.RuneDepth, &nextTimestamp)
		if err != nil {
			return 0, err
		}
		return
	}
	applyNext := func() {
		poolDepths[next.pool] = next.depths
	}
	saveBucket := func(idx int, bucketWindow db.Window) {
		runePriceUSD := runePriceUSDForDepths(poolDepths)
		depths := poolDepths[pool]

		ret[idx].Window = bucketWindow
		ret[idx].Depths = depths
		ret[idx].AssetPriceUSD = depths.AssetPrice() * runePriceUSD
	}

	err = queryBucketedGeneral(ctx, buckets, readNext, applyNext, saveBucket, q, qargs...)

	if err != nil {
		return nil, err
	}

	return ret, nil
}

func depthBefore(ctx context.Context, pools []string, time db.Nano) (
	ret timeseries.DepthMap, err error) {

	const firstValueQuery = `
		SELECT
			pool,
			last(asset_e8, block_timestamp) AS asset_e8,
			last(rune_e8, block_timestamp) AS rune_e8
		FROM block_pool_depths
		WHERE pool = ANY($1) AND block_timestamp < $2
		GROUP BY pool
	`
	rows, err := db.Query(ctx, firstValueQuery, pools, time)
	if err != nil {
		return
	}
	defer rows.Close()

	ret = timeseries.DepthMap{}
	for rows.Next() {
		var pool string
		var depths timeseries.DepthPair
		err = rows.Scan(&pool, &depths.AssetDepth, &depths.RuneDepth)
		if err != nil {
			return
		}

		ret[pool] = depths
	}
	for _, pool := range pools {
		_, present := ret[pool]
		if !present {
			ret[pool] = timeseries.DepthPair{}
		}
	}
	return
}
