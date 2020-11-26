package stat

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

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

var intervalFromJSONParamMap = map[string]Interval{
	"5min":    Min5,
	"hour":    Hour,
	"day":     Day,
	"week":    Week,
	"month":   Month,
	"quarter": Quarter,
	"year":    Year,
}

func intervalFromJSONParam(param string) (Interval, error) {
	ret, ok := intervalFromJSONParamMap[strings.ToLower(param)]
	if !ok {
		return UndefinedInterval, errors.New("Invalid interval (" + param +
			"), accepted values: 5min, hour, day, week, month, quarter, year")
	}
	return ret, nil
}

const maxIntervalCount = 101

// We want to limit the respons intervals, but we want to restrict the
// Database lookup range too so we don't do all the work unnecessarily.
var maxDuration = map[Interval]time.Duration{
	Min5:    time.Minute * 5 * maxIntervalCount,
	Hour:    time.Hour * maxIntervalCount,
	Day:     time.Hour * 24 * maxIntervalCount,
	Week:    time.Hour * 24 * 7 * maxIntervalCount,
	Month:   time.Hour * 24 * 31 * maxIntervalCount,
	Quarter: time.Hour * 24 * 31 * 3 * maxIntervalCount,
	Year:    time.Hour * 24 * 365 * maxIntervalCount,
}

// A reasonable period for gapfil which guaranties that date_trunc will
// create all the needed entries.
var reasonableGapfillParam = map[Interval]string{
	Min5:    "300::BIGINT",      // 5 minutes
	Hour:    "3600::BIGINT",     // 1 hour
	Day:     "86400::BIGINT",    // 24 hours
	Week:    "604800::BIGINT",   // 7 days
	Month:   "2160000::BIGINT",  // 25 days
	Quarter: "7344000::BIGINT",  // 85 days
	Year:    "25920000::BIGINT", // 300 days

}

// In addition of setting sane default values it also restricts window length.
// TODO(acsaba): filling default seems not to be used, delete,
//     keep only setting max duration.
func fillMissingFromTo(w Window, inv Interval) Window {
	max := maxDuration[inv]

	if w.From.Unix() != 0 && w.Until.Unix() == 0 {
		// if only since is defined
		limitedTime := w.From.Add(max)
		w.Until = limitedTime
	} else if w.From.Unix() == 0 && w.Until.Unix() != 0 {
		// if only until is defined
		limitedTime := w.Until.Add(-max)
		w.From = limitedTime
	} else if w.From.Unix() == 0 && w.Until.Unix() == 0 {
		// if neither is defined
		w.Until = time.Now()
	}

	// if the starting time lies outside the limit
	limitedTime := w.Until.Add(-max)
	if limitedTime.After(w.From) {
		// limit the value
		w.From = limitedTime
	}

	return w
}

// Returns all the buckets for the window, so other queries don't have to care about gapfill functionality.
func generateBuckets(ctx context.Context, interval Interval, w Window) ([]int64, Window, error) {
	// We use an SQL query to use the date_trunc of sql.
	// It's not important which table we select we just need a timestamp type and we use WHERE 1=0
	// in order not to actually select any data.
	// We could consider writing an sql function instead or programming dategeneration in go.

	w = fillMissingFromTo(w, interval)
	gapfill := reasonableGapfillParam[interval]

	q := fmt.Sprintf(`
		WITH gapfill AS (
			SELECT
				time_bucket_gapfill(%s, block_timestamp, $1::BIGINT, $2::BIGINT) as bucket
			FROM block_pool_depths
			WHERE 1=0
			GROUP BY bucket)
		SELECT
			date_trunc($3, to_timestamp(bucket/300*300)) as truncated
		FROM gapfill
		GROUP BY truncated
		ORDER BY truncated ASC
	`, gapfill)
	// TODO(acsaba): change the gapfill parameter to seconds, and pass seconds here too.
	rows, err := DBQuery(ctx, q, w.From.Unix(), w.Until.Unix()-1, dbIntervalName[interval])
	if err != nil {
		return nil, w, err
	}
	defer rows.Close()

	ret := []int64{}
	for rows.Next() {
		var timestamp time.Time
		err := rows.Scan(&timestamp)
		if err != nil {
			return nil, w, err
		}
		// skip first
		if !timestamp.Before(w.From) {
			if len(ret) == 0 {
				w.From = timestamp
			}
			ret = append(ret, timestamp.Unix())
		}
	}
	return ret, w, nil
}
