package stat

import (
	"context"
	"testing"
	"time"

	"github.com/pascaldekloe/sqltest"
)

func TestSwapsFromRuneLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).QueryContext
	got, err := SwapsFromRuneLookup(context.Background(), testWindow)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}

func TestSwapsToRuneLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).QueryContext
	got, err := SwapsToRuneLookup(context.Background(), testWindow)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}

func TestPoolSwapsFromRuneLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).QueryContext
	got, err := PoolSwapsFromRuneLookup(context.Background(), "BNB.MATIC-416", testWindow)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}

func TestPoolSwapsToRuneLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).QueryContext
	got, err := PoolSwapsToRuneLookup(context.Background(), "BNB.MATIC-416", testWindow)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}

func TestPoolSwapsFromRuneBucketsLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).QueryContext
	got, err := PoolSwapsFromRuneBucketsLookup(context.Background(), "BNB.MATIC-416", 24*time.Hour, testWindow)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %d buckets", len(got))
}

func TestPoolSwapsToRuneBucketsLookup(t *testing.T) {
	DBQuery = sqltest.NewTx(t).QueryContext
	got, err := PoolSwapsToRuneBucketsLookup(context.Background(), "BNB.MATIC-416", 24*time.Hour, testWindow)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %d buckets", len(got))
}
