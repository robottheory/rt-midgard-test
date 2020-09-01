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

type PoolDetails struct {
	AssetDepth int64
	RuneDepth  int64
	PoolDepth  int64
	AssetROI   float64
	RuneROI    float64
	PriceRune  float64
}

func GetPoolDetails(pool string, w Window) (PoolDetails, error) {
	poolDetails := PoolDetails{}

	stakes, err := PoolStakesLookup(pool, w)
	if err != nil {
		return poolDetails, err
	}
	swaps, err := PoolSwapsLookup(pool, w)
	if err != nil {
		return poolDetails, err
	}
	adds, err := PoolAddsLookup(pool, w)
	if err != nil {
		return poolDetails, err
	}
	gas, err := PoolGasLookup(pool, w)
	if err != nil {
		return poolDetails, err
	}
	fee, err := PoolFeesLookup(pool, w)
	if err != nil {
		return poolDetails, err
	}
	slash, err := PoolSlashesLookup(pool, w)
	if err != nil {
		return poolDetails, err
	}
	errata, err := PoolErratasLookup(pool, w)
	if err != nil {
		return poolDetails, err
	}

	poolDetails.AssetDepth = stakes.AssetE8Total + swaps.AssetE8Total + adds.AssetE8Total - gas.AssetE8Total + fee.AssetE8Total + slash.AssetE8Total + errata.AssetE8Total
	poolDetails.RuneDepth = stakes.RuneE8Total + adds.RuneE8Total + gas.RuneE8Total + errata.RuneE8Total // TODO (manolodewiner) + fee.RuneE8Total + reward.RuneE8Total + swaps.RuneE8Total + slash.RuneE8Total
	poolDetails.PoolDepth = 2 * poolDetails.RuneDepth

	assetStaked := float64(stakes.AssetE8Total)
	assetDepth := float64(poolDetails.AssetDepth)
	runeStaked := float64(stakes.RuneE8Total)
	runeDepth := float64(poolDetails.RuneDepth)

	var assetROI, runeROI float64
	if assetStaked > 0 {
		assetROI = (assetDepth - assetStaked) / assetStaked
	}

	if runeStaked > 0 {
		runeROI = (runeDepth - runeStaked) / runeStaked
	}

	if poolDetails.AssetDepth > 0 {
		poolDetails.PriceRune = runeDepth / assetDepth
	}

	poolDetails.AssetROI = assetROI
	poolDetails.RuneROI = runeROI

	return poolDetails, nil
}
