// Package http provides primitives to interact the openapi HTTP API.
//
// Code generated by github.com/deepmap/oapi-codegen DO NOT EDIT.
package http

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/deepmap/oapi-codegen/pkg/runtime"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v4"
	"github.com/pascaldekloe/metrics"
	"github.com/pascaldekloe/metrics/gostat"
)

// AssetDetail defines model for AssetDetail.
type AssetDetail struct {
	Asset       *Asset  `json:"asset,omitempty"`
	DateCreated *int64  `json:"dateCreated,omitempty"`
	PriceRune   *string `json:"priceRune,omitempty"`
}

// BlockRewards defines model for BlockRewards.
type BlockRewards struct {
	BlockReward *string `json:"blockReward,omitempty"`
	BondReward  *string `json:"bondReward,omitempty"`
	StakeReward *string `json:"stakeReward,omitempty"`
}

// BondMetrics defines model for BondMetrics.
type BondMetrics struct {
	AverageActiveBond  *string `json:"averageActiveBond,omitempty"`
	AverageStandbyBond *string `json:"averageStandbyBond,omitempty"`
	MaximumActiveBond  *string `json:"maximumActiveBond,omitempty"`
	MaximumStandbyBond *string `json:"maximumStandbyBond,omitempty"`
	MedianActiveBond   *string `json:"medianActiveBond,omitempty"`
	MedianStandbyBond  *string `json:"medianStandbyBond,omitempty"`
	MinimumActiveBond  *string `json:"minimumActiveBond,omitempty"`
	MinimumStandbyBond *string `json:"minimumStandbyBond,omitempty"`
	TotalActiveBond    *string `json:"totalActiveBond,omitempty"`
	TotalStandbyBond   *string `json:"totalStandbyBond,omitempty"`
}

// Error defines model for Error.
type Error struct {
	Error string `json:"error"`
}

// NetworkInfo defines model for NetworkInfo.
type NetworkInfo struct {
	ActiveBonds             *[]string     `json:"activeBonds,omitempty"`
	ActiveNodeCount         *int          `json:"activeNodeCount,omitempty"`
	BlockRewards            *BlockRewards `json:"blockRewards,omitempty"`
	BondMetrics             *BondMetrics  `json:"bondMetrics,omitempty"`
	BondingROI              *string       `json:"bondingROI,omitempty"`
	NextChurnHeight         *string       `json:"nextChurnHeight,omitempty"`
	PoolActivationCountdown *int64        `json:"poolActivationCountdown,omitempty"`
	PoolShareFactor         *string       `json:"poolShareFactor,omitempty"`
	StakingROI              *string       `json:"stakingROI,omitempty"`
	StandbyBonds            *[]string     `json:"standbyBonds,omitempty"`
	StandbyNodeCount        *int          `json:"standbyNodeCount,omitempty"`
	TotalReserve            *string       `json:"totalReserve,omitempty"`
	TotalStaked             *string       `json:"totalStaked,omitempty"`
}

// NodeKey defines model for NodeKey.
type NodeKey struct {
	Ed25519   *string `json:"ed25519,omitempty"`
	Secp256k1 *string `json:"secp256k1,omitempty"`
}

