// Package oapigen provides primitives to interact the openapi HTTP API.
//
// Code generated by github.com/deepmap/oapi-codegen DO NOT EDIT.
package oapigen

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// BlockRewards defines model for BlockRewards.
type BlockRewards struct {
	BlockReward string `json:"blockReward"`
	BondReward  string `json:"bondReward"`
	PoolReward  string `json:"poolReward"`
}

// BondMetrics defines model for BondMetrics.
type BondMetrics struct {

	// Int64, Average bond of active nodes
	AverageActiveBond string `json:"averageActiveBond"`

	// Int64, Average bond of standby nodes
	AverageStandbyBond string `json:"averageStandbyBond"`

	// Int64, Maxinum bond of active nodes
	MaximumActiveBond string `json:"maximumActiveBond"`

	// Int64, Maximum bond of standby nodes
	MaximumStandbyBond string `json:"maximumStandbyBond"`

	// Int64, Median bond of active nodes
	MedianActiveBond string `json:"medianActiveBond"`

	// Int64, Median bond of standby nodes
	MedianStandbyBond string `json:"medianStandbyBond"`

	// Int64, Minumum bond of active nodes
	MinimumActiveBond string `json:"minimumActiveBond"`

	// Int64, Minumum bond of standby nodes
	MinimumStandbyBond string `json:"minimumStandbyBond"`

	// Int64, Total bond of active nodes
	TotalActiveBond string `json:"totalActiveBond"`

	// Int64, Total bond of standby nodes
	TotalStandbyBond string `json:"totalStandbyBond"`
}

// BoolConstants defines model for BoolConstants.
type BoolConstants struct {
	StrictBondLiquidityRatio bool `json:"StrictBondLiquidityRatio"`
}

// Constants defines model for Constants.
type Constants struct {
	BoolValues   BoolConstants   `json:"bool_values"`
	Int64Values  Int64Constants  `json:"int_64_values"`
	StringValues StringConstants `json:"string_values"`
}

// DepthHistory defines model for DepthHistory.
type DepthHistory struct {
	Intervals DepthHistoryIntervals `json:"intervals"`
	Meta      DepthHistoryMeta      `json:"meta"`
}

// DepthHistoryIntervals defines model for DepthHistoryIntervals.
type DepthHistoryIntervals []DepthHistoryItem

// DepthHistoryItem defines model for DepthHistoryItem.
type DepthHistoryItem struct {

	// Int64, the amount of Asset in the pool
	AssetDepth string `json:"assetDepth"`

	// Float, price of asset in rune. I.e. rune amount / asset amount
	AssetPrice string `json:"assetPrice"`

	// Int64, The end time of bucket in unix timestamp
	EndTime string `json:"endTime"`

	// Int64, the amount of Rune in the pool
	RuneDepth string `json:"runeDepth"`

	// Int64, The beginning time of bucket in unix timestamp
	StartTime string `json:"startTime"`
}

// DepthHistoryMeta defines model for DepthHistoryMeta.
type DepthHistoryMeta struct {

	// Int64, The end time of bucket in unix timestamp
	EndTime string `json:"endTime"`

	// Int64, The beginning time of bucket in unix timestamp
	StartTime string `json:"startTime"`
}

// EarningsHistory defines model for EarningsHistory.
type EarningsHistory struct {
	Intervals EarningsHistoryIntervals `json:"intervals"`
	Meta      EarningsHistoryItem      `json:"meta"`
}

// EarningsHistoryIntervals defines model for EarningsHistoryIntervals.
type EarningsHistoryIntervals []EarningsHistoryItem

