package stat_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

func TestSwitchedRuneStat(t *testing.T) {
	testdb.InitTest(t)

	testdb.InsertSwitchEvent(t, testdb.FakeSwitch{
		BurnE8:         42,
		BlockTimestamp: "2020-01-01 12:00:00"})

	body := testdb.CallJSON(t, "http://localhost:8080/v2/stats")

	var jsonResult oapigen.StatsResponse
	testdb.MustUnmarshal(t, body, &jsonResult)

	require.Equal(t, "42", jsonResult.SwitchedRune)
}