// PoolDetail defines model for PoolDetail.
type PoolDetail struct {
	Asset            *Asset  `json:"asset,omitempty"`
	AssetDepth       *string `json:"assetDepth,omitempty"`
	AssetROI         *string `json:"assetROI,omitempty"`
	AssetStakedTotal *string `json:"assetStakedTotal,omitempty"`
	BuyAssetCount    *string `json:"buyAssetCount,omitempty"`
	BuyFeeAverage    *string `json:"buyFeeAverage,omitempty"`
	BuyFeesTotal     *string `json:"buyFeesTotal,omitempty"`
	BuySlipAverage   *string `json:"buySlipAverage,omitempty"`
	BuyTxAverage     *string `json:"buyTxAverage,omitempty"`
	BuyVolume        *string `json:"buyVolume,omitempty"`
	PoolDepth        *string `json:"poolDepth,omitempty"`
	PoolFeeAverage   *string `json:"poolFeeAverage,omitempty"`
	PoolFeesTotal    *string `json:"poolFeesTotal,omitempty"`
	PoolROI          *string `json:"poolROI,omitempty"`
	PoolROI12        *string `json:"poolROI12,omitempty"`
	PoolSlipAverage  *string `json:"poolSlipAverage,omitempty"`
	PoolStakedTotal  *string `json:"poolStakedTotal,omitempty"`
	PoolTxAverage    *string `json:"poolTxAverage,omitempty"`
	PoolUnits        *string `json:"poolUnits,omitempty"`
	PoolVolume       *string `json:"poolVolume,omitempty"`
	PoolVolume24hr   *string `json:"poolVolume24hr,omitempty"`
	Price            *string `json:"price,omitempty"`
	RuneDepth        *string `json:"runeDepth,omitempty"`
	RuneROI          *string `json:"runeROI,omitempty"`
	RuneStakedTotal  *string `json:"runeStakedTotal,omitempty"`
	SellAssetCount   *string `json:"sellAssetCount,omitempty"`
	SellFeeAverage   *string `json:"sellFeeAverage,omitempty"`
	SellFeesTotal    *string `json:"sellFeesTotal,omitempty"`
	SellSlipAverage  *string `json:"sellSlipAverage,omitempty"`
	SellTxAverage    *string `json:"sellTxAverage,omitempty"`
	SellVolume       *string `json:"sellVolume,omitempty"`
	StakeTxCount     *string `json:"stakeTxCount,omitempty"`
	StakersCount     *string `json:"stakersCount,omitempty"`
	StakingTxCount   *string `json:"stakingTxCount,omitempty"`
	Status           *string `json:"status,omitempty"`
	SwappersCount    *string `json:"swappersCount,omitempty"`
	SwappingTxCount  *string `json:"swappingTxCount,omitempty"`
	WithdrawTxCount  *string `json:"withdrawTxCount,omitempty"`
}

// Stakers defines model for Stakers.
type Stakers string

// StakersAddressData defines model for StakersAddressData.
type StakersAddressData struct {
	PoolsArray  *[]Asset `json:"poolsArray,omitempty"`
	TotalEarned *string  `json:"totalEarned,omitempty"`
	TotalROI    *string  `json:"totalROI,omitempty"`
	TotalStaked *string  `json:"totalStaked,omitempty"`
}

// StakersAssetData defines model for StakersAssetData.
type StakersAssetData struct {
	Asset            *Asset  `json:"asset,omitempty"`
	DateFirstStaked  *int64  `json:"dateFirstStaked,omitempty"`
	HeightLastStaked *int64  `json:"heightLastStaked,omitempty"`
	StakeUnits       *string `json:"stakeUnits,omitempty"`
}

// StatsData defines model for StatsData.
type StatsData struct {
	DailyActiveUsers   *string `json:"dailyActiveUsers,omitempty"`
	DailyTx            *string `json:"dailyTx,omitempty"`
	MonthlyActiveUsers *string `json:"monthlyActiveUsers,omitempty"`
	MonthlyTx          *string `json:"monthlyTx,omitempty"`
	PoolCount          *string `json:"poolCount,omitempty"`
	TotalAssetBuys     *string `json:"totalAssetBuys,omitempty"`
	TotalAssetSells    *string `json:"totalAssetSells,omitempty"`
	TotalDepth         *string `json:"totalDepth,omitempty"`
	TotalEarned        *string `json:"totalEarned,omitempty"`
	TotalStakeTx       *string `json:"totalStakeTx,omitempty"`
	TotalStaked        *string `json:"totalStaked,omitempty"`
	TotalTx            *string `json:"totalTx,omitempty"`
	TotalUsers         *string `json:"totalUsers,omitempty"`
	TotalVolume        *string `json:"totalVolume,omitempty"`
	TotalVolume24hr    *string `json:"totalVolume24hr,omitempty"`
	TotalWithdrawTx    *string `json:"totalWithdrawTx,omitempty"`
}

// ThorchainBooleanConstants defines model for ThorchainBooleanConstants.
type ThorchainBooleanConstants struct {
	StrictBondStakeRatio *bool `json:"StrictBondStakeRatio,omitempty"`
}

// ThorchainConstants defines model for ThorchainConstants.
type ThorchainConstants struct {
	BoolValues   *ThorchainBooleanConstants `json:"bool_values,omitempty"`
	Int64Values  *ThorchainInt64Constants   `json:"int_64_values,omitempty"`
	StringValues *ThorchainStringConstants  `json:"string_values,omitempty"`
}

