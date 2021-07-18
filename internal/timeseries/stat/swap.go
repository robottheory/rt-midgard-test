package stat

import (
	"context"
	"strconv"

	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/util/miderr"
)

// Swaps are generic swap statistics.
type Swaps struct {
	TxCount       int64
	RuneAddrCount int64 // Number of unique addresses involved.
	RuneE8Total   int64
}

// TODO(muninn): consider removing unique counts or making them approximations
func SwapsFromRuneLookup(ctx context.Context, w db.Window) (*Swaps, error) {
	// TODO(muninn): direction seems wrong, test and fix.
	const q = `SELECT COALESCE(COUNT(*), 0), COALESCE(COUNT(DISTINCT(from_addr)), 0), COALESCE(SUM(from_E8), 0)
        FROM swap_events
        WHERE mid_direction = 1 AND $1 <= block_timestamp AND block_timestamp < $2`

	return querySwaps(ctx, q, w.From.ToNano(), w.Until.ToNano())
}

// TODO(muninn): consider removing unique counts or making them approximations
func SwapsToRuneLookup(ctx context.Context, w db.Window) (*Swaps, error) {
	// TODO(muninn): direction seems wrong, test and fix.
	const q = `SELECT COALESCE(COUNT(*), 0), COALESCE(COUNT(DISTINCT(from_addr)), 0), COALESCE(SUM(to_E8), 0)
        FROM swap_events
        WHERE mid_direction = 0 AND $1 <= block_timestamp AND block_timestamp < $2`

	return querySwaps(ctx, q, w.From.ToNano(), w.Until.ToNano())
}

func querySwaps(ctx context.Context, q string, args ...interface{}) (*Swaps, error) {
	rows, err := db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var swaps Swaps
	if rows.Next() {
		err := rows.Scan(&swaps.TxCount, &swaps.RuneAddrCount, &swaps.RuneE8Total)
		if err != nil {
			return nil, err
		}
	}
	return &swaps, rows.Err()
}

func GetUniqueSwapperCount(ctx context.Context, pool string, window db.Window) (int64, error) {
	q := `
		SELECT
			COUNT(DISTINCT from_addr) AS unique
		FROM swap_events
		WHERE
			pool = $1
			AND block_timestamp >= $2 AND block_timestamp < $3`
	rows, err := db.Query(ctx, q, pool, window.From.ToNano(), window.Until.ToNano())
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	if !rows.Next() {
		return 0, miderr.InternalErrF("Failed to fetch uniqueSwaperCount")
	}
	var ret int64
	err = rows.Scan(&ret)
	return ret, err
}

type SwapBucket struct {
	StartTime         db.Second
	EndTime           db.Second
	RuneToAssetCount  int64
	AssetToRuneCount  int64
	TotalCount        int64
	RuneToAssetVolume int64
	AssetToRuneVolume int64
	TotalVolume       int64
	RuneToAssetFees   int64
	AssetToRuneFees   int64
	TotalFees         int64
	RuneToAssetSlip   int64
	AssetToRuneSlip   int64
	TotalSlip         int64
	RunePriceUSD      float64
}

func (meta *SwapBucket) AddBucket(bucket SwapBucket) {
	meta.RuneToAssetCount += bucket.RuneToAssetCount
	meta.AssetToRuneCount += bucket.AssetToRuneCount
	meta.TotalCount += bucket.TotalCount
	meta.RuneToAssetVolume += bucket.RuneToAssetVolume
	meta.AssetToRuneVolume += bucket.AssetToRuneVolume
	meta.TotalVolume += bucket.TotalVolume
	meta.RuneToAssetFees += bucket.RuneToAssetFees
	meta.AssetToRuneFees += bucket.AssetToRuneFees
	meta.TotalFees += bucket.TotalFees
	meta.RuneToAssetSlip += bucket.RuneToAssetSlip
	meta.AssetToRuneSlip += bucket.AssetToRuneSlip
	meta.TotalSlip += bucket.TotalSlip
}

type oneDirectionSwapBucket struct {
	Time         db.Second
	Count        int64
	VolumeInRune int64
	TotalFees    int64
	TotalSlip    int64
}

var SwapsAggregate = db.RegisterAggregate(db.NewAggregate("swaps", "swap_events").
	AddGroupColumn("pool").
	AddSumlikeExpression("volume_e8",
		`SUM(CASE
			WHEN mid_direction = 0 THEN from_e8
			WHEN mid_direction = 1 THEN to_e8 + liq_fee_in_rune_e8
			ELSE 0 END)::BIGINT`).
	AddSumlikeExpression("swap_count", "COUNT(1)").
	// On swapping from asset to rune fees are collected in rune.
	AddSumlikeExpression("rune_fees_e8",
		"SUM(CASE WHEN mid_direction = 1 THEN liq_fee_e8 ELSE 0 END)::BIGINT").
	// On swapping from rune to asset fees are collected in asset.
	AddSumlikeExpression("asset_fees_e8",
		"SUM(CASE WHEN mid_direction = 0 THEN liq_fee_e8 ELSE 0 END)::BIGINT").
	AddBigintSumColumn("liq_fee_in_rune_e8").
	AddBigintSumColumn("swap_slip_bp"))

// TODO(muninn): fix for synths, use direction selector int
func volumeSelector(direction db.SwapDirection) (volumeSelect, directionFilter string) {
	directionFilter = "mid_direction = " + strconv.Itoa(int(direction))
	switch direction {
	case db.RuneToAsset, db.RuneToSynth:
		volumeSelect = `COALESCE(SUM(from_E8), 0)`
	case db.AssetToRune, db.SynthToRune:
		volumeSelect = `COALESCE(SUM(to_e8), 0) + COALESCE(SUM(liq_fee_in_rune_e8), 0)`
	}
	return
}

