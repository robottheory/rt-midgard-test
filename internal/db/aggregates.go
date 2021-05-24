package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"gitlab.com/thorchain/midgard/internal/util/jobs"
	"gitlab.com/thorchain/midgard/internal/util/timer"
)

const (
	aggregatesInitialDelay    = 10 * time.Second
	aggregatesRefreshInterval = 5 * time.Minute
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

func AggregatesDdl() string {
	var b strings.Builder
	fmt.Fprint(&b, `
-- version 1

DROP SCHEMA IF EXISTS midgard_agg CASCADE;
CREATE SCHEMA midgard_agg;

`)

	for name, aggregate := range aggregates {
		for _, bucket := range intervals {
			if bucket.exact {
				bucketField := fmt.Sprintf("time_bucket('%d', block_timestamp)",
					bucket.minDuration*1000000000)
				q := strings.TrimSpace(fmt.Sprintf(aggregate.lowerQuery, bucketField))
				fmt.Fprintf(&b, `
CREATE MATERIALIZED VIEW midgard_agg.%s_%s
WITH (timescaledb.continuous) AS
%s
WITH NO DATA;
`, name, bucket.name, q)
			} else {
				bucketField := fmt.Sprintf("nano_trunc('%s', d.bucket_start)",
					bucket.name)
				q := strings.TrimSpace(fmt.Sprintf(aggregate.higherQuery, bucketField))
				fmt.Fprintf(&b, `
CREATE VIEW midgard_agg.%s_%s AS
%s;
`, name, bucket.name, q)
			}
		}
	}
	return b.String()
}

func DropAggregates() (err error) {
	_, err = theDB.Exec(`
		DROP SCHEMA IF EXISTS midgard_agg CASCADE;
		DELETE FROM midgard.constants WHERE key = '` + aggregatesDdlHashKey + `';
	`)
	return
}

var aggregatesRefreshTimer = timer.NewTimer("aggregates_refresh")

func refreshAggregates(ctx context.Context) {
	stop := aggregatesRefreshTimer.One()
	log.Debug().Msg("Refreshing aggregates")

	refreshEnd := LastBlockTimestamp() - 5*60*1000000000
	for name := range aggregates {
		for _, bucket := range intervals {
			if !bucket.exact {
				continue
			}
			if ctx.Err() != nil {
				return
			}
			q := fmt.Sprintf("CALL refresh_continuous_aggregate('midgard_agg.%s_%s', NULL, '%d')",
				name, bucket.name, refreshEnd)
			_, err := theDB.Exec(q)
			if err != nil {
				log.Error().Err(err).Msgf("Refreshing %s_%s", name, bucket.name)
			}
		}
	}

	log.Debug().Msg("Refreshing done")
	stop()
}

func StartAggregatesRefresh(ctx context.Context) *jobs.Job {
	log.Info().Msg("Starting aggregates refresh job")
	job := jobs.Start("AggregatesRefresh", func() {
		time.Sleep(aggregatesInitialDelay)
		for {
			if ctx.Err() != nil {
				log.Info().Msg("Shutdown aggregates refresh job")
				return
			}
			refreshAggregates(ctx)
			jobs.Sleep(ctx, aggregatesRefreshInterval)
		}
	})
	return &job
}
