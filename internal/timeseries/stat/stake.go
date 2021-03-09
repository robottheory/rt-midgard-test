package stat

import (
	"context"
	"time"

	"gitlab.com/thorchain/midgard/internal/db"
)

// TODO(acsaba): This file should be renamed to addLiquidity.go once the terminology of all
// functions is updated

// Stakes are generic stake statistics.
type Stakes struct {
	TxCount         int64
	RuneAddrCount   int64 // Number of unique staker addresses involved.
	RuneE8Total     int64
	StakeUnitsTotal int64
	First, Last     time.Time
}

func StakesLookup(ctx context.Context, w db.Window) (*Stakes, error) {
	const q = `SELECT COALESCE(COUNT(*), 0), COALESCE(COUNT(DISTINCT(rune_addr))), COALESCE(SUM(rune_e8), 0), COALESCE(SUM(stake_units), 0), COALESCE(MIN(block_timestamp), 0), COALESCE(MAX(block_timestamp), 0)
FROM stake_events
WHERE block_timestamp >= $1 AND block_timestamp < $2`

	return queryStakes(ctx, q, w.From.ToNano(), w.Until.ToNano())
}

func queryStakes(ctx context.Context, q string, args ...interface{}) (*Stakes, error) {
	rows, err := db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var r Stakes
	if rows.Next() {
		var first, last int64
		err := rows.Scan(&r.TxCount, &r.RuneAddrCount, &r.StakeUnitsTotal, &r.RuneE8Total, &first, &last)
		if err != nil {
			return nil, err
		}
		if first != 0 {
			r.First = time.Unix(0, first)
		}
		if last != 0 {
			r.Last = time.Unix(0, last)
		}
	}
	return &r, rows.Err()
}