// ThorchainEndpoint defines model for ThorchainEndpoint.
type ThorchainEndpoint struct {
	Address *string `json:"address,omitempty"`
	Chain   *string `json:"chain,omitempty"`
	PubKey  *string `json:"pub_key,omitempty"`
}

// ThorchainEndpoints defines model for ThorchainEndpoints.
type ThorchainEndpoints struct {
	Current *[]ThorchainEndpoint `json:"current,omitempty"`
}

// ThorchainInt64Constants defines model for ThorchainInt64Constants.
type ThorchainInt64Constants struct {
	BadValidatorRate                *int64 `json:"BadValidatorRate,omitempty"`
	BlocksPerYear                   *int64 `json:"BlocksPerYear,omitempty"`
	DesireValidatorSet              *int64 `json:"DesireValidatorSet,omitempty"`
	DoubleSignMaxAge                *int64 `json:"DoubleSignMaxAge,omitempty"`
	EmissionCurve                   *int64 `json:"EmissionCurve,omitempty"`
	FailKeySignSlashPoints          *int64 `json:"FailKeySignSlashPoints,omitempty"`
	FailKeygenSlashPoints           *int64 `json:"FailKeygenSlashPoints,omitempty"`
	FundMigrationInterval           *int64 `json:"FundMigrationInterval,omitempty"`
	JailTimeKeygen                  *int64 `json:"JailTimeKeygen,omitempty"`
	JailTimeKeysign                 *int64 `json:"JailTimeKeysign,omitempty"`
	LackOfObservationPenalty        *int64 `json:"LackOfObservationPenalty,omitempty"`
	MinimumBondInRune               *int64 `json:"MinimumBondInRune,omitempty"`
	MinimumNodesForBFT              *int64 `json:"MinimumNodesForBFT,omitempty"`
	MinimumNodesForYggdrasil        *int64 `json:"MinimumNodesForYggdrasil,omitempty"`
	NewPoolCycle                    *int64 `json:"NewPoolCycle,omitempty"`
	ObserveSlashPoints              *int64 `json:"ObserveSlashPoints,omitempty"`
	OldValidatorRate                *int64 `json:"OldValidatorRate,omitempty"`
	RotatePerBlockHeight            *int64 `json:"RotatePerBlockHeight,omitempty"`
	RotateRetryBlocks               *int64 `json:"RotateRetryBlocks,omitempty"`
	SigningTransactionPeriod        *int64 `json:"SigningTransactionPeriod,omitempty"`
	StakeLockUpBlocks               *int64 `json:"StakeLockUpBlocks,omitempty"`
	TransactionFee                  *int64 `json:"TransactionFee,omitempty"`
	ValidatorRotateInNumBeforeFull  *int64 `json:"ValidatorRotateInNumBeforeFull,omitempty"`
	ValidatorRotateNumAfterFull     *int64 `json:"ValidatorRotateNumAfterFull,omitempty"`
	ValidatorRotateOutNumBeforeFull *int64 `json:"ValidatorRotateOutNumBeforeFull,omitempty"`
	WhiteListGasAsset               *int64 `json:"WhiteListGasAsset,omitempty"`
	YggFundLimit                    *int64 `json:"YggFundLimit,omitempty"`
}

// ThorchainLastblock defines model for ThorchainLastblock.
type ThorchainLastblock struct {
	Chain          *string `json:"chain,omitempty"`
	Lastobservedin *int64  `json:"lastobservedin,omitempty"`
	Lastsignedout  *int64  `json:"lastsignedout,omitempty"`
	Thorchain      *int64  `json:"thorchain,omitempty"`
}

// ThorchainStringConstants defines model for ThorchainStringConstants.
type ThorchainStringConstants struct {
	DefaultPoolStatus *string `json:"DefaultPoolStatus,omitempty"`
}

// TotalVolChanges defines model for TotalVolChanges.
type TotalVolChanges struct {
	BuyVolume   *string `json:"buyVolume,omitempty"`
	SellVolume  *string `json:"sellVolume,omitempty"`
	Time        *int64  `json:"time,omitempty"`
	TotalVolume *string `json:"totalVolume,omitempty"`
}

