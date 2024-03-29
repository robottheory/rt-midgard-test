package timeseries

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"math"
	"sync/atomic"
	"time"

	"gitlab.com/thorchain/midgard/internal/fetch/sync/chain"

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

var blockFlushTimer = timer.NewTimer("block_write_flush")

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
	UnitsPerPool        map[string]int64
	PricePerPool        map[string]float64
	PriceUSDPPerPool    map[string]float64
}

var usdPoolWhitelist = []string{}

// Setup initializes the package. The previous state is restored (if there was any).
func Setup(whitelist []string) error {
	usdPoolWhitelist = whitelist
	const q = "SELECT height, timestamp, hash, agg_state FROM block_log ORDER BY height DESC LIMIT 1"
	rows, err := db.Query(context.Background(), q)
	if err != nil {
		return fmt.Errorf("last block lookup: %w", err)
	}
	defer rows.Close()

	var track blockTrack
	if rows.Next() {
		var ns int64
		var aggSerial []byte
		err := rows.Scan(&track.Height, &ns, &track.Hash, &aggSerial)
		if err != nil {
			return err
		}
		track.Timestamp = time.Unix(0, ns)
		if err := gob.NewDecoder(bytes.NewReader(aggSerial)).Decode(&track.aggTrack); err != nil {
			return fmt.Errorf("restore with malformed aggregation state denied on %w", err)
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
	for pool, E8 := range track.UnitsPerPool {
		record.Recorder.SetPoolUnit(pool, E8)
	}

	return rows.Err()
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

func RunePriceUSDForDepths(depths DepthMap) float64 {
	ret := math.NaN()
	var maxdepth int64 = -1

	for _, pool := range usdPoolWhitelist {
		poolInfo, ok := depths[pool]
		if ok && maxdepth < poolInfo.RuneDepth {
			maxdepth = poolInfo.RuneDepth
			ret = 1 / poolInfo.AssetPrice()
		}
	}
	return ret
}

func ProcessBlock(block *chain.Block, commit bool) (err error) {
	err = db.Inserter.StartBlock()
	if err != nil {
		return
	}

	poolPrice := make(map[string]float64)
	poolPriceUSD := make(map[string]float64)
	// Record all the events
	record.ProcessBlock(block)

	depths := make(map[string]PoolDepths)
	for pool := range record.Recorder.AssetE8DepthPerPool() {
		depths[pool] = PoolDepths{
			AssetDepth: record.Recorder.AssetE8DepthPerPool()[pool],
			RuneDepth:  record.Recorder.RuneE8DepthPerPool()[pool],
			PoolUnit:   record.Recorder.UnitsPerPool()[pool],
		}
	}
	for pool := range record.Recorder.AssetE8DepthPerPool() {
		if _, ok := record.Recorder.AssetE8DepthPerPool()[pool]; ok {
			if _, ok := record.Recorder.RuneE8DepthPerPool()[pool]; ok {
				poolPrice[pool] = AssetPrice(record.Recorder.AssetE8DepthPerPool()[pool], record.Recorder.RuneE8DepthPerPool()[pool])
				poolPriceUSD[pool] = RunePriceUSDForDepths(depths) * poolPrice[pool]
			}
		}
	}
	record.Recorder.SetPoolPriceUSD(poolPriceUSD)
	// in-memory snapshot
	track := blockTrack{
		Height:    block.Height,
		Timestamp: block.Time,
		Hash:      block.Hash,
		aggTrack: aggTrack{
			AssetE8DepthPerPool: record.Recorder.AssetE8DepthPerPool(),
			RuneE8DepthPerPool:  record.Recorder.RuneE8DepthPerPool(),
			SynthE8DepthPerPool: record.Recorder.SynthE8DepthPerPool(),
			UnitsPerPool:        record.Recorder.UnitsPerPool(),
			PricePerPool:        poolPrice,
			PriceUSDPPerPool:    poolPriceUSD,
		},
	}

	firstBlockHeight := db.FirstBlock.Get().Height
	// We know that this is the first block if:
	// - db.FirstBlock was not set yet
	// - it was set and this block is it
	thisIsTheFirstBlock := firstBlockHeight == 0 || block.Height <= firstBlockHeight

	var aggSerial bytes.Buffer
	if commit || thisIsTheFirstBlock {
		// Persist the current state to the DB on "commit" blocks.
		// This way we can continue after being interrupted, but not waste space on intermediary
		// blocks in the batch.
		if err := gob.NewEncoder(&aggSerial).Encode(&track.aggTrack); err != nil {
			// won't bring the service down, but prevents state recovery
			log.Error().Err(err).Msg("Failed to persist tracking state")
		}
	}
	cols := []string{"height", "timestamp", "hash", "agg_state"}
	err = db.Inserter.Insert("block_log", cols, block.Height, block.Time.UnixNano(), block.Hash, aggSerial.Bytes())
	if err != nil {
		return fmt.Errorf("persist block height %d: %w", block.Height, err)
	}

	err = depthRecorder.update(block.Time,
		track.aggTrack.AssetE8DepthPerPool,
		track.aggTrack.RuneE8DepthPerPool,
		track.aggTrack.SynthE8DepthPerPool,
		track.aggTrack.UnitsPerPool, track.PricePerPool, track.PriceUSDPPerPool)
	if err != nil {
		return
	}

	err = db.Inserter.EndBlock()
	if err != nil {
		return
	}

	if commit || thisIsTheFirstBlock {
		defer blockFlushTimer.One()()

		err = db.Inserter.Flush()
		if err != nil {
			db.MarkBatchInserterFail()
			log.Fatal().Err(err).Msg("Inserter.Flush() failed. Marking BatchInserter as failed and exiting to switch to TxInserter.")
			return
		}
		// update global in-memory state
		setLastBlock(&track)

		// For the first block:
		firstBlockHeight := db.FirstBlock.Get().Height
		if firstBlockHeight == 0 || block.Height <= firstBlockHeight {
			hash := db.PrintableHash(string(block.Hash))
			log.Info().Int64("height", block.Height).Str("hash", hash).Msg("Processed first block")
			db.SetAndCheckFirstBlock(hash, block.Height, db.TimeToNano(block.Time))
		}
	}
	return nil
}

func setLastBlock(track *blockTrack) {
	lastBlockTrack.Store(track)
	db.LastCommittedBlock.Set(track.Height, db.TimeToNano(track.Timestamp))
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
