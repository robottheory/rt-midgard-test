package stat

import (
	"context"
	"fmt"
	"strconv"

	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

func querySelectTimestampInSeconds(targetColumn, intervalValueNumber string) string {
	return fmt.Sprintf(
		"EXTRACT(EPOCH FROM (date_trunc(%s, to_timestamp(%s/1000000000/300*300))))::BIGINT AS start_time", intervalValueNumber, targetColumn)
}

func GetEarningsTimeSeries(ctx context.Context, buckets db.Buckets) (oapigen.EarningsHistoryResponse, error) {
	window := buckets.Window()
	timestamps := buckets.Timestamps[:len(buckets.Timestamps)-1]

	// GET DATA
	liquidityFeesByPoolQ := fmt.Sprintf(`
		SELECT SUM(liq_fee_in_rune_E8), %s, pool
		FROM swap_events
		WHERE block_timestamp >= $1 AND block_timestamp < $2
		GROUP BY start_time, pool
	`, querySelectTimestampInSeconds("block_timestamp", "$3"))

	liquidityFeesByPoolRows, err := db.Query(ctx,
		liquidityFeesByPoolQ, window.From.ToNano(), window.Until.ToNano(),
		db.DBIntervalName[buckets.Interval])
	if err != nil {
		return oapigen.EarningsHistoryResponse{}, err
	}
	defer liquidityFeesByPoolRows.Close()

	bondingRewardsQ := fmt.Sprintf(`
	SELECT SUM(bond_e8), %s
	FROM rewards_events
	WHERE block_timestamp >= $1 AND block_timestamp < $2
	GROUP BY start_time
	`, querySelectTimestampInSeconds("block_timestamp", "$3"))

	bondingRewardsRows, err := db.Query(ctx,
		bondingRewardsQ, window.From.ToNano(), window.Until.ToNano(),
		db.DBIntervalName[buckets.Interval])
	if err != nil {
		return oapigen.EarningsHistoryResponse{}, err
	}

	poolRewardsQ := fmt.Sprintf(`
	SELECT SUM(rune_E8), %s, pool
	FROM rewards_event_entries
	WHERE block_timestamp >= $1 AND block_timestamp < $2
	GROUP BY start_time, pool
	`, querySelectTimestampInSeconds("block_timestamp", "$3"))

	poolRewardsRows, err := db.Query(ctx,
		poolRewardsQ, window.From.ToNano(), window.Until.ToNano(),
		db.DBIntervalName[buckets.Interval])
	if err != nil {
		return oapigen.EarningsHistoryResponse{}, err
	}
	defer poolRewardsRows.Close()

	nodeStartCountQ := `
	SELECT SUM (CASE WHEN current = 'active' THEN 1 WHEN former = 'active' THEN -1 else 0 END)
	FROM update_node_account_status_events
	WHERE block_timestamp <= $1
	`
	var nodeStartCount int64
	err = timeseries.QueryOneValue(&nodeStartCount, ctx, nodeStartCountQ, window.From.ToNano())
	if err != nil {
		return oapigen.EarningsHistoryResponse{}, err
	}

	nodeDeltasQ := `
	SELECT
	SUM(CASE WHEN current = 'active' THEN 1 WHEN former = 'active' THEN -1 else 0 END) AS delta,
	(block_timestamp/1000000000)::BIGINT AS seconds_timestamp
	FROM update_node_account_status_events
	WHERE $1 < block_timestamp AND block_timestamp < $2
	GROUP BY seconds_timestamp
	ORDER BY seconds_timestamp
	`
	nodeDeltasRows, err := db.Query(ctx, nodeDeltasQ, window.From.ToNano(), window.Until.ToNano())
	if err != nil {
		return oapigen.EarningsHistoryResponse{}, err
	}
	defer nodeDeltasRows.Close()

	// PROCESS DATA
	// Create aggregate variables to be filled with row results

	intervalTotalLiquidityFees := make(map[db.Second]int64)
	var metaTotalLiquidityFees int64

	// NOTE: BondingRewards are total bonding rewards sent from reserve to nodes. They equal
	// the exact earnings  (BondingRewards = BondingEarnings = share of fees + block rewards)
	intervalTotalBondingRewards := make(map[db.Second]int64)
	var metaTotalBondingRewards int64

	// NOTE: Pool rewards are pool rewards sent from reserve (+) or sent to nodes (-). They
	// are the difference between share of rewards (fees + block) and the fees collected by
	// the pool
	intervalTotalPoolRewards := make(map[db.Second]int64)
	var metaTotalPoolRewards int64

	// NOTE: PoolEarnings = PoolRewards + LiquidityFees
	intervalEarningsByPool := make(map[db.Second]map[string]int64)
	metaEarningsByPool := make(map[string]int64)

	intervalNodeCountWeightedSum := make(map[db.Second]int64)
	var metaNodeCountWeightedSum int64

	// Store query results into aggregate variables
	for liquidityFeesByPoolRows.Next() {
		var liquidityFeeE8 int64
		var startTime db.Second
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
		var startTime db.Second
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
		var startTime db.Second
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

	// NOTE: Node Weighted Sums:
	// Transverse node updates calculating aggregated weighted sums
	// Start with node count from genesis up to the first interval
	// timestamp (NodeStartCount). Look for the next timestamp where a change in node
	// count happens and add weighted sum using where we started up to a
	// second before the change is detected for all time intervals in between.
	// Then update the counts and starting timestamps and keep
	// on interating until there are no more changes.
	// This is needed to get node avg count for each interval
	nodesCurrentTimestampIndex := 0
	nodesLastCount := nodeStartCount
	nodesLastCountTimestamp := timestamps[0]
	for nodeDeltasRows.Next() {
		var delta int64
		var deltaTimestamp db.Second
		err := nodeDeltasRows.Scan(&delta, &deltaTimestamp)
		if err != nil {
			return oapigen.EarningsHistoryResponse{}, err
		}

		for (nodesCurrentTimestampIndex < len(timestamps)-1) && timestamps[nodesCurrentTimestampIndex+1] < deltaTimestamp {
			// if delta timestamp is greater than the interval timestamp, the node count
			// didn't change from current timestamp to the start of next interval so weights
			// for the remaining of the interval are computed using nodesLastCount

			// Add weighted count up to the end of the interval
			weightedCount := (timestamps[nodesCurrentTimestampIndex+1].ToI() - nodesLastCountTimestamp.ToI()) * nodesLastCount
			intervalNodeCountWeightedSum[timestamps[nodesCurrentTimestampIndex]] += weightedCount
			metaNodeCountWeightedSum += weightedCount
			// Move to the next interval
			nodesCurrentTimestampIndex++
			nodesLastCountTimestamp = timestamps[nodesCurrentTimestampIndex]
		}

		// Add last weighted sum to interval and global aggregates (last count happend up to deltaTimestamp - 1)
		weightedCount := (deltaTimestamp.ToI() - nodesLastCountTimestamp.ToI()) * nodesLastCount
		intervalNodeCountWeightedSum[timestamps[nodesCurrentTimestampIndex]] += weightedCount
		metaNodeCountWeightedSum += weightedCount

		// Update Count and Last timestamps
		nodesLastCount += delta
		nodesLastCountTimestamp = deltaTimestamp
	}

	// Advance until last interval adding corresponding weighted counts
	for nodesCurrentTimestampIndex < (len(timestamps) - 1) {
		// Add weighted count up to the end of the interval
		weightedCount := (timestamps[nodesCurrentTimestampIndex+1].ToI() - nodesLastCountTimestamp.ToI()) * nodesLastCount
		intervalNodeCountWeightedSum[timestamps[nodesCurrentTimestampIndex]] += weightedCount
		metaNodeCountWeightedSum += weightedCount
		// Move to the next interval
		nodesCurrentTimestampIndex++
		nodesLastCountTimestamp = timestamps[nodesCurrentTimestampIndex]
	}
	// Add last weighted count
	endTimeInt := window.Until
	if nodesLastCountTimestamp < (endTimeInt - 1) {
		weightedCount := (endTimeInt - nodesLastCountTimestamp).ToI() * nodesLastCount
		intervalNodeCountWeightedSum[timestamps[nodesCurrentTimestampIndex]] += weightedCount
		metaNodeCountWeightedSum += weightedCount
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
		Meta: buildEarningsInterval(
			timestamps[0], window.Until, metaTotalLiquidityFees, metaTotalPoolRewards,
			metaTotalBondingRewards, metaNodeCountWeightedSum, metaEarningsIntervalPools),
		Intervals: make([]oapigen.EarningsHistoryInterval, 0, len(timestamps)),
	}

	// Build and add Intervals to Response
	for timestampIndex, timestamp := range timestamps {
		// get end timestamp
		var endTime db.Second
		if timestampIndex >= (len(timestamps) - 1) {
			endTime = window.Until
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
		earningsIntervalAddr := buildEarningsInterval(timestamp, endTime, intervalTotalLiquidityFees[timestamp], intervalTotalPoolRewards[timestamp], intervalTotalBondingRewards[timestamp], intervalNodeCountWeightedSum[timestamp], earningsIntervalPools)

		earnings.Intervals = append(earnings.Intervals, earningsIntervalAddr)
	}

	return earnings, nil
}

func buildEarningsInterval(startTime, endTime db.Second,
	totalLiquidityFees, totalPoolRewards, totalBondingRewards, nodeCountWeightedSum int64,
	earningsIntervalPools []oapigen.EarningsHistoryIntervalPool) oapigen.EarningsHistoryInterval {

	liquidityEarnings := totalPoolRewards + totalLiquidityFees
	earnings := liquidityEarnings + totalBondingRewards
	blockRewards := earnings - totalLiquidityFees

	avgNodeCount := float64(nodeCountWeightedSum) / float64(endTime-startTime)

	return oapigen.EarningsHistoryInterval{
		StartTime:         intStr(startTime.ToI()),
		EndTime:           intStr(endTime.ToI()),
		LiquidityFees:     intStr(totalLiquidityFees),
		BlockRewards:      intStr(blockRewards),
		BondingEarnings:   intStr(totalBondingRewards),
		LiquidityEarnings: intStr(liquidityEarnings),
		Earnings:          intStr(earnings),
		AvgNodeCount:      floatStr(avgNodeCount),
		Pools:             earningsIntervalPools,
	}
}

func floatStr(f float64) string {
	return strconv.FormatFloat(f, 'f', 2, 64)
}
