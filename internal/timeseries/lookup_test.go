package timeseries

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSynthUnitsBigNum(t *testing.T) {
	synthUnits := GetSinglePoolSynthUnits(context.Background(), 1878459169, 1909971564, 35168185551)
	require.Equal(t, int64(36368256684), synthUnits)
}

func TestGetSinglePoolSynthUnits(t *testing.T) {
	// 10% of the pool is owned by synth holders
	synthUnits := GetSinglePoolSynthUnits(context.Background(), 10000, 2000, 90)
	require.Equal(t, int64(10), synthUnits)
}
