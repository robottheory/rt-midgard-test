package websockets_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.com/thorchain/midgard/chain"
	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/internal/websockets"
)

func TestWebsockets(t *testing.T) {
	testdb.InitTest(t)
	channel := make(chan websockets.Payload, 100)
	websockets.TestChannel = &channel
	chain.CreateWebsocketChannel()
	timeseries.SetDepthsForTest([]timeseries.Depth{
		{Pool: "POOLA", AssetDepth: 10, RuneDepth: 20},
	})

	defer websockets.Start(10).MustQuit()

	*chain.WebsocketNotify <- struct{}{}

	select {
	case payload := <-*websockets.TestChannel:
		assert.Equal(t, "POOLA", payload.Asset)
		assert.Equal(t, "2", payload.Price)
	case <-time.After(1000 * time.Millisecond):
		assert.Fail(t, "didn't get websoket reply")
	}

	timeseries.SetDepthsForTest([]timeseries.Depth{
		{Pool: "POOLA", AssetDepth: 40, RuneDepth: 20},
	})
	*chain.WebsocketNotify <- struct{}{}

	select {
	case payload := <-*websockets.TestChannel:
		assert.Equal(t, "POOLA", payload.Asset)
		assert.Equal(t, "0.5", payload.Price)
	case <-time.After(1000 * time.Millisecond):
		assert.Fail(t, "didn't get websoket reply")
	}
}

// TODO(acsaba): just a second test to test that there is no goroutine problem.
func TestWebsocketSecond(t *testing.T) {
	testdb.InitTest(t)
	channel := make(chan websockets.Payload, 100)
	websockets.TestChannel = &channel
	chain.CreateWebsocketChannel()
	timeseries.SetDepthsForTest([]timeseries.Depth{
		{Pool: "POOLA", AssetDepth: 10, RuneDepth: 20},
	})

	defer websockets.Start(10).MustQuit()

	*chain.WebsocketNotify <- struct{}{}

	select {
	case payload := <-*websockets.TestChannel:
		assert.Equal(t, "POOLA", payload.Asset)
		assert.Equal(t, "2", payload.Price)
	case <-time.After(1000 * time.Millisecond):
		assert.Fail(t, "didn't get websoket reply")
	}
}
