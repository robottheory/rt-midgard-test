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
        WHERE _direction = 1 AND $1 <= block_timestamp AND block_timestamp < $2`

	return querySwaps(ctx, q, w.From.ToNano(), w.Until.ToNano())
}

// TODO(muninn): consider removing unique counts or making them approximations
func SwapsToRuneLookup(ctx context.Context, w db.Window) (*Swaps, error) {
	// TODO(muninn): direction seems wrong, test and fix.
	const q = `SELECT COALESCE(COUNT(*), 0), COALESCE(COUNT(DISTINCT(from_addr)), 0), COALESCE(SUM(to_E8), 0)
        FROM swap_events
        WHERE _direction = 0 AND $1 <= block_timestamp AND block_timestamp < $2`

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
			WHEN _direction = 0 THEN from_e8
			WHEN _direction = 1 THEN to_e8 + liq_fee_in_rune_e8
			ELSE 0 END)::BIGINT`).
	AddSumlikeExpression("swap_count", "COUNT(1)").
	// On swapping from asset to rune fees are collected in rune.
	AddSumlikeExpression("rune_fees_e8",
		"SUM(CASE WHEN _direction = 1 THEN liq_fee_e8 ELSE 0 END)::BIGINT").
	// On swapping from rune to asset fees are collected in asset.
	AddSumlikeExpression("asset_fees_e8",
		"SUM(CASE WHEN _direction = 0 THEN liq_fee_e8 ELSE 0 END)::BIGINT").
	AddBigintSumColumn("liq_fee_in_rune_e8").
	AddBigintSumColumn("swap_slip_bp"))

func volumeSelector(direction db.SwapDirection) (volumeSelect, directionFilter string) {
	directionFilter = "_direction = " + strconv.Itoa(int(direction))
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
	runeToAsset, err := getSwapBuckets(ctx, pool, buckets, db.RuneToAsset)
	if err != nil {
		return nil, err
	}

	assetToRune, err := getSwapBuckets(ctx, pool, buckets, db.AssetToRune)
	if err != nil {
		return nil, err
	}

	runeToSynth, err := getSwapBuckets(ctx, pool, buckets, db.RuneToSynth)
	if err != nil {
		return nil, err
	}

	synthToRune, err := getSwapBuckets(ctx, pool, buckets, db.SynthToRune)
	if err != nil {
		return nil, err
	}

	usdPrice, err := USDPriceHistory(ctx, buckets)
	if err != nil {
		return nil, err
	}

	return mergeSwapsGapfill(runeToAsset, assetToRune, runeToSynth, synthToRune, usdPrice), nil
}

func mergeSwapsGapfill(
	sparseRuneToAsset, sparseAssetToRune, sparseRuneToSynth, sparseSynthToRune []oneDirectionSwapBucket,
	denseUSDPrices []USDPriceBucket) []SwapBucket {
	ret := make([]SwapBucket, len(denseUSDPrices))

	timeAfterLast := denseUSDPrices[len(denseUSDPrices)-1].Window.Until + 1
	sparseRuneToAsset = append(sparseRuneToAsset, oneDirectionSwapBucket{Time: timeAfterLast})
	sparseAssetToRune = append(sparseAssetToRune, oneDirectionSwapBucket{Time: timeAfterLast})
	sparseRuneToSynth = append(sparseRuneToSynth, oneDirectionSwapBucket{Time: timeAfterLast})
	sparseSynthToRune = append(sparseSynthToRune, oneDirectionSwapBucket{Time: timeAfterLast})

	rtaIdx, atrIdx, rtsIdx, strIdx := 0, 0, 0, 0
	for i, usdPrice := range denseUSDPrices {
		current := &ret[i]
		current.StartTime = usdPrice.Window.From
		current.EndTime = usdPrice.Window.Until
		rta := sparseRuneToAsset[rtaIdx]
		atr := sparseAssetToRune[atrIdx]
		rts := sparseRuneToSynth[rtsIdx]
		str := sparseSynthToRune[strIdx]

		if current.StartTime == rta.Time {
			// we have rune to asset swap in this bucket
			current.RuneToAssetCount = rta.Count
			current.RuneToAssetVolume = rta.VolumeInRune
			current.RuneToAssetFees = rta.TotalFees
			current.RuneToAssetSlip += rta.TotalSlip
			rtaIdx++
		}
		if current.StartTime == atr.Time {
			// We have asset to rune swap in this bucket
			current.AssetToRuneCount = atr.Count
			current.AssetToRuneVolume = atr.VolumeInRune
			current.AssetToRuneFees += atr.TotalFees
			current.AssetToRuneSlip += atr.TotalSlip
			atrIdx++
		}
		if current.StartTime == rts.Time {
			// We have rune to synth swap in this bucket
			current.RuneToSynthCount = rts.Count
			current.RuneToSynthVolume = rts.VolumeInRune
			current.RuneToSynthFees += rts.TotalFees
			current.RuneToSynthSlip += rts.TotalSlip
			rtsIdx++
		}
		if current.StartTime == str.Time {
			// We have rune to synth swap in this bucket
			current.SynthToRuneCount = str.Count
			current.SynthToRuneVolume = str.VolumeInRune
			current.SynthToRuneFees += str.TotalFees
			current.SynthToRuneSlip += str.TotalSlip
			strIdx++
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
	err = addVolumes(ctx, pools, w, db.AssetToRune, &poolVolumes)
	if err != nil {
		return nil, err
	}

	return poolVolumes, nil
}
