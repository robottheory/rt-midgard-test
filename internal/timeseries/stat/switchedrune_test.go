package stat_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

// TODO(muninn): test/fix value from Mint
func TestSwitchedRuneStat(t *testing.T) {
	blocks := testdb.InitTestBlocks(t)

	blocks.NewBlock(t, "2020-01-01 12:00:00",
		testdb.Switch{
			FromAddress: "b2",
			ToAddress:   "thor2",
			Burn:        "42 BNB.RUNE-B1A",
		})

	body := testdb.CallJSON(t, "http://localhost:8080/v2/stats")

	var jsonResult oapigen.StatsResponse
	testdb.MustUnmarshal(t, body, &jsonResult)

	require.Equal(t, "42", jsonResult.SwitchedRune)
}

