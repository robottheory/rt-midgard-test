package stat

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gitlab.com/thorchain/midgard/internal/graphql/model"
)

func jsonParamToDbInterval(str string) (model.Interval, error) {
	switch str {
	case "5min":
		return model.IntervalMinute5, nil
	case "hour":
		return model.IntervalHour, nil
	case "day":
		return model.IntervalDay, nil
	case "week":
		return model.IntervalWeek, nil
	case "month":
		return model.IntervalMonth, nil
	case "quarter":
		return model.IntervalQuarter, nil
	case "year":
		return model.IntervalYear, nil
	}
	return "", errors.New("the requested interval is invalid: " + str)
}

const maxIntervalCount = 101

// We want to limit the respons intervals, but we want to restrict the
// Database lookup range too so we don't do all the work unnecessarily.
func getMaxDuration(inv model.Interval) (time.Duration, error) {
	switch inv {
	case model.IntervalMinute5:
		return time.Minute * 5 * maxIntervalCount, nil
	case model.IntervalHour:
		return time.Hour * maxIntervalCount, nil
	case model.IntervalDay:
		return time.Hour * 24 * maxIntervalCount, nil
	case model.IntervalWeek:
		return time.Hour * 24 * 7 * maxIntervalCount, nil
	case model.IntervalMonth:
		return time.Hour * 24 * 31 * maxIntervalCount, nil
	case model.IntervalQuarter:
		return time.Hour * 24 * 31 * 3 * maxIntervalCount, nil
	case model.IntervalYear:
		return time.Hour * 24 * 365 * maxIntervalCount, nil
	}
	return time.Duration(0), errors.New(string("the requested interval is invalid: " + inv))
}

// A reasonable period for gapfil which guaranties that date_trunc will
// create all the needed entries.
func reasonableGapfillParam(inv model.Interval) (string, error) {
	switch inv {
	case model.IntervalMinute5:
		return "300::BIGINT", nil // 5 minutes
	case model.IntervalHour:
		return "3600::BIGINT", nil // 1 hour
	case model.IntervalDay:
		return "86400::BIGINT", nil // 24 hours
	case model.IntervalWeek:
		return "604800::BIGINT", nil // 7 days
	case model.IntervalMonth:
		return "2160000::BIGINT", nil // 25 days
	case model.IntervalQuarter:
		return "7344000::BIGINT", nil // 85 days
	case model.IntervalYear:
		return "25920000::BIGINT", nil // 300 days
	}
	return "", errors.New(string("the requested interval is invalid: " + inv))
}

// In addition of setting sane default values it also restricts window length.
func fillMissingFromTo(w Window, inv model.Interval) (Window, error) {
	maxDuration, err := getMaxDuration(inv)
	if err != nil {
		return Window{}, err
	}

	if w.From.Unix() != 0 && w.Until.Unix() == 0 {
		// if only since is defined
		limitedTime := w.From.Add(maxDuration)
		w.Until = limitedTime
	} else if w.From.Unix() == 0 && w.Until.Unix() != 0 {
		// if only until is defined
		limitedTime := w.Until.Add(-maxDuration)
		w.From = limitedTime
	} else if w.From.Unix() == 0 && w.Until.Unix() == 0 {
		// if neither is defined
		w.Until = time.Now()
	}

	// if the starting time lies outside the limit
	limitedTime := w.Until.Add(-maxDuration)
	if limitedTime.After(w.From) {
		// limit the value
		w.From = limitedTime
	}

	return w, nil
}

// Returns all the buckets for the window, so other queries don't have to care about gapfill functionality.
func generateBuckets(ctx context.Context, interval model.Interval, w Window) ([]int64, Window, error) {
	// We use an SQL query to use the date_trunc of sql.
	// It's not important which table we select we just need a timestamp type and we use WHERE 1=0
	// in order not to actually select any data.
	// We could consider writing an sql function instead or programming dategeneration in go.

	w, err := fillMissingFromTo(w, interval)
	if err != nil {
		return nil, w, err
	}
	gapfill, err := reasonableGapfillParam(interval)
	if err != nil {
		return nil, w, err
	}

	q := fmt.Sprintf(`
		WITH gapfill AS (
			SELECT
				time_bucket_gapfill(%s, block_timestamp, $1::BIGINT, $2::BIGINT) as bucket
			FROM block_pool_depths
			WHERE 1=0
			GROUP BY bucket)
		SELECT
			date_trunc($3, to_timestamp(bucket)) as truncated
		FROM gapfill
		GROUP BY truncated
		ORDER BY truncated ASC
	`, gapfill)

	// TODO(acsaba): change the gapfill parameter to seconds, and pass seconds here too.
	rows, err := DBQuery(ctx, q, w.From.Unix(), w.Until.Unix()-1, interval)
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
