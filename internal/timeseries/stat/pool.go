package stat

func PoolStatusLookup(pool string) (string, error) {
	var status string
	const q = `SELECT last(status, block_timestamp) FROM pool_events WHERE asset = $1`

	rows, err := DBQuery(q, pool)
	if err != nil {
		return status, err
	}
	defer rows.Close()

	if !rows.Next() {
		return status, rows.Err()
	}

	if err := rows.Scan(&status); err != nil {
		return status, err
	}

	// TODO (manolodewiner) Query THORChain if we haven't received any pool event for the specified pool --> usecase.go

	return status, rows.Err()
}

type PoolDetails struct {
	AssetDepth  int64
	RuneDepth   int64
	PoolDepth   int64
	AssetROI    float64
	RuneROI     float64
	PriceRune   float64
	StakedTotal float64
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

	poolDetails.AssetDepth = stakes.AssetE8Total + swaps.FromE8Total + adds.AssetE8Total - gas.AssetE8Total + fee.AssetE8Total + slash.AssetE8Total + errata.AssetE8Total
	poolDetails.RuneDepth = stakes.RuneE8Total + adds.RuneE8Total + gas.RuneE8Total + errata.RuneE8Total + swaps.LiqFeeRuneE8 // TODO (manolodewiner) + reward.RuneE8Total + swaps.RuneE8Total + slash.RuneE8Total
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

	poolDetails.StakedTotal = runeStaked + (assetStaked * poolDetails.PriceRune)

	return poolDetails, nil
}