// EarningsHistoryItem defines model for EarningsHistoryItem.
type EarningsHistoryItem struct {

	// float64, Average amount of active nodes during the time interval
	AvgNodeCount string `json:"avgNodeCount"`

	// Int64, Total block rewards emitted during the time interval
	BlockRewards string `json:"blockRewards"`

	// Int64, Share of earnings sent to nodes during the time interval
	BondingEarnings string `json:"bondingEarnings"`

	// Int64, System income generated during the time interval. It is the sum of liquidity fees and block rewards
	Earnings string `json:"earnings"`

	// Int64, The end time of interval in unix timestamp
	EndTime string `json:"endTime"`

	// Int64, Share of earnings sent to pools during the time interval
	LiquidityEarnings string `json:"liquidityEarnings"`

	// Int64, Total liquidity fees, converted to RUNE, collected during the time interval
	LiquidityFees string `json:"liquidityFees"`

	// Earnings data for each pool for the time interval
	Pools []EarningsHistoryItemPool `json:"pools"`

	// Int64, The beginning time of interval in unix timestamp
	StartTime string `json:"startTime"`
}

// EarningsHistoryItemPool defines model for EarningsHistoryItemPool.
type EarningsHistoryItemPool struct {

	// Int64, Share of earnings sent to the pool during the time interval
	Earnings string `json:"earnings"`

	// asset for the given pool
	Pool string `json:"pool"`
}

// Health defines model for Health.
type Health struct {

	// True means healthy, connected to database
	Database bool `json:"database"`

	// True means healthy. False means Midgard is still catching up to the chain
	InSync bool `json:"inSync"`

	// Int64, the current block count
	ScannerHeight string `json:"scannerHeight"`
}

// InboundAddresses defines model for InboundAddresses.
type InboundAddresses struct {
	Current []InboundAddressesItem `json:"current"`
}

// InboundAddressesItem defines model for InboundAddressesItem.
type InboundAddressesItem struct {
	Address string `json:"address"`
	Chain   string `json:"chain"`

	// indicate whether this chain has halted
	Halted bool   `json:"halted"`
	PubKey string `json:"pub_key"`
}

// Int64Constants defines model for Int64Constants.
type Int64Constants struct {
	AsgardSize                  int64 `json:"AsgardSize"`
	BadValidatorRate            int64 `json:"BadValidatorRate"`
	BlocksPerYear               int64 `json:"BlocksPerYear"`
	ChurnInterval               int64 `json:"ChurnInterval"`
	ChurnRetryInterval          int64 `json:"ChurnRetryInterval"`
	CliTxCost                   int64 `json:"CliTxCost"`
	DesiredValidatorSet         int64 `json:"DesiredValidatorSet"`
	DoubleSignMaxAge            int64 `json:"DoubleSignMaxAge"`
	EmissionCurve               int64 `json:"EmissionCurve"`
	FailKeygenSlashPoints       int64 `json:"FailKeygenSlashPoints"`
	FailKeysignSlashPoints      int64 `json:"FailKeysignSlashPoints"`
	FundMigrationInterval       int64 `json:"FundMigrationInterval"`
	JailTimeKeygen              int64 `json:"JailTimeKeygen"`
	JailTimeKeysign             int64 `json:"JailTimeKeysign"`
	LackOfObservationPenalty    int64 `json:"LackOfObservationPenalty"`
	LiquidityLockUpBlocks       int64 `json:"LiquidityLockUpBlocks"`
	MinimumBondInRune           int64 `json:"MinimumBondInRune"`
	MinimumNodesForBFT          int64 `json:"MinimumNodesForBFT"`
	MinimumNodesForYggdrasil    int64 `json:"MinimumNodesForYggdrasil"`
	NativeChainGasFee           int64 `json:"NativeChainGasFee"`
	NewPoolCycle                int64 `json:"NewPoolCycle"`
	ObservationDelayFlexibility int64 `json:"ObservationDelayFlexibility"`
	ObserveSlashPoints          int64 `json:"ObserveSlashPoints"`
	OldValidatorRate            int64 `json:"OldValidatorRate"`
	OutboundTransactionFee      int64 `json:"OutboundTransactionFee"`
	SigningTransactionPeriod    int64 `json:"SigningTransactionPeriod"`
	YggFundLimit                int64 `json:"YggFundLimit"`
}

// Lastblock defines model for Lastblock.
type Lastblock struct {
	Current []LastblockItem `json:"current"`
}

