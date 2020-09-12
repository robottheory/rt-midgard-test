package stat

import (
	"testing"
	"time"

	"github.com/pascaldekloe/sqltest"
)

func TestSwapsFromRuneLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).Query
	got, err := SwapsFromRuneLookup(testWindow)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}

func TestSwapsToRuneLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).Query
	got, err := SwapsToRuneLookup(testWindow)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}

func TestPoolSwapsFromRuneLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).Query
	got, err := PoolSwapsFromRuneLookup("BNB.MATIC-416", testWindow)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}

func TestPoolSwapsToRuneLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).Query
	got, err := PoolSwapsToRuneLookup("BNB.MATIC-416", testWindow)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}

func TestPoolSwapsFromRuneBucketsLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).Query
	got, err := PoolSwapsFromRuneBucketsLookup("BNB.MATIC-416", 24*time.Hour, testWindow)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %d buckets", len(got))
}

func TestPoolSwapsToRuneBucketsLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).Query
	got, err := PoolSwapsToRuneBucketsLookup("BNB.MATIC-416", 24*time.Hour, testWindow)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %d buckets", len(got))
}
