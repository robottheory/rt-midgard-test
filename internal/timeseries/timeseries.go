package timeseries

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/fetch/record"
	"gitlab.com/thorchain/midgard/internal/util/timer"
)

// OutboundTimeout is an upperboundary for the amount of time for a followup on outbound events.
const OutboundTimeout = time.Hour * 48

// LastBlockTrack is an in-memory copy of the write state.
// TODO(acsaba): migrate users to using BlockState wherever it's possible.
var lastBlockTrack atomic.Value

// BlockTrack is a write state.
type blockTrack struct {
	Height int64
	// TODO(acsaba): rewrite to db.Nano
	Timestamp time.Time
	Hash      []byte
	aggTrack
}

// AggTrack has a snapshot of runningTotals.
type aggTrack struct {
	AssetE8DepthPerPool map[string]int64
	RuneE8DepthPerPool  map[string]int64
	SynthE8DepthPerPool map[string]int64
}

// Setup initializes the package. The previous state is restored (if there was any).
func Setup() (lastBlockHeight int64, lastBlockTimestamp time.Time, lastBlockHash []byte, err error) {
	const q = "SELECT height, timestamp, hash, agg_state FROM block_log ORDER BY height DESC LIMIT 1"
	rows, err := db.Query(context.Background(), q)
	if err != nil {
		return 0, time.Time{}, nil, fmt.Errorf("last block lookup: %w", err)
	}
	defer rows.Close()

	var track blockTrack
	if rows.Next() {
		var ns int64
		var aggSerial []byte
		err := rows.Scan(&track.Height, &ns, &track.Hash, &aggSerial)
		if err != nil {
			return 0, time.Time{}, nil, err
		}
		track.Timestamp = time.Unix(0, ns)
		if err := gob.NewDecoder(bytes.NewReader(aggSerial)).Decode(&track.aggTrack); err != nil {
			return 0, time.Time{}, nil, fmt.Errorf("restore with malformed aggregation state denied on %w", err)
		}
	}

	// sync in-memory tracker
	setLastBlock(&track)

	// apply aggregation state to recorder
	for pool, E8 := range track.AssetE8DepthPerPool {
		record.Recorder.SetAssetDepth(pool, E8)
	}
	for pool, E8 := range track.RuneE8DepthPerPool {
		record.Recorder.SetRuneDepth(pool, E8)
	}
	for pool, E8 := range track.SynthE8DepthPerPool {
		record.Recorder.SetSynthDepth(pool, E8)
	}

	return track.Height, track.Timestamp, track.Hash, rows.Err()
}

// QueryOneValue is a helper to make store single value queries
// result into dest
func QueryOneValue(dest interface{}, ctx context.Context, query string, args ...interface{}) error {
	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	ok := rows.Next()

	if !ok {
		return errors.New("Expected one result from query but got none.")
	}

	err = rows.Scan(dest)
	if err != nil {
		return err
	}

	return nil
}

var blockCommitTimer = timer.NewTimer("block_write_commit")

// CommitBlock marks the given height as done.
// Invokation of EventListener during CommitBlock causes race conditions!
func CommitBlock(height int64, timestamp time.Time, hash []byte) error {
	defer blockCommitTimer.One()()
	// in-memory snapshot
	track := blockTrack{
		Height:    height,
		Timestamp: timestamp,
		Hash:      make([]byte, len(hash)),
		aggTrack: aggTrack{
			AssetE8DepthPerPool: record.Recorder.AssetE8DepthPerPool(),
			RuneE8DepthPerPool:  record.Recorder.RuneE8DepthPerPool(),
			SynthE8DepthPerPool: record.Recorder.SynthE8DepthPerPool(),
		},
	}
	copy(track.Hash, hash)

	// persist to database
	var aggSerial bytes.Buffer
	if err := gob.NewEncoder(&aggSerial).Encode(&track.aggTrack); err != nil {
		// won't bing the service down, but prevents state recovery
		log.Error().Err(err).Msg("aggregation state ommited from persistence")
	}
	// TODO(huginn): Can it happen that we have a CONFLICT, that is try to insert the same block twice?
	// If no, remove this comment.
	// If yes, reinstate the warning that we have a duplicate block.
	q := []string{"height", "timestamp", "hash", "agg_state"}
	err := db.Inserter.Insert("block_log", q, height, timestamp.UnixNano(), hash, aggSerial.Bytes())
	if err != nil {
		return fmt.Errorf("persist block height %d: %w", height, err)
	}

	err = depthRecorder.update(timestamp, track.aggTrack.AssetE8DepthPerPool, track.aggTrack.RuneE8DepthPerPool, track.aggTrack.SynthE8DepthPerPool)
	if err != nil {
		return err
	}

	// commit in-memory state
	setLastBlock(&track)

	if height == 1 {
		db.SetFirstBlockTimestamp(db.TimeToNano(timestamp))
		db.SetFirstBlochHash(string(track.Hash))
		record.LoadCorrections(db.ChainID())
	}
	return nil
}

func setLastBlock(track *blockTrack) {
	lastBlockTrack.Store(track)
	db.SetLastBlockTimestamp(db.TimeToNano(track.Timestamp))
	Latest.setLatestStates(track)
}

func getLastBlock() *blockTrack {
	interfacePtr := lastBlockTrack.Load()
	if interfacePtr == nil {
		log.Panic().Msg("LastBlock not loaded yet")
	}
	return interfacePtr.(*blockTrack)
}

// LastBlock gets the most recent commit.
func LastBlock() (height int64, timestamp time.Time, hash []byte) {
	track := getLastBlock()
	return track.Height, track.Timestamp, track.Hash
}

// AssetAndRuneDepths gets the current snapshot handle.
// The asset price is the asset depth divided by the RUNE depth.
func AssetAndRuneDepths() (assetE8PerPool, runeE8PerPool map[string]int64, timestamp time.Time) {
	track := getLastBlock()
	return track.aggTrack.AssetE8DepthPerPool, track.aggTrack.RuneE8DepthPerPool, track.Timestamp
}

// AllDepths gets the current snapshot handle.
// The asset price is the asset depth divided by the RUNE depth.
func AllDepths() (assetE8PerPool, runeE8PerPool, synthE8PerPool map[string]int64, timestamp time.Time) {
	track := getLastBlock()
	return track.aggTrack.AssetE8DepthPerPool, track.aggTrack.RuneE8DepthPerPool, track.aggTrack.SynthE8DepthPerPool, track.Timestamp
}
