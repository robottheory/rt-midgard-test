// Package api provides primitives to interact the openapi HTTP API.
//
// Code generated by github.com/deepmap/oapi-codegen DO NOT EDIT.
package api

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/deepmap/oapi-codegen/pkg/runtime"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v4"
)

// AssetDetail defines model for AssetDetail.
type AssetDetail struct {
	Asset       *Asset  `json:"asset,omitempty"`
	DateCreated *int64  `json:"dateCreated,omitempty"`
	Logo        *string `json:"logo,omitempty"`
	Name        *string `json:"name,omitempty"`
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

	// Average bond of active nodes
	AverageActiveBond *string `json:"averageActiveBond,omitempty"`

	// Average bond of standby nodes
	AverageStandbyBond *string `json:"averageStandbyBond,omitempty"`

	// Maxinum bond of active nodes
	MaximumActiveBond *string `json:"maximumActiveBond,omitempty"`

	// Maximum bond of standby nodes
	MaximumStandbyBond *string `json:"maximumStandbyBond,omitempty"`

	// Median bond of active nodes
	MedianActiveBond *string `json:"medianActiveBond,omitempty"`

	// Median bond of standby nodes
	MedianStandbyBond *string `json:"medianStandbyBond,omitempty"`

	// Minumum bond of active nodes
	MinimumActiveBond *string `json:"minimumActiveBond,omitempty"`

	// Minumum bond of standby nodes
	MinimumStandbyBond *string `json:"minimumStandbyBond,omitempty"`

	// Total bond of active nodes
	TotalActiveBond *string `json:"totalActiveBond,omitempty"`

	// Total bond of standby nodes
	TotalStandbyBond *string `json:"totalStandbyBond,omitempty"`
}

// Error defines model for Error.
type Error struct {
	Error string `json:"error"`
}

// NetworkInfo defines model for NetworkInfo.
type NetworkInfo struct {

	// Array of Active Bonds
	ActiveBonds *[]string `json:"activeBonds,omitempty"`

	// Number of Active Nodes
	ActiveNodeCount *int          `json:"activeNodeCount,omitempty"`
	BlockRewards    *BlockRewards `json:"blockRewards,omitempty"`
	BondMetrics     *BondMetrics  `json:"bondMetrics,omitempty"`
	BondingROI      *string       `json:"bondingROI,omitempty"`
	NextChurnHeight *string       `json:"nextChurnHeight,omitempty"`
	PoolShareFactor *string       `json:"poolShareFactor,omitempty"`
	StakingROI      *string       `json:"stakingROI,omitempty"`

	// Array of Standby Bonds
	StandbyBonds *[]string `json:"standbyBonds,omitempty"`

	// Number of Standby Nodes
	StandbyNodeCount *int `json:"standbyNodeCount,omitempty"`

	// Total left in Reserve
	TotalReserve *string `json:"totalReserve,omitempty"`

	// Total Rune Staked in Pools
	TotalStaked *string `json:"totalStaked,omitempty"`
}