// TxDetails defines model for TxDetails.
type TxDetails struct {
	Date    *int64  `json:"date,omitempty"`
	Events  *Event  `json:"events,omitempty"`
	Gas     *Gas    `json:"gas,omitempty"`
	Height  *string `json:"height,omitempty"`
	In      *Tx     `json:"in,omitempty"`
	Options *Option `json:"options,omitempty"`
	Out     *[]Tx   `json:"out,omitempty"`
	Pool    *Asset  `json:"pool,omitempty"`
	Status  *string `json:"status,omitempty"`
	Type    *string `json:"type,omitempty"`
}

// Asset defines model for asset.
type Asset string

// Coin defines model for coin.
type Coin struct {
	Amount *string `json:"amount,omitempty"`
	Asset  *Asset  `json:"asset,omitempty"`
}

// Coins defines model for coins.
type Coins []Coin

// Event defines model for event.
type Event struct {
	Fee        *string `json:"fee,omitempty"`
	Slip       *string `json:"slip,omitempty"`
	StakeUnits *string `json:"stakeUnits,omitempty"`
}

// Gas defines model for gas.
type Gas struct {
	Amount *string `json:"amount,omitempty"`
	Asset  *Asset  `json:"asset,omitempty"`
}

// Option defines model for option.
type Option struct {
	Asymmetry           *string `json:"asymmetry,omitempty"`
	PriceTarget         *string `json:"priceTarget,omitempty"`
	WithdrawBasisPoints *string `json:"withdrawBasisPoints,omitempty"`
}

// Tx defines model for tx.
type Tx struct {
	Address *string `json:"address,omitempty"`
	Coins   *Coins  `json:"coins,omitempty"`
	Memo    *string `json:"memo,omitempty"`
	TxID    *string `json:"txID,omitempty"`
}

// AssetsDetailedResponse defines model for AssetsDetailedResponse.
type AssetsDetailedResponse []AssetDetail

// GeneralErrorResponse defines model for GeneralErrorResponse.
type GeneralErrorResponse Error

// HealthResponse defines model for HealthResponse.
type HealthResponse struct {
	CatchingUp    *bool  `json:"catching_up,omitempty"`
	Database      *bool  `json:"database,omitempty"`
	ScannerHeight *int64 `json:"scannerHeight,omitempty"`
}

// NetworkResponse defines model for NetworkResponse.
type NetworkResponse NetworkInfo

// NodeKeyResponse defines model for NodeKeyResponse.
type NodeKeyResponse []NodeKey

// PoolsDetailedResponse defines model for PoolsDetailedResponse.
type PoolsDetailedResponse []PoolDetail

// PoolsResponse defines model for PoolsResponse.
type PoolsResponse []Asset

// StakersAddressDataResponse defines model for StakersAddressDataResponse.
type StakersAddressDataResponse StakersAddressData

// StakersAssetDataResponse defines model for StakersAssetDataResponse.
type StakersAssetDataResponse []StakersAssetData

// StakersResponse defines model for StakersResponse.
type StakersResponse []Stakers

// StatsResponse defines model for StatsResponse.
type StatsResponse StatsData

// ThorchainConstantsResponse defines model for ThorchainConstantsResponse.
type ThorchainConstantsResponse ThorchainConstants

// ThorchainEndpointsResponse defines model for ThorchainEndpointsResponse.
type ThorchainEndpointsResponse ThorchainEndpoints

// ThorchainLastblockResponse defines model for ThorchainLastblockResponse.
type ThorchainLastblockResponse ThorchainLastblock

// TotalVolChangesResponse defines model for TotalVolChangesResponse.
type TotalVolChangesResponse []TotalVolChanges

// TxsResponse defines model for TxsResponse.
type TxsResponse struct {
	Count *int64       `json:"count,omitempty"`
	Txs   *[]TxDetails `json:"txs,omitempty"`
}

// GetAssetInfoParams defines parameters for GetAssetInfo.
type GetAssetInfoParams struct {

	// One or more comma separated unique asset (CHAIN.SYMBOL)
	Asset string `json:"asset"`
}

// GetTotalVolChangesParams defines parameters for GetTotalVolChanges.
type GetTotalVolChangesParams struct {

	// Interval of calculations
	Interval string `json:"interval"`

	// Start time of the query as unix timestamp
	From int64 `json:"from"`

	// End time of the query as unix timestamp
	To int64 `json:"to"`
}

