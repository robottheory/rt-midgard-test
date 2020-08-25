package stat

type PoolFees struct {
	AssetE8Total    int64
	AssetE8Avg      float64
	PoolDeductTotal int64
}

func PoolFeesLookup(pool string, w Window) (PoolFees, error) {
	w.normalize()

	const q = `SELECT COALESCE(SUM(asset_e8), 0), COALESCE(AVG(asset_E8), 0), COALESCE(SUM(pool_deduct), 0) FROM fee_events
WHERE asset = $1 AND block_timestamp >= $2 AND block_timestamp < $3`

	rows, err := DBQuery(q, pool, w.Start.UnixNano(), w.End.UnixNano())
	if err != nil {
		return PoolFees{}, err
	}
	defer rows.Close()

	if !rows.Next() {
		return PoolFees{}, rows.Err()
	}

	var r PoolFees
	if err := rows.Scan(&r.AssetE8Total, &r.AssetE8Avg, &r.PoolDeductTotal); err != nil {
		return PoolFees{}, err
	}
	return r, rows.Err()
}

type PoolErratas struct {
	AssetE8Total int64
	RuneE8Total  int64
}

func PoolErratasLookup(pool string, w Window) (PoolErratas, error) {
	w.normalize()

	const q = `SELECT COALESCE(SUM(asset_e8), 0), COALESCE(SUM(rune_e8), 0) FROM errata_events
WHERE asset = $1 AND block_timestamp >= $2 AND block_timestamp < $3`

	rows, err := DBQuery(q, pool, w.Start.UnixNano(), w.End.UnixNano())
	if err != nil {
		return PoolErratas{}, err
	}
	defer rows.Close()

	if !rows.Next() {
		return PoolErratas{}, rows.Err()
	}

	var r PoolErratas
	if err := rows.Scan(&r.AssetE8Total, &r.RuneE8Total); err != nil {
		return PoolErratas{}, err
	}
	return r, rows.Err()
}
