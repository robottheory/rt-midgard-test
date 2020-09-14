package timeseries

import (
	"context"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"testing"
	"time"

	"gitlab.com/thorchain/midgard/event"
)

// TestPools ensures new pools are visible immediately.
func TestPools(t *testing.T) {
	mustSetup(t)

	// snapshot
	offset, err := Pools(context.Background(), time.Time{})
	if err != nil {
		t.Fatal(err)
	}

	// change
	newAsset := fmt.Sprintf("BTC.RUNE-%d", rand.Int())
	EventListener.OnStake(&event.Stake{
		Pool:       []byte(newAsset),
		AssetTx:    []byte("EUR"),
		AssetChain: []byte("EU"),
		RuneTx:     []byte("123"),
		RuneChain:  []byte("THOR"),
		RuneAddr:   []byte("home"),
		RuneE8:     42,
		StakeUnits: 1,
	}, new(event.Metadata))

	// verify
	got, err := Pools(context.Background(), time.Time{})
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

func TestPoolStatus(t *testing.T) {
	mustSetup(t)

	got, err := PoolStatus(context.Background(), "BNB.MATIC-416", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}

func TestStakeAddrs(t *testing.T) {
	mustSetup(t)

	got, err := StakeAddrs(context.Background(), time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}

func TestStatusPerNode(t *testing.T) {
	mustSetup(t)

	got, err := StatusPerNode(context.Background(), time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v", got)
}

func TestNodesSecpAndEd(t *testing.T) {
	mustSetup(t)

	secp, ed, err := NodesSecpAndEd(context.Background(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("got %+v and %+v", secp, ed)
}
