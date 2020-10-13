package timeseries

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"gitlab.com/thorchain/midgard/event"
)

type TxTransactions struct {
	Count uint            `json:"count"`
	Txs   []TxTransaction `json:"txs"`
}

// In multichain there will be also multiple IN transations, we probably need to update this.
type TxTransaction struct {
	EventType string    `json:"type"`
	Date      int64     `json:"date"`
	Height    int64     `json:"height"`
	Events    TxEvents  `json:"events"`
	In        TxInOut   `json:"in"`
	Out       []TxInOut `json:"out"`
	Pool      string    `json:"pool"`
	Status    string    `json:"status"`
}

type TxEvents struct {
	Fee        uint64  `json:"fee"`
	StakeUnits int64   `json:"stakeUnits"`
	Slip       float64 `json:"slip"`
}

type TxCoin struct {
	Amount int64  `json:"amount"`
	Asset  string `json:"asset"`
}

// PriceTarget is not set currently, we should probably remove it.
// Also Assymetry and Withdraw is set only for some event types,
// we may consider redesigning the API or returning values only if it makes sense.
type TxOptions struct {
	Asymmetry           int64 `json:"asymmetry"`
	PriceTarget         int64 `json:"priceTarget"`
	WithdrawBasisPoints int64 `json:"withdrawBasisPoints"`
}

type TxInOut struct {
	Address string    `json:"address"`
	Coins   []TxCoin  `json:"coins"`
	Memo    string    `json:"memo"`
	TxId    string    `json:"txId"`
	Options TxOptions `json:"options"`
}

/* NOTE(elfedy): In the docs for ("/v1/doc") event and tx are used almost interchangeably,
 but there seem to be three different conceps regarding this endpoint that should probably
 be well understood (and perhaps more clearly documented):
	- "inbound transaction event": event that the doc for this endpoint refers to as "event".
		Describes an operation that is triggerd by one (or sometimes two in case of stakes)
		inbound transactions sent from an external address to a Vault address.
	- "Thorchain event": events emitted by Thorchain that Midgard parses to calculate the state
		of the network at a given point in time. An "inbound transaction event" generates several
		Thorchain events to be emited. These are the events that are stored in Midgard tables with
		*_event prefixes.
	- "Transactions": transactions that Thorchain validators observe and process. There are
	"in" transactions that are sent from an extrenal address to a Thorchain vault and there are
	"out" transactions that are sent form a Thorchain vault to an external address.
*/

