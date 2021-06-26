package timeseries_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/internal/fetch/record"
	"gitlab.com/thorchain/midgard/internal/timeseries"
)

// TestPools ensures new pools are visible immediately.
func TestPools(t *testing.T) {
	timeseries.SetLastTimeForTest(testdb.StrToSec("2020-09-30 23:00:00"))

	testdb.SetupTestDB(t)
	testdb.MustExec(t, "DELETE FROM stake_events")

	newAsset := "BTC.RUNE-4242"
	record.Recorder.OnStake(&record.Stake{
		AddBase: record.AddBase{
			Pool:       []byte(newAsset),
			AssetTx:    []byte("EUR"),
			AssetChain: []byte("EU"),
			AssetAddr:  []byte("assetAddr"),
			RuneTx:     []byte("123"),
			RuneChain:  []byte("THOR"),
			RuneAddr:   []byte("home"),
			RuneE8:     42,
		},
		StakeUnits: 1,
	}, new(record.Metadata))

	// verify
	got, err := timeseries.PoolsWithDeposit(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []string{newAsset}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %q", got)
		t.Errorf("want %q", want)
	}
}

func TestGetSinglePoolSynthUnits(t *testing.T) {
	synthUnits := timeseries.GetSinglePoolSynthUnits(context.Background(), 1878459169, 1909971564, 35168185551)
	require.Equal(t, int64(36368256684), synthUnits)
}

// TODO(acsaba): have tests to check that these functions don't fail on production data.
// - PoolStatus
// - StatusPerNode
// - NodesSecpAndEd
