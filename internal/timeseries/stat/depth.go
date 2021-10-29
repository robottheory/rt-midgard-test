package stat

import (
	"context"
	"database/sql"

	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/internal/util/miderr"
)

type PoolDepthBucket struct {
	Window        db.Window
	Depths        timeseries.PoolDepths
	AssetPriceUSD float64
}

type TVLDepthBucket struct {
	Window         db.Window
	TotalPoolDepth int64
	RunePriceUSD   float64
}

type OHLCVBucket struct {
	Window db.Window
	Depths timeseries.PoolOHLCV
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

var poolDepthsAggregate = db.RegisterAggregate(
	db.NewAggregate("pool_depths", "block_pool_depths").
		AddGroupColumn("pool").
		AddLastColumn("asset_e8").
		AddLastColumn("rune_e8").
		AddLastColumn("synth_e8").
		AddLastColumn("units").
		AddFirstColumn("block_timestamp").
		AddLastColumn("block_timestamp").
		AddFirstColumn("priceUSD").
		AddLastColumn("priceUSD").
		AddMinColumn("priceUSD").
		AddMaxColumn("priceUSD"))

func getDepthsHistory(ctx context.Context, buckets db.Buckets, pools []string,
	saveDepths func(idx int, bucketWindow db.Window, depths timeseries.DepthMap)) (err error) {
	var poolDepths timeseries.DepthMap
	if buckets.OneInterval() {
		// We only interested in the state at the end of the single interval:
		poolDepths, err = depthBefore(ctx, pools, buckets.Timestamps[1].ToNano())
		if err != nil {
			return err
		}
		saveDepths(0, buckets.BucketWindow(0), poolDepths)
		return
	}

	// last rune and asset depths before the first bucket
	poolDepths, err = depthBefore(ctx, pools, buckets.Timestamps[0].ToNano())
	if err != nil {
		return err
	}
	poolFilter := ""
	qargs := []interface{}{buckets.Start().ToNano(), buckets.End().ToNano()}
	if pools != nil {
		poolFilter = "pool = ANY($3)"
		qargs = []interface{}{buckets.Start().ToNano(), buckets.End().ToNano(), pools}
	}

	q := `
		SELECT
			pool,
			asset_e8,
			rune_e8,
			synth_e8,
			aggregate_timestamp / 1000000000 AS truncated
		FROM midgard_agg.pool_depths_` + buckets.AggregateName() + `
		` + db.Where("$1 <= aggregate_timestamp", "aggregate_timestamp < $2", poolFilter) + `
		ORDER BY aggregate_timestamp ASC
	`

	var next struct {
		pool   string
		depths timeseries.PoolDepths
	}

	readNext := func(rows *sql.Rows) (nextTimestamp db.Second, err error) {
		err = rows.Scan(&next.pool, &next.depths.AssetDepth, &next.depths.RuneDepth, &next.depths.SynthDepth, &nextTimestamp)
		if err != nil {
			return 0, err
		}
		return
	}
	applyNext := func() {
		poolDepths[next.pool] = next.depths
	}
	saveBucket := func(idx int, bucketWindow db.Window) {
		saveDepths(idx, bucketWindow, poolDepths)
	}

	return queryBucketedGeneral(ctx, buckets, readNext, applyNext, saveBucket, q, qargs...)
}

func getMinDate(ctx context.Context, pool string, startDate db.Nano, endDate db.Nano) int64 {
	poolFilter := `midgard_agg.pool_depths_5min.pool = $3`
	qargs := []interface{}{startDate, endDate, pool}
	q := `
		SELECT
			 min(first_priceusd),
			first_block_timestamp
		FROM midgard_agg.pool_depths_5min
		` + db.Where("$1 <= aggregate_timestamp", "aggregate_timestamp < $2", poolFilter) + `
		GROUP BY first_block_timestamp
		ORDER BY min(first_priceusd) ASC LIMIT 1
	`
	rows, err := db.Query(ctx, q, qargs...)
	if err != nil {
		return -1
	}
	defer rows.Close()
	if rows.Next() {
		var firstPrice5min float64
		var startTimestamp5min int64
		err = rows.Scan(&firstPrice5min, &startTimestamp5min)
		if err != nil {
			return -1
		}
		return startTimestamp5min
	}
	return -1
}

func getMaxDate(ctx context.Context, pool string, startDate db.Nano, endDate db.Nano) int64 {
	poolFilter := `midgard_agg.pool_depths_5min.pool = $3`
	qargs := []interface{}{startDate, endDate, pool}
	q := `
		SELECT
			 max(first_priceusd),
			first_block_timestamp
		FROM midgard_agg.pool_depths_5min
		` + db.Where("$1 <= aggregate_timestamp", "aggregate_timestamp < $2", poolFilter) + `
		GROUP BY first_block_timestamp
		ORDER BY max(first_priceusd) DESC LIMIT 1
	`
	rows, err := db.Query(ctx, q, qargs...)
	if err != nil {
		return -1
	}
	defer rows.Close()
	if rows.Next() {
		var firstPrice5min float64
		var startTimestamp5min int64
		err = rows.Scan(&firstPrice5min, &startTimestamp5min)
		if err != nil {
			return -1
		}
		return startTimestamp5min
	}
	return -1
}

func getOHCLVSimpleHistory(ctx context.Context, buckets db.Buckets, pool string,
	saveDepths func(idx int, bucketWindow db.Window, depths timeseries.OHLCVMap)) (err error) {
	var poolDepths timeseries.OHLCVMap

	poolDepths = make(timeseries.OHLCVMap, 0)
	poolFilter := `midgard_agg.pool_depths_` + buckets.AggregateName() + `.pool = $3`
	qargs := []interface{}{buckets.Start().ToNano(), buckets.End().ToNano(), pool}

	q := ``

	var next struct {
		pool   string
		depths timeseries.PoolOHLCV
	}
	if buckets.AggregateName() == "5min" {
		q = `
		SELECT
			first_priceusd,
			priceusd,
			first_block_timestamp,
			block_timestamp,
			pool,
			asset_e8,
			rune_e8,
			synth_e8,
			first_block_timestamp,
			block_timestamp,
			first_priceusd,
			priceusd,
			min_priceusd,
			max_priceusd,
			aggregate_timestamp / 1000000000 AS truncated
		FROM midgard_agg.pool_depths_` + buckets.AggregateName() + `
		` + db.Where("$1 <= aggregate_timestamp", "aggregate_timestamp < $2", poolFilter) + `
		ORDER BY aggregate_timestamp ASC
	`
	} else {
		q = `
		SELECT
			-1,
			-1,
			-1,
			-1,
			midgard_agg.pool_depths_` + buckets.AggregateName() + `.pool,
			midgard_agg.pool_depths_` + buckets.AggregateName() + `.asset_e8,
			midgard_agg.pool_depths_` + buckets.AggregateName() + `.rune_e8,
			midgard_agg.pool_depths_` + buckets.AggregateName() + `.synth_e8,
			midgard_agg.pool_depths_` + buckets.AggregateName() + `.first_block_timestamp,
			midgard_agg.pool_depths_` + buckets.AggregateName() + `.block_timestamp,
			midgard_agg.pool_depths_` + buckets.AggregateName() + `.first_priceusd,
			midgard_agg.pool_depths_` + buckets.AggregateName() + `.priceusd,
			midgard_agg.pool_depths_` + buckets.AggregateName() + `.min_priceusd,
			midgard_agg.pool_depths_` + buckets.AggregateName() + `.max_priceusd,
			midgard_agg.pool_depths_` + buckets.AggregateName() + `.aggregate_timestamp / 1000000000 AS truncated
		FROM midgard_agg.pool_depths_` + buckets.AggregateName() + `
		` + db.Where(`$1 <= midgard_agg.pool_depths_`+buckets.AggregateName()+`.aggregate_timestamp`, `midgard_agg.pool_depths_`+buckets.AggregateName()+`.aggregate_timestamp < $2`, poolFilter) + `
		ORDER BY midgard_agg.pool_depths_` + buckets.AggregateName() + `.aggregate_timestamp ASC`
	}

	readNext := func(rows *sql.Rows) (nextTimestamp db.Second, err error) {
		var firstPrice5min, lastPrice5min float64
		var startTimestamp5min, lastTimestamp5min int64
		err = rows.Scan(&firstPrice5min, &lastPrice5min, &startTimestamp5min, &lastTimestamp5min, &next.pool, &next.depths.AssetDepth, &next.depths.RuneDepth, &next.depths.SynthDepth, &next.depths.FirstDate, &next.depths.LastDate, &next.depths.FirstPrice, &next.depths.LastPrice, &next.depths.MinPrice, &next.depths.MaxPrice, &nextTimestamp)
		if err != nil {
			return 0, err
		}
		if next.depths.FirstPrice < next.depths.LastPrice {
			next.depths.MaxDate = int64(float64(next.depths.LastDate) * 1e-9)
			next.depths.MinDate = int64(float64(next.depths.FirstDate) * 1e-9)
		} else {
			next.depths.MaxDate = int64(float64(next.depths.FirstDate) * 1e-9)
			next.depths.MinDate = int64(float64(next.depths.LastDate) * 1e-9)
		}
		return
	}
	applyNext := func() {
		poolDepths[next.pool] = next.depths
	}
	saveBucket := func(idx int, bucketWindow db.Window) {
		saveDepths(idx, bucketWindow, poolDepths)
	}

	return queryBucketedGeneral(ctx, buckets, readNext, applyNext, saveBucket, q, qargs...)
}

// Each bucket contains the latest depths before the timestamp.
// Returns dense results (i.e. not sparse).
func PoolDepthHistory(ctx context.Context, buckets db.Buckets, pool string) (
	ret []PoolDepthBucket, err error) {
	allPools := addUsdPools(pool)
	ret = make([]PoolDepthBucket, buckets.Count())

	saveDepths := func(idx int, bucketWindow db.Window, poolDepths timeseries.DepthMap) {
		runePriceUSD := runePriceUSDForDepths(poolDepths)
		depths := poolDepths[pool]

		ret[idx].Window = bucketWindow
		ret[idx].Depths = depths
		ret[idx].AssetPriceUSD = depths.AssetPrice() * runePriceUSD
	}

	err = getDepthsHistory(ctx, buckets, allPools, saveDepths)
	return ret, err
}

func PoolOHLCVHistory(ctx context.Context, buckets db.Buckets, pool string) (
	ret []OHLCVBucket, err error) {
	ret = make([]OHLCVBucket, buckets.Count())
	saveDepths := func(idx int, bucketWindow db.Window, poolDepths timeseries.OHLCVMap) {
		depths := poolDepths[pool]
		ret[idx].Window = bucketWindow
		ret[idx].Depths = depths
		if depths.MaxDate == 0 && depths.MinDate == 0 {
			return
		}
		maxDate := getMaxDate(ctx, pool, bucketWindow.From.ToNano(), bucketWindow.Until.ToNano())
		minDate := getMinDate(ctx, pool, bucketWindow.From.ToNano(), bucketWindow.Until.ToNano())
		if maxDate == -1 || minDate == -1 {
			return
		}
		ret[idx].Depths.MaxDate = int64(float64(maxDate) * 1e-9)
		ret[idx].Depths.MinDate = int64(float64(minDate) * 1e-9)
	}

	err = getOHCLVSimpleHistory(ctx, buckets, pool, saveDepths)
	if err != nil {
		return
	}
	usdPrices, err := USDPriceHistory(ctx, buckets)
	if err != nil {
		return
	}

	if len(usdPrices) != buckets.Count() {
		err = miderr.InternalErr("Misalligned buckets")
		return
	}
	for i := 0; i < buckets.Count(); i++ {
		timestamp, _ := buckets.Bucket(i)
		if usdPrices[i].Window.From != timestamp {
			err = miderr.InternalErr("Misalligned buckets")
		}
		ret[i].Depths.Liquidity = int64(float64(ret[i].Depths.AssetDepth)*ret[i].Depths.LastPrice + float64(ret[i].Depths.RuneDepth)*usdPrices[i].RunePriceUSD)
	}
	return ret, err
}

func TVLDepthHistory(ctx context.Context, buckets db.Buckets) (
	ret []TVLDepthBucket, err error) {
	ret = make([]TVLDepthBucket, buckets.Count())

	saveDepths := func(idx int, bucketWindow db.Window, poolDepths timeseries.DepthMap) {
		runePriceUSD := runePriceUSDForDepths(poolDepths)
		var depth int64 = 0
		for _, pair := range poolDepths {
			depth += pair.RuneDepth
		}

		ret[idx].Window = bucketWindow
		ret[idx].TotalPoolDepth = depth
		ret[idx].RunePriceUSD = runePriceUSD
	}

	err = getDepthsHistory(ctx, buckets, nil, saveDepths)
	return ret, err
}

type USDPriceBucket struct {
	Window       db.Window
	RunePriceUSD float64
}

// Each bucket contains the latest depths before the timestamp.
// Returns dense results (i.e. not sparse).
func USDPriceHistory(ctx context.Context, buckets db.Buckets) (
	ret []USDPriceBucket, err error) {
	if len(usdPoolWhitelist) == 0 {
		return nil, miderr.InternalErr("No USD pools defined")
	}

	ret = make([]USDPriceBucket, buckets.Count())

	saveDepths := func(idx int, bucketWindow db.Window, poolDepths timeseries.DepthMap) {
		ret[idx].Window = bucketWindow
		ret[idx].RunePriceUSD = runePriceUSDForDepths(poolDepths)
	}

	err = getDepthsHistory(ctx, buckets, usdPoolWhitelist, saveDepths)
	return ret, err
}

func depthBefore(ctx context.Context, pools []string, time db.Nano) (
	ret timeseries.DepthMap, err error) {
	whereConditions := []string{}
	qargs := []interface{}{}
	if pools != nil {
		whereConditions = append(whereConditions, "pool = ANY($1)")
		qargs = append(qargs, pools)
	}

	subQuery, qargs := poolDepthsAggregate.UnionQuery(0, time, whereConditions, qargs)

	firstValueQuery := `
		SELECT
			pool,
			last(asset_e8, aggregate_timestamp) AS asset_e8,
			last(rune_e8, aggregate_timestamp) AS rune_e8,
			last(synth_e8, aggregate_timestamp) AS synth_e8
		FROM ` + subQuery + ` AS u
		GROUP BY pool
	`

	rows, err := db.Query(ctx, firstValueQuery, qargs...)
	if err != nil {
		return
	}
	defer rows.Close()

	ret = timeseries.DepthMap{}
	for rows.Next() {
		var pool string
		var depths timeseries.PoolDepths
		err = rows.Scan(&pool, &depths.AssetDepth, &depths.RuneDepth, &depths.SynthDepth)
		if err != nil {
			return
		}

		ret[pool] = depths
	}
	for _, pool := range pools {
		_, present := ret[pool]
		if !present {
			ret[pool] = timeseries.PoolDepths{}
		}
	}
	return
}

/*func ohclvBefore(ctx context.Context, pools []string, time db.Nano) (
	ret timeseries.OHLCVMap, err error) {
	whereConditions := []string{}
	qargs := []interface{}{}
	if pools != nil {
		whereConditions = append(whereConditions, "pool = ANY($1)")
		qargs = append(qargs, pools)
	}

	subQuery, qargs := poolDepthsAggregate.UnionQuery(0, time, whereConditions, qargs)

	firstValueQuery := `
		SELECT
			pool,
			last(asset_e8, aggregate_timestamp) AS asset_e8,
			last(rune_e8, aggregate_timestamp) AS rune_e8,
			last(synth_e8, aggregate_timestamp) AS synth_e8,

		FROM ` + subQuery + ` AS u
		GROUP BY pool
	`

	rows, err := db.Query(ctx, firstValueQuery, qargs...)
	if err != nil {
		return
	}
	defer rows.Close()

	ret = timeseries.OHLCVMap{}
	for rows.Next() {
		var pool string
		var depths timeseries.PoolOHLCV
		err = rows.Scan(&pool, &depths.AssetDepth, &depths.RuneDepth, &depths.SynthDepth)
		if err != nil {
			return
		}

		ret[pool] = depths
	}
	for _, pool := range pools {
		_, present := ret[pool]
		if !present {
			ret[pool] = timeseries.PoolOHLCV{}
		}
	}
	return
}*/
