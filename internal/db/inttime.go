package db

import "time"

type Second int64
type Nano int64

// TODO(acsaba): get rid of this function, remove time dependency.
func TimeToSecond(t time.Time) Second {
	return Second(t.Unix())
}

func (s Second) ToNano() Nano {
	return Nano(s * 1000000000)
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