// GetPoolsDetailsParams defines parameters for GetPoolsDetails.
type GetPoolsDetailsParams struct {

	// Specifies the returning view
	View *string `json:"view,omitempty"`

	// One or more comma separated unique asset (CHAIN.SYMBOL)
	Asset string `json:"asset"`
}

// GetStakersAddressAndAssetDataParams defines parameters for GetStakersAddressAndAssetData.
type GetStakersAddressAndAssetDataParams struct {

	// One or more comma separated unique asset (CHAIN.SYMBOL)
	Asset string `json:"asset"`
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

	// pagination offset
	Offset int64 `json:"offset"`

	// pagination limit
	Limit int64 `json:"limit"`
}

// ServerInterface represents all server handlers.
type ServerInterface interface {
	// Get Asset Information// (GET /v1/assets)
	GetAssetInfo(ctx echo.Context, params GetAssetInfoParams) error
	// Get Documents// (GET /v1/doc)
	GetDocs(ctx echo.Context) error
	// Get Health// (GET /v1/health)
	GetHealth(ctx echo.Context) error
	// Get Total Volume Changes
	// (GET /v1/history/total_volume)
	GetTotalVolChanges(ctx echo.Context, params GetTotalVolChangesParams) error
	// Get Network Data
	// (GET /v1/network)
	GetNetworkData(ctx echo.Context) error
	// Get Node public keys// (GET /v1/nodes)
	GetNodes(ctx echo.Context) error
	// Get Asset Pools// (GET /v1/pools)
	GetPools(ctx echo.Context) error
	// Get Pools Details
	// (GET /v1/pools/detail)
	GetPoolsDetails(ctx echo.Context, params GetPoolsDetailsParams) error
	// Get Stakers
	// (GET /v1/stakers)
	GetStakersData(ctx echo.Context) error
	// Get Staker Data// (GET /v1/stakers/{address})
	GetStakersAddressData(ctx echo.Context, address string) error
	// Get Staker Pool Data// (GET /v1/stakers/{address}/pools)
	GetStakersAddressAndAssetData(ctx echo.Context, address string, params GetStakersAddressAndAssetDataParams) error
	// Get Global Stats// (GET /v1/stats)
	GetStats(ctx echo.Context) error
	// Get Swagger// (GET /v1/swagger.json)
	GetSwagger(ctx echo.Context) error
	// Get the Proxied THORChain Constants// (GET /v1/thorchain/constants)
	GetThorchainProxiedConstants(ctx echo.Context) error
	// Get the Proxied THORChain Lastblock// (GET /v1/thorchain/lastblock)
	GetThorchainProxiedLastblock(ctx echo.Context) error
	// Get the Proxied Pool Addresses// (GET /v1/thorchain/pool_addresses)
	GetThorchainProxiedEndpoints(ctx echo.Context) error
	// Get details of a tx by address, asset or tx-id// (GET /v1/txs)
	GetTxDetails(ctx echo.Context, params GetTxDetailsParams) error
}

// ServerInterfaceWrapper converts echo contexts to parameters.
type ServerInterfaceWrapper struct {
	Handler ServerInterface
}

// GetAssetInfo converts echo context to params.
func (w *ServerInterfaceWrapper) GetAssetInfo(ctx echo.Context) error {
	var err error

	// Parameter object where we will unmarshal all parameters from the context
	var params GetAssetInfoParams
	// ------------- Required query parameter "asset" -------------
	if paramValue := ctx.QueryParam("asset"); paramValue != "" {

	} else {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Query argument asset is required, but not found"))
	}

	err = runtime.BindQueryParameter("form", true, true, "asset", ctx.QueryParams(), &params.Asset)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter asset: %s", err))
	}

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.GetAssetInfo(ctx, params)
	return err
}

// GetDocs converts echo context to params.
func (w *ServerInterfaceWrapper) GetDocs(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.GetDocs(ctx)
	return err
}

// GetHealth converts echo context to params.
func (w *ServerInterfaceWrapper) GetHealth(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.GetHealth(ctx)
	return err
}