// Gets a list of operations generated by external transactions and return its associated data
func TxList(ctx context.Context, moment time.Time, params map[string]string) (TxTransactions, error) {
	// CHECK PARAMS
	// give latest value if zero moment
	_, timestamp, _ := LastBlock()
	if moment.IsZero() {
		moment = timestamp
	} else if timestamp.Before(moment) {
		return TxTransactions{}, errBeyondLast
	}

	// check limit param
	if params["limit"] == "" {
		return TxTransactions{}, errors.New("Query parameter limit is required")
	}
	limit, err := strconv.ParseUint(params["limit"], 10, 64)
	if err != nil || limit < 1 || limit > 50 {
		return TxTransactions{}, errors.New("limit must be an integer between 1 and 50")
	}

	// check offset param
	if params["offset"] == "" {
		return TxTransactions{}, errors.New("Query parameter offset is required")
	}
	offset, err := strconv.ParseUint(params["offset"], 10, 64)
	if err != nil || offset < 0 {
		return TxTransactions{}, errors.New("offset must be a positive integer")
	}

	// build types from type param
	types := make([]string, 0)
	for k := range txInSelectQueries {
		types = append(types, k)
	}
	if params["type"] != "" {
		types = strings.Split(params["type"], ",")
	}

	// EXECUTE QUERIES
	countPS, resultsPS, err := txPreparedStatements(
		moment,
		params["txid"],
		params["address"],
		params["asset"],
		types,
		limit,
		offset)
	if err != nil {
		return TxTransactions{}, fmt.Errorf("tx prepared statements error: %w", err)
	}

	// Get count
	countRows, err := DBQuery(ctx, countPS.Query, countPS.Values...)

	if err != nil {
		return TxTransactions{}, fmt.Errorf("tx count lookup: %w", err)
	}
	defer countRows.Close()
	var txCount uint
	countRows.Next()
	err = countRows.Scan(&txCount)
	if err != nil {
		return TxTransactions{}, fmt.Errorf("tx count resolve: %w", err)
	}

	// Get results subset
	rows, err := DBQuery(ctx, resultsPS.Query, resultsPS.Values...)
	if err != nil {
		return TxTransactions{}, fmt.Errorf("tx lookup: %w", err)
	}
	defer rows.Close()

	// PROCESS RESULTS
	transactions := []TxTransaction{}
	// TODO(elfedy): This is a hack to get block heights in a semi-performant way,
	// where we get min and max timestamp and get all the relevant heights
	// If we want to make this operation faster we should consider indexing
	// the block_log table by timestamp or making it an hypertable
	var minTimestamp, maxTimestamp int64
	minTimestamp = math.MaxInt64

	for rows.Next() {
		var result txQueryResult
		err := rows.Scan(
			&result.tx,
			&result.fromAddr,
			&result.toAddr,
			&result.asset,
			&result.assetE8,
			&result.asset_2nd,
			&result.asset_2nd_E8,
			&result.memo,
			&result.pool,
			&result.liqFee,
			&result.stakeUnits,
			&result.tradeSlip,
			&result.asymmetry,
			&result.basisPoints,
			&result.eventType,
			&result.blockTimestamp)
		if err != nil {
			return TxTransactions{}, fmt.Errorf("tx resolve: %w", err)
		}
		var transaction TxTransaction

		transaction, minTimestamp, maxTimestamp, err = txProcessQueryResult(ctx, result, minTimestamp, maxTimestamp)
		if err != nil {
			return TxTransactions{}, fmt.Errorf("tx resolve: %w", err)
		}
		transactions = append(transactions, transaction)
	}

	// get heights and store them in a map
	heights := make(map[int64]int64)
	heightsQuery := "SELECT timestamp, height FROM block_log WHERE TIMESTAMP >= $1 AND TIMESTAMP <= $2"
	heightRows, err := DBQuery(ctx, heightsQuery, minTimestamp, maxTimestamp)
	if err != nil {
		return TxTransactions{}, fmt.Errorf("tx height lookup: %w", err)
	}
	defer heightRows.Close()

	for heightRows.Next() {
		var timestamp, height int64
		err = heightRows.Scan(&timestamp, &height)
		if err != nil {
			return TxTransactions{}, fmt.Errorf("tx height resolve: %w", err)
		}
		heights[timestamp] = height
	}

	// Add height to each result set
	for _, transaction := range transactions {
		transaction.Height = heights[transaction.Date]
	}

	return TxTransactions{Count: txCount, Txs: transactions}, rows.Err()
}

// Helper structs to build needed queries
// Query key is used in the query to then be replaced when parsed
// This way arguments can be dynamically inserted in query strings
type namedSqlValue struct {
	QueryKey string
	Value    interface{}
}

type preparedSqlStatement struct {
	Query  string
	Values []interface{}
}

