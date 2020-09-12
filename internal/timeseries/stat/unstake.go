package stat

import (
	"log"

	"gitlab.com/thorchain/midgard/event"
)

// Unstakes are generic unstake statistics.
type Unstakes struct {
	TxCount       int64
	RuneAddrCount int64 // Number of unique staker addresses involved.
	RuneE8Total   int64
}

func UnstakesLookup(w Window) (*Unstakes, error) {
	// BUG(pascaldekloe): No way for asset declarations in unstake events to detect RUNE.
	const q = `SELECT COALESCE(COUNT(*), 0), COALESCE(COUNT(DISTINCT(to_addr)), 0), COALESCE(SUM(asset_e8), 0)
	FROM unstake_events
	WHERE block_timestamp >= $1 AND block_timestamp <= $2 AND asset IN ('THOR.RUNE', 'BNB.RUNE-67C', 'BNB.RUNE-B1A')`

	rows, err := DBQuery(q, w.Since.UnixNano(), w.Until.UnixNano())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var r Unstakes
	if rows.Next() {
		err := rows.Scan(&r.TxCount, &r.RuneAddrCount, &r.RuneE8Total)
		if err != nil {
			return nil, err
		}
	}
	return &r, rows.Err()
}

// PoolUnstakes are unstake statistics for a specific asset.
type PoolUnstakes struct {
	TxCount          int64
	AssetE8Total     int64
	RuneE8Total      int64
	StakeUnitsTotal  int64
	BasisPointsTotal int64
}

func PoolUnstakesLookup(pool string, w Window) (*PoolUnstakes, error) {
	const q = `SELECT asset, COALESCE(COUNT(*), 0), COALESCE(SUM(asset_e8), 0), COALESCE(SUM(stake_units), 0), COALESCE(SUM(basis_points), 0)
FROM unstake_events
WHERE pool = $1 AND block_timestamp >= $2 AND block_timestamp < $3
GROUP BY asset`

	rows, err := DBQuery(q, pool, w.Since.UnixNano(), w.Until.UnixNano())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var unstakes PoolUnstakes
	for rows.Next() {
		var asset []byte
		var txCount, assetE8Total, stakeUnitsTotal, basisPointsTotal int64
		err = rows.Scan(&asset, &txCount, &assetE8Total, &stakeUnitsTotal, &basisPointsTotal)
		if err != nil {
			return nil, err
		}

		unstakes.TxCount += txCount
		unstakes.StakeUnitsTotal += stakeUnitsTotal
		unstakes.BasisPointsTotal += basisPointsTotal

		switch {
		case event.IsRune(asset):
			unstakes.RuneE8Total = assetE8Total
		case string(asset) == pool:
			unstakes.AssetE8Total = assetE8Total
		default:
			// BUG(pascaldekloe): Unstake assets are ignored when they don't
			// match the pool name. How should they be applied?
			log.Printf("unstake asset %q for pool %q ignored for lookup", asset, pool)
		}
	}

	return &unstakes, rows.Err()
}
