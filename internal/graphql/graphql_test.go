package graphql

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/samsarahq/thunder/graphql"
	"gitlab.com/thorchain/midgard/internal/timeseries/stat"
)

const testAsset = "TEST.COIN"

var lastBlockTimestamp = time.Unix(0, 1597635651176263382)

func resetStubs(t *testing.T) {
	lastBlock = func() (height int64, timestamp time.Time, hash []byte) {
		return 1001, lastBlockTimestamp, []byte{1, 3, 3, 7}
	}

	// reject all by default; prevents accidental mock reuse too
	allPoolStakesAddrLookup = func(addr string, w stat.Window) ([]stat.PoolStakes, error) {
		t.Errorf("allPoolStakesAddrLookup invoked with %q, %+v", addr, w)
		return nil, nil
	}
	poolSwapsFromRuneLookup = func(asset string, w stat.Window) (*stat.PoolSwaps, error) {
		t.Errorf("poolSwapsFromRuneLookup invoked with %q, %+v", asset, w)
		return new(stat.PoolSwaps), nil
	}
	poolSwapsFromRuneBucketsLookup = func(pool string, bucketSize time.Duration, w stat.Window) ([]*stat.PoolSwaps, error) {
		t.Errorf("poolSwapsFromRuneBucketsLookup invoked with %q, %s %+v", pool, bucketSize, w)
		return nil, nil
	}
	poolGasLookup = func(asset string, w stat.Window) (*stat.PoolGas, error) {
		t.Errorf("poolGasLookup invoked with %q, %+v", asset, w)
		return new(stat.PoolGas), nil
	}
	poolSwapsToRuneLookup = func(asset string, w stat.Window) (*stat.PoolSwaps, error) {
		t.Errorf("poolSwapsToRuneLookup invoked with %q, %+v", asset, w)
		return new(stat.PoolSwaps), nil
	}
	poolSwapsToRuneBucketsLookup = func(pool string, bucketSize time.Duration, w stat.Window) ([]*stat.PoolSwaps, error) {
		t.Errorf("poolSwapsToRuneBucketsLookup invoked with %q, %s %+v", pool, bucketSize, w)
		return nil, nil
	}
	poolStakesBucketsLookup = func(asset string, bucketSize time.Duration, w stat.Window) ([]stat.PoolStakes, error) {
		t.Errorf("poolStakesBucketsLookup invoked with %q, %s, %+v", asset, bucketSize, w)
		return nil, nil
	}
	poolStakesLookup = func(asset string, w stat.Window) (*stat.PoolStakes, error) {
		t.Errorf("poolStakesLookup invoked with %q, %+v", asset, w)
		return new(stat.PoolStakes), nil
	}
	stakesAddrLookup = func(addr string, w stat.Window) (*stat.Stakes, error) {
		t.Errorf("stakesAddrLookup invoked with %q, %+v", addr, w)
		return new(stat.Stakes), nil
	}
}

func queryServer(t *testing.T, query string) (responseBody []byte) {
	t.Helper()

	reqBuf := bytes.NewBufferString(fmt.Sprintf(`{"query": %q}`, query))
	req := httptest.NewRequest("POST", `/arbitrary/location`, reqBuf)
	req.Header.Set("Content-Type", "application/json")

	resp := httptest.NewRecorder()
	graphql.HTTPHandler(Schema).ServeHTTP(resp, req)
	if resp.Code != 200 {
		t.Fatalf("HTTP %d: %s", resp.Code, resp.Body)
	}

	var buf bytes.Buffer
	err := json.Indent(&buf, resp.Body.Bytes(), "", "\t")
	if err != nil {
		t.Fatal("malformed response:", err)
	}
	return buf.Bytes()
}

func TestPoolBuyStats(t *testing.T) {
	resetStubs(t)

	// mockup
	poolSwapsFromRuneLookup = func(asset string, w stat.Window) (*stat.PoolSwaps, error) {
		if asset != testAsset {
			t.Errorf("lookup for pool %q, want %q", asset, testAsset)
		}
		if !w.Since.IsZero() || !w.Until.Equal(lastBlockTimestamp) {
			t.Errorf("lookup with time constraints %+v, want (0, %s)", w, lastBlockTimestamp)
		}

		return &stat.PoolSwaps{TxCount: 99, AssetE8Total: 1, RuneE8Total: 2}, nil
	}

	got := queryServer(t, `{query: pool(asset: "TEST.COIN") { swapsFromRuneStats { txCount assetE8Total runeE8Total }}}`)
	const want = `{
	"data": {
		"query": {
			"__key": "TEST.COIN",
			"swapsFromRuneStats": {
				"assetE8Total": 1,
				"runeE8Total": 2,
				"txCount": 99
			}
		}
	},
	"errors": null
}
`
	if string(got) != want {
		t.Errorf("got:  %s", got)
		t.Errorf("want: %s", want)
	}
}

func TestPoolGas(t *testing.T) {
	resetStubs(t)

	// mockup
	poolGasLookup = func(asset string, w stat.Window) (*stat.PoolGas, error) {
		if asset != testAsset {
			t.Errorf("lookup for pool %q, want %q", asset, testAsset)
		}
		if !w.Since.IsZero() || !w.Until.Equal(lastBlockTimestamp) {
			t.Errorf("lookup with time constraints %+v, want (0, %s)", w, lastBlockTimestamp)
		}

		return &stat.PoolGas{AssetE8Total: 1, RuneE8Total: 2}, nil
	}

	got := queryServer(t, `{query: pool(asset: "TEST.COIN") { gasStats { assetE8Total runeE8Total }}}`)
	const want = `{
	"data": {
		"query": {
			"__key": "TEST.COIN",
			"gasStats": {
				"assetE8Total": 1,
				"runeE8Total": 2
			}
		}
	},
	"errors": null
}
`
	if string(got) != want {
		t.Errorf("got:  %s", got)
		t.Errorf("want: %s", want)
	}
}