// Builds prepared statements for Tx lookup. Two queries are needed, one to get the count
// of the total entries for the query, and one to get the subset that will actually be
// returned to the caller.
// The two queries are built form a base query with the structure:
// SELECT * FROM (inTxType1Query UNION_ALL inTxType2Query...inTxTypeNQuery) WHERE <<conditions>>
func txPreparedStatements(moment time.Time,
	txid,
	address,
	asset string,
	types []string,
	limit,
	offset uint64) (preparedSqlStatement, preparedSqlStatement, error) {

	var countPS, resultsPS preparedSqlStatement
	// Initialize query param slices (to dynamically insert query params)
	baseValues := make([]namedSqlValue, 0)
	subsetValues := make([]namedSqlValue, 0)

	baseValues = append(baseValues, namedSqlValue{"#MOMENT#", moment.UnixNano()})
	subsetValues = append(subsetValues, namedSqlValue{"#LIMIT#", limit}, namedSqlValue{"#OFFSET#", offset})

	// Build select part of the query by taking the tx in queries from the selected types
	// and joining them using UNION ALL
	usedSelectQueries := make([]string, 0)
	for _, eventType := range types {
		q := txInSelectQueries[eventType]
		if q == nil {
			return countPS, resultsPS, fmt.Errorf("invalid type %q", eventType)
		}

		usedSelectQueries = append(usedSelectQueries, q...)
	}
	selectQuery := "SELECT * FROM (" + strings.Join(usedSelectQueries, " UNION ALL ") + ") union_results"

	// TODO(elfedy): this is a temporary hack as for some reason the count query that has
	// a single select query is much slower when no UNIONS happen, and making a union into
	// itself makes it faster. Profiling and optimizing should be done for this at a later stage
	countSelectQuery := selectQuery
	if len(usedSelectQueries) == 1 {
		countSelectQuery = "SELECT * FROM (" + usedSelectQueries[0] + " UNION " + usedSelectQueries[0] + ") union_results"
	}

	// Replace all #RUNE# values with actual asset
	selectQuery = strings.ReplaceAll(selectQuery, "#RUNE#", `'`+event.RuneAsset()+`'`)
	countSelectQuery = strings.ReplaceAll(countSelectQuery, "#RUNE#", `'`+event.RuneAsset()+`'`)

	// build WHERE clause applied to the union_all result, based on filter arguments
	// (txid, address, asset)
	whereQuery := `
	WHERE union_results.block_timestamp <= #MOMENT#`

	if txid != "" {
		baseValues = append(baseValues, namedSqlValue{"#TXID#", txid})
		whereQuery += ` AND (
			union_results.tx = #TXID# OR
			union_results.tx IN (
				SELECT in_tx FROM outbound_events WHERE
					outbound_events.tx = #TXID#
			)
		)`
	}

	if address != "" {
		baseValues = append(baseValues, namedSqlValue{"#ADDRESS#", address})
		whereQuery += ` AND (
			union_results.to_addr = #ADDRESS# OR
			union_results.from_addr = #ADDRESS# OR
			union_results.tx IN (
				SELECT in_tx FROM outbound_events WHERE
					outbound_events.to_addr = #ADDRESS# OR
					outbound_events.from_addr = #ADDRESS#
			)
		)`
	}

	if asset != "" {
		baseValues = append(baseValues, namedSqlValue{"#ASSET#", asset})
		whereQuery += ` AND (
			union_results.asset = #ASSET# OR
			union_results.asset_2nd = #ASSET# OR 
			union_results.tx IN (
				SELECT in_tx FROM outbound_events WHERE
					outbound_events.asset = #ASSET#
			)
		)`
	}

	// build subset query for the results being shown (based on limit and offset)
	subsetQuery := `
	ORDER BY union_results.block_timestamp DESC
	LIMIT #LIMIT# 
	OFFSET #OFFSET# 
	`
	// build and return final queries
	countTxQuery := countSelectQuery + " " + whereQuery
	countQuery := "SELECT count(*) FROM (" + countTxQuery + ") AS count"
	countQueryValues := make([]interface{}, 0)
	for i, queryValue := range baseValues {
		position := i + 1
		positionLabel := fmt.Sprintf("$%d", position)
		countQuery = strings.ReplaceAll(countQuery, queryValue.QueryKey, positionLabel)
		countQueryValues = append(countQueryValues, queryValue.Value)
	}
	countPS = preparedSqlStatement{countQuery, countQueryValues}

	txQuery := selectQuery + " " + whereQuery
	resultsQuery := txQuery + subsetQuery
	resultsQueryValues := make([]interface{}, 0)
	for i, queryValue := range append(baseValues, subsetValues...) {
		position := i + 1
		positionLabel := fmt.Sprintf("$%d", position)
		resultsQuery = strings.ReplaceAll(resultsQuery, queryValue.QueryKey, positionLabel)
		resultsQueryValues = append(resultsQueryValues, queryValue.Value)
	}
	resultsPS = preparedSqlStatement{resultsQuery, resultsQueryValues}

	return countPS, resultsPS, nil
}

type txQueryResult struct {
	tx             string
	fromAddr       string
	toAddr         string
	asset          sql.NullString
	assetE8        int64
	asset_2nd      sql.NullString
	asset_2nd_E8   int64
	memo           string
	pool           string
	liqFee         uint64
	stakeUnits     int64
	tradeSlip      uint64
	asymmetry      int64
	basisPoints    int64
	eventType      string
	blockTimestamp int64
}

