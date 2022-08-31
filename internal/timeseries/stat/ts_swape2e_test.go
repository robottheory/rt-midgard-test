package stat_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

func TestTsSwapsHistoryE2E(t *testing.T) {
	blocks := testdb.InitTestBlocks(t)
	// if 111 is true just written '1' on comment and if special string is true written 'A'
	//1
	blocks.NewBlock(t, "2022-07-01 00:10:00",
		testdb.PoolActivate{Pool: "BNB.ASSET1"},
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
	//1
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
	//0,A
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
	//0,A
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
	//0,A
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
	//0,A
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
	//0,A
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
	//0,A
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
	//0,A
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
	//1
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
	//1
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
	//
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

	{
		// 2022-06-28 to 2022-08-29
		body := testdb.CallJSON(t, "http://localhost:8080/v2/history/ts-swaps?from=1656374400&to=1661731200")

		var jsonApiResult oapigen.SwapHistoryResponse
		testdb.MustUnmarshal(t, body, &jsonApiResult)

		require.Equal(t, "22", jsonApiResult.Meta.TotalVolume)
	}

	{
		// 2022-08-01 to 2022-08-23
		body := testdb.CallJSON(t, "http://localhost:8080/v2/history/ts-swaps?from=1659312000&to=1661212800")

		var jsonApiResult oapigen.SwapHistoryResponse
		testdb.MustUnmarshal(t, body, &jsonApiResult)

		require.Equal(t, "4", jsonApiResult.Meta.TotalVolume)
	}

	{
		// 2022-08-23 to 2022-08-29
		body := testdb.CallJSON(t, "http://localhost:8080/v2/history/ts-swaps?from=1661212800&to=1661731200")

		var jsonApiResult oapigen.SwapHistoryResponse
		testdb.MustUnmarshal(t, body, &jsonApiResult)

		require.Equal(t, "10", jsonApiResult.Meta.TotalVolume)
	}
}