// LastblockItem defines model for LastblockItem.
type LastblockItem struct {
	Chain          string `json:"chain"`
	LastObservedIn string `json:"last_observed_in"`
	LastSignedOut  string `json:"last_signed_out"`
	Thorchain      string `json:"thorchain"`
}

// LiquidityHistory defines model for LiquidityHistory.
type LiquidityHistory struct {
	Intervals LiquidityHistoryIntervals `json:"intervals"`
	Meta      LiquidityHistoryItem      `json:"meta"`
}

// LiquidityHistoryIntervals defines model for LiquidityHistoryIntervals.
type LiquidityHistoryIntervals []LiquidityHistoryItem

// LiquidityHistoryItem defines model for LiquidityHistoryItem.
type LiquidityHistoryItem struct {

	// Int64, total deposits (liquidity additions) during the time interval
	Deposits string `json:"deposits"`

	// Int64, The end time of bucket in unix timestamp
	EndTime string `json:"endTime"`

	// Int64, net liquidity changes (withdrawals - deposits) during the time interval
	Net string `json:"net"`

	// Int64, The beginning time of bucket in unix timestamp
	StartTime string `json:"startTime"`

	// Int64, total withdrawals during the time interval
	Withdrawals string `json:"withdrawals"`
}

// MemberDetails defines model for MemberDetails.
type MemberDetails struct {

	// Liquidity provider data for all the pools of a given member
	Pools []MemberPoolDetails `json:"pools"`
}

// MemberPoolDetails defines model for MemberPoolDetails.
type MemberPoolDetails struct {

	// Int64, total asset added to the pool by member
	AssetAdded string `json:"assetAdded"`

	// Int64, total asset withdrawn from the pool by member
	AssetWithdrawn string `json:"assetWithdrawn"`

	// Int64, Unix timestamp for the first time member deposited into the pool
	DateFirstAdded string `json:"dateFirstAdded"`

	// Int64, Unix timestamp for the last time member deposited into the pool
	DateLastAdded string `json:"dateLastAdded"`

	// Int64, pool liquidity units that belong the the member
	LiquidityUnits string `json:"liquidityUnits"`

	// Pool rest of the data refers to
	Pool string `json:"pool"`

	// Int64, total RUNE added to the pool by member
	RuneAdded string `json:"runeAdded"`

	// Int64, total RUNE withdrawn from the pool by member
	RuneWithdrawn string `json:"runeWithdrawn"`
}

// Members defines model for Members.
type Members []string

// Network defines model for Network.
type Network struct {
	ActiveBonds []string `json:"activeBonds"`

	// Int64, Number of Active Nodes
	ActiveNodeCount string       `json:"activeNodeCount"`
	BlockRewards    BlockRewards `json:"blockRewards"`
	BondMetrics     BondMetrics  `json:"bondMetrics"`

	// Float, (1 + (bondReward * blocksPerMonth/totalActiveBond)) ^ 12 -1
	BondingAPY string `json:"bondingAPY"`

	// Float, (1 + (stakeReward * blocksPerMonth/totalDepth of active pools)) ^ 12 -1
	LiquidityAPY string `json:"liquidityAPY"`

	// Int64, next height of blocks
	NextChurnHeight string `json:"nextChurnHeight"`

	// Int64, the remaining time of pool activation (in blocks)
	PoolActivationCountdown string `json:"poolActivationCountdown"`
	PoolShareFactor         string `json:"poolShareFactor"`

	// Array of Standby Bonds
	StandbyBonds []string `json:"standbyBonds"`

	// Int64, Number of Standby Nodes
	StandbyNodeCount string `json:"standbyNodeCount"`

	// Int64, Total Rune pooled in all pools
	TotalPooledRune string `json:"totalPooledRune"`

	// Int64, Total left in Reserve
	TotalReserve string `json:"totalReserve"`
}

// NodeKey defines model for NodeKey.
type NodeKey struct {

	// ed25519 public key
	Ed25519 string `json:"ed25519"`

	// node thorchain address
	NodeAddress string `json:"nodeAddress"`

	// secp256k1 public key
	Secp256k1 string `json:"secp256k1"`
}

