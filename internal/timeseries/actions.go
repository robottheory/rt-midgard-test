package timeseries

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/fetch/record"
	"gitlab.com/thorchain/midgard/internal/util"
	"gitlab.com/thorchain/midgard/internal/util/miderr"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

const (
	MaxAddresses = 50
	MaxLimit     = 50
	DefaultLimit = 50
)

func floatStr(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

type action struct {
	pools      []string
	actionType string
	status     string
	in         []transaction
	out        []transaction
	date       int64
	height     int64
	metadata   oapigen.Metadata
}

func (a action) toOapigen() oapigen.Action {
	oapigenIn := make([]oapigen.Transaction, len(a.in))
	oapigenOut := make([]oapigen.Transaction, len(a.out))

	for i, tx := range a.in {
		oapigenIn[i] = tx.toOapigen()
	}

	for i, tx := range a.out {
		oapigenOut[i] = tx.toOapigen()
	}

	return oapigen.Action{
		Pools:    a.pools,
		Type:     oapigen.ActionType(a.actionType),
		Status:   oapigen.ActionStatus(a.status),
		In:       oapigenIn,
		Out:      oapigenOut,
		Date:     util.IntStr(a.date),
		Height:   util.IntStr(a.height),
		Metadata: a.metadata,
	}
}

type transaction struct {
	Address string   `json:"address"`
	Coins   coinList `json:"coins"`
	TxID    string   `json:"txID"`
}

func (tx transaction) toOapigen() oapigen.Transaction {
	return oapigen.Transaction{
		Address: tx.Address,
		TxID:    tx.TxID,
		Coins:   tx.Coins.toOapigen(),
	}
}

type transactionList []transaction

func (a *transactionList) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &a)
}

type coin struct {
	Asset  string `json:"asset"`
	Amount int64  `json:"amount"`
}

func (c coin) toOapigen() oapigen.Coin {
	return oapigen.Coin{
		Asset:  c.Asset,
		Amount: util.IntStr(c.Amount),
	}
}

type coinList []coin

func (coins coinList) toOapigen() []oapigen.Coin {
	oapigenCoins := make([]oapigen.Coin, len(coins))
	for i, c := range coins {
		oapigenCoins[i] = c.toOapigen()
	}
	return oapigenCoins
}

func (a *coinList) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &a)
}

// Combination of possible fields the jsonb `meta` column can have
type actionMeta struct {
	// refund:
	Reason string `json:"reason"`
	// withdraw:
	Asymmetry      float64 `json:"asymmetry"`
	BasisPoints    int64   `json:"basisPoints"`
	ImpLossProt    int64   `json:"impermanentLossProtection"`
	LiquidityUnits int64   `json:"liquidityUnits"`
	EmitAssetE8    int64   `json:"emitAssetE8"`
	EmitRuneE8     int64   `json:"emitRuneE8"`
	// swap:
	SwapSingle   bool  `json:"swapSingle"`
	LiquidityFee int64 `json:"liquidityFee"`
	SwapTarget   int64 `json:"swapTarget"`
	SwapSlip     int64 `json:"swapSlip"`
	// addLiquidity:
	Status string `json:"status"`
	// also LiquidityUnits
}

// TODO(huginn): switch to using native pgx interface, this would allow us to scan
// jsonb and array data automatically, without writing these methods and using libpq.
// It's also more efficient.
func (a *actionMeta) Scan(value interface{}) error {
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, &a)
	case nil:
		*a = actionMeta{}
		return nil
	}

	return errors.New("unsupported scan type for actionMeta")
}

type ActionsParams struct {
	Limit      string
	Offset     string
	ActionType string
	Address    string
	TXId       string
	Asset      string
	AssetType  string
}

