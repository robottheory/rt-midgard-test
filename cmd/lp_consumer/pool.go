package main

import (
	"encoding/json"
	"fmt"
)

type pool struct {
	AddCount      int64
	WithdrawCount int64

	lastHeight int64
	lastOffset int16
}

func (p *pool) Encode(value interface{}) ([]byte, error) {
	if _, isUser := value.(*pool); !isUser {
		return nil, fmt.Errorf("Codec requires value *pool, got %T", value)
	}
	return json.Marshal(value)
}

func (p *pool) Decode(data []byte) (interface{}, error) {
	var (
		c   pool
		err error
	)
	err = json.Unmarshal(data, &c)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshaling pool: %v", err)
	}
	return &c, nil
}
