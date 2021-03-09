package stat

import (
	"context"

	"gitlab.com/thorchain/midgard/internal/db"
)

// TODO(acsaba): This file should be renamed to addLiquidity.go once the terminology of all
// functions is updated

func StakesLookup(ctx context.Context, w db.Window) (ret CountAndTotal, err error) {
	return liquidityChange(ctx, w, "stake_events", "asset_E8", "rune_E8")
}
