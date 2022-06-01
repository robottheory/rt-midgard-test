package kafka

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/tendermint/tendermint/abci/types"
)

// IndexedEvent is the blockchain event plus the height and offset in that block so we know where it came from.
// Why are these signed, you ask?  See:
// https://blog.cosmos.network/choosing-a-type-for-blockchain-height-beware-of-unsigned-integers-714804dddf1d
type IndexedEvent struct {
	Height int64
	Offset int16
	Event  *types.Event
}

type IndexedEventCodec struct{}

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

	// Size: 1 for version byte, 8 for uint64 (Height), 2 for uint16 (Offset)
	//       plus bytes for Event
	buf := make([]byte, 1+8+2+iEvent.Event.Size())

	// Write Version
	buf[0] = V0

	// Write Height and Offset
	h := uint64(iEvent.Height)
	o := uint16(iEvent.Offset)
	binary.LittleEndian.PutUint64(buf[1:], h)
	binary.LittleEndian.PutUint16(buf[9:], o)

	// Write Event
	//if data, err := iEvent.Event.Marshal(); err != nil {
	//	midlog.InfoF("Reported length: %v, actual length: %v", iEvent.Event.Size(), len(data))
	//}
	if _, err := iEvent.Event.MarshalTo(buf[11:]); err != nil {
		return nil, err
	}

	return buf, nil
}

func (e *IndexedEventCodec) Decode(data []byte) (interface{}, error) {
	version := data[0]
	if version != V0 {
		return nil, errors.New("unknown version while decoding message")
	}

	iEvent := IndexedEvent{}

	// Height starts after version, Offset starts after Height
	// Version is 1 byte, Height is 8 bytes, Offset is 2 bytes
	h := binary.LittleEndian.Uint64(data[1:])
	o := binary.LittleEndian.Uint16(data[9:])
	iEvent.Height = int64(h)
	iEvent.Offset = int16(o)

	event := &types.Event{}
	// Event data starts at 11, after the version, height, and offset
	if err := event.Unmarshal(data[11:]); err != nil {
		return nil, fmt.Errorf("error unmarshaling event: %v", err)
	}
	iEvent.Event = event

	return iEvent, nil
}
