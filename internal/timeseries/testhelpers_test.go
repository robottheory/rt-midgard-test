// Exports functionality for tests only.
// Because the filename has _test, it won't be included in production builds.

package timeseries

import (
	"time"
)

// Often current height or timestamp is read from the last track, this function helps
// to set them for tests.
func SetLastTrackForTest(height int64, timestamp time.Time, hash string) {
	track := blockTrack{
		Height:    height,
		Timestamp: timestamp,
		Hash:      make([]byte, len(hash)),
		aggTrack: aggTrack{
			AssetE8DepthPerPool: map[string]int64{},
			RuneE8DepthPerPool:  map[string]int64{},
		},
	}
	copy(track.Hash, hash)
	lastBlockTrack.Store(&track)
}
