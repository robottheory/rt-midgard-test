package stat

import (
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/pascaldekloe/sqltest"

	"gitlab.com/thorchain/midgard/event"
	"gitlab.com/thorchain/midgard/internal/timeseries"
)

func init() {
	sqltest.Setup("postgres", "user=midgard password=password host=localhost port=5432 sslmode=disable dbname=midgard")
}

var testWindow = Window{Since: time.Unix(0, 0), Until: time.Now()}

func testSetup(t *testing.T) {
	// run all in transaction with automated rollbacks
	tx := sqltest.NewTx(t)
	DBQuery = tx.Query
	timeseries.DBQuery = tx.Query
	timeseries.DBExec = tx.Exec
	timeseries.Setup()
}

// TestPoolsLookupNone ensures new pools are visible immediately.
func TestPoolsLookupAdd(t *testing.T) {
	testSetup(t)
	// snapshot
	offset, err := PoolsLookup()
	if err != nil {
		t.Fatal(err)
	}

	// change
	newAsset := fmt.Sprintf("BTC.RUNE-%d", rand.Int())
	timeseries.EventListener.OnStake(&event.Stake{Pool: []byte(newAsset)}, new(event.Metadata))

	// verify
	got, err := PoolsLookup()
	if err != nil {
		t.Fatal(err)
	}
	want := append(offset, newAsset)
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %q", got)
		t.Errorf("want %q", want)
	}
}

// TestPoolsLookupNone ensures no errors on an empty database.
func TestPoolsLookupNone(t *testing.T) {
	testSetup(t)

	// whipeout
	_, err := timeseries.DBExec("DELETE FROM stake_events")
	if err != nil {
		t.Fatal(err)
	}

	// verify
	got, err := PoolsLookup()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("got %q, want none", got)
	}
}
