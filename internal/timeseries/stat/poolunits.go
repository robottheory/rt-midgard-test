package stat

import (
	"context"

	"gitlab.com/thorchain/midgard/internal/db"
)

// PoolUnits gets net stake units in pools
func PoolsUnits(ctx context.Context, pools []string) (map[string]int64, error) {
	q := `SELECT
		stake_events.pool,
		(
			COALESCE(SUM(stake_events.stake_units), 0) -
			(SELECT COALESCE(SUM(unstake_events.stake_units), 0) FROM unstake_events WHERE unstake_events.pool = stake_events.pool)
		)
		FROM stake_events
		WHERE pool = ANY($1)
		GROUP BY stake_events.pool
	`

	poolsUnits := make(map[string]int64)
	rows, err := db.Query(ctx, q, pools)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var pool string
		var units int64
		err := rows.Scan(&pool, &units)
		if err != nil {
			return nil, err
		}
		poolsUnits[pool] = units
	}

	if err != nil {
		return nil, err
	}

	return poolsUnits, nil
}
