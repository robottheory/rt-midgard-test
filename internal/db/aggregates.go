package db

import (
	"fmt"
	"strings"
)

// TimescaleDB does not support continuous aggregates where the time buckets
// are not constant size (ie., month, year), so for those we need to aggregate
// the daily aggregate into higher aggregates.
//
// The `lowerQuery` is the query template for creating the materialized view
// from a hypertable. Should contain a single %s hole which will be filled by
// a `time_bucket(..., block_timestamp)`, and this column should be named
// `bucket_start`.
//
// The `higherQuery` is the query template for creating views from the daily aggregate.
// Should contain a single %s hole which will be filled by a `nano_trunc(..., d.bucket_start)`,
// so the daily aggregate should be aliased as `d`.
//
// TODO(huginn): think on how to simplify this.
type aggregateParams struct {
	lowerQuery  string
	higherQuery string
}

var aggregates = map[string]aggregateParams{
	"pool_depths": {`
SELECT
    pool,
    last(asset_e8, block_timestamp) as asset_e8,
    last(rune_e8, block_timestamp) as rune_e8,
    %s as bucket_start
FROM block_pool_depths
GROUP BY bucket_start, pool`, `
SELECT
    pool,
    last(asset_e8, d.bucket_start) as asset_e8,
    last(rune_e8, d.bucket_start) as rune_e8,
    %s as bucket_start
FROM midgard_agg.pool_depths_day d
GROUP BY bucket_start, pool`},
}

// TODO(huginn): consolidate all these maps
var fixedBuckets = map[string]int64{
	"5min": 60 * 5,
	"hour": 60 * 60,
	"day":  60 * 60 * 24,
}

var higherBuckets = []string{"week", "month", "quarter", "year"}

func AggregatesDdl() string {
	var b strings.Builder
	fmt.Fprint(&b, `
-- version 1

DROP SCHEMA IF EXISTS midgard_agg CASCADE;
CREATE SCHEMA midgard_agg;

`)

	for name, aggregate := range aggregates {
		for bucketName, bucket := range fixedBuckets {
			bucketField := fmt.Sprintf("time_bucket('%d', block_timestamp)",
				bucket*1000000000)
			q := strings.TrimSpace(fmt.Sprintf(aggregate.lowerQuery, bucketField))
			fmt.Fprintf(&b, `
CREATE MATERIALIZED VIEW midgard_agg.%s_%s
WITH (timescaledb.continuous) AS
%s
WITH NO DATA;
`, name, bucketName, q)
		}

		for _, bucket := range higherBuckets {
			bucketField := fmt.Sprintf("nano_trunc('%s', d.bucket_start)",
				bucket)
			q := strings.TrimSpace(fmt.Sprintf(aggregate.higherQuery, bucketField))
			fmt.Fprintf(&b, `
CREATE VIEW midgard_agg.%s_%s AS
%s;
`, name, bucket, q)
		}
	}
	return b.String()
}
