package stat

import (
	"context"

	"gitlab.com/thorchain/midgard/internal/db"
)

// Swaps are generic swap statistics.
type Swaps struct {
	TxCount     int64
	RuneE8Total int64
}

type SwapBucket struct {
	StartTime         db.Second
	EndTime           db.Second
	RuneToAssetCount  int64
	AssetToRuneCount  int64
	RuneToSynthCount  int64
	SynthToRuneCount  int64
	TotalCount        int64
	RuneToAssetVolume int64
	AssetToRuneVolume int64
	RuneToSynthVolume int64
	SynthToRuneVolume int64
	TotalVolume       int64
	RuneToAssetFees   int64
	AssetToRuneFees   int64
	RuneToSynthFees   int64
	SynthToRuneFees   int64
	TotalFees         int64
	RuneToAssetSlip   int64
	AssetToRuneSlip   int64
	RuneToSynthSlip   int64
	SynthToRuneSlip   int64
	TotalSlip         int64
	RunePriceUSD      float64
}

func (meta *SwapBucket) AddBucket(bucket SwapBucket) {
	meta.RuneToAssetCount += bucket.RuneToAssetCount
	meta.AssetToRuneCount += bucket.AssetToRuneCount
	meta.RuneToSynthCount += bucket.RuneToSynthCount
	meta.SynthToRuneCount += bucket.SynthToRuneCount
	meta.TotalCount += bucket.TotalCount
	meta.RuneToAssetVolume += bucket.RuneToAssetVolume
	meta.AssetToRuneVolume += bucket.AssetToRuneVolume
	meta.RuneToSynthVolume += bucket.RuneToSynthVolume
	meta.SynthToRuneVolume += bucket.SynthToRuneVolume
	meta.TotalVolume += bucket.TotalVolume
	meta.RuneToAssetFees += bucket.RuneToAssetFees
	meta.AssetToRuneFees += bucket.AssetToRuneFees
	meta.RuneToSynthFees += bucket.RuneToSynthFees
	meta.SynthToRuneFees += bucket.SynthToRuneFees
	meta.TotalFees += bucket.TotalFees
	meta.RuneToAssetSlip += bucket.RuneToAssetSlip
	meta.AssetToRuneSlip += bucket.AssetToRuneSlip
	meta.RuneToSynthSlip += bucket.RuneToSynthSlip
	meta.SynthToRuneSlip += bucket.SynthToRuneSlip
	meta.TotalSlip += bucket.TotalSlip
}

type OneDirectionSwapBucket struct {
	Time         db.Second
	Count        int64
	VolumeInRune int64
	TotalFees    int64
	TotalSlip    int64
	Direction    db.SwapDirection
}

type swapFeesTotal struct {
	RuneAmount  int64
	AssetAmount int64
}

var SwapsAggregate = db.RegisterAggregate(db.NewAggregate("swaps", "swap_events").
	AddGroupColumn("pool").
	AddGroupColumn("_direction").
	AddSumlikeExpression("volume_e8",
		`SUM(CASE
			WHEN _direction%2 = 0 THEN from_e8
			WHEN _direction%2 = 1 THEN to_e8 + liq_fee_in_rune_e8
			ELSE 0 END)::BIGINT`).
	AddSumlikeExpression("swap_count", "COUNT(1)").
	// On swapping from asset to rune fees are collected in rune.
	AddSumlikeExpression("rune_fees_e8",
		"SUM(CASE WHEN _direction%2 = 1 THEN liq_fee_e8 ELSE 0 END)::BIGINT").
	// On swapping from rune to asset fees are collected in asset.
	AddSumlikeExpression("asset_fees_e8",
		"SUM(CASE WHEN _direction%2 = 0 THEN liq_fee_e8 ELSE 0 END)::BIGINT").
	AddBigintSumColumn("liq_fee_in_rune_e8").
	AddBigintSumColumn("swap_slip_bp"))

var TSSwapsAggregate = db.RegisterAggregate(db.NewAggregate("tsswaps", "swap_events").
	AddGroupColumn("pool").
	AddGroupColumn("_direction").
	AddSumlikeExpression("volume_e8",
		`SUM(CASE
			WHEN _direction%2 = 0 THEN from_e8
			WHEN _direction%2 = 1 THEN to_e8 + liq_fee_in_rune_e8
			ELSE 0 END)::BIGINT`).
	AddSumlikeExpression("swap_count", "COUNT(1)").
	// On swapping from asset to rune fees are collected in rune.
	AddSumlikeExpression("rune_fees_e8",
		"SUM(CASE WHEN _direction%2 = 1 THEN liq_fee_e8 ELSE 0 END)::BIGINT").
	// On swapping from rune to asset fees are collected in asset.
	AddSumlikeExpression("asset_fees_e8",
		"SUM(CASE WHEN _direction%2 = 0 THEN liq_fee_e8 ELSE 0 END)::BIGINT").
	AddBigintSumColumn("liq_fee_in_rune_e8").
	AddBigintSumColumn("swap_slip_bp"))

