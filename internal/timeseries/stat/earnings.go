package stat

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

func querySelectTimestampInSeconds(targetColumn, intervalValueNumber string) string {
	return fmt.Sprintf(
		"EXTRACT(EPOCH FROM (date_trunc(%s, to_timestamp(%s/1000000000/300*300))))::BIGINT AS startTime", intervalValueNumber, targetColumn)
}

func GetEarningsTimeSeries(ctx context.Context, intervalStr string, from, to time.Time) (oapigen.EarningsHistoryResponse, error) {
	interval, err := intervalFromJSONParam(intervalStr)
	if err != nil {
		return oapigen.EarningsHistoryResponse{}, err
	}
	window := Window{
		From:  from,
		Until: to,
	}

	timestamps, window, err := generateBuckets(ctx, interval, window)
	if len(timestamps) == 0 {
		return oapigen.EarningsHistoryResponse{}, errors.New("no buckets were generated for the given timeframe")
	}

	// GET DATA
	liquidityFeesByPoolQ := fmt.Sprintf(`
		SELECT SUM(liq_fee_in_rune_E8), %s, pool
		FROM swap_events
		WHERE block_timestamp >= $1 AND block_timestamp < $2
		GROUP BY startTime, pool
	`, querySelectTimestampInSeconds("block_timestamp", "$3"))

	liquidityFeesByPoolRows, err := DBQuery(ctx, liquidityFeesByPoolQ, window.From.UnixNano(), window.Until.UnixNano(), dbIntervalName[interval])
	if err != nil {
		return oapigen.EarningsHistoryResponse{}, err
	}
	defer liquidityFeesByPoolRows.Close()

	bondingRewardsQ := fmt.Sprintf(`
	SELECT SUM(bond_e8), %s
	FROM rewards_events
	WHERE block_timestamp >= $1 AND block_timestamp < $2
	GROUP BY startTime
	`, querySelectTimestampInSeconds("block_timestamp", "$3"))

	bondingRewardsRows, err := DBQuery(ctx, bondingRewardsQ, window.From.UnixNano(), window.Until.UnixNano(), dbIntervalName[interval])
	if err != nil {
		return oapigen.EarningsHistoryResponse{}, err
	}

	poolRewardsQ := fmt.Sprintf(`
	SELECT SUM(rune_E8), %s, pool
	FROM rewards_event_entries
	WHERE block_timestamp >= $1 AND block_timestamp < $2
	GROUP BY startTime, pool
	`, querySelectTimestampInSeconds("block_timestamp", "$3"))

	poolRewardsRows, err := DBQuery(ctx, poolRewardsQ, window.From.UnixNano(), window.Until.UnixNano(), dbIntervalName[interval])
	if err != nil {
		return oapigen.EarningsHistoryResponse{}, err
	}
	defer poolRewardsRows.Close()

	// PROCESS DATA
	// Create aggregate variables to be filled with row results

	intervalTotalLiquidityFees := make(map[Second]int64)
	var metaTotalLiquidityFees int64

	// NOTE: BondingRewards are total bonding rewards sent from reserve to nodes. They equal
	// the exact earnings  (BondingRewards = BondingEarnings = share of fees + block rewards)
	intervalTotalBondingRewards := make(map[Second]int64)
	var metaTotalBondingRewards int64

	// NOTE: Pool rewards are pool rewards sent from reserve (+) or sent to nodes (-). They
	// are the difference between share of rewards (fees + block) and the fees collected by
	// the pool
	intervalTotalPoolRewards := make(map[Second]int64)
	var metaTotalPoolRewards int64

	// NOTE: PoolEarnings = PoolRewards + LiquidityFees
	intervalEarningsByPool := make(map[Second]map[string]int64)
	metaEarningsByPool := make(map[string]int64)

	// Store query results into aggregate variables
	for liquidityFeesByPoolRows.Next() {
		var liquidityFeeE8 int64
		var startTime Second
		var pool string
		err := liquidityFeesByPoolRows.Scan(&liquidityFeeE8, &startTime, &pool)
		if err != nil {
			return oapigen.EarningsHistoryResponse{}, err
		}

		if intervalEarningsByPool[startTime] == nil {
			intervalEarningsByPool[startTime] = make(map[string]int64)
		}

		// Add fees to earnings by pool
		intervalEarningsByPool[startTime][pool] += liquidityFeeE8
		metaEarningsByPool[pool] += liquidityFeeE8

		// Add fees to total fees aggregate
		intervalTotalLiquidityFees[startTime] += liquidityFeeE8
		metaTotalLiquidityFees += liquidityFeeE8
	}

	for bondingRewardsRows.Next() {
		var bondingRewards int64
		var startTime Second
		err := bondingRewardsRows.Scan(&bondingRewards, &startTime)
		if err != nil {
			return oapigen.EarningsHistoryResponse{}, err
		}

		// Add rewards to total bonding rewards
		intervalTotalBondingRewards[startTime] += bondingRewards
		metaTotalBondingRewards += bondingRewards
	}

	for poolRewardsRows.Next() {
		var runeE8 int64
		var startTime Second
		var pool string
		err := poolRewardsRows.Scan(&runeE8, &startTime, &pool)
		if err != nil {
			return oapigen.EarningsHistoryResponse{}, err
		}

		if intervalEarningsByPool[startTime] == nil {
			intervalEarningsByPool[startTime] = make(map[string]int64)
		}

		// Add rewards to earnings by pool
		intervalEarningsByPool[startTime][pool] += runeE8
		metaEarningsByPool[pool] += runeE8

		// Add rewards to total pool rewards
		intervalTotalPoolRewards[startTime] += runeE8
		metaTotalPoolRewards += runeE8
	}

	// From earnings by pool get all Pools and build meta EarningsHistoryIntervalPools
	poolsList := make([]string, 0, len(metaEarningsByPool))
	metaEarningsIntervalPools := make([]oapigen.EarningsHistoryIntervalPool, 0, len(metaEarningsByPool))
	for pool, earnings := range metaEarningsByPool {
		poolsList = append(poolsList, pool)
		metaEarningsIntervalPool := oapigen.EarningsHistoryIntervalPool{
			Pool:     pool,
			Earnings: intStr(earnings),
		}
		metaEarningsIntervalPools = append(metaEarningsIntervalPools, metaEarningsIntervalPool)
	}

	// Build Response and Meta
	earnings := oapigen.EarningsHistoryResponse{
		Meta:      buildEarningsInterval(timestamps[0], TimeToSecond(window.Until), metaTotalLiquidityFees, metaTotalPoolRewards, metaTotalBondingRewards, metaEarningsIntervalPools),
		Intervals: make([]oapigen.EarningsHistoryInterval, 0, len(timestamps)),
	}

	// Build and add Intervals to Response
	for timestampIndex, timestamp := range timestamps {
		// get end timestamp
		var endTime Second
		if timestampIndex >= (len(timestamps) - 1) {
			endTime = TimeToSecond(window.Until)
		} else {
			endTime = timestamps[timestampIndex+1]
		}

		earningsByPool := intervalEarningsByPool[timestamp]

		// Process pools
		earningsIntervalPools := make([]oapigen.EarningsHistoryIntervalPool, 0, len(poolsList))
		for _, pool := range poolsList {
			var earningsIntervalPool oapigen.EarningsHistoryIntervalPool
			earningsIntervalPool.Earnings = intStr(earningsByPool[pool])
			earningsIntervalPool.Pool = pool
			earningsIntervalPools = append(earningsIntervalPools, earningsIntervalPool)
		}

		// build resulting interval
		earningsIntervalAddr := buildEarningsInterval(timestamp, endTime, intervalTotalLiquidityFees[timestamp], intervalTotalPoolRewards[timestamp], intervalTotalBondingRewards[timestamp], earningsIntervalPools)

		earnings.Intervals = append(earnings.Intervals, earningsIntervalAddr)
	}

	return earnings, nil
}