// PoolDetail defines model for PoolDetail.
type PoolDetail struct {
	Asset *Asset `json:"asset,omitempty"`

	// Total current Asset balance
	AssetDepth *string `json:"assetDepth,omitempty"`

	// Asset return on investment
	AssetROI *string `json:"assetROI,omitempty"`

	// Total Asset staked
	AssetStakedTotal *string `json:"assetStakedTotal,omitempty"`

	// Number of RUNE->ASSET transactions
	BuyAssetCount *string `json:"buyAssetCount,omitempty"`

	// Average sell Asset fee size for RUNE->ASSET (in ASSET)
	BuyFeeAverage *string `json:"buyFeeAverage,omitempty"`

	// Total fees (in Asset)
	BuyFeesTotal *string `json:"buyFeesTotal,omitempty"`

	// Average trade slip for RUNE->ASSET in %
	BuySlipAverage *string `json:"buySlipAverage,omitempty"`

	// Average Asset buy transaction size for (RUNE->ASSET) (in ASSET)
	BuyTxAverage *string `json:"buyTxAverage,omitempty"`

	// Total Asset buy volume (RUNE->ASSET) (in Asset)
	BuyVolume *string `json:"buyVolume,omitempty"`

	// Total depth of both sides (in RUNE)
	PoolDepth *string `json:"poolDepth,omitempty"`

	// Average pool fee
	PoolFeeAverage *string `json:"poolFeeAverage,omitempty"`

	// Total fees
	PoolFeesTotal *string `json:"poolFeesTotal,omitempty"`

	// Pool ROI (average of RUNE and Asset ROI)
	PoolROI *string `json:"poolROI,omitempty"`

	// Pool ROI over 12 months
	PoolROI12 *string `json:"poolROI12,omitempty"`

	// Average pool slip
	PoolSlipAverage *string `json:"poolSlipAverage,omitempty"`

	// Rune value staked Total
	PoolStakedTotal *string `json:"poolStakedTotal,omitempty"`

	// Average pool transaction
	PoolTxAverage *string `json:"poolTxAverage,omitempty"`

	// Total pool units outstanding
	PoolUnits *string `json:"poolUnits,omitempty"`

	// Two-way volume of all-time (in RUNE)
	PoolVolume *string `json:"poolVolume,omitempty"`

	// Two-way volume in 24hrs (in RUNE)
	PoolVolume24hr *string `json:"poolVolume24hr,omitempty"`

	// Price of Asset (in RUNE).
	Price *string `json:"price,omitempty"`

	// Total current Rune balance
	RuneDepth *string `json:"runeDepth,omitempty"`

	// RUNE return on investment
	RuneROI *string `json:"runeROI,omitempty"`

	// Total RUNE staked
	RuneStakedTotal *string `json:"runeStakedTotal,omitempty"`

	// Number of ASSET->RUNE transactions
	SellAssetCount *string `json:"sellAssetCount,omitempty"`

	// Average buy Asset fee size for ASSET->RUNE (in RUNE)
	SellFeeAverage *string `json:"sellFeeAverage,omitempty"`

	// Total fees (in RUNE)
	SellFeesTotal *string `json:"sellFeesTotal,omitempty"`

	// Average trade slip for ASSET->RUNE in %
	SellSlipAverage *string `json:"sellSlipAverage,omitempty"`

	// Average Asset sell transaction size (ASSET>RUNE) (in RUNE)
	SellTxAverage *string `json:"sellTxAverage,omitempty"`

	// Total Asset sell volume (ASSET>RUNE) (in RUNE).
	SellVolume *string `json:"sellVolume,omitempty"`

	// Number of stake transactions
	StakeTxCount *string `json:"stakeTxCount,omitempty"`

	// Number of unique stakers
	StakersCount *string `json:"stakersCount,omitempty"`

	// Number of stake & withdraw transactions
	StakingTxCount *string `json:"stakingTxCount,omitempty"`
	Status         *string `json:"status,omitempty"`

	// Number of unique swappers interacting with pool
	SwappersCount *string `json:"swappersCount,omitempty"`

	// Number of swapping transactions in the pool (buys and sells)
	SwappingTxCount *string `json:"swappingTxCount,omitempty"`

	// Number of withdraw transactions
	WithdrawTxCount *string `json:"withdrawTxCount,omitempty"`
}

// Stakers defines model for Stakers.
type Stakers string

// StakersAddressData defines model for StakersAddressData.
type StakersAddressData struct {
	PoolsArray *[]Asset `json:"poolsArray,omitempty"`

	// Total value of earnings (in RUNE) across all pools.
	TotalEarned *string `json:"totalEarned,omitempty"`

	// Average of all pool ROIs.
	TotalROI *string `json:"totalROI,omitempty"`

	// Total staked (in RUNE) across all pools.
	TotalStaked *string `json:"totalStaked,omitempty"`
}

// StakersAssetData defines model for StakersAssetData.
type StakersAssetData struct {
	Asset *Asset `json:"asset,omitempty"`

	// Value of Assets earned from the pool.
	AssetEarned *string `json:"assetEarned,omitempty"`

	// ROI of the Asset side
	AssetROI *string `json:"assetROI,omitempty"`

	// Amount of Assets staked.
	AssetStaked     *string `json:"assetStaked,omitempty"`
	DateFirstStaked *int64  `json:"dateFirstStaked,omitempty"`

	// Total value of earnings (in RUNE).
	PoolEarned *string `json:"poolEarned,omitempty"`

	// Average ROI (in RUNE) of both sides
	PoolROI *string `json:"poolROI,omitempty"`

	// RUNE value staked.
	PoolStaked *string `json:"poolStaked,omitempty"`

	// Value of RUNE earned from the pool.
	RuneEarned *string `json:"runeEarned,omitempty"`

	// ROI of the Rune side.
	RuneROI *string `json:"runeROI,omitempty"`

	// Amount of RUNE staked.
	RuneStaked *string `json:"runeStaked,omitempty"`

	// Represents ownership of a pool.
	StakeUnits *string `json:"stakeUnits,omitempty"`
}

