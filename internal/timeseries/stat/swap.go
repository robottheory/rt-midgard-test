package stat

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"gitlab.com/thorchain/midgard/internal/graphql/model"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
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
// TODO(donfrigo) remove unnecessary fields in order to use ToRune and FromRune instead
type PoolSwaps struct {
	// TODO(acsaba): change time to int64 unix sec
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

const maxIntervalCount = 101

// We want to limit the respons intervals, but we want to restrict the
// Database lookup range too so we don't do all the work unnecessarily.
func getMaxDuration(inv model.Interval) (time.Duration, error) {
	switch inv {
	case model.IntervalMinute5:
		return time.Minute * 5 * maxIntervalCount, nil
	case model.IntervalHour:
		return time.Hour * maxIntervalCount, nil
	case model.IntervalDay:
		return time.Hour * 24 * maxIntervalCount, nil
	case model.IntervalWeek:
		return time.Hour * 24 * 7 * maxIntervalCount, nil
	case model.IntervalMonth:
		return time.Hour * 24 * 31 * maxIntervalCount, nil
	case model.IntervalQuarter:
		return time.Hour * 24 * 31 * 3 * maxIntervalCount, nil
	case model.IntervalYear:
		return time.Hour * 24 * 365 * maxIntervalCount, nil
	}
	return time.Duration(0), errors.New(string("the requested interval is invalid: " + inv))
}

// A reasonable period for gapfil which guaranties that date_trunc will
// create all the needed entries.
func reasonableGapfillParam(inv model.Interval) (string, error) {
	switch inv {
	case model.IntervalMinute5:
		return "300::BIGINT", nil // 5 minutes
	case model.IntervalHour:
		return "3600::BIGINT", nil // 1 hour
	case model.IntervalDay:
		return "86400::BIGINT", nil // 24 hours
	case model.IntervalWeek:
		return "604800::BIGINT", nil // 7 days
	case model.IntervalMonth:
		return "2160000::BIGINT", nil // 25 days
	case model.IntervalQuarter:
		return "7344000::BIGINT", nil // 85 days
	case model.IntervalYear:
		return "25920000::BIGINT", nil // 300 days
	}
	return "", errors.New(string("the requested interval is invalid: " + inv))
}

// Function to get asset volumes from all (*) or  given pool, for given asset with given interval
func getPoolSwapsSparse(ctx context.Context, pool string, interval model.Interval, w Window, swapToRune bool) ([]PoolSwaps, error) {
	var q, poolQuery string
	if pool != "*" {
		poolQuery = fmt.Sprintf(`swap.pool = '%s' AND`, pool)
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
				date_trunc($3, to_timestamp(swap.block_timestamp/1000000000)) AS truncated
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
			GROUP BY truncated
			ORDER BY truncated ASC`,
			poolQuery)
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
                date_trunc($3, to_timestamp(swap.block_timestamp/1000000000)) AS truncated
            FROM swap_events as swap
            WHERE %s from_asset <> pool AND block_timestamp >= $1 AND block_timestamp < $2
            GROUP BY truncated
            ORDER BY truncated ASC`,
			poolQuery)
	}

	return appendPoolSwaps(ctx, []PoolSwaps{}, q, swapToRune, w.From.UnixNano(), w.Until.UnixNano(), interval)
}

