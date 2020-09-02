package graphql

import (
	"net/http/httptest"
	"net/url"
	"testing"

	"gitlab.com/thorchain/midgard/internal/timeseries/stat"
)

func resetStubs(t *testing.T) {
	// reject all by default; prevents accidental mock reuse too
	poolStakesLookup = func(poolID string, w stat.Window) (stat.PoolStakes, error) {
		t.Errorf("poolStakesLookup invoked with %q, %+v", poolID, w)
		return stat.PoolStakes{}, nil
	}
}

func queryServer(t *testing.T, query string) (responseBody []byte) {
	t.Helper()
	req := httptest.NewRequest("GET", `/arbitrary/location?query=`+url.QueryEscape(query), nil)
	resp := httptest.NewRecorder()
	Server.ServeHTTP(resp, req)
	if resp.Code != 200 {
		t.Fatalf("HTTP %d: %s", resp.Code, resp.Body)
	}
	if got := resp.HeaderMap.Get("Content-Type"); got != "application/json" {
		t.Errorf("got Content-Type %q, want JSON", got)
	}
	return resp.Body.Bytes()
}

func TestPoolByID(t *testing.T) {
	resetStubs(t)

	// mockup
	poolStakesLookup = func(poolID string, w stat.Window) (stat.PoolStakes, error) {
		if poolID != "test-asset" {
			t.Errorf("lookup for pool %q, want test-asset", poolID)
		}
		if !w.Start.IsZero() || !w.End.IsZero() {
			t.Errorf("lookup with time constraints %+v", w)
		}

		return stat.PoolStakes{AssetE8Total: 1, RuneE8Total: 2, UnitsTotal: 3}, nil
	}

	got := queryServer(t, `{ pool(poolId: "test-asset") {asset poolStakedTotal runeStakedTotal poolUnits} }`)
	const want = `{"data":{"pool":{"asset":"test-asset","poolStakedTotal":1,"runeStakedTotal":2,"poolUnits":3}}}`
	if string(got) != want {
		t.Errorf("got  %q", got)
		t.Errorf("want %q", want)
	}
}
