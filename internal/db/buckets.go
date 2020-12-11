package db

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	"gitlab.com/thorchain/midgard/internal/util/miderr"
)

// Window specifies the applicable time period.
type Window struct {
	From  Second // lower bound [inclusive]
	Until Second // upper bound [exclusive]
}

type Interval int

type Seconds []Second

// It contains count+1 timestamps, so the last timestamp should be the endTime of the last bucket.
type Buckets struct {
	Timestamps Seconds
	Interval   Interval
}

func (b Buckets) Start() Second {
	return b.Timestamps[0]
}

func (b Buckets) End() Second {
	return b.Timestamps[len(b.Timestamps)-1]
}

func (b Buckets) Count() int {
	return len(b.Timestamps) - 1
}

func (b Buckets) Bucket(idx int) (startTime, endTime Second) {
	return b.Timestamps[idx], b.Timestamps[idx+1]
}

func (b Buckets) Window() Window {
	return Window{b.Start(), b.End()}
}

const (
	Min5 Interval = iota
	Hour
	Day
	Week
	Month
	Quarter
	Year
	UndefinedInterval
)

// This name is used for the sql date_trunc function.
// date_trunc can not accept '5 minute' as a parameter.
// Instead we round every timestamp to the nearest 5min
// with (timestamp / 300) * 300
var DBIntervalName = []string{
	Min5:    "minute",
	Hour:    "hour",
	Day:     "day",
	Week:    "week",
	Month:   "month",
	Quarter: "quarter",
	Year:    "year",
}

const maxIntervalCount = 100

// Used for extending bounds at least to the next occurance.
var maxDuration = map[Interval]Second{
	Min5:    60 * 5,
	Hour:    60 * 60,
	Day:     60 * 60 * 24,
	Week:    60 * 60 * 24 * 7,
	Month:   60 * 60 * 24 * 31,
	Quarter: 60 * 60 * 24 * 31 * 3,
	Year:    60 * 60 * 24 * 31 * 12,
}

// Used for gapfill, to make sure every interval has one value.
var minDuration = map[Interval]Second{
	Min5:    60 * 5,
	Hour:    60 * 60,
	Day:     60 * 60 * 24,
	Week:    60 * 60 * 24 * 7,
	Month:   60 * 60 * 24 * 28,
	Quarter: 60 * 60 * 24 * 28 * 3,
	Year:    60 * 60 * 24 * 28 * 12,
}

// Returns all the buckets for the window, so other queries don't have to care about gapfill functionality.
func generateBuckets(ctx context.Context, interval Interval, w Window) (Seconds, miderr.Err) {
	// We use an SQL query to use the date_trunc of sql.
	// It's not important which table we select we just need a timestamp type and we use WHERE 1=0
	// in order not to actually select any data.
	// We could consider writing an sql function instead or programming dategeneration in go.

	// TODO(acsaba): return error if window too big before we do the query

	q := `
		WITH gapfill AS (
			SELECT
				time_bucket_gapfill($1::BIGINT, block_timestamp, $2::BIGINT, $3::BIGINT) as bucket
			FROM block_pool_depths
			WHERE 1=0
			GROUP BY bucket)
		SELECT
			EXTRACT(EPOCH FROM
				date_trunc($4, to_timestamp(bucket/300*300)))::BIGINT as truncated
		FROM gapfill
		GROUP BY truncated
		ORDER BY truncated ASC
	`

	rows, err := Query(ctx, q, minDuration[interval],
		w.From, w.Until+maxDuration[interval],
		DBIntervalName[interval])
	if err != nil {
		return nil, miderr.InternalErrE(err)
	}
	defer rows.Close()

	timestamps := []Second{}
	for rows.Next() {
		var timestamp Second
		err := rows.Scan(&timestamp)
		if err != nil {
			return nil, miderr.InternalErrE(err)
		}
		// TODO(acsaba): include the w.From also in the results
		// skip first
		if w.From <= timestamp {
			timestamps = append(timestamps, timestamp)
		}
	}

	// Leave exactly one timestamp bigger than TO
	lastIdx := len(timestamps) - 1
	for ; 0 < lastIdx && w.Until <= timestamps[lastIdx-1]; lastIdx-- {
	}
	ret := timestamps[:lastIdx+1]

	if len(ret) < 2 {
		// We need at least 2 elements to have an [from, to) interval.
		return nil, miderr.BadRequest("No interval requested")
	}
	if maxIntervalCount < len(ret) {
		return nil, miderr.BadRequestF("Too many intervals requested: %d, max allowed (%d)",
			len(ret), maxIntervalCount)
	}
	return timestamps[:lastIdx+1], nil
}

func BucketsFromWindow(ctx context.Context, window Window, interval Interval) (ret Buckets, merr miderr.Err) {
	ret.Interval = interval
	ret.Timestamps, merr = generateBuckets(ctx, ret.Interval, window)
	return
}

var intervalFromJSONParamMap = map[string]Interval{
	"5min":    Min5,
	"hour":    Hour,
	"day":     Day,
	"week":    Week,
	"month":   Month,
	"quarter": Quarter,
	"year":    Year,
}

func getParam(query url.Values, name string) (string, miderr.Err) {
	v := query.Get(name)
	if v == "" {
		return "", miderr.BadRequestF("'%s' parameter is required", name)
	}
	return v, nil
}

func secParam(query url.Values, name string) (s Second, merr miderr.Err) {
	input, merr := getParam(query, name)
	i, err := strconv.ParseInt(input, 10, 64)
	if err != nil {
		return 0, miderr.BadRequestF(
			"Bad unix timestamp for param '%s': %s", name, input)
	}
	return Second(i), nil
}

func BucketsFromQuery(ctx context.Context, query url.Values) (ret Buckets, merr miderr.Err) {
	from, merr := secParam(query, "from")
	if merr != nil {
		return
	}
	to, merr := secParam(query, "to")
	if merr != nil {
		return
	}

	intervalStr, merr := getParam(query, "interval")
	if merr != nil {
		return
	}
	interval, ok := intervalFromJSONParamMap[strings.ToLower(intervalStr)]
	if !ok {
		merr = miderr.BadRequestF(
			"Invalid interval '(%s)', accepted values: 5min, hour, day, week, month, quarter, year",
			intervalStr)
		return
	}

	origWindow := Window{
		From:  from,
		Until: to,
	}
	ret, merr = BucketsFromWindow(ctx, origWindow, interval)
	return
}
