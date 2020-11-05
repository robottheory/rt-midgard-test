package stat

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"gitlab.com/thorchain/midgard/internal/graphql/model"
	"gitlab.com/thorchain/midgard/internal/timeseries"
)

// Swaps are generic swap statistics.
type Swaps struct {
	TxCount       int64
	RuneAddrCount int64 // Number of unique addresses involved.
	RuneE8Total   int64
}

func SwapsFromRuneLookup(ctx context.Context, w Window) (*Swaps, error) {
	const q = `SELECT COALESCE(COUNT(*), 0), COALESCE(COUNT(DISTINCT(from_addr)), 0), COALESCE(SUM(from_E8), 0)
        FROM swap_events
        WHERE pool = from_asset AND block_timestamp >= $1 AND block_timestamp <= $2`

	return querySwaps(ctx, q, w.From.UnixNano(), w.Until.UnixNano())
}

func SwapsToRuneLookup(ctx context.Context, w Window) (*Swaps, error) {
	const q = `SELECT COALESCE(COUNT(*), 0), COALESCE(COUNT(DISTINCT(swap.from_addr)), 0), COALESCE(SUM(out.asset_E8), 0)
        FROM swap_events swap
    JOIN outbound_events out ON
        /* limit comparison set—no indinces */
        swap.block_timestamp <= out.block_timestamp AND
        swap.block_timestamp + 36000000000000 >= out.block_timestamp AND
        swap.tx = out.in_tx
        WHERE swap.block_timestamp >= $1 AND swap.block_timestamp <= $2 AND swap.pool <> swap.from_asset`

	return querySwaps(ctx, q, w.From.UnixNano(), w.Until.UnixNano())
}