// GetTotalVolChanges converts echo context to params.
func (w *ServerInterfaceWrapper) GetTotalVolChanges(ctx echo.Context) error {
	var err error

	// Parameter object where we will unmarshal all parameters from the context
	var params GetTotalVolChangesParams
	// ------------- Required query parameter "interval" -------------

	err = runtime.BindQueryParameter("form", true, true, "interval", ctx.QueryParams(), &params.Interval)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter interval: %s", err))
	}

	// ------------- Required query parameter "from" -------------

	err = runtime.BindQueryParameter("form", true, true, "from", ctx.QueryParams(), &params.From)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter from: %s", err))
	}

	// ------------- Required query parameter "to" -------------

	err = runtime.BindQueryParameter("form", true, true, "to", ctx.QueryParams(), &params.To)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter to: %s", err))
	}

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.GetTotalVolChanges(ctx, params)
	return err
}

// GetNetworkData converts echo context to params.
func (w *ServerInterfaceWrapper) GetNetworkData(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.GetNetworkData(ctx)
	return err
}

// GetNodes converts echo context to params.
func (w *ServerInterfaceWrapper) GetNodes(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.GetNodes(ctx)
	return err
}

// GetPools converts echo context to params.
func (w *ServerInterfaceWrapper) GetPools(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.GetPools(ctx)
	return err
}

// GetPoolsDetails converts echo context to params.
func (w *ServerInterfaceWrapper) GetPoolsDetails(ctx echo.Context) error {
	var err error

	// Parameter object where we will unmarshal all parameters from the context
	var params GetPoolsDetailsParams
	// ------------- Optional query parameter "view" -------------

	err = runtime.BindQueryParameter("form", true, false, "view", ctx.QueryParams(), &params.View)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter view: %s", err))
	}

	// ------------- Required query parameter "asset" -------------
	if paramValue := ctx.QueryParam("asset"); paramValue != "" {

	} else {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Query argument asset is required, but not found"))
	}

	err = runtime.BindQueryParameter("form", true, true, "asset", ctx.QueryParams(), &params.Asset)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter asset: %s", err))
	}

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.GetPoolsDetails(ctx, params)
	return err
}

// GetStakersData converts echo context to params.
func (w *ServerInterfaceWrapper) GetStakersData(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.GetStakersData(ctx)
	return err
}

// GetStakersAddressData converts echo context to params.
func (w *ServerInterfaceWrapper) GetStakersAddressData(ctx echo.Context) error {
	var err error
	// ------------- Path parameter "address" -------------
	var address string

	err = runtime.BindStyledParameter("simple", false, "address", ctx.Param("address"), &address)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter address: %s", err))
	}

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.GetStakersAddressData(ctx, address)
	return err
}

// GetStakersAddressAndAssetData converts echo context to params.
func (w *ServerInterfaceWrapper) GetStakersAddressAndAssetData(ctx echo.Context) error {
	var err error
	// ------------- Path parameter "address" -------------
	var address string

	err = runtime.BindStyledParameter("simple", false, "address", ctx.Param("address"), &address)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter address: %s", err))
	}

	// Parameter object where we will unmarshal all parameters from the context
	var params GetStakersAddressAndAssetDataParams
	// ------------- Required query parameter "asset" -------------
	if paramValue := ctx.QueryParam("asset"); paramValue != "" {

	} else {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Query argument asset is required, but not found"))
	}

	err = runtime.BindQueryParameter("form", true, true, "asset", ctx.QueryParams(), &params.Asset)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter asset: %s", err))
	}

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.GetStakersAddressAndAssetData(ctx, address, params)
	return err
}

// GetStats converts echo context to params.
func (w *ServerInterfaceWrapper) GetStats(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.GetStats(ctx)
	return err
}

// GetSwagger converts echo context to params.
func (w *ServerInterfaceWrapper) GetSwagger(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.GetSwagger(ctx)
	return err
}

// GetThorchainProxiedConstants converts echo context to params.
func (w *ServerInterfaceWrapper) GetThorchainProxiedConstants(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.GetThorchainProxiedConstants(ctx)
	return err
}

// GetThorchainProxiedLastblock converts echo context to params.
func (w *ServerInterfaceWrapper) GetThorchainProxiedLastblock(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.GetThorchainProxiedLastblock(ctx)
	return err
}

// GetThorchainProxiedEndpoints converts echo context to params.
func (w *ServerInterfaceWrapper) GetThorchainProxiedEndpoints(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.GetThorchainProxiedEndpoints(ctx)
	return err
}

