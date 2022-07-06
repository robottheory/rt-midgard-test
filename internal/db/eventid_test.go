package db_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/thorchain/midgard/internal/db"
)

// To run with fuzzing, run:
// go test -v -fuzz=Fuzz -fuzztime=30s ./internal/db/eventid_test.go
func FuzzInvolution(f *testing.F) {
	cases := []int64{2220000000003, 2221000040004, 2227000050005, 2229000000009}
	for _, c := range cases {
		f.Add(c)
	}
	f.Fuzz(func(t *testing.T, eid int64) {
		require.Equal(t, eid, db.ParseEventId(eid).AsBigint())
	})
}
