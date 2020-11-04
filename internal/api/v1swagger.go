package api

import (
	"io"
	"net/http"
)

func serveV1SwaggerJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	// error discarded
	_, _ = io.WriteString(w, `{
   "components": {
      "responses": {
         "AssetsDetailedResponse": {
            "content": {
               "application/json": {
                  "schema": {
                     "items": {
                        "$ref": "#/components/schemas/AssetDetail"
                     },
                     "type": "array"
                  }
               }
            },
            "description": "object containing detailed asset information"
         },
         "GeneralErrorResponse": {
            "content": {
               "application/json": {
                  "schema": {
                     "$ref": "#/components/schemas/Error"
                  }
               }
            },
            "description": "error message"
         },
         "HealthResponse": {
            "content": {
               "application/json": {
                  "schema": {
                     "properties": {
                        "catching_up": {
                           "type": "boolean"
                        },
                        "database": {
                           "type": "boolean"
                        },
                        "scannerHeight": {
                           "format": "int64",
                           "type": "integer"
                        }
                     },
                     "type": "object"
                  }
               }
            },
            "description": "Returns an health status of Midgard"
         },
         "NetworkResponse": {
            "content": {
               "application/json": {
                  "schema": {
                     "$ref": "#/components/schemas/NetworkInfo"
                  }
               }
            },
            "description": "Returns an object containing Network data"
         },
         "NodeKeyResponse": {
            "content": {
               "application/json": {
                  "schema": {
                     "items": {
                        "$ref": "#/components/schemas/NodeKey"
                     },
                     "type": "array"
                  }
               }
            },
            "description": "Returns an object containing Network data"
         },
         "PoolsDetailedResponse": {
            "content": {
               "application/json": {
                  "schema": {
                     "items": {
                        "$ref": "#/components/schemas/PoolDetail"
                     },
                     "type": "array"
                  }
               }
            },
            "description": "object containing pool data for that asset"
         },
         "PoolsResponse": {
            "content": {
               "application/json": {
                  "schema": {
                     "items": {
                        "$ref": "#/components/schemas/asset"
                     },
                     "type": "array"
                  }
               }
            },
            "description": "array of assets"
         },
         "StakersAddressDataResponse": {
            "content": {
               "application/json": {
                  "schema": {
                     "$ref": "#/components/schemas/StakersAddressData"
                  }
               }
            },
            "description": "array of all the pools the staker is staking in"
         },
         "StakersAssetDataResponse": {
            "content": {
               "application/json": {
                  "schema": {
                     "items": {
                        "$ref": "#/components/schemas/StakersAssetData"
                     },
                     "type": "array"
                  }
               }
            },
            "description": "object containing staking data for a specific staker and asset"
         },
         "StakersResponse": {
            "content": {
               "application/json": {
                  "schema": {
                     "items": {
                        "$ref": "#/components/schemas/Stakers"
                     },
                     "type": "array"
                  }
               }
            },
            "description": "array of all the stakers"
         },
         "StatsResponse": {
            "content": {
               "application/json": {
                  "schema": {
                     "$ref": "#/components/schemas/StatsData"
                  }
               }
            },
            "description": "object containing global BEPSwap data"
         },
         "ThorchainConstantsResponse": {
            "content": {
               "application/json": {
                  "schema": {
                     "$ref": "#/components/schemas/ThorchainConstants"
                  }
               }
            },
            "description": "Get Return an object for the proxied constants endpoint."
         },
         "ThorchainEndpointsResponse": {
            "content": {
               "application/json": {
                  "schema": {
                     "$ref": "#/components/schemas/ThorchainEndpoints"
                  }
               }
            },
            "description": "Get Return an object for the proxied pools_addresses endpoint."
         },
         "ThorchainLastblockResponse": {
            "content": {
               "application/json": {
                  "schema": {
                     "$ref": "#/components/schemas/ThorchainLastblock"
                  }
               }
            },
            "description": "Get Return an object for the proxied lastblock endpoint."
         },
         "TotalVolChangesResponse": {
            "content": {
               "application/json": {
                  "schema": {
                     "items": {
                        "$ref": "#/components/schemas/TotalVolChanges"
                     },
                     "type": "array"
                  }
               }
            },
            "description": "Get Return an array of total volume changes."
         },
         "TxsResponse": {
            "content": {
               "application/json": {
                  "schema": {
                     "properties": {
                        "count": {
                           "format": "int64",
                           "type": "integer"
                        },
                        "txs": {
                           "items": {
                              "$ref": "#/components/schemas/TxDetails"
                           },
                           "type": "array"
                        }
                     },
                     "type": "object"
                  }
               }
            },
            "description": "Returns an array of transactions"
         }
      },
      "schemas": {
         "AssetDetail": {
            "properties": {
               "asset": {
                  "$ref": "#/components/schemas/asset"
               },
               "dateCreated": {
                  "format": "int64",
                  "type": "integer"
               },
               "priceRune": {
                  "type": "string"
               }
            },
            "type": "object"
         },
         "BlockRewards": {
            "properties": {
               "blockReward": {
                  "type": "string"
               },
               "bondReward": {
                  "type": "string"
               },
               "stakeReward": {
                  "type": "string"
               }
            },
            "type": "object"
         },
         "BondMetrics": {
            "properties": {
               "averageActiveBond": {
                  "description": "Average bond of active nodes",
                  "type": "string"
               },
               "averageStandbyBond": {
                  "description": "Average bond of standby nodes",
                  "type": "string"
               },
               "maximumActiveBond": {
                  "description": "Maxinum bond of active nodes",
                  "type": "string"
               },
               "maximumStandbyBond": {
                  "description": "Maximum bond of standby nodes",
                  "type": "string"
               },
               "medianActiveBond": {
                  "description": "Median bond of active nodes",
                  "type": "string"
               },
               "medianStandbyBond": {
                  "description": "Median bond of standby nodes",
                  "type": "string"
               },
               "minimumActiveBond": {
                  "description": "Minumum bond of active nodes",
                  "type": "string"
               },
               "minimumStandbyBond": {
                  "description": "Minumum bond of standby nodes",
                  "type": "string"
               },
               "totalActiveBond": {
                  "description": "Total bond of active nodes",
                  "type": "string"
               },
               "totalStandbyBond": {
                  "description": "Total bond of standby nodes",
                  "type": "string"
               }
            },
            "type": "object"
         },
         "Error": {
            "properties": {
               "error": {
                  "type": "string"
               }
            },
            "required": [
               "error"
            ],
            "type": "object"
         },
         "NetworkInfo": {
            "properties": {
               "activeBonds": {
                  "description": "Array of Active Bonds",
                  "items": {
                     "type": "string"
                  },
                  "type": "array"
               },
               "activeNodeCount": {
                  "description": "Number of Active Nodes",
                  "type": "integer"
               },
               "blockRewards": {
                  "$ref": "#/components/schemas/BlockRewards"
               },
               "bondMetrics": {
                  "$ref": "#/components/schemas/BondMetrics"
               },
               "bondingROI": {
                  "type": "string"
               },
               "nextChurnHeight": {
                  "type": "string"
               },
               "poolActivationCountdown": {
                  "description": "The remaining time of pool activation (in blocks)",
                  "format": "int64",
                  "type": "integer"
               },
               "poolShareFactor": {
                  "type": "string"
               },
               "stakingROI": {
                  "type": "string"
               },
               "standbyBonds": {
                  "description": "Array of Standby Bonds",
                  "items": {
                     "type": "string"
                  },
                  "type": "array"
               },
               "standbyNodeCount": {
                  "description": "Number of Standby Nodes",
                  "type": "integer"
               },
               "totalReserve": {
                  "description": "Total left in Reserve",
                  "type": "string"
               },
               "totalStaked": {
                  "description": "Total Rune Staked in Pools",
                  "type": "string"
               }
            },
            "type": "object"
         },
         "NodeKey": {
            "properties": {
               "ed25519": {
                  "description": "ed25519 public key",
                  "type": "string"
               },
               "secp256k1": {
                  "description": "secp256k1 public key",
                  "type": "string"
               }
            },
            "type": "object"
         },
         "PoolDetail": {
            "properties": {
               "asset": {
                  "$ref": "#/components/schemas/asset"
               },
               "assetDepth": {
                  "description": "Total current Asset balance",
                  "type": "string"
               },
               "assetROI": {
                  "description": "Asset return on investment",
                  "type": "string"
               },
               "assetStakedTotal": {
                  "description": "Total Asset staked",
                  "type": "string"
               },
               "buyAssetCount": {
                  "description": "Number of RUNE-\u003eASSET transactions",
                  "type": "string"
               },
               "buyFeeAverage": {
                  "description": "Average sell Asset fee size for RUNE-\u003eASSET (in ASSET)",
                  "type": "string"
               },
               "buyFeesTotal": {
                  "description": "Total fees (in Asset)",
                  "type": "string"
               },
               "buySlipAverage": {
                  "description": "Average trade slip for RUNE-\u003eASSET in %",
                  "type": "string"
               },
               "buyTxAverage": {
                  "description": "Average Asset buy transaction size for (RUNE-\u003eASSET) (in ASSET)",
                  "type": "string"
               },
               "buyVolume": {
                  "description": "Total Asset buy volume (RUNE-\u003eASSET) (in Asset)",
                  "type": "string"
               },
               "poolDepth": {
                  "description": "Total depth of both sides (in RUNE)",
                  "type": "string"
               },
               "poolFeeAverage": {
                  "description": "Average pool fee",
                  "type": "string"
               },
               "poolFeesTotal": {
                  "description": "Total fees",
                  "type": "string"
               },
               "poolROI": {
                  "description": "Pool ROI (average of RUNE and Asset ROI)",
                  "type": "string"
               },
               "poolROI12": {
                  "description": "Pool ROI over 12 months",
                  "type": "string"
               },
               "poolSlipAverage": {
                  "description": "Average pool slip",
                  "type": "string"
               },
               "poolStakedTotal": {
                  "description": "Rune value staked Total",
                  "type": "string"
               },
               "poolTxAverage": {
                  "description": "Average pool transaction",
                  "type": "string"
               },
               "poolUnits": {
                  "description": "Total pool units outstanding",
                  "type": "string"
               },
               "poolVolume": {
                  "description": "Two-way volume of all-time (in RUNE)",
                  "type": "string"
               },
               "poolVolume24hr": {
                  "description": "Two-way volume in 24hrs (in RUNE)",
                  "type": "string"
               },
               "price": {
                  "description": "Price of Asset (in RUNE).",
                  "type": "string"
               },
               "runeDepth": {
                  "description": "Total current Rune balance",
                  "type": "string"
               },
               "runeROI": {
                  "description": "RUNE return on investment",
                  "type": "string"
               },
               "runeStakedTotal": {
                  "description": "Total RUNE staked",
                  "type": "string"
               },
               "sellAssetCount": {
                  "description": "Number of ASSET-\u003eRUNE transactions",
                  "type": "string"
               },
               "sellFeeAverage": {
                  "description": "Average buy Asset fee size for ASSET-\u003eRUNE (in RUNE)",
                  "type": "string"
               },
               "sellFeesTotal": {
                  "description": "Total fees (in RUNE)",
                  "type": "string"
               },
               "sellSlipAverage": {
                  "description": "Average trade slip for ASSET-\u003eRUNE in %",
                  "type": "string"
               },
               "sellTxAverage": {
                  "description": "Average Asset sell transaction size (ASSET\u003eRUNE) (in RUNE)",
                  "type": "string"
               },
               "sellVolume": {
                  "description": "Total Asset sell volume (ASSET\u003eRUNE) (in RUNE).",
                  "type": "string"
               },
               "stakeTxCount": {
                  "description": "Number of stake transactions",
                  "type": "string"
               },
               "stakersCount": {
                  "description": "Number of unique stakers",
                  "type": "string"
               },
               "stakingTxCount": {
                  "description": "Number of stake \u0026 withdraw transactions",
                  "type": "string"
               },
               "status": {
                  "enum": [
                     "bootstrapped",
                     "enabled",
                     "disabled"
                  ],
                  "type": "string"
               },
               "swappersCount": {
                  "description": "Number of unique swappers interacting with pool",
                  "type": "string"
               },
               "swappingTxCount": {
                  "description": "Number of swapping transactions in the pool (buys and sells)",
                  "type": "string"
               },
               "withdrawTxCount": {
                  "description": "Number of withdraw transactions",
                  "type": "string"
               }
            },
            "type": "object"
         },
         "Stakers": {
            "description": "Staker address",
            "example": "tbnb1fj2lqj8dvr5pumfchc7ntlfqd2v6zdxqwjewf5",
            "type": "string"
         },
         "StakersAddressData": {
            "properties": {
               "poolsArray": {
                  "items": {
                     "$ref": "#/components/schemas/asset"
                  },
                  "type": "array"
               },
               "totalEarned": {
                  "description": "Total value of earnings (in RUNE) across all pools.",
                  "type": "string"
               },
               "totalROI": {
                  "description": "Average of all pool ROIs.",
                  "type": "string"
               },
               "totalStaked": {
                  "description": "Total staked (in RUNE) across all pools.",
                  "type": "string"
               }
            },
            "type": "object"
         },
         "StakersAssetData": {
            "properties": {
               "asset": {
                  "$ref": "#/components/schemas/asset"
               },
               "dateFirstStaked": {
                  "format": "int64",
                  "type": "integer"
               },
               "heightLastStaked": {
                  "format": "int64",
                  "type": "integer"
               },
               "stakeUnits": {
                  "description": "Represents ownership of a pool.",
                  "type": "string"
               }
            },
            "type": "object"
         },
         "StatsData": {
            "properties": {
               "dailyActiveUsers": {
                  "description": "Daily active users (unique addresses interacting)",
                  "type": "string"
               },
               "dailyTx": {
                  "description": "Daily transactions",
                  "type": "string"
               },
               "monthlyActiveUsers": {
                  "description": "Monthly active users",
                  "type": "string"
               },
               "monthlyTx": {
                  "description": "Monthly transactions",
                  "type": "string"
               },
               "poolCount": {
                  "description": "Number of active pools",
                  "type": "string"
               },
               "totalAssetBuys": {
                  "description": "Total buying transactions",
                  "type": "string"
               },
               "totalAssetSells": {
                  "description": "Total selling transactions",
                  "type": "string"
               },
               "totalDepth": {
                  "description": "Total RUNE balances",
                  "type": "string"
               },
               "totalEarned": {
                  "description": "Total earned (in RUNE Value).",
                  "type": "string"
               },
               "totalStakeTx": {
                  "description": "Total staking transactions",
                  "type": "string"
               },
               "totalStaked": {
                  "description": "Total staked (in RUNE Value).",
                  "type": "string"
               },
               "totalTx": {
                  "description": "Total transactions",
                  "type": "string"
               },
               "totalUsers": {
                  "description": "Total unique swappers \u0026 stakers",
                  "type": "string"
               },
               "totalVolume": {
                  "description": "Total (in RUNE Value) of all assets swapped since start.",
                  "type": "string"
               },
               "totalVolume24hr": {
                  "description": "Total (in RUNE Value) of all assets swapped in 24hrs",
                  "type": "string"
               },
               "totalWithdrawTx": {
                  "description": "Total withdrawing transactions",
                  "type": "string"
               }
            },
            "type": "object"
         },
         "ThorchainBooleanConstants": {
            "properties": {
               "StrictBondStakeRatio": {
                  "type": "boolean"
               }
            },
            "type": "object"
         },
         "ThorchainConstants": {
            "properties": {
               "bool_values": {
                  "$ref": "#/components/schemas/ThorchainBooleanConstants"
               },
               "int_64_values": {
                  "$ref": "#/components/schemas/ThorchainInt64Constants"
               },
               "string_values": {
                  "$ref": "#/components/schemas/ThorchainStringConstants"
               }
            },
            "type": "object"
         },
         "ThorchainEndpoint": {
            "properties": {
               "address": {
                  "type": "string"
               },
               "chain": {
                  "type": "string"
               },
               "pub_key": {
                  "type": "string"
               }
            },
            "type": "object"
         },
         "ThorchainEndpoints": {
            "properties": {
               "current": {
                  "items": {
                     "$ref": "#/components/schemas/ThorchainEndpoint"
                  },
                  "type": "array"
               }
            },
            "type": "object"
         },
         "ThorchainInt64Constants": {
            "properties": {
               "BadValidatorRate": {
                  "format": "int64",
                  "type": "integer"
               },
               "BlocksPerYear": {
                  "format": "int64",
                  "type": "integer"
               },
               "DesireValidatorSet": {
                  "format": "int64",
                  "type": "integer"
               },
               "DoubleSignMaxAge": {
                  "format": "int64",
                  "type": "integer"
               },
               "EmissionCurve": {
                  "format": "int64",
                  "type": "integer"
               },
               "FailKeySignSlashPoints": {
                  "format": "int64",
                  "type": "integer"
               },
               "FailKeygenSlashPoints": {
                  "format": "int64",
                  "type": "integer"
               },
               "FundMigrationInterval": {
                  "format": "int64",
                  "type": "integer"
               },
               "JailTimeKeygen": {
                  "format": "int64",
                  "type": "integer"
               },
               "JailTimeKeysign": {
                  "format": "int64",
                  "type": "integer"
               },
               "LackOfObservationPenalty": {
                  "format": "int64",
                  "type": "integer"
               },
               "MinimumBondInRune": {
                  "format": "int64",
                  "type": "integer"
               },
               "MinimumNodesForBFT": {
                  "format": "int64",
                  "type": "integer"
               },
               "MinimumNodesForYggdrasil": {
                  "format": "int64",
                  "type": "integer"
               },
               "NewPoolCycle": {
                  "format": "int64",
                  "type": "integer"
               },
               "ObserveSlashPoints": {
                  "format": "int64",
                  "type": "integer"
               },
               "OldValidatorRate": {
                  "format": "int64",
                  "type": "integer"
               },
               "RotatePerBlockHeight": {
                  "format": "int64",
                  "type": "integer"
               },
               "RotateRetryBlocks": {
                  "format": "int64",
                  "type": "integer"
               },
               "SigningTransactionPeriod": {
                  "format": "int64",
                  "type": "integer"
               },
               "StakeLockUpBlocks": {
                  "format": "int64",
                  "type": "integer"
               },
               "TransactionFee": {
                  "format": "int64",
                  "type": "integer"
               },
               "ValidatorRotateInNumBeforeFull": {
                  "format": "int64",
                  "type": "integer"
               },
               "ValidatorRotateNumAfterFull": {
                  "format": "int64",
                  "type": "integer"
               },
               "ValidatorRotateOutNumBeforeFull": {
                  "format": "int64",
                  "type": "integer"
               },
               "WhiteListGasAsset": {
                  "format": "int64",
                  "type": "integer"
               },
               "YggFundLimit": {
                  "format": "int64",
                  "type": "integer"
               }
            },
            "type": "object"
         },
         "ThorchainLastblock": {
            "properties": {
               "chain": {
                  "type": "string"
               },
               "lastobservedin": {
                  "format": "int64",
                  "type": "integer"
               },
               "lastsignedout": {
                  "format": "int64",
                  "type": "integer"
               },
               "thorchain": {
                  "format": "int64",
                  "type": "integer"
               }
            },
            "type": "object"
         },
         "ThorchainStringConstants": {
            "properties": {
               "DefaultPoolStatus": {
                  "type": "string"
               }
            },
            "type": "object"
         },
         "TotalVolChanges": {
            "properties": {
               "buyVolume": {
                  "type": "string"
               },
               "sellVolume": {
                  "type": "string"
               },
               "time": {
                  "format": "int64",
                  "type": "integer"
               },
               "totalVolume": {
                  "type": "string"
               }
            },
            "type": "object"
         },
         "TxDetails": {
            "properties": {
               "date": {
                  "format": "int64",
                  "type": "integer"
               },
               "events": {
                  "$ref": "#/components/schemas/event"
               },
               "gas": {
                  "$ref": "#/components/schemas/gas"
               },
               "height": {
                  "type": "string"
               },
               "in": {
                  "$ref": "#/components/schemas/tx"
               },
               "options": {
                  "$ref": "#/components/schemas/option"
               },
               "out": {
                  "items": {
                     "$ref": "#/components/schemas/tx"
                  },
                  "type": "array"
               },
               "pool": {
                  "$ref": "#/components/schemas/asset"
               },
               "status": {
                  "enum": [
                     "success",
                     "refund"
                  ],
                  "type": "string"
               },
               "type": {
                  "enum": [
                     "swap",
                     "stake",
                     "unstake",
                     "rewards",
                     "add",
                     "pool",
                     "gas",
                     "refund",
                     "doubleSwap"
                  ],
                  "type": "string"
               }
            }
         },
         "asset": {
            "type": "string"
         },
         "coin": {
            "properties": {
               "amount": {
                  "type": "string"
               },
               "asset": {
                  "$ref": "#/components/schemas/asset"
               }
            },
            "type": "object"
         },
         "coins": {
            "items": {
               "$ref": "#/components/schemas/coin"
            },
            "type": "array"
         },
         "event": {
            "properties": {
               "fee": {
                  "type": "string"
               },
               "slip": {
                  "type": "string"
               },
               "stakeUnits": {
                  "type": "string"
               }
            },
            "type": "object"
         },
         "gas": {
            "properties": {
               "amount": {
                  "type": "string"
               },
               "asset": {
                  "$ref": "#/components/schemas/asset"
               }
            },
            "type": "object"
         },
         "option": {
            "properties": {
               "asymmetry": {
                  "type": "string"
               },
               "priceTarget": {
                  "type": "string"
               },
               "withdrawBasisPoints": {
                  "type": "string"
               }
            },
            "type": "object"
         },
         "tx": {
            "properties": {
               "address": {
                  "type": "string"
               },
               "coins": {
                  "$ref": "#/components/schemas/coins"
               },
               "memo": {
                  "type": "string"
               },
               "txID": {
                  "type": "string"
               }
            },
            "type": "object"
         }
      }
   },
   "info": {
      "contact": {
         "email": "devs@thorchain.org"
      },
      "description": "The Midgard Public API queries THORChain and any chains linked via the Bifröst and prepares information about the network to be readily available for public users. The API parses transaction event data from THORChain and stores them in a time-series database to make time-dependent queries easy. Midgard does not hold critical information. To interact with BEPSwap and Asgardex, users should query THORChain directly.",
      "title": "Midgard Public API",
      "version": "1.0.0-oas3"
   },
   "openapi": "3.0.0",
   "paths": {
      "/v2/assets": {
         "get": {
            "description": "Detailed information about a specific asset. Returns enough information to display a unique asset in various user interfaces, including latest price.",
            "operationId": "GetAssetInfo",
            "parameters": [
               {
                  "description": "One or more comma separated unique asset (CHAIN.SYMBOL)",
                  "example": [
                     "BNB.TOMOB-1E1",
                     "BNB.TCAN-014"
                  ],
                  "in": "query",
                  "name": "asset",
                  "required": true,
                  "schema": {
                     "type": "string"
                  }
               }
            ],
            "responses": {
               "200": {
                  "$ref": "#/components/responses/AssetsDetailedResponse"
               },
               "400": {
                  "$ref": "#/components/responses/GeneralErrorResponse"
               }
            },
            "summary": "Get Asset Information"
         }
      },
      "/v2/doc": {
         "get": {
            "description": "Swagger/openapi 3.0 specification generated documents.",
            "operationId": "GetDocs",
            "responses": {
               "200": {
                  "description": "swagger/openapi 3.0 spec generated docs"
               }
            },
            "summary": "Get Documents",
            "tags": [
               "Documentation"
            ]
         }
      },
      "/v2/health": {
         "get": {
            "description": "Returns an object containing the health response of the API.",
            "operationId": "GetHealth",
            "responses": {
               "200": {
                  "$ref": "#/components/responses/HealthResponse"
               }
            },
            "summary": "Get Health"
         }
      },
      "/v2/history/total_volume": {
         "get": {
            "description": "Returns total volume changes of all pools in specified interval",
            "operationId": "GetTotalVolChanges",
            "parameters": [
               {
                  "description": "Interval of calculations",
                  "in": "query",
                  "name": "interval",
                  "required": true,
                  "schema": {
                     "enum": [
                        "5min",
                        "hour",
                        "day",
                        "week",
                        "month",
                        "year"
                     ],
                     "type": "string"
                  }
               },
               {
                  "description": "Start time of the query as unix timestamp",
                  "in": "query",
                  "name": "from",
                  "required": true,
                  "schema": {
                     "format": "int64",
                     "type": "integer"
                  }
               },
               {
                  "description": "End time of the query as unix timestamp",
                  "in": "query",
                  "name": "to",
                  "required": true,
                  "schema": {
                     "format": "int64",
                     "type": "integer"
                  }
               }
            ],
            "responses": {
               "200": {
                  "$ref": "#/components/responses/TotalVolChangesResponse"
               }
            },
            "summary": "Get Total Volume Changes"
         }
      },
      "/v2/network": {
         "get": {
            "description": "Returns an object containing Network data",
            "operationId": "GetNetworkData",
            "responses": {
               "200": {
                  "$ref": "#/components/responses/NetworkResponse"
               }
            },
            "summary": "Get Network Data"
         }
      },
      "/v2/nodes": {
         "get": {
            "description": "Returns an object containing Node public keys",
            "operationId": "GetNodes",
            "responses": {
               "200": {
                  "$ref": "#/components/responses/NodeKeyResponse"
               }
            },
            "summary": "Get Node public keys"
         }
      },
      "/v2/pools": {
         "get": {
            "description": "Returns an array containing all the assets supported on BEPSwap pools",
            "operationId": "GetPools",
            "responses": {
               "200": {
                  "$ref": "#/components/responses/PoolsResponse"
               },
               "400": {
                  "$ref": "#/components/responses/GeneralErrorResponse"
               }
            },
            "summary": "Get Asset Pools"
         }
      },
      "/v2/pools/detail": {
         "get": {
            "description": "Returns an object containing all the pool details for that asset.",
            "operationId": "GetPoolsDetails",
            "parameters": [
               {
                  "description": "Specifies the returning view",
                  "in": "query",
                  "name": "view",
                  "schema": {
                     "default": "full",
                     "enum": [
                        "balances",
                        "simple",
                        "full"
                     ],
                     "type": "string"
                  }
               },
               {
                  "description": "One or more comma separated unique asset (CHAIN.SYMBOL)",
                  "example": [
                     "BNB.TOMOB-1E1",
                     "BNB.TCAN-014"
                  ],
                  "in": "query",
                  "name": "asset",
                  "required": true,
                  "schema": {
                     "type": "string"
                  }
               }
            ],
            "responses": {
               "200": {
                  "$ref": "#/components/responses/PoolsDetailedResponse"
               }
            },
            "summary": "Get Pools Details"
         }
      },
      "/v2/stakers": {
         "get": {
            "description": "Returns an array containing the addresses for all stakers.",
            "operationId": "GetStakersData",
            "responses": {
               "200": {
                  "$ref": "#/components/responses/StakersResponse"
               }
            },
            "summary": "Get Stakers"
         }
      },
      "/v2/stakers/{address}": {
         "get": {
            "description": "Returns an array containing all the pools the staker is staking in.",
            "operationId": "GetStakersAddressData",
            "parameters": [
               {
                  "description": "Unique staker address",
                  "example": "bnb1jxfh2g85q3v0tdq56fnevx6xcxtcnhtsmcu64m",
                  "in": "path",
                  "name": "address",
                  "required": true,
                  "schema": {
                     "type": "string"
                  }
               }
            ],
            "responses": {
               "200": {
                  "$ref": "#/components/responses/StakersAddressDataResponse"
               }
            },
            "summary": "Get Staker Data"
         }
      },
      "/v2/stakers/{address}/pools": {
         "get": {
            "description": "Returns an object containing staking data for the specified staker and pool.",
            "operationId": "GetStakersAddressAndAssetData",
            "parameters": [
               {
                  "description": "Unique staker address",
                  "example": "bnb1jxfh2g85q3v0tdq56fnevx6xcxtcnhtsmcu64m",
                  "in": "path",
                  "name": "address",
                  "required": true,
                  "schema": {
                     "type": "string"
                  }
               },
               {
                  "description": "One or more comma separated unique asset (CHAIN.SYMBOL)",
                  "example": [
                     "BNB.TOMOB-1E1",
                     "BNB.TCAN-014"
                  ],
                  "in": "query",
                  "name": "asset",
                  "required": true,
                  "schema": {
                     "type": "string"
                  }
               }
            ],
            "responses": {
               "200": {
                  "$ref": "#/components/responses/StakersAssetDataResponse"
               }
            },
            "summary": "Get Staker Pool Data"
         }
      },
      "/v2/stats": {
         "get": {
            "description": "Returns an object containing global stats for all pools and all transactions.",
            "operationId": "GetStats",
            "responses": {
               "200": {
                  "$ref": "#/components/responses/StatsResponse"
               }
            },
            "summary": "Get Global Stats"
         }
      },
      "/v2/swagger.json": {
         "get": {
            "description": "Returns human and machine readable swagger/openapi specification.",
            "operationId": "GetSwagger",
            "responses": {
               "200": {
                  "description": "human and machine readable swagger/openapi specification"
               }
            },
            "summary": "Get Swagger",
            "tags": [
               "Specification"
            ]
         }
      },
      "/v2/thorchain/constants": {
         "get": {
            "description": "Returns a proxied endpoint for the constants endpoint from a local thornode",
            "operationId": "GetThorchainProxiedConstants",
            "responses": {
               "200": {
                  "$ref": "#/components/responses/ThorchainConstantsResponse"
               }
            },
            "summary": "Get the Proxied THORChain Constants"
         }
      },
      "/v2/thorchain/lastblock": {
         "get": {
            "description": "Returns a proxied endpoint for the lastblock endpoint from a local thornode",
            "operationId": "GetThorchainProxiedLastblock",
            "responses": {
               "200": {
                  "$ref": "#/components/responses/ThorchainLastblockResponse"
               }
            },
            "summary": "Get the Proxied THORChain Lastblock"
         }
      },
      "/v2/thorchain/pool_addresses": {
         "get": {
            "description": "Returns a proxied endpoint for the pool_addresses endpoint from a local thornode",
            "operationId": "GetThorchainProxiedEndpoints",
            "responses": {
               "200": {
                  "$ref": "#/components/responses/ThorchainEndpointsResponse"
               }
            },
            "summary": "Get the Proxied Pool Addresses"
         }
      },
      "/v2/txs": {
         "get": {
            "description": "Return an array containing the event details",
            "operationId": "GetTxDetails",
            "parameters": [
               {
                  "description": "Address of sender or recipient of any in/out tx in event",
                  "example": "tbnb1fj2lqj8dvr5pumfchc7ntlfqd2v6zdxqwjewf5",
                  "in": "query",
                  "name": "address",
                  "required": false,
                  "schema": {
                     "type": "string"
                  }
               },
               {
                  "description": "ID of any in/out tx in event",
                  "example": "2F624637DE179665BA3322B864DB9F30001FD37B4E0D22A0B6ECE6A5B078DAB4",
                  "in": "query",
                  "name": "txid",
                  "required": false,
                  "schema": {
                     "type": "string"
                  }
               },
               {
                  "description": "Any asset used in event (CHAIN.SYMBOL)",
                  "example": "BNB.TOMOB-1E1",
                  "in": "query",
                  "name": "asset",
                  "required": false,
                  "schema": {
                     "type": "string"
                  }
               },
               {
                  "description": "One or more comma separated unique types of event",
                  "example": [
                     "swap",
                     "stake",
                     "unstake",
                     "add",
                     "refund",
                     "doubleSwap"
                  ],
                  "in": "query",
                  "name": "type",
                  "required": false,
                  "schema": {
                     "type": "string"
                  }
               },
               {
                  "description": "pagination offset",
                  "in": "query",
                  "name": "offset",
                  "required": true,
                  "schema": {
                     "format": "int64",
                     "minimum": 0,
                     "type": "integer"
                  }
               },
               {
                  "description": "pagination limit",
                  "in": "query",
                  "name": "limit",
                  "required": true,
                  "schema": {
                     "format": "int64",
                     "maximum": 50,
                     "minimum": 0,
                     "type": "integer"
                  }
               }
            ],
            "responses": {
               "200": {
                  "$ref": "#/components/responses/TxsResponse"
               }
            },
            "summary": "Get details of a tx by address, asset or tx-id"
         }
      }
   },
   "servers": [
      {
         "url": "http://127.0.0.1:8080"
      },
      {
         "url": "https://127.0.0.1:8080"
      }
   ]
}`)
}
