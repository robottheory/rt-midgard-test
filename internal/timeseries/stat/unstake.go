package stat

type Unstakes struct {
	TxCount          int64
	E8Total          int64
	StakeUnitsTotal  int64
	BasisPointsTotal int64
}

func PoolAssetUnstakesLookup(pool string, w Window) (Unstakes, error) {
	const q = `SELECT COALESCE(COUNT(*), 0), COALESCE(SUM(asset_e8), 0), COALESCE(SUM(stake_units), 0), COALESCE(SUM(basis_points), 0)
FROM unstake_events
WHERE pool = $1 AND asset = $1 AND block_timestamp >= $2 AND block_timestamp < $3`

	return queryUnstakes(q, pool, w.Since.UnixNano(), w.Until.UnixNano())
}

func PoolRuneUnstakesLookup(pool string, w Window) (Unstakes, error) {
	const q = `SELECT COALESCE(COUNT(*), 0), COALESCE(SUM(asset_e8), 0), COALESCE(SUM(stake_units), 0), COALESCE(SUM(basis_points), 0)
FROM unstake_events
WHERE pool = $1 AND asset <> $1 AND block_timestamp >= $2 AND block_timestamp < $3`

	return queryUnstakes(q, pool, w.Since.UnixNano(), w.Until.UnixNano())
}

func queryUnstakes(q string, args ...interface{}) (Unstakes, error) {
	rows, err := DBQuery(q, args...)
	if err != nil {
		return Unstakes{}, err
	}
	defer rows.Close()

	if !rows.Next() {
		return Unstakes{}, rows.Err()
	}

	var r Unstakes
	err = rows.Scan(&r.TxCount, &r.E8Total, &r.StakeUnitsTotal, &r.BasisPointsTotal)
	return r, err
}
