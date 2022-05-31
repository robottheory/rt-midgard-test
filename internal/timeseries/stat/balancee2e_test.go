package stat_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/thorchain/midgard/config"
	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

type expectation struct {
	addr  string
	coins oapigen.Coins
}

func TestEmptyBalanceE2E(t *testing.T) {
	config.Global.EventRecorder.OnTransferEnabled = true
	_ = testdb.InitTestBlocks(t)

	checkExpected(t, []expectation{{"thorAddr1", oapigen.Coins{}}})
}

func TestZeroBalanceE2E(t *testing.T) {
	config.Global.EventRecorder.OnTransferEnabled = true

	blocks := testdb.InitTestBlocks(t)

	blocks.NewBlock(t, "2000-01-01 00:00:00",
		testdb.Transfer{
			FromAddr:    "thorAddr1",
			ToAddr:      "thorAddr2",
			AssetAmount: "1 THOR.RUNE",
		},
	)

	blocks.NewBlock(t, "2000-01-01 00:00:01",
		testdb.Transfer{
			FromAddr:    "thorAddr2",
			ToAddr:      "thorAddr1",
			AssetAmount: "1 THOR.RUNE",
		},
	)

	checkExpected(t, []expectation{
		{"thorAddr1", oapigen.Coins{{Amount: "0", Asset: "THOR.RUNE"}}},
		{"thorAddr2", oapigen.Coins{{Amount: "0", Asset: "THOR.RUNE"}}},
	})

}

func TestBalanceE2E(t *testing.T) {
	config.Global.EventRecorder.OnTransferEnabled = true

	blocks := testdb.InitTestBlocks(t)

	blocks.NewBlock(t, "2000-01-01 00:00:00",
		testdb.Transfer{
			FromAddr:    "thorAddr1",
			ToAddr:      "thorAddr2",
			AssetAmount: "1 THOR.RUNE",
		},
	)
	blocks.NewBlock(t, "2000-01-01 00:00:01",
		testdb.Transfer{
			FromAddr:    "thorAddr1",
			ToAddr:      "thorAddr2",
			AssetAmount: "20 THOR.RUNE",
		},
	)
	blocks.NewBlock(t, "2000-01-01 00:00:03",
		testdb.Transfer{
			FromAddr:    "thorAddr2",
			ToAddr:      "thorAddr1",
			AssetAmount: "3 THOR.RUNE",
		},
	)
	blocks.NewBlock(t, "2000-01-01 00:00:04",
		testdb.Transfer{
			FromAddr:    "thorAddr2",
			ToAddr:      "thorAddr1",
			AssetAmount: "300 BTC/BTC",
		},
	)

	checkExpected(t, []expectation{
		{"thorAddr1", oapigen.Coins{{Amount: "300", Asset: "BTC/BTC"}, {Amount: "-18", Asset: "THOR.RUNE"}}},
		{"thorAddr2", oapigen.Coins{{Amount: "-300", Asset: "BTC/BTC"}, {Amount: "18", Asset: "THOR.RUNE"}}},
	})

}

func checkExpected(t *testing.T, expectations []expectation) {
	balanceQueryUrl := "http://localhost:8080/v2/balance/%v"

	for _, expectedBalance := range expectations {
		actualJson := testdb.CallJSON(t, fmt.Sprintf(balanceQueryUrl, expectedBalance.addr))
		var actualBalance oapigen.BalanceResponse
		testdb.MustUnmarshal(t, actualJson, &actualBalance)
		require.Equal(t, expectedBalance.coins, actualBalance.Coins)
	}
}
