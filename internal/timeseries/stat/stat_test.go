package stat

import (
	"testing"
	"time"

	"gitlab.com/thorchain/midgard/internal/db"
)

var GoldenBuckets = []struct {
	Size time.Duration
	db.Window
	N   int64
	Err string
}{
	{time.Second, db.Window{}, 0, "bucket size 1s smaller than resolution 5m0s"},
	{time.Hour + time.Second, db.Window{}, 0, "bucket size 1h0m1s not a multiple of 5m0s"},

	{time.Hour, db.Window{From: time.Unix(0, 0), Until: time.Unix(119*60, 0)}, 2, ""},
	{time.Hour, db.Window{From: time.Unix(0, 0), Until: time.Unix(120*60, 0)}, 3, ""},
	{time.Hour, db.Window{From: time.Unix(0, 0), Until: time.Unix(121*60, 0)}, 3, ""},
	{time.Hour, db.Window{From: time.Unix(59*60, 0), Until: time.Unix(119*60, 0)}, 2, ""},
	{time.Hour, db.Window{From: time.Unix(60*60, 0), Until: time.Unix(119*60, 0)}, 1, ""},
	{time.Hour, db.Window{From: time.Unix(61*60, 0), Until: time.Unix(119*60, 0)}, 1, ""},
}

func TestGoldenBuckets(t *testing.T) {
	for _, gold := range GoldenBuckets {
		n, err := bucketsFor(gold.Size, gold.Window)
		if n != gold.N || err == nil && gold.Err != "" || err != nil && err.Error() != gold.Err {
			t.Errorf("%s for (t+%ds, t+%ds) got [%d & %v] want [%d & %v]", gold.Size, gold.Window.From.Unix(), gold.Window.Until.Unix(), n, err, gold.N, gold.Err)
		}
	}
}