func runActionsQuery(ctx context.Context, q preparedSqlStatement) ([]action, error) {
	rows, err := db.Query(ctx, q.Query, q.Values...)
	if err != nil {
		return nil, fmt.Errorf("actions query: %w", err)
	}
	defer rows.Close()

	actions := []action{}

	for rows.Next() {
		var result action
		var ins transactionList
		var outs transactionList
		var fees coinList
		var meta actionMeta
		err := rows.Scan(
			&result.height,
			&result.date,
			&result.actionType,
			pq.Array(&result.pools),
			&ins,
			&outs,
			&fees,
			&meta,
		)
		if err != nil {
			return nil, fmt.Errorf("actions read: %w", err)
		}

		result.in = ins
		result.out = outs
		result.completeFromDBRead(&meta, fees)

		actions = append(actions, result)
	}

	return actions, nil
}

func (a *action) completeFromDBRead(meta *actionMeta, fees coinList) {
	if a.pools == nil {
		a.pools = []string{}
	}

	isDoubleSwap := a.actionType == "swap" && !meta.SwapSingle
	outs := []transaction{}
	for _, t := range a.out {
		// When we have a double swap: Asset1->Rune ; Rune->Asset2
		// then we have two outbound_events: one for the Rune and one for Asset2.
		// We hide the Rune outbound and show only the Asset2 outbound to the user.
		// TxID is "" for THOR.RUNE transactions.
		if !(t.TxID == "" && isDoubleSwap) {
			outs = append(outs, t)
		} else {
			isSynth := len(t.Coins) != 0 &&
				record.GetCoinType([]byte(t.Coins[0].Asset)) == record.AssetSynth
			if isSynth {
				outs = append(outs, t)
			}
		}
	}
	a.out = outs

	// process status
	a.status = "success"
	if meta.Status != "" {
		a.status = meta.Status
	}

	switch a.actionType {
	case "swap":
		// There might be multiple outs. Maybe we could check if the full sum was sent out.
		// toe8 of last swap (1st or 2nd) <= sum(outTxs.coin.amount) + networkfee.amount
		// We would need to query toe8 in txInSelectQueries.
		if len(a.out) == 0 {
			a.status = "pending"
		}
	case "refund":
		// success: either fee is greater than in amount or both
		// outbound and fees are present.
		// TODO(elfedy): Sometimes fee + outbound not equals in amount
		// The resons behind this must be investigated
		inBalances := make(map[string]int64)
		outBalances := make(map[string]int64)
		outFees := make(map[string]int64)

		for _, tx := range a.in {
			for _, coin := range tx.Coins {
				inBalances[coin.Asset] = coin.Amount
			}
		}
		for _, tx := range a.out {
			for _, coin := range tx.Coins {
				outBalances[coin.Asset] = coin.Amount
			}
		}
		for _, coin := range fees {
			outFees[coin.Asset] = coin.Amount
		}

		a.status = "success"
		for k, inBalance := range inBalances {
			if inBalance > outFees[k] && outBalances[k] == 0 {
				a.status = "pending"
				break
			}
		}
	case "withdraw":
		var runeOut, assetOut, runeFee, assetFee int64
		for _, tx := range a.out {
			for _, coin := range tx.Coins {
				if coin.Asset != "THOR.RUNE" {
					assetOut = coin.Amount
				} else {
					runeOut = coin.Amount
				}
			}
		}
		for _, coin := range fees {
			if coin.Asset != "THOR.RUNE" {
				assetFee = coin.Amount
			} else {
				runeFee = coin.Amount
			}
		}
		runeOk := meta.EmitRuneE8 <= runeFee || runeOut != 0
		assetOk := meta.EmitAssetE8 <= assetFee || assetOut != 0

		a.status = "pending"
		if runeOk && assetOk {
			a.status = "success"
		}
	default:
	}

	switch a.actionType {
	case "swap":
		a.metadata.Swap = &oapigen.SwapMetadata{
			LiquidityFee: util.IntStr(meta.LiquidityFee),
			SwapSlip:     util.IntStr(meta.SwapSlip),
			SwapTarget:   util.IntStr(meta.SwapTarget),
			NetworkFees:  fees.toOapigen(),
		}
	case "addLiquidity":
		if meta.LiquidityUnits != 0 {
			a.metadata.AddLiquidity = &oapigen.AddLiquidityMetadata{
				LiquidityUnits: util.IntStr(meta.LiquidityUnits),
			}
		}
	case "withdraw":
		a.metadata.Withdraw = &oapigen.WithdrawMetadata{
			LiquidityUnits:            util.IntStr(meta.LiquidityUnits),
			Asymmetry:                 floatStr(meta.Asymmetry),
			BasisPoints:               util.IntStr(meta.BasisPoints),
			NetworkFees:               fees.toOapigen(),
			ImpermanentLossProtection: util.IntStr(meta.ImpLossProt),
		}
	case "refund":
		a.metadata.Refund = &oapigen.RefundMetadata{
			NetworkFees: fees.toOapigen(),
			Reason:      meta.Reason,
		}
	}
}

