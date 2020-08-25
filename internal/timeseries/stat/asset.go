package stat

type PoolFees struct {
	AssetE8Total    int64
	AssetE8Avg      float64
	PoolDeductTotal int64
}

func PoolFeesLookup(pool string, w Window) (PoolFees, error) {
	w.normalize()

	const q = `SELECT SUM(asset_e8), AVG(asset_E8), SUM(pool_deduct) FROM fee_events
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
	// pointer-pointers prevent errors on no matches 😖
	p1, p2, p3 := &r.AssetE8Total, &r.AssetE8Avg, &r.PoolDeductTotal
	if err := rows.Scan(&p1, &p2, &p3); err != nil {
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

	const q = `SELECT SUM(asset_e8), SUM(rune_e8) FROM errata_events
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
	// pointer-pointers prevent errors on no matches 😖
	p1, p2 := &r.AssetE8Total, &r.RuneE8Total
	if err := rows.Scan(&p1, &p2); err != nil {
		return PoolErratas{}, err
	}
	return r, rows.Err()
}
