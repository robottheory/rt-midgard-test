package graphql

import (
	"net/http/httptest"
	"testing"

	"gitlab.com/thorchain/midgard/internal/timeseries/stat"
)

func TestPoolByID(t *testing.T) {
	poolStakesLookup = func(poolID string, w stat.Window) (stat.PoolStakes, error) {
		if poolID != "test-asset" {
			t.Errorf("lookup for pool %q, want test-asset", poolID)
		}
		if !w.Start.IsZero() || !w.End.IsZero() {
			t.Errorf("lookup with time constraints %+v", w)
		}

		return stat.PoolStakes{
			AssetE8Total: 1,
			RuneE8Total:  2,
			UnitsTotal:   3,
		}, nil
	}

	req := httptest.NewRequest("GET", `/?query={pool(poolId:"test-asset"){asset%20poolStakedTotal%20runeStakedTotal%20poolUnits}}`, nil)
	resp := httptest.NewRecorder()
	Server.ServeHTTP(resp, req)
	if resp.Code != 200 {
		t.Fatalf("HTTP %d: %s", resp.Code, resp.Body)
	}

	const want = `{"data":{"pool":{"asset":"test-asset","poolStakedTotal":1,"runeStakedTotal":2,"poolUnits":3}}}`
	if got := resp.Body.String(); got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}