// Returns sparse buckets, when there are no swaps in the bucket, the bucket is missing.
// Returns several results for a given for all directions where a swap is present.
func GetSwapBuckets(ctx context.Context, pool *string, buckets db.Buckets) (
	[]OneDirectionSwapBucket, error) {
	filters := []string{}
	params := []interface{}{}
	if pool != nil {
		filters = append(filters, "pool = $1")
		params = append(params, *pool)
	}
	q, params := SwapsAggregate.BucketedQuery(`
		SELECT
			aggregate_timestamp/1000000000 as time,
			_direction,
			SUM(swap_count) AS count,
			SUM(volume_e8) AS volume,
			SUM(liq_fee_in_rune_e8) AS fee,
			SUM(swap_slip_bp) AS slip
		FROM %s
		GROUP BY _direction, time
		ORDER BY time ASC
	`, buckets, filters, params)

	rows, err := db.Query(ctx, q, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ret := []OneDirectionSwapBucket{}
	for rows.Next() {
		var bucket OneDirectionSwapBucket
		err := rows.Scan(&bucket.Time, &bucket.Direction, &bucket.Count, &bucket.VolumeInRune, &bucket.TotalFees, &bucket.TotalSlip)
		if err != nil {
			return []OneDirectionSwapBucket{}, err
		}
		ret = append(ret, bucket)
	}
	return ret, rows.Err()
}

func getSwapFees(ctx context.Context, pool string, from, to int64) (
	swapFeesTotal, error,
) {
	params := []interface{}{}
	params = append(params, pool)
	params = append(params, from*1000000000)
	if to > 0 {
		params = append(params, to*1000000000)
	}
	q := `
		SELECT
			SUM(rune_fees_e8),
			SUM(asset_fees_e8)
		FROM midgard_agg.swaps_5min
		WHERE pool=$1
		AND aggregate_timestamp>=$2
		`
	if to > 0 {
		q += `AND aggregate_timestamp<=$3`
	}

	rows, err := db.Query(ctx, q, params...)
	if err != nil {
		return swapFeesTotal{}, err
	}
	defer rows.Close()
	var swapFees swapFeesTotal
	if rows.Next() {
		var rune, asset *int64
		err := rows.Scan(&rune, &asset)
		if err != nil {
			return swapFeesTotal{}, err
		}
		if rune != nil {
			swapFees.RuneAmount = *rune
		}
		if asset != nil {
			swapFees.AssetAmount = *asset
		}
	}
	return swapFees, rows.Err()
}

func getRewards(ctx context.Context, pool string, from, to int64) (
	int64, error,
) {
	params := []interface{}{}
	params = append(params, pool)
	params = append(params, from*1000000000)
	if to > 0 {
		params = append(params, to*1000000000)
	}
	q := `
		SELECT
			SUM(rune_e8)
		FROM midgard_agg.rewards_event_entries_5min
		WHERE pool=$1
		AND aggregate_timestamp>=$2
		`
	if to > 0 {
		q += `AND aggregate_timestamp<=$3`
	}

	rows, err := db.Query(ctx, q, params...)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var rewards *int64
	if rows.Next() {

		err := rows.Scan(&rewards)
		if err != nil {
			return 0, err
		}
	}
	if rewards == nil {
		return 0, rows.Err()
	}
	return *rewards, rows.Err()
}

// TODO:Get swap target from input
func getTsSwapBuckets(ctx context.Context, pool *string, buckets db.Buckets) (
	[]OneDirectionSwapBucket, error,
) {
	filters := []string{}
	params := []interface{}{}
	if pool != nil {
		filters = append(filters, "pool = $1")
		params = append(params, *pool)
	}
	q, params := TSSwapsAggregate.BucketedQuery(`
		SELECT
			aggregate_timestamp/1000000000 as time,
			_direction,
			SUM(swap_count) AS count,
			SUM(volume_e8) AS volume,
			SUM(liq_fee_in_rune_e8) AS fee,
			SUM(swap_slip_bp) AS slip
		FROM %s
		GROUP BY _direction, time
		ORDER BY time ASC
	`, buckets, filters, params)

	rows, err := db.Query(ctx, q, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ret := []OneDirectionSwapBucket{}
	for rows.Next() {
		var bucket OneDirectionSwapBucket
		err := rows.Scan(&bucket.Time, &bucket.Direction, &bucket.Count, &bucket.VolumeInRune, &bucket.TotalFees, &bucket.TotalSlip)
		if err != nil {
			return []OneDirectionSwapBucket{}, err
		}
		ret = append(ret, bucket)
	}
	return ret, rows.Err()
}

// Returns gapfilled PoolSwaps for given pool, window and interval
func GetPoolSwaps(ctx context.Context, pool *string, buckets db.Buckets) ([]SwapBucket, error) {
	swaps, err := GetSwapBuckets(ctx, pool, buckets)
	if err != nil {
		return nil, err
	}
	usdPrice, err := USDPriceHistory(ctx, buckets)
	if err != nil {
		return nil, err
	}

	return mergeSwapsGapfill(swaps, usdPrice), nil
}

func GetPoolSwapsFee(ctx context.Context, pool string, from, to int64) (swapFeesTotal, error) {
	return getSwapFees(ctx, pool, from, to)
}

func GetPoolRewards(ctx context.Context, pool string, from, to int64) (int64, error) {
	return getRewards(ctx, pool, from, to)
}

// Returns gapfilled PoolSwaps routed via THORSwap for given pool, window and interval
func GetPoolTsSwaps(ctx context.Context, pool *string, buckets db.Buckets) ([]SwapBucket, error) {
	swaps, err := getTsSwapBuckets(ctx, pool, buckets)
	if err != nil {
		return nil, err
	}
	usdPrice, err := USDPriceHistory(ctx, buckets)
	if err != nil {
		return nil, err
	}

	return mergeSwapsGapfill(swaps, usdPrice), nil
}

func mergeSwapsGapfill(swaps []OneDirectionSwapBucket,
	denseUSDPrices []USDPriceBucket) []SwapBucket {
	ret := make([]SwapBucket, len(denseUSDPrices))

	timeAfterLast := denseUSDPrices[len(denseUSDPrices)-1].Window.Until + 1
	swaps = append(swaps, OneDirectionSwapBucket{Time: timeAfterLast})

	idx := 0
	for i, usdPrice := range denseUSDPrices {
		current := &ret[i]
		current.StartTime = usdPrice.Window.From
		current.EndTime = usdPrice.Window.Until
		for swaps[idx].Time == current.StartTime {
			swap := &swaps[idx]
			switch swap.Direction {
			case db.RuneToAsset:
				current.RuneToAssetCount += swap.Count
				current.RuneToAssetVolume += swap.VolumeInRune
				current.RuneToAssetFees += swap.TotalFees
				current.RuneToAssetSlip += swap.TotalSlip
			case db.AssetToRune:
				current.AssetToRuneCount += swap.Count
				current.AssetToRuneVolume += swap.VolumeInRune
				current.AssetToRuneFees += swap.TotalFees
				current.AssetToRuneSlip += swap.TotalSlip
			case db.RuneToSynth:
				current.RuneToSynthCount += swap.Count
				current.RuneToSynthVolume += swap.VolumeInRune
				current.RuneToSynthFees += swap.TotalFees
				current.RuneToSynthSlip += swap.TotalSlip
			case db.SynthToRune:
				current.SynthToRuneCount += swap.Count
				current.SynthToRuneVolume += swap.VolumeInRune
				current.SynthToRuneFees += swap.TotalFees
				current.SynthToRuneSlip += swap.TotalSlip
			}
			idx++
		}

		current.TotalCount = (current.RuneToAssetCount + current.AssetToRuneCount +
			current.RuneToSynthCount + current.SynthToRuneCount)
		current.TotalVolume = (current.RuneToAssetVolume + current.AssetToRuneVolume +
			current.RuneToSynthVolume + current.SynthToRuneVolume)
		current.TotalFees = (current.RuneToAssetFees + current.AssetToRuneFees +
			current.RuneToSynthFees + current.SynthToRuneFees)
		current.TotalSlip = (current.RuneToAssetSlip + current.AssetToRuneSlip +
			current.RuneToSynthSlip + current.SynthToRuneSlip)
		current.RunePriceUSD = usdPrice.RunePriceUSD
	}

	return ret
}

// Add the volume of swaps to poolVolumes.
// Note: only considers AssetToRune and RuneToAsset directions!
func addNonSynthVolumes(
	ctx context.Context,
	pools []string,
	w db.Window,
	poolVolumes *map[string]int64) error {
	bucket := db.OneIntervalBuckets(w.From, w.Until)
	wheres := []string{
		"_direction < 2",
		"pool = ANY($1)",
	}
	params := []interface{}{pools}

	q, params := SwapsAggregate.BucketedQuery(`
	SELECT
		pool,
		volume_e8 AS volume
	FROM %s
	`, bucket, wheres, params)

	swapRows, err := db.Query(ctx, q, params...)
	if err != nil {
		return err
	}
	defer swapRows.Close()

	for swapRows.Next() {
		var pool string
		var volume int64
		err := swapRows.Scan(&pool, &volume)
		if err != nil {
			return err
		}
		(*poolVolumes)[pool] += volume
	}
	return nil
}

// PoolsTotalVolume computes total volume amount for given timestamps (from/to) and pools
func PoolsTotalVolume(ctx context.Context, pools []string, w db.Window) (map[string]int64, error) {
	poolVolumes := make(map[string]int64)
	// TODO(muninn): fix for synths, decide if we should count mints and redeems in pools volume
	//   or maybe we should surface it under another field
	err := addNonSynthVolumes(ctx, pools, w, &poolVolumes)
	if err != nil {
		return nil, err
	}

	return poolVolumes, nil
}
