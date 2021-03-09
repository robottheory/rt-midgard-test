package stat

import (
	"context"

	"gitlab.com/thorchain/midgard/internal/db"
)

// TODO(acsaba): This file should be renamed to withdraw.go once the terminology of all
// functions is updated

type CountAndTotal struct {
	Count       int64
	TotalVolume int64
}

func liquidityChange(ctx context.Context, w db.Window, table, assetColumn, runeColumn string) (ret CountAndTotal, err error) {
	buckets := db.OneIntervalBuckets(w.From, w.Until)

	withdraws, err := liquidityChangesFromTable(ctx, buckets, "*",
		table, assetColumn, runeColumn)
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
	ret.Count = bucket.count
	ret.TotalVolume = bucket.runeVolume + bucket.assetVolume
	return
}

func UnstakesLookup(ctx context.Context, w db.Window) (ret CountAndTotal, err error) {
	return liquidityChange(ctx, w, "unstake_events", "emit_asset_E8", "emit_rune_E8")
}
