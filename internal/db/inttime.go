package db

import (
	"context"
	"sync/atomic"
	"time"
)

type (
	Second int64
	Nano   int64
)

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

func (n Nano) ToTime() time.Time {
	return time.Unix(0, int64(n))
}

// 0 == false ; 1 == true
var fetchCaughtUp int32 = 0

func NowNano() Nano {
	return LastCommitedBlock.Get().Timestamp + 1
}

func NowSecond() Second {
	return LastCommitedBlock.Get().Timestamp.ToSecond() + 1
}

func SetFetchCaughtUp() {
	atomic.StoreInt32(&fetchCaughtUp, 1)
}

// FetchCaughtUp returns true if we reached the height which we saw at startup.
// Doesn't check current time, doesn't check if the chain went further since.
func FetchCaughtUp() bool {
	return atomic.LoadInt32(&fetchCaughtUp) != 0
}

func SleepWithContext(ctx context.Context, duration time.Duration) {
	select {
	case <-time.After(duration):
	case <-ctx.Done():
		return
	}
}