// NodeKeys defines model for NodeKeys.
type NodeKeys []NodeKey

// PoolDetail defines model for PoolDetail.
type PoolDetail struct {
	Asset string `json:"asset"`

	// Int64, the amount of Asset in the pool.
	AssetDepth string `json:"assetDepth"`

	// Float, price of asset in rune. I.e. rune amount / asset amount.
	AssetPrice string `json:"assetPrice"`

	// Float, Average Percentage Yield: annual return estimated using last weeks income, taking compound interest into account.
	PoolAPY string `json:"poolAPY"`

	// Int64, the amount of Rune in the pool.
	RuneDepth string `json:"runeDepth"`

	// The state of the pool, e.g. Available, Staged.
	Status string `json:"status"`

	// Int64, Liquidity Units in the pool.
	Units string `json:"units"`

	// Int64, the total volume of swaps in the last 24h to and from Rune denoted in Rune.
	Volume24h string `json:"volume24h"`
}

// PoolDetails defines model for PoolDetails.
type PoolDetails []PoolDetail

// PoolLegacyDetail defines model for PoolLegacyDetail.
type PoolLegacyDetail struct {
	Asset string `json:"asset"`

	// Int64, the amount of Asset in the pool
	AssetDepth string `json:"assetDepth"`

	// Float, price of asset in rune. I.e. rune amount / asset amount
	AssetPrice string `json:"assetPrice"`

	// Float, Average Percentage Yield: annual return estimated using last weeks income, taking compound interest into account.
	PoolAPY string `json:"poolAPY"`

	// Int64, same as history/swaps:totalFees
	PoolFeesTotal string `json:"poolFeesTotal"`

	// Float, same as history/swaps:averageSlip
	PoolSlipAverage *string `json:"poolSlipAverage,omitempty"`

	// Float, same as history/swaps totalVolume / totalCount
	PoolTxAverage string `json:"poolTxAverage"`

	// Int64, the amount of Rune in the pool
	RuneDepth string `json:"runeDepth"`

	// The state of the pool, e.g. Available, Staged
	Status string `json:"status"`

	// Int64, same as history/swaps:totalCount
	SwappingTxCount string `json:"swappingTxCount"`

	// Int64, Liquidity Units in the pool
	Units string `json:"units"`

	// Int64, the total volume of swaps in the last 24h to and from Rune denoted in Rune
	Volume24h string `json:"volume24h"`
}

// Queue defines model for Queue.
type Queue struct {
	Outbound string `json:"outbound"`
	Swap     string `json:"swap"`
}

// StatsData defines model for StatsData.
type StatsData struct {

	// Daily active users (unique addresses interacting)
	DailyActiveUsers string `json:"dailyActiveUsers"`

	// Daily transactions
	DailyTx string `json:"dailyTx"`

	// Monthly active users
	MonthlyActiveUsers string `json:"monthlyActiveUsers"`

	// Monthly transactions
	MonthlyTx string `json:"monthlyTx"`

	// Total buying transactions
	TotalAssetBuys string `json:"totalAssetBuys"`

	// Total selling transactions
	TotalAssetSells string `json:"totalAssetSells"`

	// Total RUNE balances
	TotalDepth string `json:"totalDepth"`

	// Total staking transactions
	TotalStakeTx string `json:"totalStakeTx"`

	// Total staked (in RUNE Value)
	TotalStaked string `json:"totalStaked"`

	// Total transactions
	TotalTx string `json:"totalTx"`

	// Total unique swappers & members
	TotalUsers string `json:"totalUsers"`

	// Total (in RUNE Value) of all assets swapped since start
	TotalVolume string `json:"totalVolume"`

	// Total withdrawing transactions
	TotalWithdrawTx string `json:"totalWithdrawTx"`
}

// StringConstants defines model for StringConstants.
type StringConstants struct {
	DefaultPoolStatus string `json:"DefaultPoolStatus"`
}

// SwapHistory defines model for SwapHistory.
type SwapHistory struct {
	Intervals SwapHistoryIntervals `json:"intervals"`
	Meta      SwapHistoryItem      `json:"meta"`
}

