package db

import (
	"context"
	"encoding/hex"
	"strings"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
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

// Nano value
// A sane default value for test.
// If this is too high the history endpoints will cut off results.
var firstBlockTimestamp int64 = 1606780800 * 1e9 // 2020-12-01 00:00

// 0 == false ; 1 == true
var fetchCaughtUp int32 = 0

var firstBlockHash string = ""

func PrintableHash(encodedHash string) string {
	return strings.ToUpper(hex.EncodeToString([]byte(encodedHash)))
}

func SetFirstBlochHash(hash string) {
	hash = PrintableHash(hash)
	log.Info().Msgf("First block hash: %s", hash)
	firstBlockHash = hash
}

func ChainID() string {
	return firstBlockHash
}

func SetFirstBlockTimestamp(n Nano) {
	atomic.StoreInt64(&firstBlockTimestamp, n.ToI())
}

func LoadFirstBlockFromDB(ctx context.Context) {
	q := `select timestamp, hash from block_log where height = 1`
	rows, err := Query(ctx, q)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query for first timestamp")
	}
	if !rows.Next() {
		// There were no blocks yet
		return
	}
	var t0 Nano
	var hash string
	err = rows.Scan(&t0, &hash)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read for first timestamp")
	}
	SetFirstBlockTimestamp(t0)
	log.Info().Msgf("Loaded first block hash from DB: %s", PrintableHash(hash))
	SetFirstBlochHash(hash)
}

func FirstBlockNano() Nano {
	return Nano(atomic.LoadInt64(&firstBlockTimestamp))
}

func FirstBlockSecond() Second {
	return FirstBlockNano().ToSecond()
}

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
