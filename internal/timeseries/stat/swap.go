package stat

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"
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

	return querySwaps(ctx, q, w.Since.UnixNano(), w.Until.UnixNano())
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

	return querySwaps(ctx, q, w.Since.UnixNano(), w.Until.UnixNano())
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
type PoolSwaps struct {
	First               time.Time
	Last                time.Time
	TruncatedTime       time.Time
	TxCount             int64
	AssetE8Total        int64
	RuneE8Total         int64
	LiqFeeE8Total       int64
	LiqFeeInRuneE8Total int64
	TradeSlipBPTotal    int64
	VolumeInRune        float64
}

func PoolSwapsFromRuneLookup(ctx context.Context, pool string, w Window) (*PoolSwaps, error) {
	const q = `SELECT COALESCE(COUNT(*), 0), 0, COALESCE(SUM(from_E8), 0), COALESCE(SUM(liq_fee_E8), 0), COALESCE(SUM(liq_fee_in_rune_E8), 0), COALESCE(SUM(trade_slip_BP), 0), COALESCE(MIN(block_timestamp), 0), COALESCE(MAX(block_timestamp), 0), to_timestamp(0)
	FROM swap_events
	WHERE pool = $1 AND from_asset <> $1 AND block_timestamp >= $2 AND block_timestamp < $3`

	var swaps [1]PoolSwaps
	_, err := appendPoolSwaps(ctx, swaps[:0], q, false, pool, w.Since.UnixNano(), w.Until.UnixNano())
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
	_, err := appendPoolSwaps(ctx, swaps[:0], q, false, pool, w.Since.UnixNano(), w.Until.UnixNano())
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

	return appendPoolSwaps(ctx, a, q, false, pool, w.Since.UnixNano(), w.Until.UnixNano(), bucketSize)
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

	return appendPoolSwaps(ctx, a, q, false, pool, w.Since.UnixNano(), w.Until.UnixNano(), bucketSize)
}
type Interval string

// SQL parameters for date_trunc
const(
	fiveMin Interval = "minute"
	hour             = "hour"
	day              = "day"
	week             = "week"
	month            = "month"
	quarter          = "quarter"
	year             = "year"
)

// GetIntervalFromString converts string to Interval.
func GetIntervalFromString(str string) (Interval, error) {
	switch str {
	case "5min":
		return fiveMin, nil
	case "hour":
		return hour, nil
	case "day":
		return day, nil
	case "week":
		return week , nil
	case "month":
		return month, nil
	case "quarter":
		return quarter, nil
	case "year":
		return year, nil
	}
	return "", errors.New("the requested interval is invalid: " + str)
}

// Function to get asset volumes from all (*) or  given pool, for given asset with given interval
func PoolSwapsLookup(ctx context.Context, pool string, interval Interval, w Window, limit bool, convertToRune bool) ([]PoolSwaps, error) {
	var q, poolQuery string
	if pool != "*" {
		poolQuery = fmt.Sprintf(`swap.pool = '%s' AND`, pool)
	}

	// If conversion is true then it assumes that the query selects to the flowing fields in addition: TruncatedTime, volumeInRune
	if convertToRune {
		q = fmt.Sprintf(`SELECT COALESCE(COUNT(*), 0), COALESCE(SUM(from_E8), 0), 0, COALESCE(SUM(liq_fee_E8), 0), COALESCE(SUM(liq_fee_in_rune_E8), 0), COALESCE(SUM(trade_slip_BP), 0), COALESCE(MIN(swap.block_timestamp), 0), COALESCE(MAX(swap.block_timestamp), 0), time_bucket('5 min', date_trunc($3, to_timestamp(swap.block_timestamp/1000000000))) AS bucket, COALESCE(SUM(CAST(rune_e8 as NUMERIC) / CAST(asset_e8 as NUMERIC) * swap.from_e8)) as rune_volume
			FROM swap_events AS swap
			LEFT JOIN LATERAL (
				SELECT depths.asset_e8, depths.rune_e8
					FROM block_pool_depths as depths
				WHERE
				depths.block_timestamp <= swap.block_timestamp AND swap.pool = depths.pool
				ORDER  BY depths.block_timestamp DESC
				LIMIT  1
			) AS joined on TRUE
			WHERE %s swap.from_asset <> 'BNB.RUNE-B1A' AND swap.block_timestamp >= $1 AND swap.block_timestamp <= $2
			GROUP BY bucket
			ORDER BY bucket ASC`, poolQuery)
	} else {
		q = fmt.Sprintf(`SELECT COALESCE(COUNT(*), 0), 0, COALESCE(SUM(from_E8), 0), COALESCE(SUM(liq_fee_E8), 0), COALESCE(SUM(liq_fee_in_rune_E8), 0), COALESCE(SUM(trade_slip_BP), 0), COALESCE(MIN(block_timestamp), 0), COALESCE(MAX(block_timestamp), 0), time_bucket('5 min', date_trunc($3, to_timestamp(block_timestamp/1000000000))) AS bucket
			FROM swap_events as swap
			WHERE %s from_asset = 'BNB.RUNE-B1A' AND block_timestamp >= $1 AND block_timestamp <= $2
			GROUP BY bucket
			ORDER BY bucket ASC`, poolQuery)
	}

	if limit {
		q = q + ` LIMIT 100`
	}

	return appendPoolSwaps(ctx, []PoolSwaps{}, q, convertToRune, w.Since.UnixNano(), w.Until.UnixNano(), interval)
}

