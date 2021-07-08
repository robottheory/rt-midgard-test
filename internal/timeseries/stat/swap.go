package stat

import (
	"context"

	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/util/miderr"
)

// Swaps are generic swap statistics.
type Swaps struct {
	TxCount       int64
	RuneAddrCount int64 // Number of unique addresses involved.
	RuneE8Total   int64
}

// TODO(acsaba): remove this, use PoolsTotalVolume.
func SwapsFromRuneLookup(ctx context.Context, w db.Window) (*Swaps, error) {
	const q = `SELECT COALESCE(COUNT(*), 0), COALESCE(COUNT(DISTINCT(from_addr)), 0), COALESCE(SUM(from_E8), 0)
        FROM swap_events
        WHERE pool = from_asset AND block_timestamp >= $1 AND block_timestamp <= $2`

	return querySwaps(ctx, q, w.From.ToNano(), w.Until.ToNano())
}

// TODO(acsaba): remove this, use PoolsTotalVolume.
func SwapsToRuneLookup(ctx context.Context, w db.Window) (*Swaps, error) {
	const q = `SELECT COALESCE(COUNT(*), 0), COALESCE(COUNT(DISTINCT(swap.from_addr)), 0), COALESCE(SUM(out.asset_E8), 0)
        FROM swap_events swap
	JOIN outbound_events out ON
		/* limit comparison set—no indinces */
		swap.block_timestamp <= out.block_timestamp AND
		swap.block_timestamp + 36000000000000 >= out.block_timestamp AND
		swap.tx = out.in_tx
        WHERE swap.block_timestamp >= $1 AND swap.block_timestamp <= $2 AND swap.pool <> swap.from_asset`

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
	StartTime     db.Second
	EndTime       db.Second
	ToAssetCount  int64
	ToRuneCount   int64
	TotalCount    int64
	ToAssetVolume int64
	ToRuneVolume  int64
	TotalVolume   int64
	ToAssetFees   int64
	ToRuneFees    int64
	TotalFees     int64
	ToAssetSlip   int64
	ToRuneSlip    int64
	TotalSlip     int64
	RunePriceUSD  float64
}

func (meta *SwapBucket) AddBucket(bucket SwapBucket) {
	meta.ToAssetCount += bucket.ToAssetCount
	meta.ToRuneCount += bucket.ToRuneCount
	meta.TotalCount += bucket.TotalCount
	meta.ToAssetVolume += bucket.ToAssetVolume
	meta.ToRuneVolume += bucket.ToRuneVolume
	meta.TotalVolume += bucket.TotalVolume
	meta.ToAssetFees += bucket.ToAssetFees
	meta.ToRuneFees += bucket.ToRuneFees
	meta.TotalFees += bucket.TotalFees
	meta.ToAssetSlip += bucket.ToAssetSlip
	meta.ToRuneSlip += bucket.ToRuneSlip
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
	// TODO(muninn): fix for synths, classify dirrection correctly. will be misclassified
	AddSumlikeExpression("volume_e8", "SUM(CASE WHEN from_asset = pool THEN to_e8 + liq_fee_in_rune_e8 ELSE from_e8 END)::BIGINT").
	AddSumlikeExpression("swap_count", "COUNT(1)").
	// TODO(muninn): fix for synths, classify dirrection correctly. will be misclassified
	AddSumlikeExpression("rune_fees_e8", "SUM(CASE WHEN from_asset = pool THEN liq_fee_e8 ELSE 0 END)::BIGINT").
	// TODO(muninn): fix for synths, classify dirrection correctly. will be misclassified
	AddSumlikeExpression("asset_fees_e8", "SUM(CASE WHEN from_asset = pool THEN 0 ELSE liq_fee_e8 END)::BIGINT").
	AddBigintSumColumn("liq_fee_in_rune_e8").
	AddBigintSumColumn("swap_slip_bp"))

// TODO(muninn): fix for synths, classify dirrection correctly. will be misclassified
func volumeSelector(swapToAsset bool) (volumeSelect, directionFilter string) {
	if swapToAsset {
		// from rune to asset
		volumeSelect = `COALESCE(SUM(from_E8), 0)`
		directionFilter = `from_asset <> pool`
	} else {
		// from asset to Rune
		volumeSelect = `COALESCE(SUM(to_e8), 0) + COALESCE(SUM(liq_fee_in_rune_e8), 0)`
		directionFilter = `from_asset = pool`
	}
	return
}

// Returns sparse buckets, when there are no swaps in the bucket, the bucket is missing.
func getSwapBuckets(ctx context.Context, pool *string, buckets db.Buckets, swapToAsset bool) (
	[]oneDirectionSwapBucket, error) {
	queryArguments := []interface{}{buckets.Window().From.ToNano(), buckets.Window().Until.ToNano()}

	var poolFilter string
	if pool != nil {
		poolFilter = `swap.pool = $3`
		queryArguments = append(queryArguments, *pool)
	}

	volume, directionFilter := volumeSelector(swapToAsset)

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
	toAsset, err := getSwapBuckets(ctx, pool, buckets, true)
	if err != nil {
		return nil, err
	}

	toRune, err := getSwapBuckets(ctx, pool, buckets, false)
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
			current.ToAssetCount = ta.Count
			current.ToAssetVolume = ta.VolumeInRune
			current.ToAssetFees = ta.TotalFees
			current.ToAssetSlip += ta.TotalSlip
			taIdx++
		}
		if current.StartTime == tr.Time {
			// We have swap to Rune in this bucket
			current.ToRuneCount = tr.Count
			current.ToRuneVolume = tr.VolumeInRune
			current.ToRuneFees += tr.TotalFees
			current.ToRuneSlip += tr.TotalSlip
			trIdx++
		}
		current.TotalCount = current.ToAssetCount + current.ToRuneCount
		current.TotalVolume = current.ToAssetVolume + current.ToRuneVolume
		current.TotalFees = current.ToAssetFees + current.ToRuneFees
		current.TotalSlip = current.ToAssetSlip + current.ToRuneSlip
		current.RunePriceUSD = usdPrice.RunePriceUSD
	}

	return ret
}

// Add the volume of one direction to poolVolumes.
func addVolumes(
	ctx context.Context,
	pools []string,
	w db.Window,
	swapToAsset bool,
	poolVolumes *map[string]int64) error {
	volume, directionFilter := volumeSelector(swapToAsset)
	q := `
	SELECT
		pool,
		` + volume + ` AS volume
	FROM swap_events
	` +
		db.Where(
			directionFilter,
			"pool = ANY($1)",
			"block_timestamp >= $2 AND block_timestamp <= $3") + `
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
	err := addVolumes(ctx, pools, w, false, &poolVolumes)
	if err != nil {
		return nil, err
	}
	err = addVolumes(ctx, pools, w, true, &poolVolumes)
	if err != nil {
		return nil, err
	}

	return poolVolumes, nil
}
