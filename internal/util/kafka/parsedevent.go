package kafka

import (
	"bytes"
	"encoding/gob"
	"gitlab.com/thorchain/midgard/internal/fetch/record"
	"time"
)

// ParsedEvent roughly corresponds to the thorchain tendermint events,
// though they may be modified/expanded
type ParsedEvent struct {
	EventIndex     EventIdx
	BlockTimestamp time.Time

	OriginalPartition int32

	Type  string
	Event interface{}
}

type ParsedEventCodec struct{}

func init() {
	gob.Register(record.Add{})
	gob.Register(record.Errata{})
	gob.Register(record.Fee{})
	gob.Register(record.Gas{})
	gob.Register(record.PoolBalanceChange{})
	gob.Register(record.Rewards{})
	gob.Register(record.Slash{})
	gob.Register(record.Stake{})
	gob.Register(record.Swap{})
	gob.Register(record.Unstake{})
}

func (pc ParsedEventCodec) Encode(value interface{}) (b []byte, e error) {
	//return json.Marshal(value)

	var encodedEvent bytes.Buffer
	enc := gob.NewEncoder(&encodedEvent)

	// Encode the value.
	err := enc.Encode(value)
	if err != nil {
		return nil, err
	}

	return encodedEvent.Bytes(), nil
}

func (pc ParsedEventCodec) Decode(data []byte) (b interface{}, e error) {
	var (
		c   ParsedEvent
		err error
	)
	//
	//err = json.Unmarshal(data, &c)
	//return c, err

	reader := bytes.NewReader(data)
	dec := gob.NewDecoder(reader)

	if err = dec.Decode(&c); err != nil {
		return nil, err
	}

	return c, nil
}

func NewParsedEventFromIndexedEvent(event IndexedEvent) ParsedEvent {
	p := ParsedEvent{}
	p.EventIndex.Height = event.Height
	p.EventIndex.Offset = event.Offset
	p.Type = event.Event.Type
	p.Event = event.Event
	p.BlockTimestamp = event.BlockTimestamp
	return p
}
