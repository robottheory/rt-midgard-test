package db

import (
	"context"
	"log"
	"sync/atomic"
	"time"
)

type Second int64
type Nano int64

// TODO(acsaba): get rid of this function, remove time dependency.
func TimeToSecond(t time.Time) Second {
	return Second(t.Unix())
}

// TODO(acsaba): get rid of this function, remove time dependency.
func TimeToNano(t time.Time) Nano {
	return Nano(t.UnixNano())
}

func (s Second) ToNano() Nano {
	return Nano(s * 1e9)
}

func (s Second) ToI() int64 {
	return int64(s)
}

func (s Second) ToTime() time.Time {
	return time.Unix(int64(s), 0)
}

func (s Second) Add(duration time.Duration) Second {
	return s + Second(duration.Seconds())
}

func (n Nano) ToI() int64 {
	return int64(n)
}

func (n Nano) ToSecond() Second {
	return Second(n / 1e9)
}

// Nano values,
var lastBlockTimestamp int64
var firstBlockTimestamp int64

func init() {
	// A sane default value for test.
	// If this is too high the history endpoints will cut off results.
	firstBlockTimestamp = 1606780800 * 1e9 // 2020-12-01 00:00
}

func SetFirstBlockTimestamp(n Nano) {
	atomic.StoreInt64(&firstBlockTimestamp, n.ToI())
}

func LoadFirstBlockTimestampFromDB(ctx context.Context) {
	q := `select timestamp from block_log where height = 1`
	rows, err := Query(ctx, q)
	if err != nil {
		log.Println("Failed to query for first timestamp.")
	}
	if !rows.Next() {
		// There were no blocks yet
		return
	}
	var t0 Nano
	err = rows.Scan(&t0)
	if err != nil {
		log.Println("Failed to read first timestamp.")
	}
	SetFirstBlockTimestamp(t0)
}

func FirstBlockNano() Nano {
	return Nano(atomic.LoadInt64(&firstBlockTimestamp))
}

func FirstBlockSecond() Second {
	return FirstBlockNano().ToSecond()
}

func SetLastBlockTimestamp(n Nano) {
	atomic.StoreInt64(&lastBlockTimestamp, n.ToI())
}

func LastBlockTimestamp() Nano {
	return Nano(atomic.LoadInt64(&lastBlockTimestamp))
}

func NowNano() Nano {
	return LastBlockTimestamp() + 1
}

func NowSecond() Second {
	return LastBlockTimestamp().ToSecond() + 1
}
