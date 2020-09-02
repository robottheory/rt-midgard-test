package stat

import "time"

// PoolSwaps are swap statistics for a specific asset.
type PoolSwaps struct {
	TxCount      int64
	AssetE8Total int64
	PriceAverage float64
	TradeSlip    int64
	LiqFee       int64
	LiqFeeRune   int64
	First, Last  time.Time
}

func SwapsLookup(w Window) (PoolSwaps, error) {
	w.normalize()

	const q = `SELECT COALESCE(COUNT(*), 0), COALESCE(SUM(asset_e8), 0), COALESCE(AVG(price_target), 0), COALESCE(SUM(trade_slip), 0), COALESCE(SUM(liq_fee), 0), COALESCE(SUM(liq_fee_in_rune), 0), COALESCE(MIN(block_timestamp), 0), COALESCE(MAX(block_timestamp), 0)
FROM swap_events
WHERE block_timestamp >= $1 AND block_timestamp < $2`

	return querySwaps(q, w.Start.UnixNano(), w.End.UnixNano())

}

func PoolSwapsLookup(pool string, w Window) (PoolSwaps, error) {
	w.normalize()

	const q = `SELECT COALESCE(COUNT(*), 0), COALESCE(SUM(asset_e8), 0), COALESCE(AVG(price_target), 0), COALESCE(SUM(trade_slip), 0), COALESCE(SUM(liq_fee), 0), COALESCE(SUM(liq_fee_in_rune), 0), COALESCE(MIN(block_timestamp), 0), COALESCE(MAX(block_timestamp), 0)
FROM swap_events
WHERE pool = $1 AND block_timestamp >= $2 AND block_timestamp < $3`

	return querySwaps(q, pool, w.Start.UnixNano(), w.End.UnixNano())

}

func PoolSwapsFeesLookup(pool string, w Window) (PoolSwaps, error) {
	w.normalize()

	const q = `SELECT  COALESCE(AVG(price_target), 0), COALESCE(SUM(trade_slip), 0), COALESCE(SUM(liq_fee), 0), COALESCE(SUM(liq_fee_in_rune), 0), COALESCE(MIN(block_timestamp), 0), COALESCE(MAX(block_timestamp), 0)
FROM swap_events
WHERE pool = $1 AND block_timestamp >= $2 AND block_timestamp < $3`

	return querySwaps(q, pool, w.Start.UnixNano(), w.End.UnixNano())

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
	if err := rows.Scan(&r.TxCount, &r.AssetE8Total, &r.PriceAverage, &r.TradeSlip, &r.LiqFee, &r.LiqFeeRune, &first, &last); err != nil {
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
