package db

import (
	"context"
	"encoding/hex"
	"strings"
	"sync/atomic"
	"unsafe"

	"github.com/rs/zerolog/log"
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

var LastCommitedBlock StoredBlockId
var FirstBlock StoredBlockId

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
	FirstBlock.Set(1, t0)
	log.Info().Msgf("Loaded first block hash from DB: %s", PrintableHash(hash))
	SetFirstBlochHash(hash)
}

// TODO(huginn): define a better signaling, make it DB aggregate dependent
// 0 == false ; 1 == true
var fetchCaughtUp int32 = 0

func SetFetchCaughtUp() {
	atomic.StoreInt32(&fetchCaughtUp, 1)
}

// FetchCaughtUp returns true if we reached the height which we saw at startup.
// Doesn't check current time, doesn't check if the chain went further since.
func FetchCaughtUp() bool {
	return atomic.LoadInt32(&fetchCaughtUp) != 0
}