// StatsData defines model for StatsData.
type StatsData struct {

	// Daily active users (unique addresses interacting)
	DailyActiveUsers *string `json:"dailyActiveUsers,omitempty"`

	// Daily transactions
	DailyTx *string `json:"dailyTx,omitempty"`

	// Monthly active users
	MonthlyActiveUsers *string `json:"monthlyActiveUsers,omitempty"`

	// Monthly transactions
	MonthlyTx *string `json:"monthlyTx,omitempty"`

	// Number of active pools
	PoolCount *string `json:"poolCount,omitempty"`

	// Total buying transactions
	TotalAssetBuys *string `json:"totalAssetBuys,omitempty"`

	// Total selling transactions
	TotalAssetSells *string `json:"totalAssetSells,omitempty"`

	// Total RUNE balances
	TotalDepth *string `json:"totalDepth,omitempty"`

	// Total earned (in RUNE Value).
	TotalEarned *string `json:"totalEarned,omitempty"`

	// Total staking transactions
	TotalStakeTx *string `json:"totalStakeTx,omitempty"`

	// Total staked (in RUNE Value).
	TotalStaked *string `json:"totalStaked,omitempty"`

	// Total transactions
	TotalTx *string `json:"totalTx,omitempty"`

	// Total unique swappers & stakers
	TotalUsers *string `json:"totalUsers,omitempty"`

	// Total (in RUNE Value) of all assets swapped since start.
	TotalVolume *string `json:"totalVolume,omitempty"`

	// Total (in RUNE Value) of all assets swapped in 24hrs
	TotalVolume24hr *string `json:"totalVolume24hr,omitempty"`

	// Total withdrawing transactions
	TotalWithdrawTx *string `json:"totalWithdrawTx,omitempty"`
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

// NetworkResponse defines model for NetworkResponse.
type NetworkResponse NetworkInfo

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

// ThorchainEndpointsResponse defines model for ThorchainEndpointsResponse.
type ThorchainEndpointsResponse ThorchainEndpoints

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

// GetPoolsDataParams defines parameters for GetPoolsData.
type GetPoolsDataParams struct {

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

	// Requested type of events
	Type *string `json:"type,omitempty"`

	// pagination offset
	Offset int64 `json:"offset"`

	// pagination limit
	Limit int64 `json:"limit"`
}

// ServerInterface represents all server handlers.
type ServerInterface interface {
	// Get Asset Information
	// (GET /v1/assets)
	GetAssetInfo(ctx echo.Context, params GetAssetInfoParams) error
	// Get Documents
	// (GET /v1/doc)
	GetDocs(ctx echo.Context) error
	// Get Health
	// (GET /v1/health)
	GetHealth(ctx echo.Context) error
	// Get Network Data
	// (GET /v1/network)
	GetNetworkData(ctx echo.Context) error
	// Get Asset Pools
	// (GET /v1/pools)
	GetPools(ctx echo.Context) error
	// Get Pools Data
	// (GET /v1/pools/detail)
	GetPoolsData(ctx echo.Context, params GetPoolsDataParams) error
	// Get Stakers
	// (GET /v1/stakers)
	GetStakersData(ctx echo.Context) error
	// Get Staker Data
	// (GET /v1/stakers/{address})
	GetStakersAddressData(ctx echo.Context, address string) error
	// Get Staker Pool Data
	// (GET /v1/stakers/{address}/pools)
	GetStakersAddressAndAssetData(ctx echo.Context, address string, params GetStakersAddressAndAssetDataParams) error
	// Get Global Stats
	// (GET /v1/stats)
	GetStats(ctx echo.Context) error
	// Get Swagger
	// (GET /v1/swagger.json)
	GetSwagger(ctx echo.Context) error
	// Get the Proxied Pool Addresses
	// (GET /v1/thorchain/pool_addresses)
	GetThorchainProxiedEndpoints(ctx echo.Context) error
	// Get details of a tx by address, asset or tx-id
	// (GET /v1/txs)
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

// GetNetworkData converts echo context to params.
func (w *ServerInterfaceWrapper) GetNetworkData(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.GetNetworkData(ctx)
	return err
}

// GetPools converts echo context to params.
func (w *ServerInterfaceWrapper) GetPools(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.GetPools(ctx)
	return err
}

// GetPoolsData converts echo context to params.
func (w *ServerInterfaceWrapper) GetPoolsData(ctx echo.Context) error {
	var err error

	// Parameter object where we will unmarshal all parameters from the context
	var params GetPoolsDataParams
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
	err = w.Handler.GetPoolsData(ctx, params)
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
	router.GET("/v1/network", wrapper.GetNetworkData)
	router.GET("/v1/pools", wrapper.GetPools)
	router.GET("/v1/pools/detail", wrapper.GetPoolsData)
	router.GET("/v1/stakers", wrapper.GetStakersData)
	router.GET("/v1/stakers/:address", wrapper.GetStakersAddressData)
	router.GET("/v1/stakers/:address/pools", wrapper.GetStakersAddressAndAssetData)
	router.GET("/v1/stats", wrapper.GetStats)
	router.GET("/v1/swagger.json", wrapper.GetSwagger)
	router.GET("/v1/thorchain/pool_addresses", wrapper.GetThorchainProxiedEndpoints)
	router.GET("/v1/txs", wrapper.GetTxDetails)
}

// Base64 encoded, gzipped, json marshaled Swagger object
var swaggerSpec = []string{

	"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",
	"RQAA",
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
