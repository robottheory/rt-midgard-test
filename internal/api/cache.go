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
	"gitlab.com/thorchain/midgard/internal/util/jobs"
	"gitlab.com/thorchain/midgard/internal/util/miderr"
)

// BackgroundQueryTimeoutSec defines how much time one db query may run while running a
// cache refresh.
// Refresh functions should use this constant to create their own timeouts for inner steps.
const BackgroundQueryTimeoutSec = time.Second * 60

// BackgroundCalculationTotalTimeout is the time a Refresh operation of single http result may take.
const BackgroundCalculationTotalTimeout = time.Second * 60 * 5

// CacheRefreshStartupSleep is the delay at startup before starting the first cache refresh.
const CacheRefreshStartupSleep = time.Second * 2

// CacheRefreshSleepPerRound is the delay between cache recalculations.
const CacheRefreshSleepPerRound = time.Second * 30

type RefreshFunc func(ctx context.Context, w io.Writer) error

type cachedResponse struct {
	buf bytes.Buffer
	err error
}

type cache struct {
	f             RefreshFunc
	name          string
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
	response.err = c.f(ctx, &response.buf)

	c.responseMutex.Lock()
	c.response = response
	c.responseMutex.Unlock()
}

func (c *cache) getResponse() cachedResponse {
	c.responseMutex.RLock()
	defer c.responseMutex.RUnlock()
	return c.response
}

func (cs *cacheStore) RefreshAll(ctx context.Context) {
	cs.RLock()
	caches := cs.caches
	cs.RUnlock()

	for _, cache := range caches {
		ctx2, cancel := context.WithTimeout(ctx, BackgroundCalculationTotalTimeout)
		cache.Refresh(ctx2)
		cancel()
		if ctx.Err() != nil {
			// Cancelled
			return
		}
	}
}

func (cs *cacheStore) StartBackgroundRefresh(ctx context.Context) *jobs.Job {
	ret := jobs.Start("CacheRefresh", func() {
		jobs.Sleep(ctx, CacheRefreshStartupSleep)
		log.Info().Msgf("Starting background cache population")
		for {
			if ctx.Err() != nil {
				log.Info().Msgf("Shutdown background cache population")
				return
			}
			cs.RefreshAll(ctx)
			jobs.Sleep(ctx, CacheRefreshSleepPerRound)
		}
	})
	return &ret
}
