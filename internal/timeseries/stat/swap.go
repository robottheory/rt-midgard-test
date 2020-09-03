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

type TotalSwapQuant struct {
	TxCount             int64
	TotalVolume         int64   // toRuneVolume + toAssetVolume (in RUNE)
	TotalFees           int64   // fees in RUNE: toRuneFees + toAssetFees
	MeantoAssetSlippage float64 // buy slippage in RUNE
	MeanToRuneSlippage  float64 // sell slippage in RUNE
	MeanSlippage        float64 // total slippage in RUNE: buySlippage + sellSlippage
}
type PoolSwapsInterval struct {
	ToAssetCount   int64 // swaps in this period from RUNE -> ASSET
	ToRuneCount    int64 // swaps in this period from ASSET -> RUNE
	ToAssetVolume  int64 // volume for RUNE->ASSET (in RUNE)
	ToRuneVolume   int64 // volume for ASSET->RUNE (in RUNE)
	ToAssetLiqFees int64 // buy fees in RUNE
	ToRuneLiqFees  int64 // sell fees in RUNE
	Timestamp      time.Time

	TotalSwapQuant
}

func PoolSwapsIntervalLookup(pool string, interval uint64, w Window) ([]PoolSwapsInterval, error) {
	w.normalize()

	const qBuy = `SELECT COALESCE(COUNT(*), 0), COALESCE(SUM(from_e8), 0), COALESCE(SUM(liq_fee_in_rune_e8), 0), time_bucket($1, block_timestamp) AS tb
FROM swap_events
WHERE pool = $2 AND from_asset<>pool AND block_timestamp >= $3 AND block_timestamp < $4
GROUP BY tb
`
	const qSell = `SELECT COALESCE(COUNT(*), 0), COALESCE(SUM(from_e8), 0), COALESCE(SUM(liq_fee_in_rune_e8), 0), time_bucket($1, block_timestamp) AS tb
FROM swap_events
WHERE pool = $2 AND from_asset=pool AND block_timestamp >= $3 AND block_timestamp < $4
GROUP BY tb
`

	rowsBuy, err := DBQuery(qBuy, interval, pool, w.Start.UnixNano(), w.End.UnixNano())
	if err != nil {
		return []PoolSwapsInterval{}, err
	}
	defer rowsBuy.Close()
	rowsSell, err := DBQuery(qSell, interval, pool, w.Start.UnixNano(), w.End.UnixNano())
	if err != nil {
		return []PoolSwapsInterval{}, err
	}
	defer rowsSell.Close()

	var pools []PoolSwapsInterval
	for rowsBuy.Next() {
		var s PoolSwapsInterval
		if err := rowsBuy.Scan(&s.ToAssetCount, &s.ToAssetVolume, &s.ToAssetLiqFees, &s.Timestamp); err != nil {
			return pools, err
		}
		if err := rowsSell.Scan(&s.ToRuneCount, &s.ToRuneVolume, &s.ToRuneLiqFees, &s.Timestamp); err != nil {
			return pools, err
		}
		// TODO operations
		pools = append(pools, s)
	}
	return pools, rowsBuy.Err()
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
