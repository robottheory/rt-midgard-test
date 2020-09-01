package models

import (
	"fmt"
	"io"
	"strconv"

	"github.com/99designs/gqlgen/graphql"
)

type Pool struct {
	Asset            string
	Status           string
	Price            float64
	AssetStakedTotal uint64
	RuneStakedTotal  uint64
	PoolStakedTotal  uint64
	AssetDepth       uint64
	RuneDepth        uint64
	PoolDepth        uint64
	PoolUnits        uint64
	CurrentAssetROI  float64
	CurrentRuneROI   float64
}

// MarshalAssetAmount implements a uint64 value
func MarshalAssetAmount(t uint64) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		_, err := io.WriteString(w, strconv.FormatUint(t, 10))
		if err != nil {
			return
		}
	})
}

// UnmarshalAssetAmount implements a uint64 value
func UnmarshalAssetAmount(v interface{}) (uint64, error) {
	switch t := v.(type) {
	case string:
		return strconv.ParseUint(t, 10, 64)
	case int:
		return uint64(t), nil
	case int64:
		return uint64(t), nil
	case float64:
		return uint64(t), nil
	default:
		return 0, fmt.Errorf("unknown type: %T", t)
	}
}
