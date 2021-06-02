// +build linux

package websockets_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.com/thorchain/midgard/chain"
	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/internal/util/jobs"
	"gitlab.com/thorchain/midgard/internal/websockets"
)

type devNull struct{}

func (devNull) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func init() {
	websockets.Logger.SetOutput(devNull{})
}

func recieveSome(t *testing.T, count int) []websockets.Payload {
	ret := []websockets.Payload{}
	for i := 0; i < count; i++ {
		select {
		case payload := <-*websockets.TestChannel:
			ret = append(ret, payload)
		case <-time.After(1000 * time.Millisecond):
			require.Fail(t, "didn't get websoket reply")
		}
	}
	return ret
}

func initTest(t *testing.T) {
	testdb.InitTest(t)
	channel := make(chan websockets.Payload, 100)
	websockets.TestChannel = &channel
	chain.CreateWebsocketChannel()
}

func TestWebsockets(t *testing.T) {
	initTest(t)
	timeseries.SetDepthsForTest([]timeseries.Depth{
		{Pool: "POOLA", AssetDepth: 10, RuneDepth: 20},
	})

	job := jobs.StartForTests(func(ctx context.Context) {
		websockets.Start(ctx, 10)
	})
	defer job.Quit()

	*chain.WebsocketNotify <- struct{}{}

	response := recieveSome(t, 1)
	require.Equal(t, "POOLA", response[0].Asset)
	require.Equal(t, "2", response[0].Price)

	timeseries.SetDepthsForTest([]timeseries.Depth{
		{Pool: "POOLA", AssetDepth: 40, RuneDepth: 20},
	})
	*chain.WebsocketNotify <- struct{}{}

	response = recieveSome(t, 1)
	require.Equal(t, "POOLA", response[0].Asset)
	require.Equal(t, "0.5", response[0].Price)
}

func TestWebsocketTwoPools(t *testing.T) {
	initTest(t)
	timeseries.SetDepthsForTest([]timeseries.Depth{
		{Pool: "POOLA", AssetDepth: 10, RuneDepth: 20},
		{Pool: "POOLB", AssetDepth: 10, RuneDepth: 100},
	})

	job := jobs.StartForTests(func(ctx context.Context) {
		websockets.Start(ctx, 10)
	})
	defer job.Quit()

	*chain.WebsocketNotify <- struct{}{}

	response := recieveSome(t, 2)
	require.Contains(t, response, websockets.Payload{"2", "POOLA"})
	require.Contains(t, response, websockets.Payload{"10", "POOLB"})
}
