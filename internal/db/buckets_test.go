package db_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

func bucketPass(t *testing.T, getParams string) (ret []string) {
	testdb.SetupTestDB(t)
	testdb.MustExec(t, "DELETE FROM swap_events")

	body := testdb.CallV1(t, "http://localhost:8080/v2/history/swaps?"+getParams)

	var swapHistory oapigen.SwapHistoryResponse
	testdb.MustUnmarshal(t, body, &swapHistory)

	assert.NotEmpty(t, swapHistory.Intervals)
	assert.Equal(t, swapHistory.Meta.StartTime, swapHistory.Intervals[0].StartTime)
	assert.Equal(t,
		swapHistory.Meta.EndTime,
		swapHistory.Intervals[len(swapHistory.Intervals)-1].EndTime)

	for _, interval := range swapHistory.Intervals {
		i, err := strconv.Atoi(interval.StartTime)
		assert.Nil(t, err)
		ret = append(ret, testdb.SecToString(db.Second(i)))
	}
	return
}

func bucketFail(t *testing.T, getParams string, msg ...string) {
	testdb.SetupTestDB(t)
	testdb.MustExec(t, "DELETE FROM swap_events")
	testdb.CallFail(t, "http://localhost:8080/v2/history/swaps?"+getParams, msg...)
}

func TestYearExact(t *testing.T) {
	t0 := testdb.StrToSec("2015-01-01 00:00:00")
	t1 := testdb.StrToSec("2018-01-01 00:00:00")
	starts := bucketPass(t, fmt.Sprintf("interval=year&from=%d&to=%d", t0, t1))
	assert.Equal(t, []string{
		"2015-01-01 00:00:00",
		"2016-01-01 00:00:00",
		"2017-01-01 00:00:00",
	}, starts)
}

func TestYearInexact(t *testing.T) {
	t0 := testdb.StrToSec("2015-06-01 00:00:00")
	t1 := testdb.StrToSec("2018-06-01 00:00:00")
	starts := bucketPass(t, fmt.Sprintf("interval=year&from=%d&to=%d", t0, t1))
	assert.Equal(t, []string{
		"2015-01-01 00:00:00",
		"2016-01-01 00:00:00",
		"2017-01-01 00:00:00",
		"2018-01-01 00:00:00",
	}, starts)
}

func TestYearEmptyFail(t *testing.T) {
	t0 := testdb.StrToSec("2015-01-01 00:00:00")
	t1 := testdb.StrToSec("2015-01-01 00:00:00")
	bucketFail(t, fmt.Sprintf("interval=year&from=%d&to=%d", t0, t1),
		"no interval requested")
}

func TestIntervalMissing(t *testing.T) {
	t0 := testdb.StrToSec("2015-01-01 00:00:00")
	t1 := testdb.StrToSec("2018-01-01 00:00:00")
	bucketFail(t, fmt.Sprintf("from=%d&to=%d", t0, t1), "interval", "required")
}

func TestBadIntervalName(t *testing.T) {
	t0 := testdb.StrToSec("2015-01-01 00:00:00")
	t1 := testdb.StrToSec("2018-01-01 00:00:00")
	bucketFail(t, fmt.Sprintf("interval=century&from=%d&to=%d", t0, t1),
		"invalid", "century")
}
