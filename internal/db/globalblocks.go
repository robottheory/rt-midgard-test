package db

import (
	"context"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/rs/zerolog/log"
	"gitlab.com/thorchain/midgard/config"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

type BlockId struct {
	Height    int64
	Timestamp Nano
}

type StoredBlockId struct {
	ptr unsafe.Pointer
}

func (s *StoredBlockId) Set(height int64, timestamp Nano) {
	id := BlockId{
		Height:    height,
		Timestamp: timestamp,
	}
	atomic.StorePointer(&s.ptr, unsafe.Pointer(&id))
}

func (s *StoredBlockId) Get() BlockId {
	ret := (*BlockId)(atomic.LoadPointer(&s.ptr))
	if ret != nil {
		return *ret
	}
	return BlockId{}
}

func (s *StoredBlockId) AsHeightTS() oapigen.HeightTS {
	return oapigen.HeightTS{
		Height:    int(s.Get().Height),
		Timestamp: int(s.Get().Timestamp.ToSecond()),
	}
}

var (
	LastThorNodeBlock  StoredBlockId
	LastFetchedBlock   StoredBlockId
	LastCommittedBlock StoredBlockId

	// Note: the Height is updated/kept is sync with the Timestamp until fully catched up:
	LastAggregatedBlock StoredBlockId

	FirstBlock StoredBlockId
)

func firstBlockInDB(ctx context.Context) (hash []byte, height int64, timestamp Nano) {
	q := `SELECT hash, height, timestamp FROM block_log ORDER BY timestamp LIMIT 1`
	rows, err := Query(ctx, q)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to query for first timestamp")
	}
	defer rows.Close()
	if !rows.Next() {
		// There were no blocks yet
		return []byte{}, 1, 0
	}
	err = rows.Scan(&hash, &height, &timestamp)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to read for first timestamp")
	}
	if len(hash) == 0 {
		log.Fatal().Err(err).Msg("First block hash is empty in the DB")
	}
	return
}

func SetFirstBlockFromDB(ctx context.Context) bool {
	hash, height, t0 := firstBlockInDB(ctx)
	SetChain(ChainInfo{Description: "db", ChainId: PrintableHash(hash), EarliestBlockHeight: height, EarliestBlockTime: t0.ToTime().UTC()})
	return true
}

// Fatals if there is a mismatch between FirstBlock and the db values.
func CheckFirstBlockInDB(ctx context.Context, chain ChainInfo) {
	hashInDB, heightInDb, timeInDb := firstBlockInDB(ctx)
	if len(hashInDB) == 0 {
		return
	}
	chain.AssertStartMatch(
		ChainInfoFrom("db", hashInDB, heightInDb, timeInDb.ToTime().UTC(), heightInDb))
}

// TODO(huginn): define a better signaling, make it DB aggregate dependent
// 0 == false ; 1 == true
var fetchCaughtUp int32 = 0

func SetFetchCaughtUp() {
	atomic.StoreInt32(&fetchCaughtUp, 1)
}

// FullyCaughtUp returns true if the last stage of block processing (aggregation)
// is less than the configured amount of time in the past.
// At this point Midgard is fully functional and is ready to serve up-to-date data.
func FullyCaughtUp() bool {
	duration := time.Since(LastAggregatedBlock.Get().Timestamp.ToTime())
	return duration < time.Duration(config.Global.MaxBlockAge)
}
