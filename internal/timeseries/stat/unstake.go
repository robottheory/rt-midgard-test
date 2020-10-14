package stat

import (
	"context"
)

// Unstakes are generic unstake statistics.
type Unstakes struct {
	TxCount       int64
	RuneAddrCount int64 // Number of unique staker addresses involved.
	RuneE8Total   int64
}

// TODO(elfedy): Rune_E8 total should come from associated outbound events, not from
// the unstake event itself as it refers to the amount sent in the transaction that
// requests the unstake (which is a random small amount) and not to the amount actually
// being unstaked, which is calculated by the node and then sent in an outbound transaction
func UnstakesLookup(ctx context.Context, w Window) (*Unstakes, error) {
	const q = `SELECT COALESCE(COUNT(*), 0), COALESCE(COUNT(DISTINCT(to_addr)), 0), COALESCE(SUM(asset_e8), 0)
	FROM unstake_events
	WHERE block_timestamp >= $1 AND block_timestamp <= $2 AND asset IN ('THOR.RUNE', 'BNB.RUNE-67C', 'BNB.RUNE-B1A')`

	rows, err := DBQuery(ctx, q, w.Since.UnixNano(), w.Until.UnixNano())
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

func PoolUnstakesLookup(ctx context.Context, pool string, w Window) (*PoolUnstakes, error) {
	var unstakes PoolUnstakes
	// Get count, stake units and basis points
	unstakeQ := `SELECT COALESCE(COUNT(*), 0), COALESCE(SUM(stake_units), 0), COALESCE(SUM(basis_points), 0)
	FROM unstake_events
	WHERE pool = $1 AND block_timestamp >= $2 AND block_timestamp < $3`

	rows, err := DBQuery(ctx, unstakeQ, pool, w.Since.UnixNano(), w.Until.UnixNano())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var txCount, stakeUnitsTotal, basisPointsTotal int64
		err = rows.Scan(&txCount, &stakeUnitsTotal, &basisPointsTotal)
		if err != nil {
			return nil, err
		}

		unstakes.TxCount += txCount
		// TODO(elfedy): Why are we getting this totals? Is it ok to just sum them?
		unstakes.StakeUnitsTotal += stakeUnitsTotal
		unstakes.BasisPointsTotal += basisPointsTotal
	}

	// Get unstaked RUNE amount
	runeUnstakedQ := `SELECT COALESCE(SUM(outbound_events.asset_E8), 0)
	FROM unstake_events
	INNER JOIN
	outbound_events
	ON outbound_events.in_tx = unstake_events.tx
	WHERE unstake_events.pool = $1
	AND unstake_events.block_timestamp >= $2
	AND unstake_events.block_timestamp < $3
	AND outbound_events.asset IN ('THOR.RUNE', 'BNB.RUNE-67C', 'BNB.RUNE-B1A')
	AND outbound_events.block_timestamp > unstake_events.block_timestamp - 3600000000000
	AND outbound_events.block_timestamp < unstake_events.block_timestamp + 3600000000000`

	rows, err = DBQuery(ctx, runeUnstakedQ, pool, w.Since.UnixNano(), w.Until.UnixNano())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var runeE8Total int64
		err = rows.Scan(&runeE8Total)
		if err != nil {
			return nil, err
		}

		unstakes.RuneE8Total += runeE8Total
	}

	// Get unstaked asset amount
	assetUnstakedQ := `SELECT COALESCE(SUM(outbound_events.asset_E8), 0)
	FROM unstake_events
	INNER JOIN
	outbound_events
	ON outbound_events.in_tx = unstake_events.tx
	WHERE unstake_events.pool = $1
	AND unstake_events.block_timestamp >= $2
	AND unstake_events.block_timestamp < $3
	AND outbound_events.asset = pool
	AND outbound_events.block_timestamp > unstake_events.block_timestamp - 3600000000000
	AND outbound_events.block_timestamp < unstake_events.block_timestamp + 3600000000000`

	rows, err = DBQuery(ctx, assetUnstakedQ, pool, w.Since.UnixNano(), w.Until.UnixNano())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var assetE8Total int64
		err = rows.Scan(&assetE8Total)
		if err != nil {
			return nil, err
		}

		unstakes.AssetE8Total += assetE8Total
	}

	return &unstakes, rows.Err()
}
