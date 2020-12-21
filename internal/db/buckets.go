package db

import (
	"context"
	"fmt"
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

type Seconds []Second

// Bucketing has two modes:
// a) If interval is given then all timestamps are rounded to the interval boundaries.
//    Timestamps contains count+1 timestamps, so the last timestamp should be the endTime
//    of the last bucket.
// b) If interval is nill, then it's an exact search with from..to parameters.
//    In this case there are exactly to Timestamps.
type Buckets struct {
	Timestamps Seconds
	interval   *Interval
}

func OneIntervalBuckets(from, to Second) Buckets {
	return Buckets{Timestamps: Seconds{from, to}}
}

var startOfChain Second = 1606780800 // 2020-12-01 00:00

func AllHistoryBuckets() Buckets {
	return Buckets{Timestamps: Seconds{startOfChain, Now().ToSecond()}}
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

func (b Buckets) OneInterval() bool {
	return b.interval == nil
}

// This name is used for the sql date_trunc function.
// date_trunc can not accept '5 minute' as a parameter.
// Instead we round every timestamp to the nearest 5min
// with (timestamp / 300) * 300
var dbIntervalName = []string{
	Min5:    "minute",
	Hour:    "hour",
	Day:     "day",
	Week:    "week",
	Month:   "month",
	Quarter: "quarter",
	Year:    "year",
}

const maxIntervalCount = 100
const cutoffWindowLength = 200

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
func generateTimestamps(ctx context.Context, interval Interval, w Window) (Seconds, miderr.Err) {
	// We use an SQL query to use the date_trunc of sql.
	// It's not important which table we select we just need a timestamp type and we use WHERE 1=0
	// in order not to actually select any data.
	// We could consider writing an sql function instead or programming dategeneration in go.

	if maxDuration[interval]*cutoffWindowLength < (w.Until - w.From) {
		return nil, miderr.BadRequestF(
			"Too wide range requested, max allowed intervals (%d).\n%s",
			maxIntervalCount, usage)
	}

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

	// Widen from and until a bit, to make sure we don't loose any at edges.
	untilParam := w.Until + maxDuration[interval] + minDuration[interval]
	fromParam := w.From - maxDuration[interval] - minDuration[interval]
	rows, err := Query(ctx, q, minDuration[interval],
		fromParam, untilParam,
		dbIntervalName[interval])
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
		timestamps = append(timestamps, timestamp)
	}

	// Leave exactly one timestamp bigger than Until
	lastIdx := len(timestamps) - 1
	for ; 0 < lastIdx && w.Until <= timestamps[lastIdx-1]; lastIdx-- {
	}
	firstIdx := 0
	for ; firstIdx < len(timestamps) && timestamps[firstIdx+1] <= w.From; firstIdx++ {
	}
	ret := timestamps[firstIdx : lastIdx+1]

	if len(ret) < 2 {
		// We need at least 2 elements to have an [from, to) interval.
		return nil, miderr.BadRequestF(
			"No interval requested. Use count or a wider from/to range.\n%s", usage)
	}
	return ret, nil
}

// TODO(acsaba): Migrate graphql to use GenerateBuckets.
func BucketsFromWindow(ctx context.Context, window Window, interval Interval) (ret Buckets, merr miderr.Err) {
	ret.interval = &interval
	ret.Timestamps, merr = generateTimestamps(ctx, *ret.interval, window)
	if merr != nil {
		return
	}
	if maxIntervalCount < ret.Count() {
		return Buckets{}, miderr.BadRequestF("Too wide range requested: %d, max allowed intervals (%d).\n%s",
			ret.Count(), maxIntervalCount, usage)
	}
	return
}

const usage = `Usage:
Parameters:
- Interval is required, possible values: 5min, hour, day, week, month, quarter, year.
- count: optional int, (1..100)
- from/to: optional int, unix second.

Providing all count/from/to will result in error. Possible configurations:
- interval=day&count=10                    - last 10 days.
- interval=day&count=10&to=1234567890      - last 10 days before to.
- interval=day&count=10&from=1234567890    - next 10 days after from.
- interval=day&from=1100000&to=1100000     - days between from and to. It will fail if more than 100 intervals are requested.
`

func generateBucketsWithInterval(ctx context.Context, from, to *Second, count *int64, interval Interval) (ret Buckets, merr miderr.Err) {
	if count == nil {
		if from == nil || to == nil {
			return Buckets{}, miderr.BadRequestF(
				"Provide count or specify both from and to.\n%s", usage)
		}
		ret.interval = &interval
		ret.Timestamps, merr = generateTimestamps(ctx, *ret.interval, Window{From: *from, Until: *to})
		if merr != nil {
			return
		}
		if maxIntervalCount < ret.Count() {
			return Buckets{}, miderr.BadRequestF("Too wide range requested: %d, max allowed intervals (%d).\n%s",
				ret.Count(), maxIntervalCount, usage)
		}
		return ret, nil
	}
	if count != nil {
		if *count < 1 || maxIntervalCount < *count {
			return Buckets{}, miderr.BadRequestF("Count out of range: %d, allowed [1..%d].\n%s",
				*count, maxIntervalCount, usage)
		}
		countInt := (int)(*count)
		if from != nil && to != nil {
			return Buckets{}, miderr.BadRequestF(
				"Count and from and to was specified. Specify max 2 of them.\n%s", usage)
		}
		if from == nil && to == nil {
			now := Now().ToSecond()
			to = &now
		}
		ret.interval = &interval
		if to != nil {
			// to & count was given
			window := Window{From: *to - Second(*count)*maxDuration[interval], Until: *to}
			ret.Timestamps, merr = generateTimestamps(ctx, *ret.interval, window)
			if merr != nil {
				return
			}
			// We might have more intervals then requested, trim the beginning.
			ret.Timestamps = ret.Timestamps[ret.Count()-(countInt):]
			return
		} else {
			// from & count was given
			window := Window{From: *from, Until: *from + Second(*count)*maxDuration[interval]}
			ret.Timestamps, merr = generateTimestamps(ctx, *ret.interval, window)
			if merr != nil {
				return
			}
			// We might have more intervals then requested, trim the beginning.
			ret.Timestamps = ret.Timestamps[:countInt+1]
			return
		}
	}
	return
}

// No interval was provided, we do a single From..To query
func generateBucketsOnlyMeta(ctx context.Context, fromP, toP *Second, count *int64) (ret Buckets, merr miderr.Err) {
	if count != nil {
		return Buckets{}, miderr.BadRequestF(
			"count was provided but no interval parameter.\n%s", usage)
	}
	if toP == nil {
		now := Now().ToSecond()
		toP = &now
	}
	if fromP == nil {
		fromP = &startOfChain
	}
	return OneIntervalBuckets(*fromP, *toP), nil
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

func optionalIntParam(query url.Values, name string) (*int64, miderr.Err) {
	input := query.Get(name)
	if input == "" {
		return nil, nil
	}
	i, err := strconv.ParseInt(input, 10, 64)
	if err != nil {
		return nil, miderr.BadRequestF(
			"Parameter '%s' is not integer: %s\n%s", name, input, usage)
	}
	return &i, nil
}
func optionalSecParam(query url.Values, name string) (*Second, miderr.Err) {
	intp, merr := optionalIntParam(query, name)
	return (*Second)(intp), merr
}

func BucketsFromQuery(ctx context.Context, query url.Values) (Buckets, miderr.Err) {
	from, merr := optionalSecParam(query, "from")
	if merr != nil {
		return Buckets{}, merr
	}
	to, merr := optionalSecParam(query, "to")
	if merr != nil {
		return Buckets{}, merr
	}
	count, merr := optionalIntParam(query, "count")
	if merr != nil {
		return Buckets{}, merr
	}

	intervalStr := query.Get("interval")
	if intervalStr == "" {
		return generateBucketsOnlyMeta(ctx, from, to, count)
	}
	interval, ok := intervalFromJSONParamMap[strings.ToLower(intervalStr)]
	if !ok {
		return Buckets{}, miderr.BadRequestF(
			"Invalid interval '(%s)', accepted values: 5min, hour, day, week, month, quarter, year.\n%s",
			intervalStr, usage)
	}

	return generateBucketsWithInterval(ctx, from, to, count, interval)
}

// Select field that truncates the value considering the buckets.Interval
// Result is date in seconds.
func SelectTruncatedTimestamp(targetColumn string, buckets Buckets) string {
	if buckets.OneInterval() {
		return fmt.Sprintf(`(%d)::BIGINT`, buckets.Start())
	} else {
		return fmt.Sprintf(
			`EXTRACT(EPOCH FROM (date_trunc('%s', to_timestamp(%s/1000000000/300*300))))::BIGINT`,
			dbIntervalName[*buckets.interval], targetColumn)
	}
}
