package stat

import "time"

// Swaps are generic swap statistics.
type Swaps struct {
	TxCount       int64
	RuneAddrCount int64 // Number of unique addresses involved.
}

func SwapsFromRuneLookup(w Window) (*Swaps, error) {
	const q = `SELECT COALESCE(COUNT(*), 0), COALESCE(COUNT(DISTINCT(from_addr)))
        FROM swap_events
        WHERE pool = from_asset AND block_timestamp >= $1 AND block_timestamp <= $2`

	return querySwaps(q, w.Since.UnixNano(), w.Until.UnixNano())
}

func SwapsToRuneLookup(w Window) (*Swaps, error) {
	const q = `SELECT COALESCE(COUNT(*), 0), COALESCE(COUNT(DISTINCT(from_addr)))
        FROM swap_events
        WHERE pool <> from_asset AND block_timestamp >= $1 AND block_timestamp <= $2`

	return querySwaps(q, w.Since.UnixNano(), w.Until.UnixNano())
}

func querySwaps(q string, args ...interface{}) (*Swaps, error) {
	rows, err := DBQuery(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var swaps Swaps
	if rows.Next() {
		err := rows.Scan(&swaps.TxCount, &swaps.RuneAddrCount)
		if err != nil {
			return nil, err
		}
	}
	return &swaps, rows.Err()
}

// PoolSwaps are swap statistics for a specific asset.
type PoolSwaps struct {
	TxCount             int64
	AssetE8Total        int64
	RuneE8Total         int64
	LiqFeeE8Total       int64
	LiqFeeInRuneE8Total int64
	TradeSlipBPTotal    int64
}

func PoolSwapsFromRuneLookup(pool string, w Window) (*PoolSwaps, error) {
	const q = `SELECT COALESCE(COUNT(*), 0), 0, COALESCE(SUM(from_E8), 0), COALESCE(SUM(liq_fee_E8), 0), COALESCE(SUM(liq_fee_in_rune_E8), 0), COALESCE(SUM(trade_slip_BP), 0)
	FROM swap_events
	WHERE pool = $1 AND from_asset <> $1 AND block_timestamp >= $2 AND block_timestamp < $3`

	var swaps [1]PoolSwaps
	_, err := appendPoolSwaps(swaps[:0], q, pool, w.Since.UnixNano(), w.Until.UnixNano())
	if err != nil || len(swaps) == 0 {
		return nil, err
	}
	return &swaps[0], nil
}

func PoolSwapsToRuneLookup(pool string, w Window) (*PoolSwaps, error) {
	const q = `SELECT COALESCE(COUNT(*), 0), COALESCE(SUM(from_E8), 0), 0, COALESCE(SUM(liq_fee_E8), 0), COALESCE(SUM(liq_fee_in_rune_E8), 0), COALESCE(SUM(trade_slip_BP), 0)
	FROM swap_events
	WHERE pool = $1 AND from_asset = $1 AND block_timestamp >= $2 AND block_timestamp < $3`

	var swaps [1]PoolSwaps
	_, err := appendPoolSwaps(swaps[:0], q, pool, w.Since.UnixNano(), w.Until.UnixNano())
	if err != nil || len(swaps) == 0 {
		return nil, err
	}
	return &swaps[0], nil
}

func PoolSwapsFromRuneBucketsLookup(pool string, bucketSize time.Duration, w Window) ([]PoolSwaps, error) {
	n, err := bucketsFor(bucketSize, w)
	if err != nil {
		return nil, err
	}
	a := make([]PoolSwaps, 0, n)

	const q = `SELECT COALESCE(COUNT(*), 0), 0, COALESCE(SUM(from_E8), 0), COALESCE(SUM(liq_fee_E8), 0), COALESCE(SUM(liq_fee_in_rune_E8), 0), COALESCE(SUM(trade_slip_BP), 0)
	FROM swap_events
	WHERE pool = $1 AND from_asset <> $1 AND block_timestamp >= $2 AND block_timestamp < $3
	GROUP BY time_bucket($4, block_timestamp)
	ORDER BY time_bucket($4, block_timestamp)`

	return appendPoolSwaps(a, q, pool, w.Since.UnixNano(), w.Until.UnixNano(), bucketSize)
}

func PoolSwapsToRuneBucketsLookup(pool string, bucketSize time.Duration, w Window) ([]PoolSwaps, error) {
	n, err := bucketsFor(bucketSize, w)
	if err != nil {
		return nil, err
	}
	a := make([]PoolSwaps, 0, n)

	const q = `SELECT COALESCE(COUNT(*), 0), COALESCE(SUM(from_E8), 0), 0, COALESCE(SUM(liq_fee_E8), 0), COALESCE(SUM(liq_fee_in_rune_E8), 0), COALESCE(SUM(trade_slip_BP), 0)
	FROM swap_events
	WHERE pool = $1 AND from_asset = $1 AND block_timestamp >= $2 AND block_timestamp < $3
	GROUP BY time_bucket($4, block_timestamp)
	ORDER BY time_bucket($4, block_timestamp)`

	return appendPoolSwaps(a, q, pool, w.Since.UnixNano(), w.Until.UnixNano(), bucketSize)
}

func appendPoolSwaps(swaps []PoolSwaps, q string, args ...interface{}) ([]PoolSwaps, error) {
	rows, err := DBQuery(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var r PoolSwaps
		if err := rows.Scan(&r.TxCount, &r.AssetE8Total, &r.RuneE8Total, &r.LiqFeeE8Total, &r.LiqFeeInRuneE8Total, &r.TradeSlipBPTotal); err != nil {
			return swaps, err
		}
		swaps = append(swaps, r)
	}
	return swaps, rows.Err()
}
