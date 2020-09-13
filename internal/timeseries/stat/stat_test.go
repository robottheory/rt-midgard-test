package stat

import (
	"testing"
	"time"

	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/pascaldekloe/sqltest"

	"gitlab.com/thorchain/midgard/internal/timeseries"
)

func init() {
	sqltest.Setup("pgx", "user=midgard password=password host=localhost port=5432 sslmode=disable dbname=midgard")
}

var testWindow = Window{Since: time.Date(2020, 8, 1, 0, 0, 0, 0, time.UTC), Until: time.Now()}

func testSetup(t *testing.T) {
	// run all in transaction with automated rollbacks
	tx := sqltest.NewTx(t)
	DBQuery = tx.Query
	timeseries.DBQuery = tx.Query
	timeseries.DBExec = tx.Exec
	timeseries.Setup()
}

var GoldenBuckets = []struct {
	Size time.Duration
	Window
	N   int64
	Err string
}{
	{time.Second, Window{}, 0, "bucket size 1s smaller than resolution 5m0s"},
	{time.Hour + time.Second, Window{}, 0, "bucket size 1h0m1s not a multiple of 5m0s"},

	{time.Hour, Window{Since: time.Unix(0, 0), Until: time.Unix(119*60, 0)}, 2, ""},
	{time.Hour, Window{Since: time.Unix(0, 0), Until: time.Unix(120*60, 0)}, 3, ""},
	{time.Hour, Window{Since: time.Unix(0, 0), Until: time.Unix(121*60, 0)}, 3, ""},
	{time.Hour, Window{Since: time.Unix(59*60, 0), Until: time.Unix(119*60, 0)}, 2, ""},
	{time.Hour, Window{Since: time.Unix(60*60, 0), Until: time.Unix(119*60, 0)}, 1, ""},
	{time.Hour, Window{Since: time.Unix(61*60, 0), Until: time.Unix(119*60, 0)}, 1, ""},
}

func TestGoldenBuckets(t *testing.T) {
	for _, gold := range GoldenBuckets {
		n, err := bucketsFor(gold.Size, gold.Window)
		if n != gold.N || err == nil && gold.Err != "" || err != nil && err.Error() != gold.Err {
			t.Errorf("%s for (t+%ds, t+%ds) got [%d & %v] want [%d & %v]", gold.Size, gold.Window.Since.Unix(), gold.Window.Until.Unix(), n, err, gold.N, gold.Err)
		}
	}
}
