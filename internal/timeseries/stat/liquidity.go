package stat

import (
	"context"
	"fmt"
	"strconv"

	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/util/miderr"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

// Query to get amount in rune for a colunm by multiplying price * column_amount.
// Query nees a join with block_pool_depth for each row and its alias provided in as depthTableAlias
func querySelectAssetAmountInRune(assetAmountColumn, depthTableAlias string) string {
	return fmt.Sprintf("CAST((CASE WHEN %[1]s.asset_e8 <>0 "+
		"THEN (CAST(%[1]s.rune_e8 as NUMERIC) / CAST(%[1]s.asset_e8 as NUMERIC) * %[2]s) "+
		" ELSE 0 END) as bigint)",
		depthTableAlias, assetAmountColumn)
}

type liquidityBucket struct {
	assetVolume               int64
	runeVolume                int64
	volume                    int64
	impermanentLossProtection int64
	count                     int64
}

type liquidityOneTableResult struct {
	total   liquidityBucket
	buckets map[db.Second]liquidityBucket
}

func liquidityChangesFromTable(
	ctx context.Context, buckets db.Buckets, pool string,
	table, assetColumn, runeColumn, impLossProtColumn string) (
	ret liquidityOneTableResult, err error) {
	window := buckets.Window()

	// NOTE: pool filter and arguments are the same in all queries
	var poolFilter string
	queryArguments := []interface{}{window.From.ToNano(), window.Until.ToNano()}
	if pool != "*" {
		poolFilter = "base.pool = $3 AND "
		queryArguments = append(queryArguments, pool)
	}

	impLossClause := "0"
	if impLossProtColumn != "" {
		impLossClause = "SUM(base." + impLossProtColumn + ")"
	}

	// GET DATA
	// TODO(acsaba): To get the depths for a given timestamp, we join by block_timestamp, assuming
	// there will always be a row in block_pool_depths as depth is being changed by the event
	// itself on that block. This won't be the case if for some reason there are other events
	// and the depth ends up being the same than previous block as new row won't be stored
	// Even though unlikely, we need to guard against this.
	query := `
	SELECT
		COUNT(*) AS count,
		SUM(` + querySelectAssetAmountInRune("base."+assetColumn, "bpd") + `) AS asset_sum,
		SUM(base.` + runeColumn + `) as rune_sum,
		` + impLossClause + ` AS imp_loss,
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

	ret.buckets = map[db.Second]liquidityBucket{}

	// Store query results into aggregate variables
	for rows.Next() {
		var bucket liquidityBucket
		var startTime db.Second
		err = rows.Scan(
			&bucket.count, &bucket.assetVolume, &bucket.runeVolume,
			&bucket.impermanentLossProtection, &startTime)
		if err != nil {
			return
		}
		bucket.volume = bucket.assetVolume + bucket.runeVolume

		ret.buckets[startTime] = bucket
		ret.total.assetVolume += bucket.assetVolume
		ret.total.runeVolume += bucket.runeVolume
		ret.total.volume += bucket.volume
		ret.total.count += bucket.count
		ret.total.impermanentLossProtection += bucket.impermanentLossProtection
	}

	return
}

func GetLiquidityHistory(ctx context.Context, buckets db.Buckets, pool string) (
	ret oapigen.LiquidityHistoryResponse, err error) {
	window := buckets.Window()

	deposits, err := liquidityChangesFromTable(ctx, buckets, pool,
		"stake_events", "asset_E8", "rune_E8", "")
	if err != nil {
		return
	}

	withdraws, err := liquidityChangesFromTable(ctx, buckets, pool,
		"unstake_events", "emit_asset_E8", "emit_rune_E8", "imp_loss_protection_e8")
	if err != nil {
		return
	}

	usdPrices, err := USDPriceHistory(ctx, buckets)
	if err != nil {
		return
	}

	if len(usdPrices) != buckets.Count() {
		err = miderr.InternalErr("Misalligned buckets")
		return
	}

	ret = oapigen.LiquidityHistoryResponse{
		Meta: buildLiquidityItem(
			window.From, window.Until, withdraws.total, deposits.total,
			usdPrices[len(usdPrices)-1].RunePriceUSD),
		Intervals: make([]oapigen.LiquidityHistoryItem, 0, buckets.Count()),
	}

	for i := 0; i < buckets.Count(); i++ {
		timestamp, endTime := buckets.Bucket(i)
		if usdPrices[i].Window.From != timestamp {
			err = miderr.InternalErr("Misalligned buckets")
		}

		withdrawals := withdraws.buckets[timestamp]
		deposits := deposits.buckets[timestamp]

		liquidityChangesItem := buildLiquidityItem(
			timestamp, endTime, withdrawals, deposits, usdPrices[i].RunePriceUSD)
		ret.Intervals = append(ret.Intervals, liquidityChangesItem)
	}

	return ret, nil
}

func buildLiquidityItem(
	startTime, endTime db.Second,
	withdrawals, deposits liquidityBucket,
	runePriceUSD float64) oapigen.LiquidityHistoryItem {
	return oapigen.LiquidityHistoryItem{
		StartTime:                     intStr(startTime.ToI()),
		EndTime:                       intStr(endTime.ToI()),
		AddAssetLiquidityVolume:       intStr(deposits.assetVolume),
		AddRuneLiquidityVolume:        intStr(deposits.runeVolume),
		AddLiquidityVolume:            intStr(deposits.volume),
		AddLiquidityCount:             intStr(deposits.count),
		WithdrawAssetVolume:           intStr(withdrawals.assetVolume),
		WithdrawRuneVolume:            intStr(withdrawals.runeVolume),
		ImpermanentLossProtectionPaid: intStr(withdrawals.impermanentLossProtection),
		WithdrawVolume:                intStr(withdrawals.volume),
		WithdrawCount:                 intStr(withdrawals.count),
		Net:                           intStr(deposits.volume - withdrawals.volume),
		RunePriceUSD:                  floatStr(runePriceUSD),
	}
}

func floatStr(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
