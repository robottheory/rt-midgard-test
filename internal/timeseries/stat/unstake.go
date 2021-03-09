package stat

import (
	"context"

	"gitlab.com/thorchain/midgard/internal/db"
)

// TODO(elfedy): This file should be renamed to withdraw.go once the terminology of all
// functions is updated

// Unstakes are generic unstake statistics.
type Unstakes struct {
	TxCount     int64
	RuneE8Total int64
}

func UnstakesLookup(ctx context.Context, w db.Window) (ret Unstakes, err error) {
	buckets := db.OneIntervalBuckets(w.From, w.Until)

	withdraws, err := liquidityChangesFromTable(ctx, buckets, "*",
		"unstake_events", "emit_asset_E8", "emit_rune_E8")
	if err != nil {
		return
	}
	bucket, ok := withdraws.buckets[w.From]
	if !ok {
		// We didn't have withdraws yet, probably the beginning of chain.
		// If there are withdraws then maybe the block_pool_depths are missing for the exact
		// timestamps.
		return ret, nil
	}
	ret.TxCount = bucket.count
	ret.RuneE8Total = bucket.runeVolume + bucket.assetVolume
	return
}
