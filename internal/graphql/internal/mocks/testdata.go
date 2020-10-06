package mocks

import (
	"time"

	"gitlab.com/thorchain/midgard/chain/notinchain"
	"gitlab.com/thorchain/midgard/internal/graphql/model"
	"gitlab.com/thorchain/midgard/internal/timeseries/stat"
)

var (
	first    time.Time
	last     time.Time
	tCreated time.Time
	err      error
	TestData Data
)

func init() {
	first, err = time.Parse(time.RFC3339, "2020-09-21T21:05:12-09:00")
	last, err = time.Parse(time.RFC3339, "2020-09-26T22:05:12-09:00")
	tCreated, _ = time.Parse(time.RFC3339, "2020-09-21T22:05:12-09:00")

	TestData = Data{
		LastBlockHeight:    62419,
		LastBlockTimestamp: "2020-08-26 02:00:07.016810585 +0000 UTC",
		LastBlockHash:      []byte{121, 98, 160, 236, 67, 162, 92, 202, 50, 61, 27, 200, 201, 110, 174, 222, 125, 119, 12, 163, 249, 137, 247, 44, 218, 27, 206, 238, 143, 167, 147, 200},
		NodesSecpAndEdData: NodesSecpAndEdData{
			Secp: map[string]string{"thorpub1addwnpepqg3g2e933fttl9nsyavray8cs9d3jhvgwun9f4hrgj8rcqsx5c0hsrlc0dk": "thor1766mazrxs5asuscepa227r6ekr657234f8p7nf"},
			Ed:   map[string]string{"thorpub1addwnpepqg3g2e933fttl9nsyavray8cs9d3jhvgwun9f4hrgj8rcqsx5c0hsrlc0dk": "thor1766mazrxs5asuscepa227r6ekr657234f8p7nf"},
		},
		Pools: []Pool{
			Pool{
				Asset:  "TEST.COIN",
				Status: "Enabled",
				Ae8pp:  3949032195733,
				Re8pp:  135631226606311,
				Price:  34.34543449730872,

				StakeTxCount:         1236,
				StakeAssetE8Total:    5002997091788,
				StakeRuneE8Total:     341573223004012,
				StakeStakeUnitsTotal: 133851876377593,
				StakeFirst:           first,
				StakeLast:            last,

				UnstakeTxCount:          274,
				UnstakeAssetE8Total:     190,
				UnstakeRuneE8Total:      220000082,
				UnstakeStakeUnitsTotal:  67286385293693,
				UnstakeBasisPointsTotal: 2150910,

				PoolDepths: []stat.PoolDepth{},

				SwapsFromRuneBucket: []stat.PoolSwaps{
					stat.PoolSwaps{
						First:               first,
						Last:                last,
						TxCount:             94,
						AssetE8Total:        0,
						RuneE8Total:         14286622385430,
						LiqFeeE8Total:       369368251,
						LiqFeeInRuneE8Total: 20500035455,
						TradeSlipBPTotal:    1741,
					},
					stat.PoolSwaps{
						First:               first,
						Last:                last,
						TxCount:             112,
						AssetE8Total:        0,
						RuneE8Total:         23245220000000,
						LiqFeeE8Total:       665305942,
						LiqFeeInRuneE8Total: 41429706330,
						TradeSlipBPTotal:    2696,
					},
				},
				SwapsToRuneBucket: []stat.PoolSwaps{
					stat.PoolSwaps{
						First:               first,
						Last:                last,
						TxCount:             52,
						AssetE8Total:        131277554216,
						RuneE8Total:         0,
						LiqFeeE8Total:       10096073016,
						LiqFeeInRuneE8Total: 10096073016,
						TradeSlipBPTotal:    893,
					},
					stat.PoolSwaps{
						First:               first,
						Last:                last,
						TxCount:             78,
						AssetE8Total:        160846334530,
						RuneE8Total:         0,
						LiqFeeE8Total:       11983916834,
						LiqFeeInRuneE8Total: 11983916834,
						TradeSlipBPTotal:    1144,
					},
				},
				StakeHistory: []stat.PoolStakes{
					stat.PoolStakes{
						TxCount:         38,
						AssetE8Total:    92761734194,
						RuneE8Total:     6276432493131,
						StakeUnitsTotal: 2310888431619,
						First:           first,
						Last:            last,
					},
				},

				NodeAccounts: []*notinchain.NodeAccount{
					{
						NodeAddr:         "1234",
						Status:           "standby",
						Bond:             2000,
						PublicKeys:       notinchain.PublicKeys{Secp256k1: "1212", Ed25519: "34434"},
						RequestedToLeave: false,
						ForcedToLeave:    false,
						LeaveHeight:      0,
						IpAddress:        "1.2.3.4",
						Version:          "0.15.0",
						SlashPoints:      85,
						Jail: notinchain.JailInfo{
							NodeAddr:      "121212",
							ReleaseHeight: 23232,
							Reason:        "reason",
						},
						CurrentAward: 23232,
					},
					{
						NodeAddr:         "4321",
						Status:           "active",
						Bond:             2000,
						PublicKeys:       notinchain.PublicKeys{Secp256k1: "1212", Ed25519: "34434"},
						RequestedToLeave: false,
						ForcedToLeave:    false,
						LeaveHeight:      0,
						IpAddress:        "1.2.3.4",
						Version:          "0.15.0",
						SlashPoints:      85,
						Jail: notinchain.JailInfo{
							NodeAddr:      "121212",
							ReleaseHeight: 23232,
							Reason:        "reason",
						},
						CurrentAward: 23232,
					},
				},

				Expected: ExpectedResponse{
					DepthHistory: model.PoolDepthHistory{
						Meta: &model.PoolDepthHistoryBucket{
							First:      0,
							Last:       0,
							RuneFirst:  0,
							RuneLast:   0,
							AssetFirst: 0,
							AssetLast:  0,
							PriceFirst: 0,
							PriceLast:  0,
						},
					},
					PriceHistory: model.PoolPriceHistory{
						Meta: &model.PoolPriceHistoryBucket{
							First:      0,
							Last:       0,
							PriceFirst: 0,
							PriceLast:  0,
						},
					},
					Stakers: []model.Staker{
						{
							Address: "TEST.COIN",
						},
					},
					Nodes: []model.Node{
						{
							Address:          "1234",
							Status:           "standby",
							Bond:             2000,
							PublicKeys:       &model.PublicKeys{Secp256k1: "1212", Ed25519: "34434"},
							RequestedToLeave: false,
							ForcedToLeave:    false,
							LeaveHeight:      0,
							IPAddress:        "1.2.3.4",
							Version:          "0.15.0",
							SlashPoints:      85,
							Jail: &model.JailInfo{
								NodeAddr:      "121212",
								ReleaseHeight: 23232,
								Reason:        "reason",
							},
							CurrentAward: 23232,
						},
					},
					Assets: []model.Asset{
						model.Asset{
							Asset:   "TEST.COIN",
							Created: "2020-09-21 21:05:12 -0900 -0900",
							Price:   34.34543449730872,
						},
					},
					Stats: model.Stats{
						DailyActiveUsers:   1,
						DailyTx:            1,
						MonthlyActiveUsers: 1,
						MonthlyTx:          1,
						TotalAssetBuys:     0,
						TotalAssetSells:    1,
						TotalDepth:         135631226606311,
						TotalStakeTx:       107,
						TotalStaked:        2658849927746,
						TotalTx:            108,
						TotalUsers:         1,
						TotalVolume:        100000000,
						TotalWithdrawTx:    100,
					},
					Pool: model.Pool{
						Asset:  "TEST.COIN",
						Status: "Enabled",
						Price:  34.34543449730872,
						Units:  66565491083900,
						Depth: &model.PoolDepth{
							AssetDepth: 3949032195733,
							RuneDepth:  135631226606311,
							PoolDepth:  271262453212622,
						},
						Stakes: &model.PoolStakes{
							AssetStaked: 5002997091788,
							RuneStaked:  341573003003930,
							PoolStaked:  513403111903635,
						},
						Roi: &model.Roi{
							AssetRoi: -0.2106667016926757,
							RuneRoi:  -0.6029217022027046,
						},
					},

					SwapHistory: model.PoolSwapHistory{
						Meta: &model.PoolSwapHistoryBucket{
							First: 1600683392,
							Last:  1600683392,
							ToRune: &model.SwapStats{
								Count:        130,
								FeesInRune:   11983916834,
								VolumeInRune: 0,
							},
							ToAsset: &model.SwapStats{
								Count:        206,
								FeesInRune:   41429706330,
								VolumeInRune: 23245220000000,
							},
							Combined: &model.SwapStats{
								Count:        336,
								FeesInRune:   84009731635,
								VolumeInRune: 37531842385430,
							},
						},
						Intervals: []*model.PoolSwapHistoryBucket{
							&model.PoolSwapHistoryBucket{
								First: 1600683392,
								Last:  1600683392,
								ToRune: &model.SwapStats{
									Count:        52,
									FeesInRune:   10096073016,
									VolumeInRune: 0,
								},
								ToAsset: &model.SwapStats{
									Count:        94,
									FeesInRune:   20500035455,
									VolumeInRune: 14286622385430,
								},
								Combined: &model.SwapStats{
									Count:        146,
									FeesInRune:   30596108471,
									VolumeInRune: 14286622385430,
								},
							},
							&model.PoolSwapHistoryBucket{
								First: 1600683740,
								Last:  1600683740,
								ToRune: &model.SwapStats{
									Count:        78,
									FeesInRune:   11983916834,
									VolumeInRune: 0,
								},
								ToAsset: &model.SwapStats{
									Count:        112,
									FeesInRune:   41429706330,
									VolumeInRune: 23245220000000,
								},
								Combined: &model.SwapStats{
									Count:        190,
									FeesInRune:   53413623164,
									VolumeInRune: 23245220000000,
								},
							},
						},
					},
					StakeHistory: model.PoolStakeHistory{
						Intervals: []*model.PoolStakeHistoryBucket{
							&model.PoolStakeHistoryBucket{
								First:         first.Unix(),
								Last:          last.Unix(),
								Count:         38,
								VolumeInRune:  6276432493131,
								VolumeInAsset: 92761734194,
								Units:         2310888431619,
							},
						},
					},
				},
			},
		},
		Timestamp: "2020-09-08 20:01:39.74967453 +0900 JST",
	}
}
