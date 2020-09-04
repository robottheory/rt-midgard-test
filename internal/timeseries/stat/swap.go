package stat

import "time"

// PoolSwaps are swap statistics for a specific asset.
type PoolSwaps struct {
	TxCount      int64
	FromE8Total  int64
	ToE8Min      float64
	TradeSlipBP  int64
	LiqFeeE8     int64
	LiqFeeRuneE8 int64
	First, Last  time.Time
}

func SwapsLookup(w Window) (PoolSwaps, error) {
	const q = `SELECT COALESCE(COUNT(*), 0), COALESCE(SUM(from_E8), 0), COALESCE(AVG(to_E8_min), 0), COALESCE(SUM(trade_slip_BP), 0), COALESCE(SUM(liq_fee), 0), COALESCE(SUM(liq_fee_in_rune_e8), 0), COALESCE(MIN(block_timestamp), 0), COALESCE(MAX(block_timestamp), 0)
FROM swap_events
WHERE block_timestamp >= $1 AND block_timestamp < $2`

	return querySwaps(q, w.Start.UnixNano(), w.End.UnixNano())
}

func PoolSwapsLookup(pool string, w Window) (PoolSwaps, error) {
	const q = `SELECT COALESCE(COUNT(*), 0), COALESCE(SUM(from_E8), 0), COALESCE(AVG(to_E8_min), 0), COALESCE(SUM(trade_slip_BP), 0), COALESCE(SUM(liq_fee), 0), COALESCE(SUM(liq_fee_in_rune_e8), 0), COALESCE(MIN(block_timestamp), 0), COALESCE(MAX(block_timestamp), 0)
FROM swap_events
WHERE pool = $1 AND block_timestamp >= $2 AND block_timestamp < $3`

	return querySwaps(q, pool, w.Start.UnixNano(), w.End.UnixNano())
}

type PoolBuySwaps struct {
	ToAssetCount        int64   // swaps in this period from RUNE -> ASSET
	ToAssetVolume       int64   // volume for RUNE->ASSET (in RUNE)
	ToAssetLiqFees      int64   // buy fees in RUNE
	MeanToAssetSlippage float64 // buy slippage in RUNE
	First, Last         time.Time
}
type PoolSellSwaps struct {
	ToRuneCount        int64 // swaps in this period from ASSET -> RUNE
	ToRuneVolume       int64 // volume for ASSET->RUNE (in RUNE)
	ToRuneLiqFees      int64 // sell fees in RUNE
	MeanToRuneSlippage int64 // sell slippage in RUNE
	First, Last        time.Time
}

func PoolBuySwapsBucketLookup(pool string, bucketSize uint64, w Window) ([]PoolBuySwaps, error) {
	const q = `SELECT COALESCE(COUNT(*), 0), COALESCE(SUM(from_e8), 0), COALESCE(SUM(liq_fee_in_rune_e8), 0), COALESCE(SUM(trade_slip_bp), 0), COALESCE(MIN(block_timestamp), 0), COALESCE(MAX(block_timestamp), 0)
	FROM swap_events
	WHERE pool = $1 AND from_asset<>pool AND block_timestamp >= $2 AND block_timestamp < $3
	GROUP BY time_bucket($4, block_timestamp)
	ORDER BY time_bucket($4, block_timestamp)
	`
	rows, err := DBQuery(q, pool, w.Start.UnixNano(), w.End.UnixNano(), bucketSize)
	if err != nil {
		return []PoolBuySwaps{}, err
	}
	defer rows.Close()

	var pools []PoolBuySwaps
	for rows.Next() {
		var r PoolBuySwaps
		var first, last int64
		if err := rows.Scan(&r.ToAssetCount, &r.ToAssetVolume, &r.ToAssetLiqFees, &r.MeanToAssetSlippage, &first, &last); err != nil {
			return pools, err
		}
		if first != 0 {
			r.First = time.Unix(0, first)
		}
		if last != 0 {
			r.Last = time.Unix(0, last)
		}
		pools = append(pools, r)
	}
	return pools, rows.Err()
}

func PoolSellSwapsBucketLookup(pool string, bucketSize uint64, w Window) ([]PoolSellSwaps, error) {
	const q = `SELECT COALESCE(COUNT(*), 0), COALESCE(SUM(to_e8_min), 0), COALESCE(SUM(liq_fee_in_rune_e8), 0), COALESCE(SUM(trade_slip_bp), 0), COALESCE(MIN(block_timestamp), 0), COALESCE(MAX(block_timestamp), 0)
	FROM swap_events
	WHERE pool = $1 AND from_asset=pool AND block_timestamp >= $2 AND block_timestamp < $3
	GROUP BY time_bucket($4, block_timestamp)
	ORDER BY time_bucket($4, block_timestamp)
	`
	rows, err := DBQuery(q, pool, w.Start.UnixNano(), w.End.UnixNano(), bucketSize)
	if err != nil {
		return []PoolSellSwaps{}, err
	}
	defer rows.Close()

	var pools []PoolSellSwaps
	for rows.Next() {
		var r PoolSellSwaps
		var first, last int64
		if err := rows.Scan(&r.ToRuneCount, &r.ToRuneVolume, &r.ToRuneLiqFees, &r.MeanToRuneSlippage, &first, &last); err != nil {
			return pools, err
		}
		if first != 0 {
			r.First = time.Unix(0, first)
		}
		if last != 0 {
			r.Last = time.Unix(0, last)
		}
		pools = append(pools, r)
	}
	return pools, rows.Err()
}

func querySwaps(q string, args ...interface{}) (PoolSwaps, error) {
	rows, err := DBQuery(q, args...)

	if err != nil {
		return PoolSwaps{}, err
	}
	defer rows.Close()

	if !rows.Next() {
		return PoolSwaps{}, rows.Err()
	}

	var r PoolSwaps
	var first, last int64
	if err := rows.Scan(&r.TxCount, &r.FromE8Total, &r.ToE8Min, &r.TradeSlipBP, &r.LiqFeeE8, &r.LiqFeeRuneE8, &first, &last); err != nil {
		return PoolSwaps{}, err
	}
	if first != 0 {
		r.First = time.Unix(0, first)
	}
	if last != 0 {
		r.Last = time.Unix(0, last)
	}
	return r, rows.Err()

}