func buildEarningsInterval(startTime, endTime Second,
	totalLiquidityFees, totalPoolRewards, totalBondingRewards int64,
	earningsIntervalPools []oapigen.EarningsHistoryIntervalPool) oapigen.EarningsHistoryInterval {

	liquidityEarnings := totalPoolRewards + totalLiquidityFees
	earnings := liquidityEarnings + totalBondingRewards
	blockRewards := earnings - totalLiquidityFees

	return oapigen.EarningsHistoryInterval{
		StartTime:         intStr(startTime.ToI()),
		EndTime:           intStr(endTime.ToI()),
		LiquidityFees:     intStr(totalLiquidityFees),
		BlockRewards:      intStr(blockRewards),
		BondingEarnings:   intStr(totalBondingRewards),
		LiquidityEarnings: intStr(liquidityEarnings),
		Earnings:          intStr(earnings),
		Pools:             earningsIntervalPools,
	}
}

/*

	/* TODO: For depths, will use this to get the average depth for each interval
	runeDepthsByPoolQ := `WITH first_timestamp AS (
		SELECT MAX(block_timestamp) FROM block_pool_depths WHERE block_timestamp <= $1
	)
	SELECT
	SUM(2 * rune_e8),
	date_trunc($3, date_trunc($3, to_timestamp(block_timestamp/1000000000/300*300))) as startTime
	pool,
	WHERE block_timestamp >= first_timestamp AND block_timestamp <= $2
	GROUP BY startTime, pool
	`
*/
