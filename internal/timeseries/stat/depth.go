package stat

import (
	"context"
	"math/big"
	"time"
)

type PoolDepth struct {
	First      time.Time
	Last       time.Time
	RuneFirst  int64
	RuneLast   int64
	AssetFirst int64
	AssetLast  int64
	PriceFirst float64
	PriceLast  float64
}

func PoolDepthBucketsLookup(ctx context.Context, asset string, bucketSize time.Duration, w Window) ([]PoolDepth, error) {
	n, err := bucketsFor(bucketSize, w)
	if err != nil {
		return nil, err
	}
	a := make([]PoolDepth, 0, n)

	const q = `
		SELECT 
			first(rune_e8, block_timestamp), 
			last(rune_e8, block_timestamp), 
			first(asset_e8, block_timestamp), 
			last(asset_e8, block_timestamp), 
			COALESCE(MIN(block_timestamp), 0), 
			COALESCE(MAX(block_timestamp), 0)
		FROM block_pool_depths
		WHERE pool = $1 AND block_timestamp >= $2 AND block_timestamp < $3
		GROUP BY time_bucket($4, block_timestamp)
		ORDER BY time_bucket($4, block_timestamp)
	`
	return appendPoolDepths(ctx, a, q, asset, w.Since.UnixNano(), w.Until.UnixNano(), bucketSize.Nanoseconds())
}

func appendPoolDepths(ctx context.Context, a []PoolDepth, q string, args ...interface{}) ([]PoolDepth, error) {
	rows, err := DBQuery(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var r PoolDepth
		var first, last int64
		err := rows.Scan(&r.RuneFirst, &r.RuneLast, &r.AssetFirst, &r.AssetLast, &first, &last)
		if err != nil {
			return a, err
		}
		if first != 0 {
			r.First = time.Unix(0, first)
		}
		if last != 0 {
			r.Last = time.Unix(0, last)
		}
		r.PriceFirst, _ = big.NewRat(r.RuneFirst, r.AssetFirst).Float64()
		r.PriceLast, _ = big.NewRat(r.RuneLast, r.AssetLast).Float64()
		a = append(a, r)
	}
	return a, rows.Err()
}
