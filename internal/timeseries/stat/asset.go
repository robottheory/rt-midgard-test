package stat

type PoolAdds struct {
	AssetE8Total int64
	RuneE8Total  int64
}

func PoolAddsLookup(pool string, w Window) (PoolAdds, error) {
	w.normalize()

	const q = `SELECT COALESCE(SUM(asset_e8), 0), COALESCE(SUM(rune_e8), 0)
FROM add_events
WHERE pool = $1 AND block_timestamp >= $2 AND block_timestamp < $3`

	rows, err := DBQuery(q, pool, w.Start.UnixNano(), w.End.UnixNano())
	if err != nil {
		return PoolAdds{}, err
	}
	defer rows.Close()

	if !rows.Next() {
		return PoolAdds{}, rows.Err()
	}

	var r PoolAdds
	if err := rows.Scan(&r.AssetE8Total, &r.RuneE8Total); err != nil {
		return PoolAdds{}, err
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


type PoolGas struct {
	AssetE8Total int64
	RuneE8Total  int64
}

func PoolGasLookup(pool string, w Window) (PoolGas, error) {
	w.normalize()

	const q = `SELECT COALESCE(SUM(asset_e8), 0), COALESCE(SUM(rune_e8), 0)
FROM gas_events
WHERE asset = $1 AND block_timestamp >= $2 AND block_timestamp < $3`

	rows, err := DBQuery(q, pool, w.Start.UnixNano(), w.End.UnixNano())
	if err != nil {
		return PoolGas{}, err
	}
	defer rows.Close()

	if !rows.Next() {
		return PoolGas{}, rows.Err()
	}

	var r PoolGas
	if err := rows.Scan(&r.AssetE8Total, &r.RuneE8Total); err != nil {
		return PoolGas{}, err
	}
	return r, rows.Err()
}

type PoolSlashes struct {
	AssetE8Total int64
}

func PoolSlashesLookup(pool string, w Window) (PoolSlashes, error) {
	w.normalize()

	const q = `SELECT COALESCE(SUM(asset_e8), 0)
FROM slash_amounts
WHERE pool = $1 AND block_timestamp >= $2 AND block_timestamp < $3`

	rows, err := DBQuery(q, pool, w.Start.UnixNano(), w.End.UnixNano())
	if err != nil {
		return PoolSlashes{}, err
	}
	defer rows.Close()

	if !rows.Next() {
		return PoolSlashes{}, rows.Err()
	}

	var r PoolSlashes
	if err := rows.Scan(&r.AssetE8Total); err != nil {
		return PoolSlashes{}, err
	}
	return r, rows.Err()
}

