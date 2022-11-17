package kafka

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/tendermint/tendermint/abci/types"
	"time"
)

// IndexedEvent is the blockchain event plus the height and offset in that block so we know where it came from.
// Why are these signed, you ask?  See:
// https://blog.cosmos.network/choosing-a-type-for-blockchain-height-beware-of-unsigned-integers-714804dddf1d
type IndexedEvent struct {
	EventIndex     EventIdx
	Height         int64
	Offset         int16
	BlockTimestamp time.Time

	Event *types.Event
}

type IndexedEventCodec struct{}

type EventIdx struct {
	Height int64
	Offset int16
}

const (
	V0 byte = 0
)

func (ie IndexedEvent) KeyAsString() (string, error) {
	return fmt.Sprintf("%v.%06d", ie.Height, ie.Offset), nil
}

func (ie IndexedEvent) KeyAsBinary() ([10]byte, error) {
	var val [10]byte

	if ie.Height < 0 || ie.Offset < 0 {
		return val, errors.New("can't encode index with negative height or offset")
	}

	h := uint64(ie.Height)
	o := uint16(ie.Offset)

	binary.LittleEndian.PutUint64(val[0:], h)
	binary.LittleEndian.PutUint16(val[8:], o)

	return val, nil
}

func (i *IndexedEventCodec) Encode(value interface{}) ([]byte, error) {
	if _, isEvent := value.(IndexedEvent); !isEvent {
		return nil, fmt.Errorf("codec requires value types.IndexedEvent, got %T", value)
	}

	iEvent := value.(IndexedEvent)

	// 1048576 is the tendermint default max_txs_bytes, plus some space for the iEvent fields
	cbuf := NewCappedBuffer(make([]byte, 0, 1024), 1048576+256)

	// Write Version
	version := []byte{V0}
	cbuf.Write(version)

	enc := gob.NewEncoder(cbuf)
	if err := enc.Encode(iEvent); err != nil {
		return nil, err
	}

	return cbuf.Bytes(), nil
}

func (e *IndexedEventCodec) Decode(data []byte) (interface{}, error) {
	version := data[0]
	if version != V0 {
		return nil, errors.New("unknown version while decoding message")
	}

	buf := bytes.NewReader(data[1:])
	decode := gob.NewDecoder(buf)

	iEvent := IndexedEvent{}

	if err := decode.Decode(&iEvent); err != nil {
		return nil, err
	}

	return iEvent, nil
}

func (ei EventIdx) LessOrEqual(oi EventIdx) bool {
	if (ei.Height < oi.Height) ||
		(ei.Height == oi.Height && ei.Offset <= oi.Offset) {
		return true
	}

	return false
}