// SwapHistoryIntervals defines model for SwapHistoryIntervals.
type SwapHistoryIntervals []SwapHistoryItem

// SwapHistoryItem defines model for SwapHistoryItem.
type SwapHistoryItem struct {

	// Float, the average slip by swap. Big swaps have the same weight as small swaps
	AverageSlip string `json:"averageSlip"`

	// Int64, The end time of bucket in unix timestamp
	EndTime string `json:"endTime"`

	// Int64, The beginning time of bucket in unix timestamp
	StartTime string `json:"startTime"`

	// Int64, count of swaps from rune to asset
	ToAssetCount string `json:"toAssetCount"`

	// Int64, volume of swaps from rune to asset denoted in rune
	ToAssetVolume string `json:"toAssetVolume"`

	// Int64, count of swaps from asset to rune
	ToRuneCount string `json:"toRuneCount"`

	// Int64, volume of swaps from asset to rune denoted in rune
	ToRuneVolume string `json:"toRuneVolume"`

	// Int64, toAssetCount + toRuneCount
	TotalCount string `json:"totalCount"`

	// Int64, the sum of all fees collected denoted in rune
	TotalFees string `json:"totalFees"`

	// Int64, toAssetVolume + toRuneVolume (denoted in rune)
	TotalVolume string `json:"totalVolume"`
}

// TxDetails defines model for TxDetails.
type TxDetails struct {

	// Int64, Unix timestamp
	Date   string `json:"date"`
	Events Event  `json:"events"`
	Height string `json:"height"`
	In     Tx     `json:"in"`
	Out    []Tx   `json:"out"`
	Pool   string `json:"pool"`
	Status string `json:"status"`
	Type   string `json:"type"`
}

// Coin defines model for coin.
type Coin struct {
	Amount string `json:"amount"`
	Asset  string `json:"asset"`
}

// Coins defines model for coins.
type Coins []Coin

// Event defines model for event.
type Event struct {
	Fee        string `json:"fee"`
	Slip       string `json:"slip"`
	StakeUnits string `json:"stakeUnits"`
}

// Option defines model for option.
type Option struct {
	Asymmetry           string `json:"asymmetry"`
	PriceTarget         string `json:"priceTarget"`
	WithdrawBasisPoints string `json:"withdrawBasisPoints"`
}

// Tx defines model for tx.
type Tx struct {
	Address string `json:"address"`
	Coins   Coins  `json:"coins"`
	Memo    string `json:"memo"`
	Options Option `json:"options"`
	TxID    string `json:"txID"`
}

// ConstantsResponse defines model for ConstantsResponse.
type ConstantsResponse Constants

// DepthHistoryResponse defines model for DepthHistoryResponse.
type DepthHistoryResponse DepthHistory

// EarningsHistoryResponse defines model for EarningsHistoryResponse.
type EarningsHistoryResponse EarningsHistory

// HealthResponse defines model for HealthResponse.
type HealthResponse Health

// InboundAddressesResponse defines model for InboundAddressesResponse.
type InboundAddressesResponse InboundAddresses

// LastblockResponse defines model for LastblockResponse.
type LastblockResponse Lastblock

// LiquidityHistoryResponse defines model for LiquidityHistoryResponse.
type LiquidityHistoryResponse LiquidityHistory

// MemberDetailsResponse defines model for MemberDetailsResponse.
type MemberDetailsResponse MemberDetails

// MembersResponse defines model for MembersResponse.
type MembersResponse Members

// NetworkResponse defines model for NetworkResponse.
type NetworkResponse Network

// NodeKeyResponse defines model for NodeKeyResponse.
type NodeKeyResponse NodeKeys

// PoolLegacyResponse defines model for PoolLegacyResponse.
type PoolLegacyResponse PoolLegacyDetail

// PoolResponse defines model for PoolResponse.
type PoolResponse PoolDetail

// PoolsResponse defines model for PoolsResponse.
type PoolsResponse PoolDetails

// QueueResponse defines model for QueueResponse.
type QueueResponse Queue