// Gets a list of actions generated by external transactions and return its associated data
func GetActions(ctx context.Context, moment time.Time, params ActionsParams) (
	oapigen.ActionsResponse, error) {
	// CHECK PARAMS
	// give latest value if zero moment
	_, timestamp, _ := LastBlock()
	if moment.IsZero() {
		moment = timestamp
	} else if timestamp.Before(moment) {
		return oapigen.ActionsResponse{}, errBeyondLast
	}

	var limit uint64
	// check limit param
	if params.Limit != "" {
		var err error
		limit, err = strconv.ParseUint(params.Limit, 10, 64)
		if err != nil || limit < 1 || MaxLimit < limit {
			return oapigen.ActionsResponse{}, errors.New("'limit' must be an integer between 1 and 50")
		}
	} else {
		limit = DefaultLimit
	}

	var offset uint64
	// check offset param
	if params.Offset != "" {
		var err error
		offset, err = strconv.ParseUint(params.Offset, 10, 64)
		if err != nil {
			return oapigen.ActionsResponse{}, errors.New("'offset' must be a non-negative integer")
		}
	} else {
		offset = 0
	}

	// build types from type param
	types := make([]string, 0)
	if params.ActionType != "" {
		types = strings.Split(params.ActionType, ",")
	}

	var addresses []string
	if params.Address != "" {
		addresses = strings.Split(params.Address, ",")
		if MaxAddresses < len(addresses) {
			return oapigen.ActionsResponse{}, miderr.BadRequestF(
				"too many addresses: %d provided, maximum is %d",
				len(addresses), MaxAddresses)
		}
	}
	if params.AssetType != "" && params.AssetType != "native" && params.AssetType != "synthetic" {
		return oapigen.ActionsResponse{}, errors.New("'invalid assetType. assetType musth be native or synthetic")
	}
	native := true
	synth := true
	if params.AssetType == "native" {
		synth = false
	} else if params.AssetType == "synthetic" {
		native = false
	}

	// Construct queries
	countPS, resultsPS, err := actionsPreparedStatements(
		moment,
		params.TXId,
		addresses,
		params.Asset,
		types,
		limit,
		offset,
		native,
		synth)
	if err != nil {
		return oapigen.ActionsResponse{}, err
	}

	// Get count
	countRows, err := db.Query(ctx, countPS.Query, countPS.Values...)
	if err != nil {
		return oapigen.ActionsResponse{}, fmt.Errorf("actions count query: %w", err)
	}
	defer countRows.Close()
	var totalCount uint
	countRows.Next()
	err = countRows.Scan(&totalCount)
	if err != nil {
		return oapigen.ActionsResponse{}, fmt.Errorf("actions count read: %w", err)
	}

	// Get results
	actions, err := runActionsQuery(ctx, resultsPS)
	if err != nil {
		return oapigen.ActionsResponse{}, err
	}

	oapigenActions := make([]oapigen.Action, len(actions))
	for i, action := range actions {
		oapigenActions[i] = action.toOapigen()
	}
	return oapigen.ActionsResponse{Count: util.IntStr(int64(totalCount)), Actions: oapigenActions}, nil
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

// Builds SQL statements for Actions lookup. Two queries are needed, one to get the count
// of the total entries for the query, and one to get the subset that will actually be
// returned to the caller.
func actionsPreparedStatements(moment time.Time,
	txid string,
	addresses []string,
	asset string,
	types []string,
	limit,
	offset uint64,
	native bool,
	synth bool,
) (preparedSqlStatement, preparedSqlStatement, error) {
	var countPS, resultsPS preparedSqlStatement
	// Initialize query param slices (to dynamically insert query params)
	baseValues := make([]namedSqlValue, 0)
	subsetValues := make([]namedSqlValue, 0)

	baseValues = append(baseValues, namedSqlValue{"#MOMENT#", moment.UnixNano()})
	subsetValues = append(subsetValues, namedSqlValue{"#LIMIT#", limit}, namedSqlValue{"#OFFSET#", offset})

	// build WHERE which is common to both queries, based on filter arguments
	// (types, txid, address, asset)
	whereQuery := `
		WHERE block_timestamp <= #MOMENT#`

	if len(types) != 0 {
		baseValues = append(baseValues, namedSqlValue{"#TYPE#", types})
		whereQuery += `
			AND type = ANY(#TYPE#)`
	}

	if txid != "" {
		baseValues = append(baseValues, namedSqlValue{"#TXID#", strings.ToUpper(txid)})
		whereQuery += `
			AND transactions @> ARRAY[#TXID#]`
	}

	if len(addresses) != 0 {
		baseValues = append(baseValues, namedSqlValue{"#ADDRESSES#", addresses})
		whereQuery += `
			AND addresses && #ADDRESSES#`
	}

	if asset != "" {
		baseValues = append(baseValues, namedSqlValue{"#ASSET#", asset})
		whereQuery += `
			AND assets @> ARRAY[#ASSET#]`
	}

	// build and return final queries
	countQuery := `SELECT count(1) FROM midgard_agg.actions` + whereQuery
	countQueryValues := make([]interface{}, 0)
	for i, queryValue := range baseValues {
		position := i + 1
		positionLabel := fmt.Sprintf("$%d", position)
		countQuery = strings.ReplaceAll(countQuery, queryValue.QueryKey, positionLabel)
		countQueryValues = append(countQueryValues, queryValue.Value)
	}
	countPS = preparedSqlStatement{countQuery, countQueryValues}

	resultsQuery := `
		SELECT
			height,
			block_timestamp,
			type,
			pools,
			ins,
			outs,
			fees,
			meta
		FROM midgard_agg.actions
	` + whereQuery + `
		ORDER BY block_timestamp DESC
		LIMIT #LIMIT#
		OFFSET #OFFSET#
	`
	if !native || !synth {
		if synth {
			resultsQuery = strings.Replace(resultsQuery, "WHERE", "AND", -1)
			resultsQuery = strings.Replace(resultsQuery, "FROM midgard_agg.actions", `
																				FROM   midgard_agg.actions
																				WHERE  EXISTS
																					   (
																							  SELECT
																							  from   unnest(assets) elem
																							  WHERE  elem LIKE '%/%')`, -1)
			countQuery = strings.Replace(countQuery, "WHERE", "AND", -1)
			countQuery = strings.Replace(countQuery, "FROM midgard_agg.actions", `
																				FROM   midgard_agg.actions
																				WHERE  EXISTS
																					   (
																							  SELECT
																							  from   unnest(assets) elem
																							  WHERE  elem LIKE '%/%')`, -1)
		} else {
			resultsQuery = strings.Replace(resultsQuery, "WHERE", "AND", -1)
			resultsQuery = strings.Replace(resultsQuery, "FROM midgard_agg.actions", `
																				FROM   midgard_agg.actions
																				WHERE  NOT EXISTS
																					   (
																							  SELECT
																							  from   unnest(assets) elem
																							  WHERE  elem LIKE '%/%')`, -1)
			countQuery = strings.Replace(countQuery, "WHERE", "AND", -1)
			countQuery = strings.Replace(countQuery, "FROM midgard_agg.actions", `
																				FROM   midgard_agg.actions
																				WHERE  NOT EXISTS
																					   (
																							  SELECT
																							  from   unnest(assets) elem
																							  WHERE  elem LIKE '%/%')`, -1)
		}
	}
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
