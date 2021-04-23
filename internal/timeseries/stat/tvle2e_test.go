package stat_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/thorchain/midgard/internal/api"
	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/internal/timeseries/stat"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

func TestTVLHistoryE2E(t *testing.T) {
	testdb.InitTest(t)
	testdb.DeclarePools("ABC.ABC", "ABC.XYZ", "ABC.USD1", "ABC.USD2")
	stat.SetUsdPoolsForTests([]string{"ABC.USD1", "ABC.USD2"})

	// This will be skipped because we query 01-09 to 01-14
	testdb.InsertBlockPoolDepth(t, "ABC.ABC", 1000, 1, "2020-01-14 12:00:00")

	// This will be the initial value
	// Not all pools existed before the start time
	testdb.InsertBlockPoolDepth(t, "ABC.XYZ", 10, 100, "2020-01-05 12:00:00")
	testdb.InsertBlockPoolDepth(t, "ABC.USD1", 100, 10, "2020-01-05 12:00:00")

	testdb.InsertBlockPoolDepth(t, "ABC.ABC", 10, 20, "2020-01-10 12:00:05")
	testdb.InsertBlockPoolDepth(t, "ABC.ABC", 20, 30, "2020-01-10 14:00:00")

	testdb.InsertBlockPoolDepth(t, "ABC.XYZ", 20, 150, "2020-01-11 14:00:00")

	testdb.InsertBlockPoolDepth(t, "ABC.USD1", 0, 0, "2020-01-13 07:00:00")
	testdb.InsertBlockPoolDepth(t, "ABC.USD2", 20, 10, "2020-01-13 08:00:00")
	testdb.InsertBlockPoolDepth(t, "ABC.ABC", 2, 5, "2020-01-13 09:00:00")
	testdb.InsertBlockPoolDepth(t, "ABC.ABC", 6, 18, "2020-01-13 10:00:00")

	from := testdb.StrToSec("2020-01-09 00:00:00")
	to := testdb.StrToSec("2020-01-14 00:00:00")

	body := testdb.CallJSON(t, fmt.Sprintf(
		"http://localhost:8080/v2/debug/tvl?interval=day&from=%d&to=%d", from, to))

	// fmt.Println(string(body))

	var jsonResult oapigen.TVLHistoryResponse
	testdb.MustUnmarshal(t, body, &jsonResult)

	require.Equal(t, oapigen.TVLHistoryItem{
		StartTime:        epochStr("2020-01-09 00:00:00"),
		EndTime:          epochStr("2020-01-14 00:00:00"),
		TotalRuneDepth:   "178",
		TotalBonds:       "0",
		TotalValueLocked: "356",
		RunePriceUSD:     "2",
	}, jsonResult.Meta)
	require.Equal(t, 5, len(jsonResult.Intervals))
	require.Equal(t, epochStr("2020-01-09 00:00:00"), jsonResult.Intervals[0].StartTime)
	require.Equal(t, epochStr("2020-01-10 00:00:00"), jsonResult.Intervals[0].EndTime)
	require.Equal(t, epochStr("2020-01-14 00:00:00"), jsonResult.Intervals[4].EndTime)

	require.Equal(t, "110", jsonResult.Intervals[0].TotalRuneDepth) // from initial values
	require.Equal(t, "140", jsonResult.Intervals[1].TotalRuneDepth)
	require.Equal(t, "190", jsonResult.Intervals[2].TotalRuneDepth)
	require.Equal(t, "190", jsonResult.Intervals[3].TotalRuneDepth) // gapfill
	require.Equal(t, "10", jsonResult.Intervals[3].RunePriceUSD)    // initial USD price
	require.Equal(t, "178", jsonResult.Intervals[4].TotalRuneDepth)
}

func TestTVLHistoryBondsE2E(t *testing.T) {
	if !api.ShowBonds {
		return
	}
	testdb.InitTest(t)

	// This will be skipped because we query 01-09 to 01-14
	testdb.InsertBondEventForTotal(t, "bond_paid", 1000, "2020-01-14 12:00:00")

	// This will be the initial value
	testdb.InsertBondEventForTotal(t, "bond_paid", 100, "2020-01-05 12:00:00")

	testdb.InsertBondEventForTotal(t, "bond_cost", 20, "2020-01-10 12:00:00")
	testdb.InsertBondEventForTotal(t, "bond_reward", 10, "2020-01-10 14:00:00")

	testdb.InsertBondEventForTotal(t, "bond_returned", 50, "2020-01-12 09:00:00")
	testdb.InsertBondEventForTotal(t, "bond_paid", 100, "2020-01-12 16:00:00")

	from := testdb.StrToSec("2020-01-09 00:00:00")
	to := testdb.StrToSec("2020-01-13 00:00:00")

	body := testdb.CallJSON(t, fmt.Sprintf(
		"http://localhost:8080/v2/debug/tvl?interval=day&from=%d&to=%d", from, to))

	// fmt.Println(string(body))

	var jsonResult oapigen.TVLHistoryResponse
	testdb.MustUnmarshal(t, body, &jsonResult)

	require.Equal(t, oapigen.TVLHistoryItem{
		StartTime:        epochStr("2020-01-09 00:00:00"),
		EndTime:          epochStr("2020-01-13 00:00:00"),
		TotalRuneDepth:   "0",
		TotalBonds:       "140",
		TotalValueLocked: "140",
		RunePriceUSD:     "NaN",
	}, jsonResult.Meta)
	require.Equal(t, 4, len(jsonResult.Intervals))
	require.Equal(t, epochStr("2020-01-09 00:00:00"), jsonResult.Intervals[0].StartTime)
	require.Equal(t, epochStr("2020-01-10 00:00:00"), jsonResult.Intervals[0].EndTime)
	require.Equal(t, epochStr("2020-01-13 00:00:00"), jsonResult.Intervals[3].EndTime)

	require.Equal(t, "100", jsonResult.Intervals[0].TotalBonds) // from initial values
	require.Equal(t, "90", jsonResult.Intervals[1].TotalBonds)
	require.Equal(t, "90", jsonResult.Intervals[2].TotalBonds) // gapfill
	require.Equal(t, "140", jsonResult.Intervals[3].TotalBonds)
}
