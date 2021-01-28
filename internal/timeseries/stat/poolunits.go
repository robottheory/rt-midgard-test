package stat

import (
	"context"

	"gitlab.com/thorchain/midgard/internal/db"
)

func unitsChanges(ctx context.Context, pools []string, tableName string) (map[string]int64, error) {
	q := `
		SELECT
			pool,
			COALESCE(SUM(stake_units), 0) as units
		FROM ` + tableName + `
		WHERE pool = ANY($1)
		GROUP BY pool`

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

	return poolsUnits, nil
}

// PoolUnits gets net stake units in pools
func PoolsUnits(ctx context.Context, pools []string) (map[string]int64, error) {
	ret, err := unitsChanges(ctx, pools, "stake_events")
	if err != nil {
		return nil, err
	}
	withdraws, err := unitsChanges(ctx, pools, "unstake_events")
	if err != nil {
		return nil, err
	}
	for k, v := range withdraws {
		ret[k] -= v
	}
	return ret, nil
}
