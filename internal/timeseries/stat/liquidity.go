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
	return fmt.Sprintf("CAST((CAST(%s.rune_e8 as NUMERIC) / CAST(%s.asset_e8 as NUMERIC) * %s) as bigint)",
		depthTableAlias, depthTableAlias, assetAmountColumn)
}

type liquidityOneTableResult struct {
	totalVolume int64
	volume      map[db.Second]int64
}

func liquidityChangesFromTable(
	ctx context.Context, buckets db.Buckets, pool string,
	table, assetColumn, runeColumn string) (
	ret liquidityOneTableResult, err error) {
	window := buckets.Window()

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
	query := `
	SELECT
		SUM(` + querySelectAssetAmountInRune("base."+assetColumn, "bpd") + ` + base.` + runeColumn + `),
		` + db.SelectTruncatedTimestamp("base.block_timestamp", buckets) + ` AS start_time
	FROM ` + table + ` AS base
	INNER JOIN block_pool_depths bpd
	ON bpd.block_timestamp = base.block_timestamp AND bpd.pool = base.pool
	WHERE ` + poolFilter + `$1 <= base.block_timestamp AND base.block_timestamp < $2
	GROUP BY start_time
	`

	rows, err := db.Query(ctx, query, queryArguments...)

	if err != nil {
		return
	}
	defer rows.Close()

	ret.volume = map[db.Second]int64{}

	// Store query results into aggregate variables
	for rows.Next() {
		var totalVolume int64
		var startTime db.Second
		err = rows.Scan(&totalVolume, &startTime)
		if err != nil {
			return
		}
		ret.volume[startTime] = totalVolume
		ret.totalVolume += totalVolume
	}

	return
}

func GetLiquidityHistory(ctx context.Context, buckets db.Buckets, pool string) (oapigen.LiquidityHistoryResponse, error) {
	window := buckets.Window()

	deposits, err := liquidityChangesFromTable(ctx, buckets, pool,
		"stake_events", "asset_E8", "rune_E8")
	if err != nil {
		return oapigen.LiquidityHistoryResponse{}, err
	}

	withdraws, err := liquidityChangesFromTable(ctx, buckets, pool,
		"unstake_events", "emit_asset_E8", "emit_rune_E8")
	if err != nil {
		return oapigen.LiquidityHistoryResponse{}, err
	}

	liquidityChanges := oapigen.LiquidityHistoryResponse{
		Meta:      buildLiquidityItem(window.From, window.Until, withdraws.totalVolume, deposits.totalVolume),
		Intervals: make([]oapigen.LiquidityHistoryItem, 0, buckets.Count()),
	}

	for i := 0; i < buckets.Count(); i++ {
		timestamp, endTime := buckets.Bucket(i)

		withdrawals := withdraws.volume[timestamp]
		deposits := deposits.volume[timestamp]

		liquidityChangesItem := buildLiquidityItem(timestamp, endTime, withdrawals, deposits)
		liquidityChanges.Intervals = append(liquidityChanges.Intervals, liquidityChangesItem)
	}

	return liquidityChanges, nil
}

func buildLiquidityItem(startTime, endTime db.Second, withdrawals, deposits int64) oapigen.LiquidityHistoryItem {
	return oapigen.LiquidityHistoryItem{
		StartTime:         intStr(startTime.ToI()),
		EndTime:           intStr(endTime.ToI()),
		WithdrawVolume:    intStr(withdrawals),
		AddLiqudityVolume: intStr(deposits),
		Net:               intStr(deposits - withdrawals),
	}
}
