package stat

import (
	"context"
	"fmt"

	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

// Query to get amount in rune for a colunm by multiplying price * column_amount.
// Query nees a join with block_pool_depth for each row and its alias provided in as depthTableAlias
func querySelectAssetAmountInRune(assetAmountColumn, depthTableAlias string) string {
	return fmt.Sprintf("CAST((CAST(%s.rune_e8 as NUMERIC) / CAST(%s.asset_e8 as NUMERIC) * %s) as bigint)", depthTableAlias, depthTableAlias, assetAmountColumn)
}

func GetLiquidityHistory(ctx context.Context, buckets db.Buckets, pool string) (oapigen.LiquidityHistoryResponse, error) {
	window := buckets.Window()
	timestamps := buckets.Timestamps[:len(buckets.Timestamps)-1]

	// NOTE: pool filter and arguments are the same in all queries
	var poolFilter string
	queryArguments := []interface{}{window.From.ToNano(), window.Until.ToNano()}
	if pool != "*" {
		poolFilter = "base.pool = $3 AND "
		queryArguments = append(queryArguments, pool)
	}

	// GET DATA
	// TODO(acsaba): To get the depths for a given timestamp, we join by block_timestamp, assuming
	// there will always be a row in block_pool_depths as depth is being changed by the event
	// itself on that block. This won't be the case if for some reason there are other events
	// and the depth ends up being the same than previous block as new row won't be stored
	// Even though unlikely, we need to guard against this.
	depositsQ := `
	SELECT
	SUM(` + querySelectAssetAmountInRune("base.asset_E8", "bpd") + `+ base.rune_E8),` +
		db.SelectTruncatedTimestamp("base.block_timestamp", buckets) + ` AS start_time
	FROM stake_events AS base
	INNER JOIN block_pool_depths bpd ON bpd.block_timestamp = base.block_timestamp
	WHERE ` + poolFilter + `$1 <= base.block_timestamp AND base.block_timestamp < $2
	GROUP BY start_time
	`

	depositsRows, err := db.Query(ctx, depositsQ, queryArguments...)

	if err != nil {
		return oapigen.LiquidityHistoryResponse{}, err
	}
	defer depositsRows.Close()

	withdrawalsQ := `
	SELECT
	SUM(` + querySelectAssetAmountInRune("base.emit_asset_E8", "bpd") + `+ base.emit_rune_E8),` +
		db.SelectTruncatedTimestamp("base.block_timestamp", buckets) + ` AS start_time
	FROM unstake_events AS base
	INNER JOIN block_pool_depths bpd ON bpd.block_timestamp = base.block_timestamp
	WHERE ` + poolFilter + `$1 <= base.block_timestamp AND base.block_timestamp < $2
	GROUP BY start_time
	`

	withdrawalsRows, err := db.Query(ctx, withdrawalsQ, queryArguments...)

	if err != nil {
		return oapigen.LiquidityHistoryResponse{}, err
	}
	defer withdrawalsRows.Close()

	// PROCESS DATA
	// Create aggregate variables to be filled with row results
	intervalTotalDeposits := make(map[db.Second]int64)
	var metaTotalDeposits int64

	intervalTotalWithdrawals := make(map[db.Second]int64)
	var metaTotalWithdrawals int64

	// Store query results into aggregate variables
	for depositsRows.Next() {
		var totalDeposits int64
		var startTime db.Second
		err := depositsRows.Scan(&totalDeposits, &startTime)
		if err != nil {
			return oapigen.LiquidityHistoryResponse{}, err
		}
		intervalTotalDeposits[startTime] = totalDeposits
		metaTotalDeposits += totalDeposits
	}

	for withdrawalsRows.Next() {
		var totalWithdrawals int64
		var startTime db.Second
		err := withdrawalsRows.Scan(&totalWithdrawals, &startTime)
		if err != nil {
			return oapigen.LiquidityHistoryResponse{}, err
		}
		intervalTotalWithdrawals[startTime] = totalWithdrawals
		metaTotalWithdrawals += totalWithdrawals
	}

	// Build Response And Meta
	liquidityChanges := oapigen.LiquidityHistoryResponse{
		Meta:      buildLiquidityItem(timestamps[0], window.Until, metaTotalWithdrawals, metaTotalDeposits),
		Intervals: make([]oapigen.LiquidityHistoryItem, 0, len(timestamps)),
	}

	// Build and add Items to Response
	for timestampIndex, timestamp := range timestamps {
		// get end timestamp
		var endTime db.Second
		if timestampIndex >= (len(timestamps) - 1) {
			endTime = window.Until
		} else {
			endTime = timestamps[timestampIndex+1]
		}

		withdrawals := intervalTotalWithdrawals[timestamp]
		deposits := intervalTotalDeposits[timestamp]

		liquidityChangesItem := buildLiquidityItem(timestamp, endTime, withdrawals, deposits)
		liquidityChanges.Intervals = append(liquidityChanges.Intervals, liquidityChangesItem)
	}

	return liquidityChanges, nil
}

func buildLiquidityItem(startTime, endTime db.Second, withdrawals, deposits int64) oapigen.LiquidityHistoryItem {
	return oapigen.LiquidityHistoryItem{
		StartTime:   intStr(startTime.ToI()),
		EndTime:     intStr(endTime.ToI()),
		Withdrawals: intStr(withdrawals),
		Deposits:    intStr(deposits),
		Net:         intStr(deposits - withdrawals),
	}
}
