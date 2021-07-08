package timeseries

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetSinglePoolSynthUnits(t *testing.T) {
	synthUnits := GetSinglePoolSynthUnits(context.Background(), 1878459169, 1909971564, 35168185551)
	require.Equal(t, int64(36368256684), synthUnits)
}
