package stat

type PoolAdds struct {
	AssetE8Total int64
	RuneE8Total  int64
}

func PoolAddsLookup(pool string, w Window) (*PoolAdds, error) {
	const q = `SELECT COALESCE(SUM(asset_e8), 0), COALESCE(SUM(rune_e8), 0)
FROM add_events
WHERE pool = $1 AND block_timestamp >= $2 AND block_timestamp < $3`

	rows, err := DBQuery(q, pool, w.Since.UnixNano(), w.Until.UnixNano())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var r PoolAdds
	if rows.Next() {
		err := rows.Scan(&r.AssetE8Total, &r.RuneE8Total)
		if err != nil {
			return nil, err
		}
	}
	return &r, rows.Err()
}

type PoolErratas struct {
	AssetE8Total int64
	RuneE8Total  int64
}

func PoolErratasLookup(pool string, w Window) (*PoolErratas, error) {
	const q = `SELECT COALESCE(SUM(asset_e8), 0), COALESCE(SUM(rune_e8), 0) FROM errata_events
WHERE asset = $1 AND block_timestamp >= $2 AND block_timestamp < $3`

	rows, err := DBQuery(q, pool, w.Since.UnixNano(), w.Until.UnixNano())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var r PoolErratas
	if rows.Next() {
		err := rows.Scan(&r.AssetE8Total, &r.RuneE8Total)
		if err != nil {
			return nil, err
		}
	}
	return &r, rows.Err()
}

type PoolGas struct {
	AssetE8Total int64
	RuneE8Total  int64
}

func PoolGasLookup(pool string, w Window) (*PoolGas, error) {
	const q = `SELECT COALESCE(SUM(asset_e8), 0), COALESCE(SUM(rune_e8), 0)
FROM gas_events
WHERE asset = $1 AND block_timestamp >= $2 AND block_timestamp < $3`

	rows, err := DBQuery(q, pool, w.Since.UnixNano(), w.Until.UnixNano())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var r PoolGas
	if rows.Next() {
		err := rows.Scan(&r.AssetE8Total, &r.RuneE8Total)
		if err != nil {
			return nil, err
		}
	}
	return &r, rows.Err()
}

type PoolSlashes struct {
	AssetE8Total int64
}

func PoolSlashesLookup(pool string, w Window) (*PoolSlashes, error) {
	const q = `SELECT COALESCE(SUM(asset_e8), 0)
FROM slash_amounts
WHERE pool = $1 AND block_timestamp >= $2 AND block_timestamp < $3`

	rows, err := DBQuery(q, pool, w.Since.UnixNano(), w.Until.UnixNano())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var r PoolSlashes
	if rows.Next() {
		err := rows.Scan(&r.AssetE8Total)
		if err != nil {
			return nil, err
		}
	}
	return &r, rows.Err()
}
