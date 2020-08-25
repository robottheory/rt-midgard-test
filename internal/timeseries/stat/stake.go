package stat

// Stakes are statistics without asset classification.
type Stakes struct {
	TxCount     int64
	UnitsTotal  int64
	RuneE8Total int64
}

// PoolStakes are statistics for a specific asset.
type PoolStakes struct {
	TxCount      int64
	UnitsTotal   int64
	RuneE8Total  int64
	AssetE8Total int64
}

func StakesLookup(w Window) (Stakes, error) {
	w.normalize()

	const q = `SELECT COUNT(*), SUM(stake_units), SUM(rune_e8)
FROM stake_events
WHERE block_timestamp >= $1 AND block_timestamp < $2`

	return queryStakes(q, w.Start.UnixNano(), w.End.UnixNano())
}

func AddrStakesLookup(addr string, w Window) (Stakes, error) {
	w.normalize()

	const q = `SELECT COUNT(*), SUM(stake_units), SUM(rune_e8)
FROM stake_events
WHERE rune_addr = $1 AND block_timestamp >= $2 AND block_timestamp < $3`

	return queryStakes(q, addr, w.Start.UnixNano(), w.End.UnixNano())
}

func PoolStakesLookup(pool string, w Window) (PoolStakes, error) {
	w.normalize()

	const q = `SELECT COUNT(*), SUM(stake_units), SUM(rune_e8), SUM(asset_e8)
FROM stake_events
WHERE pool = $1 AND block_timestamp >= $2 AND block_timestamp < $3`

	return queryPoolStakes(q, pool, w.Start.UnixNano(), w.End.UnixNano())
}

func AddrPoolStakesLookup(addr, pool string, w Window) (PoolStakes, error) {
	w.normalize()

	const q = `SELECT COUNT(*), SUM(stake_units), SUM(rune_e8), SUM(asset_e8)
FROM stake_events
WHERE rune_addr = $1 AND pool = $2 AND block_timestamp >= $3 AND block_timestamp < $4`

	return queryPoolStakes(q, addr, pool, w.Start.UnixNano(), w.End.UnixNano())
}

func queryStakes(q string, args ...interface{}) (Stakes, error) {
	rows, err := DBQuery(q, args...)
	if err != nil {
		return Stakes{}, err
	}
	defer rows.Close()

	if !rows.Next() {
		return Stakes{}, rows.Err()
	}

	var r Stakes
	// pointer-pointers prevent errors on no matches 😖
	p1, p2, p3 := &r.TxCount, &r.UnitsTotal, &r.RuneE8Total
	if err := rows.Scan(&p1, &p2, &p3); err != nil {
		return Stakes{}, err
	}
	return r, rows.Err()
}

func queryPoolStakes(q string, args ...interface{}) (PoolStakes, error) {
	rows, err := DBQuery(q, args...)
	if err != nil {
		return PoolStakes{}, err
	}
	defer rows.Close()

	if !rows.Next() {
		return PoolStakes{}, rows.Err()
	}

	var r PoolStakes
	// pointer-pointers prevent errors on no matches 😖
	p1, p2, p3, p4 := &r.TxCount, &r.UnitsTotal, &r.RuneE8Total, &r.AssetE8Total
	if err := rows.Scan(&p1, &p2, &p3, &p4); err != nil {
		return PoolStakes{}, err
	}
	return r, rows.Err()
}
