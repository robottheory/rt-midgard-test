package stat_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

func TestTsSwapsHistoryE2E(t *testing.T) {
	blocks := testdb.InitTestBlocks(t)
	timeseries.SetUsdPoolWhitelist([]string{"BNB.ASSET1"})

	blocks.NewBlock(t, "2022-07-01 00:20:00",
		testdb.PoolActivate{Pool: "BNB.ASSET1"})

	blocks.NewBlock(t, "2022-07-02 00:30:00",
		testdb.AddLiquidity{
			Pool:                   "BNB.ASSET1",
			AssetAmount:            1000,
			RuneAmount:             600,
			AssetAddress:           "1111",
			LiquidityProviderUnits: 10,
		},
	)

	// memo match 111
	blocks.NewBlock(t, "2022-07-01 00:10:00",
		testdb.Swap{
			TxID:               "2134",
			Pool:               "BNB.ASSET1",
			EmitAsset:          "1 THOR.RUNE",
			Coin:               "1 BNB.BTCB-1DE",
			FromAddress:        "thor14nu4veuhttfs0rhmg2w06769hae22c8hxx2k5d",
			ToAddress:          "thor1g98cy3n9mmjrpn0sxmn63lztelera37n8n67c0",
			LiquidityFee:       1,
			LiquidityFeeInRune: 1,
			Slip:               1,
			Memo:               "SWAP:ETH.ETH:0x8efb3e170880dff97e4ba06c30edc45b8289eca6:146552111",
			PriceTarget:        1,
		},
	)
	// memo match 111
	blocks.NewBlock(t, "2022-07-02 00:10:00",
		testdb.Swap{
			TxID:               "2345",
			Pool:               "BNB.ASSET1",
			EmitAsset:          "1 THOR.RUNE",
			Coin:               "1 BNB.BTCB-1DE",
			FromAddress:        "thor14nu4veuhttfs0rhmg2w06769hae22c8hxx2k5d",
			ToAddress:          "thor1g98cy3n9mmjrpn0sxmn63lztelera37n8n67c0",
			LiquidityFee:       1,
			LiquidityFeeInRune: 1,
			Slip:               1,
			Memo:               "SWAP:ETH.ETH:111:thor160yye65pf9rzwrgqmtgav69n6zlsyfpgm9a7xk",
			PriceTarget:        1,
		},
	)
	// memo match 0Cec49fd2
	blocks.NewBlock(t, "2022-07-03 00:10:00",
		testdb.Swap{
			TxID:               "3456",
			Pool:               "BNB.ASSET1",
			EmitAsset:          "1 THOR.RUNE",
			Coin:               "1 BNB.BTCB-1DE",
			FromAddress:        "thor14nu4veuhttfs0rhmg2w06769hae22c8hxx2k5d",
			ToAddress:          "thor1g98cy3n9mmjrpn0sxmn63lztelera37n8n67c0",
			LiquidityFee:       1,
			LiquidityFeeInRune: 1,
			Slip:               1,
			Memo:               ":0Cec49fd2:",
			PriceTarget:        1,
		},
	)
	// memo match 0Cec49fd2
	blocks.NewBlock(t, "2022-07-04 00:10:00",
		testdb.Swap{
			TxID:               "4567",
			Pool:               "BNB.ASSET1",
			EmitAsset:          "1 THOR.RUNE",
			Coin:               "1 BNB.BTCB-1DE",
			FromAddress:        "thor14nu4veuhttfs0rhmg2w06769hae22c8hxx2k5d",
			ToAddress:          "thor1g98cy3n9mmjrpn0sxmn63lztelera37n8n67c0",
			LiquidityFee:       1,
			LiquidityFeeInRune: 1,
			Slip:               1,
			Memo:               ":7Fb5771d3:",
			PriceTarget:        1,
		},
	)
	// memo match 0Cec49fd2
	blocks.NewBlock(t, "2022-08-01 00:10:00",
		testdb.Swap{
			TxID:               "5678",
			Pool:               "BNB.ASSET1",
			EmitAsset:          "1 THOR.RUNE",
			Coin:               "1 BNB.BTCB-1DE",
			FromAddress:        "thor14nu4veuhttfs0rhmg2w06769hae22c8hxx2k5d",
			ToAddress:          "thor1g98cy3n9mmjrpn0sxmn63lztelera37n8n67c0",
			LiquidityFee:       1,
			LiquidityFeeInRune: 1,
			Slip:               1,
			Memo:               ":d71DBF121:",
			PriceTarget:        1,
		},
	)
	// memo match 0Cec49fd2
	blocks.NewBlock(t, "2022-08-02 00:10:00",
		testdb.Swap{
			TxID:               "6789",
			Pool:               "BNB.ASSET1",
			EmitAsset:          "1 THOR.RUNE",
			Coin:               "1 BNB.BTCB-1DE",
			FromAddress:        "thor14nu4veuhttfs0rhmg2w06769hae22c8hxx2k5d",
			ToAddress:          "thor1g98cy3n9mmjrpn0sxmn63lztelera37n8n67c0",
			LiquidityFee:       1,
			LiquidityFeeInRune: 1,
			Slip:               1,
			Memo:               ":E88b1EF10:",
			PriceTarget:        1,
		},
	)
	// memo match 0Cec49fd2
	blocks.NewBlock(t, "2022-08-23 00:10:00",
		testdb.Swap{
			TxID:               "7890",
			Pool:               "BNB.ASSET1",
			EmitAsset:          "1 THOR.RUNE",
			Coin:               "1 BNB.BTCB-1DE",
			FromAddress:        "thor14nu4veuhttfs0rhmg2w06769hae22c8hxx2k5d",
			ToAddress:          "thor1g98cy3n9mmjrpn0sxmn63lztelera37n8n67c0",
			LiquidityFee:       1,
			LiquidityFeeInRune: 1,
			Slip:               1,
			Memo:               ":d15bD1Dfb:",
			PriceTarget:        1,
		},
	)
	// memo match 0Cec49fd2
	blocks.NewBlock(t, "2022-08-24 00:10:00",
		testdb.Swap{
			TxID:               "4321",
			Pool:               "BNB.ASSET1",
			EmitAsset:          "1 THOR.RUNE",
			Coin:               "1 BNB.BTCB-1DE",
			FromAddress:        "thor14nu4veuhttfs0rhmg2w06769hae22c8hxx2k5d",
			ToAddress:          "thor1g98cy3n9mmjrpn0sxmn63lztelera37n8n67c0",
			LiquidityFee:       1,
			LiquidityFeeInRune: 1,
			Slip:               1,
			Memo:               ":2118375DA:",
			PriceTarget:        1,
		},
	)
	// memo match 0Cec49fd2
	blocks.NewBlock(t, "2022-08-25 00:10:00",
		testdb.Swap{
			TxID:               "5432",
			Pool:               "BNB.ASSET1",
			EmitAsset:          "1 THOR.RUNE",
			Coin:               "1 BNB.BTCB-1DE",
			FromAddress:        "thor14nu4veuhttfs0rhmg2w06769hae22c8hxx2k5d",
			ToAddress:          "thor1g98cy3n9mmjrpn0sxmn63lztelera37n8n67c0",
			LiquidityFee:       1,
			LiquidityFeeInRune: 1,
			Slip:               1,
			Memo:               ":FC4414199:",
			PriceTarget:        1,
		},
	)
	// memo match 111
	blocks.NewBlock(t, "2022-08-26 00:10:00",
		testdb.Swap{
			TxID:               "6543",
			Pool:               "BNB.ASSET1",
			EmitAsset:          "1 THOR.RUNE",
			Coin:               "1 BNB.BTCB-1DE",
			FromAddress:        "thor14nu4veuhttfs0rhmg2w06769hae22c8hxx2k5d",
			ToAddress:          "thor1g98cy3n9mmjrpn0sxmn63lztelera37n8n67c0",
			LiquidityFee:       1,
			LiquidityFeeInRune: 1,
			Slip:               1,
			Memo:               "SWAP:ETH.ETH:111:thor160yye65pf9rzwrgqmtgav69n6zlsyfpgm9a7xk",
			PriceTarget:        1,
		},
	)
	// memo match 111
	blocks.NewBlock(t, "2022-08-27 00:10:00",
		testdb.Swap{
			TxID:               "7654",
			Pool:               "BNB.ASSET1",
			EmitAsset:          "1 THOR.RUNE",
			Coin:               "1 BNB.BTCB-1DE",
			FromAddress:        "thor14nu4veuhttfs0rhmg2w06769hae22c8hxx2k5d",
			ToAddress:          "thor1g98cy3n9mmjrpn0sxmn63lztelera37n8n67c0",
			LiquidityFee:       1,
			LiquidityFeeInRune: 1,
			Slip:               1,
			Memo:               "SWAP:ETH.ETH:111",
			PriceTarget:        1,
		},
	)
	// memo doesn't match to anything
	blocks.NewBlock(t, "2022-08-28 00:10:00",
		testdb.Swap{
			TxID:               "8765",
			Pool:               "BNB.ASSET1",
			EmitAsset:          "1 THOR.RUNE",
			Coin:               "1 BNB.BTCB-1DE",
			FromAddress:        "thor14nu4veuhttfs0rhmg2w06769hae22c8hxx2k5d",
			ToAddress:          "thor1g98cy3n9mmjrpn0sxmn63lztelera37n8n67c0",
			LiquidityFee:       1,
			LiquidityFeeInRune: 1,
			Slip:               1,
			Memo:               "SWAP:ETH.ETH:1234",
			PriceTarget:        1,
		},
	)

	from := db.StrToSec("2022-06-28 00:00:00")
	to := db.StrToSec("2022-08-29 00:00:00")
	{
		// 2022-06-28 to 2022-08-29
		body := testdb.CallJSON(t,
			fmt.Sprintf("http://localhost:8080/v2/history/ts-swaps?from=%d&to=%d&pool=BNB.ASSET1", from, to))

		var jsonApiResult oapigen.SwapHistoryResponse
		testdb.MustUnmarshal(t, body, &jsonApiResult)

		require.Equal(t, "18", jsonApiResult.Meta.TotalVolume)
		require.Equal(t, "27", jsonApiResult.Meta.TotalVolumeUsd)
	}

	from = db.StrToSec("2022-08-01 00:00:00")
	to = db.StrToSec("2022-08-23 23:59:59")
	{
		// 2022-08-01 to 2022-08-23
		body := testdb.CallJSON(t,
			fmt.Sprintf("http://localhost:8080/v2/history/ts-swaps?from=%d&to=%d", from, to))

		var jsonApiResult oapigen.SwapHistoryResponse
		testdb.MustUnmarshal(t, body, &jsonApiResult)

		require.Equal(t, "6", jsonApiResult.Meta.TotalVolume)
		require.Equal(t, "9", jsonApiResult.Meta.TotalVolumeUsd)
	}

	from = db.StrToSec("2022-08-23 00:00:00")
	to = db.StrToSec("2022-08-29 00:00:00")
	{
		// 2022-08-23 to 2022-08-29
		body := testdb.CallJSON(t,
			fmt.Sprintf("http://localhost:8080/v2/history/ts-swaps?from=%d&to=%d", from, to))

		var jsonApiResult oapigen.SwapHistoryResponse
		testdb.MustUnmarshal(t, body, &jsonApiResult)

		require.Equal(t, "6", jsonApiResult.Meta.TotalVolume)
		require.Equal(t, "9", jsonApiResult.Meta.TotalVolumeUsd)
	}

	from = db.StrToSec("2022-06-28 00:00:00")
	{
		// 2022-06-28
		body := testdb.CallJSON(t,
			fmt.Sprintf("http://localhost:8080/v2/history/ts-swaps?from=%d&pool=BNB.ASSET1", from))

		var jsonApiResult oapigen.SwapHistoryResponse
		testdb.MustUnmarshal(t, body, &jsonApiResult)

		require.Equal(t, "18", jsonApiResult.Meta.TotalVolume)
		require.Equal(t, "27", jsonApiResult.Meta.TotalVolumeUsd)
	}
}
