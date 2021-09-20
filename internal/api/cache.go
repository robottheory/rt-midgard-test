package api

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/rs/zerolog/log"
	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/util/jobs"
	"gitlab.com/thorchain/midgard/internal/util/miderr"
	"gitlab.com/thorchain/midgard/internal/util/timer"
)

// BackgroundCalculationTotalTimeout is the time a Refresh operation of single http result may take.
const BackgroundCalculationTotalTimeout = time.Second * 60 * 5

// CacheRefreshStartupSleep is the delay at startup before starting the first cache refresh.
const CacheRefreshStartupSleep = time.Second * 2

// CacheRefreshSleepPerRound is the delay between cache recalculations.
const CacheRefreshSleepPerRound = time.Second * 30

const CacheRefreshSleepPerRoundDurringCatchup = time.Second * 60 * 10

type RefreshFunc func(ctx context.Context, w io.Writer) error

type cachedResponse struct {
	buf bytes.Buffer
	err error
}

type cache struct {
	f             RefreshFunc
	name          string
	timer         timer.Timer
	responseMutex sync.RWMutex
	response      cachedResponse
}

type cacheStore struct {
	sync.RWMutex
	caches []*cache
}

var GlobalCacheStore cacheStore

func CreateAndRegisterCache(f RefreshFunc, name string) *cache {
	ret := cache{
		f:        f,
		name:     name,
		timer:    timer.NewTimer("background_" + name),
		response: cachedResponse{err: miderr.InternalErr("Cache not calculated yet")},
	}

	GlobalCacheStore.Lock()
	GlobalCacheStore.caches = append(GlobalCacheStore.caches, &ret)
	GlobalCacheStore.Unlock()

	return &ret
}

func (c *cache) ServeHTTP(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	response := c.getResponse()

	if response.err != nil {
		respError(w, response.err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	// Errors not checked
	_, _ = io.Copy(w, &response.buf)
}

func (c *cache) Refresh(ctx context.Context) {
	response := cachedResponse{}

	stop := c.timer.One()
	response.err = c.f(ctx, &response.buf)
	stop()

	c.responseMutex.Lock()
	c.response = response
	c.responseMutex.Unlock()
}

func (c *cache) getResponse() cachedResponse {
	c.responseMutex.RLock()
	defer c.responseMutex.RUnlock()
	return c.response
}

var CacheLogger = log.With().Str("module", "cache").Logger()

func (cs *cacheStore) RefreshAll(ctx context.Context) {
	cs.RLock()
	caches := cs.caches
	cs.RUnlock()

	for _, cache := range caches {
		ctx2, cancel := context.WithTimeout(ctx, BackgroundCalculationTotalTimeout)
		CacheLogger.Info().Str("cache", cache.name).Msg("Refreshing cache")
		start := timer.MilliCounter()
		cache.Refresh(ctx2)
		CacheLogger.Info().Str("cache", cache.name).Float32("duration", start.SecondsElapsed()).Msg(
			"Refreshed cache.")

		cancel()
		if ctx.Err() != nil {
			// Cancelled
			return
		}
	}
}

func (cs *cacheStore) StartBackgroundRefresh(ctx context.Context) *jobs.Job {
	// TODO(huginn): remove after logging overhaul
	// Reinitialize the logger, so we use the same format as the main logger
	CacheLogger = log.With().Str("module", "cache").Logger()
	// TODO(muninn): add more logs once we have log levels
	ret := jobs.Start("CacheRefresh", func() {
		jobs.Sleep(ctx, CacheRefreshStartupSleep)
		CacheLogger.Info().Msgf("Starting background cache population")
		for {
			if ctx.Err() != nil {
				CacheLogger.Info().Msgf("Shutdown background cache population")
				return
			}
			cs.RefreshAll(ctx)
			sleepTime := CacheRefreshSleepPerRound
			if !db.InSync() {
				sleepTime = CacheRefreshSleepPerRoundDurringCatchup
			}
			jobs.Sleep(ctx, sleepTime)
		}
	})
	return &ret
}