func txProcessQueryResult(ctx context.Context, result txQueryResult, minTimestamp, maxTimestamp int64) (TxTransaction, int64, int64, error) {
	// Build events data
	events := TxEvents{
		Fee:        result.liqFee,
		StakeUnits: result.stakeUnits,
		Slip:       float64(result.tradeSlip) / 10000,
	}

	// Build incoming related transaction (from external address to vault address)
	inTx := TxInOut{
		Address: result.fromAddr,
		Memo:    result.memo,
		TxId:    result.tx,
		Options: TxOptions{
			Asymmetry:           result.asymmetry,
			PriceTarget:         0,
			WithdrawBasisPoints: result.basisPoints,
		},
	}

	// Build in tx coins
	if result.asset.Valid && result.assetE8 > 0 {
		inTx.Coins = append(inTx.Coins, TxCoin{Amount: result.assetE8, Asset: result.asset.String})
	}
	if result.asset_2nd.Valid && result.asset_2nd_E8 > 0 {
		inTx.Coins = append(inTx.Coins, TxCoin{Amount: result.asset_2nd_E8, Asset: result.asset_2nd.String})
	}

	// Get and process outbound transactions (from vault address to external address)
	blockTime := time.Unix(0, result.blockTimestamp)
	outboundTimeLower := blockTime.Add(-OutboundTimeout).UnixNano()
	outboundTimeUpper := blockTime.Add(OutboundTimeout).UnixNano()

	outboundsQuery := `
	SELECT 
	tx,
	from_addr,
	memo,
	asset,
	asset_E8
	FROM outbound_events
	WHERE in_tx = $1 AND tx IS NOT NULL AND block_timestamp > $2 AND block_timestamp < $3
	`

	outboundRows, err := DBQuery(ctx, outboundsQuery, result.tx, outboundTimeLower, outboundTimeUpper)
	if err != nil {
		return TxTransaction{}, minTimestamp, maxTimestamp, fmt.Errorf("outbound tx lookup: %w", err)
	}

	outTxs := []TxInOut{}

	for outboundRows.Next() {
		var tx,
			address,
			memo,
			asset string
		var assetE8 int64

		outboundRows.Scan(&tx, &address, &memo, &asset, &assetE8)
		outTx := TxInOut{
			Address: address,
			Coins:   []TxCoin{{Amount: assetE8, Asset: asset}},
			Memo:    memo,
			TxId:    tx,
			Options: TxOptions{Asymmetry: 0, PriceTarget: 0, WithdrawBasisPoints: 0},
		}
		outTxs = append(outTxs, outTx)
	}
	outboundRows.Close()

	status := "Pending"
	switch result.eventType {
	case "swap":
		if len(outTxs) == 1 {
			status = "Success"
		}
	case "doubleSwap":
		// The in between outbound is not part of the query
		if len(outTxs) == 1 {
			status = "Success"
		}
	case "refund":
		if len(outTxs) == len(inTx.Coins) {
			status = "Success"
		}
	case "unstake":
		if len(outTxs) == 2 {
			status = "Success"
		}
	case "add":
		status = "Success"
	case "stake":
		status = "Success"
	}

	transaction := TxTransaction{
		EventType: result.eventType,
		Date:      result.blockTimestamp,
		Events:    events,
		In:        inTx,
		Out:       outTxs,
		Pool:      result.pool,
		Status:    status,
	}

	// compute min/max timestamp to get heights later
	if result.blockTimestamp < minTimestamp {
		minTimestamp = result.blockTimestamp
	}
	if result.blockTimestamp > maxTimestamp {
		maxTimestamp = result.blockTimestamp
	}

	return transaction, minTimestamp, maxTimestamp, nil
}