// Fill from or until if it's missing. Limits if it's too long for the interval.
func calcBounds(w Window, inv model.Interval) (Window, error) {
	maxDuration, err := getMaxDuration(inv)
	if err != nil {
		return Window{}, err
	}

	if w.From.Unix() != 0 && w.Until.Unix() == 0 {
		// if only since is defined
		limitedTime := w.From.Add(maxDuration)
		w.Until = limitedTime
	} else if w.From.Unix() == 0 && w.Until.Unix() != 0 {
		// if only until is defined
		limitedTime := w.Until.Add(-maxDuration)
		w.From = limitedTime
	} else if w.From.Unix() == 0 && w.Until.Unix() == 0 {
		// if neither is defined
		w.Until = time.Now()
	}

	// if the starting time lies outside the limit
	limitedTime := w.Until.Add(-maxDuration)
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

func VolumeHistory(
	ctx context.Context,
	intervalStr string,
	pool string,
	from, to time.Time) (oapigen.SwapHistoryResponse, error) {

	interval, err := GetIntervalFromString(intervalStr)
	if err != nil {
		return oapigen.SwapHistoryResponse{}, err
	}
	window := Window{
		From:  from,
		Until: to,
	}

	mergedPoolSwaps, err := GetPoolSwaps(ctx, pool, window, interval)
	if err != nil {
		return oapigen.SwapHistoryResponse{}, err
	}

	return createVolumeIntervals(mergedPoolSwaps), nil
}

// Returns gapfilled PoolSwaps for given pool, window and interval
func GetPoolSwaps(ctx context.Context, pool string, window Window, interval model.Interval) ([]PoolSwaps, error) {
	timestamps, window, err := generateBuckets(ctx, interval, window)
	if err != nil {
		return nil, err
	}
	if 0 == len(timestamps) {
		return nil, errors.New("no buckets were generated for given timeframe")
	}

	// fromRune stores conversion from Rune to Asset -> selling Rune
	fromRune, err := getPoolSwapsSparse(ctx, pool, interval, window, false)
	if err != nil {
		return nil, err
	}

	// fromAsset stores conversion from Asset to Rune -> buying Rune
	fromAsset, err := getPoolSwapsSparse(ctx, pool, interval, window, true)
	if err != nil {
		return nil, err
	}

	// merges fromRune and fromAsset and also adds gapfill
	mergedPoolSwaps, err := mergeSwapsGapfill(timestamps, fromRune, fromAsset)
	if err != nil {
		return nil, err
	}

	return mergedPoolSwaps, nil
}

func intStr(v int64) string {
	return strconv.FormatInt(v, 10)
}

func createVolumeIntervals(mergedPoolSwaps []PoolSwaps) (result oapigen.SwapHistoryResponse) {
	var metaToAsset, metaToRune int64

	for _, ps := range mergedPoolSwaps {
		timestamp := ps.TruncatedTime.Unix()
		fr := ps.FromRune
		tr := ps.ToRune

		toAssetVolume := fr.VolumeInRune
		toRuneVolume := tr.VolumeInRune
		totalVolume := fr.VolumeInRune + tr.VolumeInRune

		metaToAsset += toAssetVolume
		metaToRune += toRuneVolume

		svc := oapigen.SwapHistoryInterval{
			ToRuneVolume:  intStr(toRuneVolume),
			ToAssetVolume: intStr(toAssetVolume),
			Time:          intStr(timestamp),
			TotalVolume:   intStr(totalVolume),
		}

		result.Intervals = append(result.Intervals, svc)
	}

	meta := &result.Meta
	meta.ToAssetVolume = intStr(metaToAsset)
	meta.ToRuneVolume = intStr(metaToRune)
	meta.TotalVolume = intStr(metaToAsset + metaToRune)
	meta.FirstTime = result.Intervals[0].Time
	meta.LastTime = result.Intervals[len(result.Intervals)-1].Time

	return
}

func mergeSwapsGapfill(timestamps []int64, fromRune, fromAsset []PoolSwaps) ([]PoolSwaps, error) {
	gapfilledPoolSwaps := make([]PoolSwaps, len(timestamps))

	timeAfterLast := time.Unix(timestamps[len(timestamps)-1]+1, 0)
	if len(fromRune) == 0 {
		fromRune = append(fromRune, PoolSwaps{TruncatedTime: timeAfterLast})
	}

	if len(fromAsset) == 0 {
		fromAsset = append(fromAsset, PoolSwaps{TruncatedTime: timeAfterLast})
	}

	for i, j, k := 0, 0, 0; k < len(timestamps); {
		// selling Rune -> volume is already in Rune
		fr := fromRune[i]
		// buying Rune -> volume is calculated from asset volume and exchange rate
		fa := fromAsset[j]
		ts := timestamps[k]
		faTime := fa.TruncatedTime.Unix()
		frTime := fr.TruncatedTime.Unix()

		if ts == frTime && ts == faTime {
			// both match the timestamp
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

			gapfilledPoolSwaps[k] = ps
			i++
			j++
		} else if ts == frTime && ts != faTime {
			gapfilledPoolSwaps[k] = fr
			i++
		} else if ts != frTime && ts == faTime {
			gapfilledPoolSwaps[k] = fa
			j++
		} else if ts != frTime && ts != faTime {
			// none match the timestamp
			gapfilledPoolSwaps[k] = PoolSwaps{
				TruncatedTime: time.Unix(ts, 0),
				ToRune:        model.VolumeStats{},
				FromRune:      model.VolumeStats{},
			}
		} else {
			return gapfilledPoolSwaps, errors.New("error occurred while merging arrays")
		}

		k++
	}

	return gapfilledPoolSwaps, nil
}

// PoolsTotalVolume computes total volume amount for given timestamps (from/to) and pools
func PoolsTotalVolume(ctx context.Context, pools []string, from, to time.Time) (map[string]int64, error) {
	toRuneVolumeQ := `SELECT pool,
		COALESCE(CAST(SUM(CAST(rune_e8 as NUMERIC) / CAST(asset_e8 as NUMERIC) * swap.from_e8) as bigint), 0)
		FROM swap_events AS swap
			LEFT JOIN LATERAL (
				SELECT depths.asset_e8, depths.rune_e8
					FROM block_pool_depths as depths
				WHERE
				depths.block_timestamp <= swap.block_timestamp AND swap.pool = depths.pool
				ORDER BY depths.block_timestamp DESC
				LIMIT 1
			) AS joined on TRUE
		WHERE swap.from_asset = swap.pool AND swap.pool = ANY($1) AND swap.block_timestamp >= $2 AND swap.block_timestamp <= $3
		GROUP BY pool
	`
	toRuneRows, err := DBQuery(ctx, toRuneVolumeQ, pools, from.UnixNano(), to.UnixNano())
	if err != nil {
		return nil, err
	}
	defer toRuneRows.Close()

	poolVolumes := make(map[string]int64)
	for toRuneRows.Next() {
		var pool string
		var volume int64
		err := toRuneRows.Scan(&pool, &volume)
		if err != nil {
			return nil, err
		}
		poolVolumes[pool] = volume
	}

	fromRuneVolumeQ := `SELECT pool, COALESCE(SUM(from_e8), 0)
	FROM swap_events
	WHERE from_asset <> pool AND pool = ANY($1) AND block_timestamp >= $2 AND block_timestamp <= $3
	GROUP BY pool
	`
	fromRuneRows, err := DBQuery(ctx, fromRuneVolumeQ, pools, from.UnixNano(), to.UnixNano())
	if err != nil {
		return nil, err
	}
	defer fromRuneRows.Close()
	for fromRuneRows.Next() {
		var pool string
		var volume int64
		err := fromRuneRows.Scan(&pool, &volume)
		if err != nil {
			return nil, err
		}
		poolVolumes[pool] = poolVolumes[pool] + volume
	}

	return poolVolumes, nil
}
