package kafka

import (
	"bytes"
	"errors"
)

var ErrBufferOverflow = errors.New("write exceeded buffer limit")

type CappedBuffer struct {
	*bytes.Buffer
	cap int
}

func NewCappedBuffer(buf []byte, cap int) *CappedBuffer {
	return &CappedBuffer{
		Buffer: bytes.NewBuffer(buf),
		cap:    cap,
	}
}

func (cb *CappedBuffer) Write(p []byte) (n int, err error) {
	if cb.cap > 0 && cb.Len()+len(p) > cb.cap {
		return 0, ErrBufferOverflow
	}
	return cb.Buffer.Write(p)
}
