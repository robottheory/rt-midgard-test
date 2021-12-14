package db

import (
	"context"
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

func NowNano() Nano {
	return LastCommittedBlock.Get().Timestamp + 1
}

func NowSecond() Second {
	return LastCommittedBlock.Get().Timestamp.ToSecond() + 1
}

func SleepWithContext(ctx context.Context, duration time.Duration) {
	select {
	case <-time.After(duration):
	case <-ctx.Done():
		return
	}
}
