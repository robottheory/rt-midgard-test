package db

import (
	"sync/atomic"
	"unsafe"
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

func init() {
	// Nano value
	// A sane default value for test.
	// If this is too high the history endpoints will cut off results.
	FirstBlock.Set(1, 1606780800*1e9) // 2020-12-01 00:00
}