// StatsResponse defines model for StatsResponse.
type StatsResponse StatsData

// SwapHistoryResponse defines model for SwapHistoryResponse.
type SwapHistoryResponse SwapHistory

// TxResponse defines model for TxResponse.
type TxResponse struct {

	// Int64, count of txs matching the filters.
	Count string      `json:"count"`
	Txs   []TxDetails `json:"txs"`
}

// GetDepthHistoryParams defines parameters for GetDepthHistory.
type GetDepthHistoryParams struct {

	// Interval of calculations
	Interval string `json:"interval"`

	// Number of intervals to return. Should be between [1..100].
	Count *int `json:"count,omitempty"`

	// End time of the query as unix timestamp. If only count is given, defaults to now.
	To *int64 `json:"to,omitempty"`

	// Start time of the query as unix timestamp
	From *int64 `json:"from,omitempty"`
}

// GetEarningsHistoryParams defines parameters for GetEarningsHistory.
type GetEarningsHistoryParams struct {

	// Interval of calculations
	Interval string `json:"interval"`

	// Number of intervals to return. Should be between [1..100].
	Count *int `json:"count,omitempty"`

	// End time of the query as unix timestamp. If only count is given, defaults to now.
	To *int64 `json:"to,omitempty"`

	// Start time of the query as unix timestamp
	From *int64 `json:"from,omitempty"`
}

// GetLiquidityHistoryParams defines parameters for GetLiquidityHistory.
type GetLiquidityHistoryParams struct {

	// Return stats for given pool. Returns sum of all pools if missing
	Pool *string `json:"pool,omitempty"`

	// Interval of calculations
	Interval string `json:"interval"`

	// Number of intervals to return. Should be between [1..100]
	Count *int `json:"count,omitempty"`

	// End time of the query as unix timestamp. If only count is given, defaults to now
	To *int64 `json:"to,omitempty"`

	// Start time of the query as unix timestamp
	From *int64 `json:"from,omitempty"`
}

// GetSwapHistoryParams defines parameters for GetSwapHistory.
type GetSwapHistoryParams struct {

	// Return history given pool. Returns sum of all pools if missing.
	Pool *string `json:"pool,omitempty"`

	// Interval of calculations
	Interval string `json:"interval"`

	// Number of intervals to return. Should be between [1..100].
	Count *int `json:"count,omitempty"`

	// End time of the query as unix timestamp. If only count is given, defaults to now.
	To *int64 `json:"to,omitempty"`

	// Start time of the query as unix timestamp
	From *int64 `json:"from,omitempty"`
}

// GetPoolsParams defines parameters for GetPools.
type GetPoolsParams struct {

	// Filter for only pools with this status
	Status *string `json:"status,omitempty"`
}

// GetTxDetailsParams defines parameters for GetTxDetails.
type GetTxDetailsParams struct {

	// Address of sender or recipient of any in/out tx in event
	Address *string `json:"address,omitempty"`

	// ID of any in/out tx in event
	Txid *string `json:"txid,omitempty"`

	// Any asset used in event (CHAIN.SYMBOL)
	Asset *string `json:"asset,omitempty"`

	// One or more comma separated unique types of event
	Type *string `json:"type,omitempty"`

	// pagination limit
	Limit int64 `json:"limit"`

	// pagination offset
	Offset int64 `json:"offset"`
}

// Base64 encoded, gzipped, json marshaled Swagger object
var swaggerSpec = []string{

	"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",
}

// GetSwagger returns the Swagger specification corresponding to the generated code
// in this file.
func GetSwagger() (*openapi3.Swagger, error) {
	zipped, err := base64.StdEncoding.DecodeString(strings.Join(swaggerSpec, ""))
	if err != nil {
		return nil, fmt.Errorf("error base64 decoding spec: %s", err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(zipped))
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %s", err)
	}
	var buf bytes.Buffer
	_, err = buf.ReadFrom(zr)
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %s", err)
	}

	swagger, err := openapi3.NewSwaggerLoader().LoadSwaggerFromData(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("error loading Swagger: %s", err)
	}
	return swagger, nil
}