// GetTxDetails converts echo context to params.
func (w *ServerInterfaceWrapper) GetTxDetails(ctx echo.Context) error {
	var err error

	// Parameter object where we will unmarshal all parameters from the context
	var params GetTxDetailsParams
	// ------------- Optional query parameter "address" -------------
	if paramValue := ctx.QueryParam("address"); paramValue != "" {

	}

	err = runtime.BindQueryParameter("form", true, false, "address", ctx.QueryParams(), &params.Address)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter address: %s", err))
	}

	// ------------- Optional query parameter "txid" -------------
	if paramValue := ctx.QueryParam("txid"); paramValue != "" {

	}

	err = runtime.BindQueryParameter("form", true, false, "txid", ctx.QueryParams(), &params.Txid)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter txid: %s", err))
	}

	// ------------- Optional query parameter "asset" -------------
	if paramValue := ctx.QueryParam("asset"); paramValue != "" {

	}

	err = runtime.BindQueryParameter("form", true, false, "asset", ctx.QueryParams(), &params.Asset)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter asset: %s", err))
	}

	// ------------- Optional query parameter "type" -------------
	if paramValue := ctx.QueryParam("type"); paramValue != "" {

	}

	err = runtime.BindQueryParameter("form", true, false, "type", ctx.QueryParams(), &params.Type)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter type: %s", err))
	}

	// ------------- Required query parameter "offset" -------------
	if paramValue := ctx.QueryParam("offset"); paramValue != "" {

	} else {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Query argument offset is required, but not found"))
	}

	err = runtime.BindQueryParameter("form", true, true, "offset", ctx.QueryParams(), &params.Offset)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter offset: %s", err))
	}

	// ------------- Required query parameter "limit" -------------
	if paramValue := ctx.QueryParam("limit"); paramValue != "" {

	} else {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Query argument limit is required, but not found"))
	}

	err = runtime.BindQueryParameter("form", true, true, "limit", ctx.QueryParams(), &params.Limit)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter limit: %s", err))
	}

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.GetTxDetails(ctx, params)
	return err
}

// This is a simple interface which specifies echo.Route addition functions which
// are present on both echo.Echo and echo.Group, since we want to allow using
// either of them for path registration
type EchoRouter interface {
	CONNECT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	DELETE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	HEAD(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	OPTIONS(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	PATCH(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	POST(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	PUT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	TRACE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
}

// RegisterHandlers adds each server route to the EchoRouter.
func RegisterHandlers(router EchoRouter, si ServerInterface) {

	wrapper := ServerInterfaceWrapper{
		Handler: si,
	}

	router.GET("/v1/assets", wrapper.GetAssetInfo)
	router.GET("/v1/doc", wrapper.GetDocs)
	router.GET("/v1/health", wrapper.GetHealth)
	router.GET("/v1/history/total_volume", wrapper.GetTotalVolChanges)
	router.GET("/v1/network", wrapper.GetNetworkData)
	router.GET("/v1/nodes", wrapper.GetNodes)
	router.GET("/v1/pools", wrapper.GetPools)
	router.GET("/v1/pools/detail", wrapper.GetPoolsDetails)
	router.GET("/v1/stakers", wrapper.GetStakersData)
	router.GET("/v1/stakers/:address", wrapper.GetStakersAddressData)
	router.GET("/v1/stakers/:address/pools", wrapper.GetStakersAddressAndAssetData)
	router.GET("/v1/stats", wrapper.GetStats)
	router.GET("/v1/swagger.json", wrapper.GetSwagger)
	router.GET("/v1/thorchain/constants", wrapper.GetThorchainProxiedConstants)
	router.GET("/v1/thorchain/lastblock", wrapper.GetThorchainProxiedLastblock)
	router.GET("/v1/thorchain/pool_addresses", wrapper.GetThorchainProxiedEndpoints)
	router.GET("/v1/txs", wrapper.GetTxDetails)

	// TODO(pascaldekloe): Configure the HTTP server in main (and main only).
	// This Echo framework is so viral we need to add in here.
	router.GET("/metrics", echo.WrapHandler(http.HandlerFunc(metrics.ServeHTTP)))
	gostat.CaptureEvery(10*time.Second)
}

// Base64 encoded, gzipped, json marshaled Swagger object
var swaggerSpec = []string{

	"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",
	"V54Bf778TwAAAP//4UkwCBtbAAA=",
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
