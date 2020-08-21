package stat

import (
	"testing"

	_ "github.com/lib/pq"

	"github.com/pascaldekloe/sqltest"
)

func init() {
	sqltest.Setup("postgres", "user=midgard password=password host=localhost port=5432 sslmode=disable dbname=midgard")
}

func TestPoolsLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).Query
	got, err := PoolsLookup()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %q", got)
}