// Returns sparse buckets, when there are no swaps in the bucket, the bucket is missing.
func getSwapBuckets(ctx context.Context, pool *string, buckets db.Buckets, direction db.SwapDirection) (
	[]oneDirectionSwapBucket, error) {
	queryArguments := []interface{}{buckets.Window().From.ToNano(), buckets.Window().Until.ToNano()}

	var poolFilter string
	if pool != nil {
		poolFilter = `swap.pool = $3`
		queryArguments = append(queryArguments, *pool)
	}

	volume, directionFilter := volumeSelector(direction)

	q := `
		SELECT
			` + db.SelectTruncatedTimestamp("swap.block_timestamp", buckets) + ` AS time,
			COALESCE(COUNT(*), 0) AS count,
			` + volume + ` AS volume,
			COALESCE(SUM(liq_fee_in_rune_E8), 0) AS fee,
			COALESCE(SUM(swap_slip_bp), 0) AS slip
		FROM swap_events AS swap
		` +
		db.Where(
			poolFilter,
			directionFilter, "block_timestamp >= $1 AND block_timestamp < $2") + `
		GROUP BY time
		ORDER BY time ASC`

	rows, err := db.Query(ctx, q, queryArguments...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ret := []oneDirectionSwapBucket{}
	for rows.Next() {
		var bucket oneDirectionSwapBucket
		err := rows.Scan(&bucket.Time, &bucket.Count, &bucket.VolumeInRune, &bucket.TotalFees, &bucket.TotalSlip)
		if err != nil {
			return []oneDirectionSwapBucket{}, err
		}
		ret = append(ret, bucket)
	}
	return ret, rows.Err()
}

// Returns gapfilled PoolSwaps for given pool, window and interval
func GetPoolSwaps(ctx context.Context, pool *string, buckets db.Buckets) ([]SwapBucket, error) {
	toAsset, err := getSwapBuckets(ctx, pool, buckets, db.RuneToAsset)
	if err != nil {
		return nil, err
	}

	toRune, err := getSwapBuckets(ctx, pool, buckets, db.AssetToRune)
	if err != nil {
		return nil, err
	}

	usdPrice, err := USDPriceHistory(ctx, buckets)
	if err != nil {
		return nil, err
	}

	return mergeSwapsGapfill(toAsset, toRune, usdPrice), nil
}

func mergeSwapsGapfill(
	sparseToAsset, sparseToRune []oneDirectionSwapBucket,
	denseUSDPrices []USDPriceBucket) []SwapBucket {
	ret := make([]SwapBucket, len(denseUSDPrices))

	timeAfterLast := denseUSDPrices[len(denseUSDPrices)-1].Window.Until + 1
	sparseToAsset = append(sparseToAsset, oneDirectionSwapBucket{Time: timeAfterLast})
	sparseToRune = append(sparseToRune, oneDirectionSwapBucket{Time: timeAfterLast})

	trIdx, taIdx := 0, 0
	for i, usdPrice := range denseUSDPrices {
		current := &ret[i]
		current.StartTime = usdPrice.Window.From
		current.EndTime = usdPrice.Window.Until
		ta := sparseToAsset[taIdx]
		tr := sparseToRune[trIdx]

		if current.StartTime == ta.Time {
			// We have swap to Asset in this bucket
			current.RuneToAssetCount = ta.Count
			current.RuneToAssetVolume = ta.VolumeInRune
			current.RuneToAssetFees = ta.TotalFees
			current.RuneToAssetSlip += ta.TotalSlip
			taIdx++
		}
		if current.StartTime == tr.Time {
			// We have swap to Rune in this bucket
			current.AssetToRuneCount = tr.Count
			current.AssetToRuneVolume = tr.VolumeInRune
			current.AssetToRuneFees += tr.TotalFees
			current.AssetToRuneSlip += tr.TotalSlip
			trIdx++
		}
		current.TotalCount = current.RuneToAssetCount + current.AssetToRuneCount
		current.TotalVolume = current.RuneToAssetVolume + current.AssetToRuneVolume
		current.TotalFees = current.RuneToAssetFees + current.AssetToRuneFees
		current.TotalSlip = current.RuneToAssetSlip + current.AssetToRuneSlip
		current.RunePriceUSD = usdPrice.RunePriceUSD
	}

	return ret
}

// Add the volume of one direction to poolVolumes.
func addVolumes(
	ctx context.Context,
	pools []string,
	w db.Window,
	direction db.SwapDirection,
	poolVolumes *map[string]int64) error {
	volume, directionFilter := volumeSelector(direction)
	q := `
	SELECT
		pool,
		` + volume + ` AS volume
	FROM swap_events
	` +
		db.Where(
			directionFilter,
			"pool = ANY($1)",
			"block_timestamp >= $2 AND block_timestamp < $3") + `
	GROUP BY pool
	`
	fromRuneRows, err := db.Query(ctx, q, pools, w.From.ToNano(), w.Until.ToNano())
	if err != nil {
		return err
	}
	defer fromRuneRows.Close()

	for fromRuneRows.Next() {
		var pool string
		var volume int64
		err := fromRuneRows.Scan(&pool, &volume)
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
	err := addVolumes(ctx, pools, w, db.RuneToAsset, &poolVolumes)
	if err != nil {
		return nil, err
	}
	err = addVolumes(ctx, pools, w, db.RuneToSynth, &poolVolumes)
	if err != nil {
		return nil, err
	}

	return poolVolumes, nil
}
