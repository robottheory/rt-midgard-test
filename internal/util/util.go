package util

import (
	"bytes"
	"net/url"
	"strconv"

	"gitlab.com/thorchain/midgard/internal/util/miderr"
)

func IntStr(v int64) string {
	return strconv.FormatInt(v, 10)
}

func ConsumeUrlParam(urlParams *url.Values, key string) (value string) {
	value = urlParams.Get(key)
	urlParams.Del(key)
	return
}

func CheckUrlEmpty(urlParams url.Values) miderr.Err {
	for k := range urlParams {
		return miderr.BadRequestF("Unkown key: %s", k)
	}
	return nil
}

// It's like bytes.ToLower but returns nil for nil.
func ToLowerBytes(b []byte) []byte {
	if b == nil {
		return nil
	}
	return bytes.ToLower(b)
}

type Number interface {
	int64 | float64
}

func Max[T Number](x, y T) T {
	if y < x {
		return x
	} else {
		return y
	}
}
