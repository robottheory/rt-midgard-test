package stat

import (
	"testing"
	"time"

	"github.com/pascaldekloe/sqltest"
)

func TestPoolBuySwapsLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).Query
	got, err := PoolBuySwapsLookup("BNB.MATIC-416", testWindow)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}

func TestPoolSellSwapsLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).Query
	got, err := PoolSellSwapsLookup("BNB.MATIC-416", testWindow)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}

func TestPoolBuySwapsBucketsLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).Query
	got, err := PoolBuySwapsBucketsLookup("BNB.MATIC-416", time.Hour, testWindow)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %d buckets", len(got))
}

func TestPoolSellSwapsBucketsLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).Query
	got, err := PoolSellSwapsBucketsLookup("BNB.MATIC-416", time.Hour, testWindow)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %d buckets", len(got))
}