func appendPoolSwaps(ctx context.Context, swaps []PoolSwaps, q string, conversion bool,  args ...interface{}) ([]PoolSwaps, error) {
	rows, err := DBQuery(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var r PoolSwaps
		var first, last int64
		if conversion {
			if err := rows.Scan(&r.TxCount, &r.AssetE8Total, &r.RuneE8Total, &r.LiqFeeE8Total, &r.LiqFeeInRuneE8Total, &r.TradeSlipBPTotal, &first, &last, &r.TruncatedTime, &r.VolumeInRune); err != nil {
				return swaps, err
			}
		} else {
			if err := rows.Scan(&r.TxCount, &r.AssetE8Total, &r.RuneE8Total, &r.LiqFeeE8Total, &r.LiqFeeInRuneE8Total, &r.TradeSlipBPTotal, &first, &last, &r.TruncatedTime); err != nil {
				return swaps, err
			}
		}
		if first != 0 {
			r.First = time.Unix(0, first)
		}
		if last != 0 {
			r.Last = time.Unix(0, last)
		}
		swaps = append(swaps, r)
	}
	return swaps, rows.Err()
}

// struct returned from v1/history/total_volume endpoint
type SwapVolumeChanges struct {
	BuyVolume   string `json:"buyVolume"`	// volume RUNE bought in given a timeframe
	SellVolume  string `json:"sellVolume"`	// volume of RUNE sold in given a timeframe
	Time        int64  `json:"time"`		// beginning of the timeframe
	TotalVolume string `json:"totalVolume"` // sum of bought and sold volume
}

func TotalVolumeChanges(ctx context.Context, inv, pool string,  from, to time.Time) ([]SwapVolumeChanges, error){
	interval, err := GetIntervalFromString(inv)
	if err != nil {
		return nil, err
	}
	window := Window{
		Since: from,
		Until: to,
	}

	// fromRune stores conversion from Rune to Asset -> selling Rune
	fromRune, err := PoolSwapsLookup(ctx, pool, interval, window, false, false)
	if err != nil {
		return nil, err
	}

	// fromAsset stores conversion from Asset to Rune -> buying Rune
	fromAsset, err := PoolSwapsLookup(ctx, pool, interval, window, false, true)
	if err != nil {
		return nil, err
	}

	result, err := mergeSwaps(fromRune, fromAsset)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func mergeSwaps(fromRune, fromAsset []PoolSwaps) ([]SwapVolumeChanges, error){
	result := make([]SwapVolumeChanges, 0)

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
			timestamp := fr.TruncatedTime.Unix()
			runeSellVolume := strconv.FormatInt(fr.RuneE8Total, 10)

			svc := SwapVolumeChanges{
				BuyVolume:   "0",
				SellVolume:  runeSellVolume,
				Time:        timestamp,
				TotalVolume: runeSellVolume,
			}

			result = append(result, svc)
			i++
		} else if fa.TruncatedTime.Before(fr.TruncatedTime){
			timestamp := fa.TruncatedTime.Unix()
			runeBuyVolume := fmt.Sprintf("%f", fa.VolumeInRune)

			svc := SwapVolumeChanges{
				BuyVolume:   runeBuyVolume,
				SellVolume:  "0",
				Time:        timestamp,
				TotalVolume: runeBuyVolume,
			}

			result = append(result, svc)
			j++
		} else if fr.TruncatedTime.Equal(fa.TruncatedTime) {
			timestamp := fr.TruncatedTime.Unix()
			runeSellVolume := strconv.FormatInt(fr.RuneE8Total, 10)
			runeBuyVolume := fmt.Sprintf("%f", fa.VolumeInRune)
			totalVolume := fmt.Sprintf("%f", float64(fr.RuneE8Total) + fa.VolumeInRune)

			svc := SwapVolumeChanges{
				BuyVolume:   runeBuyVolume,
				SellVolume:  runeSellVolume,
				Time:        timestamp,
				TotalVolume: totalVolume,
			}

			result = append(result, svc)
			i++; j++
		} else {
			return result, errors.New("error occurred while merging arrays")
		}
	}

	return result, nil
}