func querySwaps(ctx context.Context, q string, args ...interface{}) (*Swaps, error) {
	rows, err := DBQuery(ctx, q, args...)
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

// PoolSwaps are swap statistics for a specific asset.
// todo(donfrigo) remove unnecessary fields in order to use ToRune and FromRune instead
type PoolSwaps struct {
	TruncatedTime       time.Time
	TxCount             int64
	AssetE8Total        int64
	RuneE8Total         int64
	LiqFeeE8Total       int64
	LiqFeeInRuneE8Total int64
	TradeSlipBPTotal    int64
	ToRune              model.VolumeStats
	FromRune            model.VolumeStats
}

func PoolSwapsFromRuneLookup(ctx context.Context, pool string, w Window) (*PoolSwaps, error) {
	const q = `SELECT COALESCE(COUNT(*), 0), 0, COALESCE(SUM(from_E8), 0), COALESCE(SUM(liq_fee_E8), 0), COALESCE(SUM(liq_fee_in_rune_E8), 0), COALESCE(SUM(trade_slip_BP), 0), COALESCE(MIN(block_timestamp), 0), COALESCE(MAX(block_timestamp), 0), to_timestamp(0)
    FROM swap_events
    WHERE pool = $1 AND from_asset <> $1 AND block_timestamp >= $2 AND block_timestamp < $3`

	var swaps [1]PoolSwaps
	_, err := appendPoolSwaps(ctx, swaps[:0], q, false, pool, w.From.UnixNano(), w.Until.UnixNano())
	if err != nil || len(swaps) == 0 {
		return nil, err
	}
	return &swaps[0], nil
}

func PoolSwapsToRuneLookup(ctx context.Context, pool string, w Window) (*PoolSwaps, error) {
	const q = `SELECT COALESCE(COUNT(*), 0), COALESCE(SUM(from_E8), 0), 0, COALESCE(SUM(liq_fee_E8), 0), COALESCE(SUM(liq_fee_in_rune_E8), 0), COALESCE(SUM(trade_slip_BP), 0), COALESCE(MIN(block_timestamp), 0), COALESCE(MAX(block_timestamp), 0), to_timestamp(0)
    FROM swap_events
    WHERE pool = $1 AND from_asset = $1 AND block_timestamp >= $2 AND block_timestamp < $3`

	var swaps [1]PoolSwaps
	_, err := appendPoolSwaps(ctx, swaps[:0], q, false, pool, w.From.UnixNano(), w.Until.UnixNano())
	if err != nil || len(swaps) == 0 {
		return nil, err
	}
	return &swaps[0], nil
}

func PoolSwapsFromRuneBucketsLookup(ctx context.Context, pool string, bucketSize time.Duration, w Window) ([]PoolSwaps, error) {
	n, err := bucketsFor(bucketSize, w)
	if err != nil {
		return nil, err
	}
	a := make([]PoolSwaps, 0, n)

	const q = `SELECT COALESCE(COUNT(*), 0), 0, COALESCE(SUM(from_E8), 0), COALESCE(SUM(liq_fee_E8), 0), COALESCE(SUM(liq_fee_in_rune_E8), 0), COALESCE(SUM(trade_slip_BP), 0), COALESCE(MIN(block_timestamp), 0), COALESCE(MAX(block_timestamp), 0), to_timestamp(0)
    FROM swap_events
    WHERE pool = $1 AND from_asset <> $1 AND block_timestamp >= $2 AND block_timestamp < $3
    GROUP BY time_bucket($4, block_timestamp)
    ORDER BY time_bucket($4, block_timestamp)`

	return appendPoolSwaps(ctx, a, q, false, pool, w.From.UnixNano(), w.Until.UnixNano(), bucketSize)
}

func PoolSwapsToRuneBucketsLookup(ctx context.Context, pool string, bucketSize time.Duration, w Window) ([]PoolSwaps, error) {
	n, err := bucketsFor(bucketSize, w)
	if err != nil {
		return nil, err
	}
	a := make([]PoolSwaps, 0, n)

	const q = `SELECT COALESCE(COUNT(*), 0), COALESCE(SUM(from_E8), 0), 0, COALESCE(SUM(liq_fee_E8), 0), COALESCE(SUM(liq_fee_in_rune_E8), 0), COALESCE(SUM(trade_slip_BP), 0), COALESCE(MIN(block_timestamp), 0), COALESCE(MAX(block_timestamp), 0), to_timestamp(0)
    FROM swap_events
    WHERE pool = $1 AND from_asset = $1 AND block_timestamp >= $2 AND block_timestamp < $3
    GROUP BY time_bucket($4, block_timestamp)
    ORDER BY time_bucket($4, block_timestamp)`

	return appendPoolSwaps(ctx, a, q, false, pool, w.From.UnixNano(), w.Until.UnixNano(), bucketSize)
}

// GetIntervalFromString converts string to Interval.
func GetIntervalFromString(str string) (model.Interval, error) {
	switch str {
	case "5min":
		return model.IntervalMinute5, nil
	case "hour":
		return model.IntervalHour, nil
	case "day":
		return model.IntervalDay, nil
	case "week":
		return model.IntervalWeek, nil
	case "month":
		return model.IntervalMonth, nil
	case "quarter":
		return model.IntervalQuarter, nil
	case "year":
		return model.IntervalYear, nil
	}
	return "", errors.New("the requested interval is invalid: " + str)
}

// GetDurationFromInterval returns the the limited duration for given interval (duration = interval * limit)
func getDurationFromInterval(inv model.Interval) (time.Duration, error) {
	switch inv {
	case model.IntervalMinute5:
		return time.Minute * 5 * 101, nil
	case model.IntervalHour:
		return time.Hour * 101, nil
	case model.IntervalDay:
		return time.Hour * 24 * 31, nil
	case model.IntervalWeek:
		return time.Hour * 24 * 7 * 6, nil
	case model.IntervalMonth:
		return time.Hour * 24 * 31 * 3, nil
	case model.IntervalQuarter:
		return time.Hour * 24 * 122 * 3, nil
	case model.IntervalYear:
		return time.Hour * 24 * 365 * 3, nil
	}
	return time.Duration(0), errors.New(string("the requested interval is invalid: " + inv))
}

// Function that converts interval to a string necessary for the gapfill functionality in the SQL query
// 300E9 stands for 300*10^9 -> 5 minutes in nanoseconds and same logic for the rest of the entries
func getGapfillFromLimit(inv model.Interval) (string, error) {
	switch inv {
	case model.IntervalMinute5:
		return "300E9::BIGINT", nil // 5 minutes
	case model.IntervalHour:
		return "3600E9::BIGINT", nil // 1 hour
	case model.IntervalDay:
		return "864E11::BIGINT", nil // 24 hours
	// TODO(acsaba): Investigate if 7day boundaries ar not breaking logic.
	case model.IntervalWeek:
		return "604800E9::BIGINT", nil // 7 days
	case model.IntervalMonth:
		return "604800E9::BIGINT", nil // 7 days
	case model.IntervalQuarter:
		return "604800E9::BIGINT", nil // 7 days
	case model.IntervalYear:
		return "604800E9::BIGINT", nil // 7 days
	}
	return "", errors.New(string("the requested interval is invalid: " + inv))
}

// Function that converts interval to a string necessary for the time bucket functionality in the SQL query
func getBucketFromInterval(inv model.Interval) (string, error) {
	switch inv {
	case model.IntervalMinute5:
		return "5 min", nil
	case model.IntervalHour:
		return "1 hour", nil
	case model.IntervalDay:
		return "1 day", nil
	case model.IntervalWeek:
		return "1 day", nil
	case model.IntervalMonth:
		return "1 day", nil
	case model.IntervalQuarter:
		return "1 day", nil
	case model.IntervalYear:
		return "1 day", nil
	}
	return "", errors.New(string("the requested interval is invalid: " + inv))
}

// Function to get asset volumes from all (*) or  given pool, for given asset with given interval
func PoolSwapsLookup(ctx context.Context, pool string, interval model.Interval, w Window, swapToRune bool) ([]PoolSwaps, error) {
	var q, poolQuery string
	if pool != "*" {
		poolQuery = fmt.Sprintf(`swap.pool = '%s' AND`, pool)
	}
	bucket, err := getBucketFromInterval(interval)
	if err != nil {
		return nil, err
	}

	// If conversion is true then it assumes that the query selects to the flowing fields in addition: TruncatedTime, volumeInRune
	if swapToRune {
		q = fmt.Sprintf(
			`
            SELECT
                COALESCE(COUNT(*), 0),
                COALESCE(SUM(from_E8), 0),
                COALESCE(CAST(SUM(CAST(rune_e8 as NUMERIC) / CAST(asset_e8 as NUMERIC) * swap.from_e8) as bigint), 0) as rune_volume,
                COALESCE(SUM(liq_fee_E8), 0),
                COALESCE(SUM(liq_fee_in_rune_E8), 0),
                COALESCE(SUM(trade_slip_BP), 0),
                COALESCE(MIN(swap.block_timestamp), 0),
                COALESCE(MAX(swap.block_timestamp), 0),
                time_bucket('%s',date_trunc($3, to_timestamp(swap.block_timestamp/1000000000))) AS bucket
            FROM swap_events AS swap
            LEFT JOIN LATERAL (
                SELECT
                    depths.asset_e8,
                    depths.rune_e8
                FROM block_pool_depths as depths
                WHERE
                    depths.block_timestamp <= swap.block_timestamp AND swap.pool = depths.pool
                ORDER  BY depths.block_timestamp DESC
                LIMIT  1
            ) AS joined on TRUE
            WHERE
                %s swap.from_asset = swap.pool
                AND $1 <= swap.block_timestamp AND swap.block_timestamp <= $2
            GROUP BY bucket
            ORDER BY bucket ASC`,
			bucket, poolQuery)
	} else {
		q = fmt.Sprintf(`
            SELECT
                COALESCE(COUNT(*), 0) as count,
                0,
                COALESCE(SUM(from_E8), 0) as from_E8,
                COALESCE(SUM(liq_fee_E8), 0) as liq_fee_E8,
                COALESCE(SUM(liq_fee_in_rune_E8), 0) as liq_fee_in_rune_E8,
                COALESCE(SUM(trade_slip_BP), 0) as trade_slip_BP,
                COALESCE(MIN(swap.block_timestamp), 0) as min,
                COALESCE(MAX(swap.block_timestamp), 0) as max,
                time_bucket('%s', date_trunc($3, to_timestamp(swap.block_timestamp/1000000000))) AS bucket
            FROM swap_events as swap
            WHERE %s from_asset <> pool AND block_timestamp >= $1 AND block_timestamp < $2
            GROUP BY bucket
            ORDER BY bucket ASC`,
			bucket, poolQuery)
	}

	return appendPoolSwaps(ctx, []PoolSwaps{}, q, swapToRune, w.From.UnixNano(), w.Until.UnixNano(), interval)
}

// Fill from or until if it's missing. Limits if it's too long for the interval.
func calcBounds(w Window, inv model.Interval) (Window, error) {
	duration, err := getDurationFromInterval(inv)
	if err != nil {
		return Window{}, err
	}

	if w.From.Unix() != 0 && w.Until.Unix() == 0 {
		// if only since is defined
		limitedTime := w.From.Add(duration)
		w.Until = limitedTime
	} else if w.From.Unix() == 0 && w.Until.Unix() != 0 {
		// if only until is defined
		limitedTime := w.Until.Add(-duration)
		w.From = limitedTime
	} else if w.From.Unix() == 0 && w.Until.Unix() == 0 {
		// if neither is defined
		w.Until = time.Now()
	}

	// if the starting time lies outside the limit
	limitedTime := w.Until.Add(-duration)
	if limitedTime.After(w.From) {
		// limit the value
		w.From = limitedTime
	}

	return w, nil
}

func appendPoolSwaps(ctx context.Context, swaps []PoolSwaps, q string, swapToRune bool, args ...interface{}) ([]PoolSwaps, error) {
	rows, err := DBQuery(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var r PoolSwaps
		var first, last int64
		if swapToRune {
			if err := rows.Scan(&r.TxCount, &r.AssetE8Total, &r.RuneE8Total, &r.LiqFeeE8Total, &r.LiqFeeInRuneE8Total, &r.TradeSlipBPTotal, &first, &last, &r.TruncatedTime); err != nil {
				return swaps, err
			}
			r.ToRune = model.VolumeStats{
				Count:        r.TxCount,
				VolumeInRune: r.RuneE8Total,
				FeesInRune:   r.LiqFeeInRuneE8Total,
			}
		} else {
			if err := rows.Scan(&r.TxCount, &r.AssetE8Total, &r.RuneE8Total, &r.LiqFeeE8Total, &r.LiqFeeInRuneE8Total, &r.TradeSlipBPTotal, &first, &last, &r.TruncatedTime); err != nil {
				return swaps, err
			}
			r.FromRune = model.VolumeStats{
				Count:        r.TxCount,
				VolumeInRune: r.RuneE8Total,
				FeesInRune:   r.LiqFeeInRuneE8Total,
			}
		}
		swaps = append(swaps, r)
	}
	return swaps, rows.Err()
}

// struct returned from v1/history/total_volume endpoint
type SwapVolumeChanges struct {
	BuyVolume   string `json:"buyVolume"`   // volume RUNE bought in given a timeframe
	SellVolume  string `json:"sellVolume"`  // volume of RUNE sold in given a timeframe
	Time        int64  `json:"time"`        // beginning of the timeframe
	TotalVolume string `json:"totalVolume"` // sum of bought and sold volume
}

func TotalVolumeChanges(ctx context.Context, inv, pool string, from, to time.Time) ([]SwapVolumeChanges, error) {
	interval, err := GetIntervalFromString(inv)
	if err != nil {
		return nil, err
	}
	window := Window{
		From:  from,
		Until: to,
	}

	timestamps, fromRune, fromAsset, err := GetPoolSwaps(ctx, pool, window, interval)
	if err != nil {
		return nil, err
	}

	result, err := createSwapVolumeChanges(timestamps, fromRune, fromAsset)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func GetPoolSwaps(ctx context.Context, pool string, window Window, interval model.Interval) ([]int64, []PoolSwaps, []PoolSwaps, error) {
	timestamps, err := generateBuckets(ctx, interval, window)
	if err != nil {
		return nil, nil, nil, err
	}
	if 0 == len(timestamps) {
		return nil, nil, nil, errors.New("no buckets were generated for given timeframe")
	}

	// fromRune stores conversion from Rune to Asset -> selling Rune
	fromRune, err := PoolSwapsLookup(ctx, pool, interval, window, false)
	if err != nil {
		return nil, nil, nil, err
	}

	// fromAsset stores conversion from Asset to Rune -> buying Rune
	fromAsset, err := PoolSwapsLookup(ctx, pool, interval, window, true)
	if err != nil {
		return nil, nil, nil, err
	}

	return timestamps, fromRune, fromAsset, nil
}

func createSwapVolumeChanges(timestamps []int64, fromRune, fromAsset []PoolSwaps) ([]SwapVolumeChanges, error) {
	result := make([]SwapVolumeChanges, 0)

	mergedPoolSwaps, err := MergeSwaps(timestamps, fromRune, fromAsset)
	if err != nil {
		return nil, err
	}

	for _, ps := range mergedPoolSwaps {
		timestamp := ps.TruncatedTime.Unix()
		fr := ps.FromRune
		tr := ps.ToRune

		runeSellVolume := strconv.FormatInt(fr.VolumeInRune, 10)
		runeBuyVolume := strconv.FormatInt(tr.VolumeInRune, 10)
		totalVolume := strconv.FormatInt(fr.VolumeInRune+tr.VolumeInRune, 10)

		svc := SwapVolumeChanges{
			BuyVolume:   runeBuyVolume,
			SellVolume:  runeSellVolume,
			Time:        timestamp,
			TotalVolume: totalVolume,
		}

		result = append(result, svc)
	}
	return result, nil
}

func MergeSwaps(timestamps []int64, fromRune, fromAsset []PoolSwaps) ([]PoolSwaps, error) {
	mergedArray := make([]PoolSwaps, 0)
	result := make([]PoolSwaps, len(timestamps))

	if len(fromRune) == 0 {
		fromRune = append(fromRune, PoolSwaps{TruncatedTime: time.Now()})
	}

	if len(fromAsset) == 0 {
		fromAsset = append(fromAsset, PoolSwaps{TruncatedTime: time.Now()})
	}

	for i, j := 0, 0; i < len(fromRune) && j < len(fromAsset); {
		// selling Rune -> volume is already in Rune
		fr := fromRune[i]
		// buying Rune -> volume is calculated from asset volume and exchange rate
		fa := fromAsset[j]

		if fr.TruncatedTime.Before(fa.TruncatedTime) {
			result = append(result, fr)
			i++
		} else if fa.TruncatedTime.Before(fr.TruncatedTime) {
			result = append(result, fa)
			j++
		} else if fr.TruncatedTime.Equal(fa.TruncatedTime) {
			toRuneStats := model.VolumeStats{
				Count:        fa.ToRune.Count,
				VolumeInRune: fa.ToRune.VolumeInRune,
				FeesInRune:   fa.ToRune.FeesInRune,
			}

			fromRuneStats := model.VolumeStats{
				Count:        fr.FromRune.Count,
				VolumeInRune: fr.FromRune.VolumeInRune,
				FeesInRune:   fr.FromRune.FeesInRune,
			}

			ps := PoolSwaps{
				TruncatedTime: fr.TruncatedTime,
				FromRune:      fromRuneStats,
				ToRune:        toRuneStats,
			}

			mergedArray = append(mergedArray, ps)
			i++
			j++
		} else {
			return result, errors.New("error occurred while merging arrays")
		}
	}

	if len(mergedArray) == 0 {
		for i, ts := range timestamps {
			ps := PoolSwaps{
				TruncatedTime: time.Unix(ts, 0),
				ToRune:        model.VolumeStats{},
				FromRune:      model.VolumeStats{},
			}
			result[i] = ps
		}
		return result, nil
	}

	maCounter := 0
	for i, ts := range timestamps {
		maResult := mergedArray[maCounter]
		prTimestamp := maResult.TruncatedTime

		ps := PoolSwaps{
			TruncatedTime: time.Unix(ts, 0),
			ToRune:        model.VolumeStats{},
			FromRune:      model.VolumeStats{},
		}

		if prTimestamp.Unix() == ts {
			fr := maResult.FromRune
			tr := maResult.ToRune

			ps = PoolSwaps{
				ToRune:        tr,
				FromRune:      fr,
				TruncatedTime: prTimestamp,
			}
			if maCounter < len(mergedArray)-1 {
				maCounter++
			}
		}
		result[i] = ps
	}

	return result, nil
}

// PoolTotalVolume computes total volume amount for given timestamps (from/to) and pool
func PoolTotalVolume(ctx context.Context, pool string, from, to time.Time) (int64, error) {
	toRuneVolumeQ := `SELECT
        COALESCE(CAST(SUM(CAST(rune_e8 as NUMERIC) / CAST(asset_e8 as NUMERIC) * swap.from_e8) as bigint), 0)
        FROM swap_events AS swap
            LEFT JOIN LATERAL (
                SELECT depths.asset_e8, depths.rune_e8
                    FROM block_pool_depths as depths
                WHERE
                depths.block_timestamp <= swap.block_timestamp AND swap.pool = depths.pool
                ORDER  BY depths.block_timestamp DESC
                LIMIT  1
            ) AS joined on TRUE
        WHERE swap.from_asset = swap.pool AND swap.pool = $1 AND swap.block_timestamp >= $2 AND swap.block_timestamp <= $3
    `
	var toRuneVolume int64
	err := timeseries.QueryOneValue(&toRuneVolume, ctx, toRuneVolumeQ, pool, from.UnixNano(), to.UnixNano())
	if err != nil {
		return 0, err
	}

	fromRuneVolumeQ := `SELECT COALESCE(SUM(from_e8), 0)
    FROM swap_events
    WHERE from_asset <> pool AND pool = $1 AND block_timestamp >= $2 AND block_timestamp <= $3
    `
	var fromRuneVolume int64
	err = timeseries.QueryOneValue(&fromRuneVolume, ctx, fromRuneVolumeQ, pool, from.UnixNano(), to.UnixNano())
	if err != nil {
		return 0, err
	}

	return toRuneVolume + fromRuneVolume, nil
}