// txIn select queries: list of queries that have inbound
// transactions as rows. They are given a type based on the operation they relate to.
// These queries are built using data from events sent by Thorchain
var txInSelectQueries = map[string][]string{
	"swap": {`SELECT 
				tx,
				from_addr,
				to_addr,
				from_asset as asset,
				from_E8 as asset_E8,
				'' as asset_2nd,
				0 as asset_2nd_E8,
				memo,
				pool,
				liq_fee_E8,
				0 as stake_units,
				trade_slip_BP,
				0 as asymmetry,
				0 as basis_points,
				'swap' as type,
				block_timestamp
			FROM swap_events AS single_swaps
			WHERE NOT EXISTS (
				SELECT tx FROM swap_events WHERE block_timestamp = single_swaps.block_timestamp AND tx = single_swaps.tx AND from_asset <> single_swaps.from_asset
			)`},
	"doubleSwap": {`SELECT
				swap_in.tx as tx,
				swap_in.from_addr as from_addr,
				swap_in.to_addr as to_addr,
				swap_in.from_asset as asset,
				swap_in.from_E8 as asset_E8,
				NULL as asset_2nd,
				0 as asset_2nd_E8,
				swap_in.memo as memo,
				swap_in.pool as pool,
				(swap_in.liq_fee_E8 + swap_out.liq_fee_E8) as liq_fee_E8,
				0 as stake_units,
				(swap_in.trade_slip_BP + swap_out.trade_slip_BP) as trade_slip_BP,
				0 as asymmetry,
				0 as basis_points,
				'doubleSwap' as type,
				swap_in.block_timestamp as block_timestamp
			FROM
			swap_events AS swap_in
			INNER JOIN
			swap_events AS swap_out
			ON swap_in.tx = swap_out.tx
			WHERE swap_in.from_asset = swap_in.pool AND swap_out.from_asset <> swap_out.pool`},
	"stake": {
		// TODO(elfedy): v1 queries thorchain to get some tx details when it parses the events
		// (i.e: the memo, non rune address) those are currently missing in this implementation.
		// Tx with both RUNE and asset
		`SELECT 
					rune_tx as tx,
					rune_addr as from_addr,
					'' as to_addr,
					pool as asset,
					asset_E8 as asset_E8,
					#RUNE# as asset_2nd,
					rune_E8 as asset_2nd_E8,
					'' as memo,
					pool,
					0 as liq_fee_E8,
					stake_units,
					0 as trade_slip_BP,
					0 as asymmetry,
					0 as basis_points,
					'stake' as type,
					block_timestamp
				FROM stake_events
				WHERE rune_tx = asset_tx`,
		// Txs with RUNE only
		`SELECT 
					rune_tx as tx,
					rune_addr as from_addr,
					'' as to_addr,
					#RUNE# as asset,
					rune_E8 as asset_E8,
					NULL as asset_2nd,
					0 as asset_2nd_E8,
					'' as memo,
					pool,
					0 as liq_fee_E8,
					stake_units,
					0 as trade_slip_BP,
					0 as asymmetry,
					0 as basis_points,
					'stake' as type,
					block_timestamp
				FROM stake_events
				WHERE rune_tx <> asset_tx`,
		// Txs with asset only
		// TODO(elfedy): check if rune_addr is from_addr here. Doesn't seem like it. If it isn't
		// we also need to query that from the node transactions
		`SELECT 
					asset_tx as tx,
					rune_addr as from_addr,
					'' as to_addr,
					pool as asset,
					asset_E8 as asset_E8,
					NULL as asset_2nd,
					0 as asset_2nd_E8,
					'' as memo,
					pool,
					0 as liq_fee_E8,
					stake_units,
					0 as trade_slip_BP,
					0 as asymmetry,
					0 as basis_points,
					'stake' as type,
					block_timestamp
				FROM stake_events
				WHERE rune_tx <> asset_tx`},
	"unstake": {`
			SELECT 
				tx,
				from_addr,
				to_addr,
				asset,
				asset_E8,
				'' as asset_2nd,
				0 as asset_2nd_E8,
				memo,
				pool,
				0 as liq_fee_E8,
				(stake_units * -1) as stake_units,
				0 as trade_slip_BP,
				asymmetry,
				basis_points,
				'unstake' as type,
				block_timestamp
			FROM unstake_events`},
	"add": {`
			SELECT 
				tx,
				from_addr,
				to_addr,
				asset,
				asset_E8,
				#RUNE# as asset_2nd,
				rune_E8 as asset_2nd_E8,
				memo,
				pool,
				0 as liq_fee_E8,
				0 as stake_units,
				0 as trade_slip_BP,
				0 as asymmetry,
				0 as basis_points,
				'add' as type,
				block_timestamp
			FROM add_events`},
	"refund": {`SELECT 
				tx,
				from_addr,
				to_addr,
				asset,
				asset_E8,
				asset_2nd,
				asset_2nd_E8,
				memo,
				'.' as pool,
				0 as liq_fee_E8,
				0 as stake_units,
				0 as trade_slip_BP,
				0 as asymmetry,
				0 as basis_points,
				'refund' as type,
				block_timestamp
			FROM refund_events`},
}
