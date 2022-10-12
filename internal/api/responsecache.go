package api

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"gitlab.com/thorchain/midgard/config"

	"gitlab.com/thorchain/midgard/internal/util/jobs"

	"github.com/julienschmidt/httprouter"
)

const ApiCacheLifetime = time.Minute * 10

type (
	ApiCacheRefreshFunc func(w http.ResponseWriter, r *http.Request, params httprouter.Params)
	apiCache            struct {
		lastUsed        time.Time
		name            string
		responseMutex   sync.RWMutex
		response        cacheResponseWriter
		refreshInterval time.Duration
		lastRefreshed   time.Time
		runnerMutex     sync.Mutex
	}
)

type cacheResponseWriter struct {
	statusCode int
	body       []byte
	header     http.Header
}

func (c *apiCache) expired() bool {
	if c.refreshInterval <= 0 {
		return true
	}
	return c.lastRefreshed.Add(c.refreshInterval).Before(time.Now())
}

func (c *apiCache) expiredTime() float64 {
	if !c.expired() {
		return 0
	}
	refreshTime := c.lastRefreshed.Add(c.refreshInterval)
	return time.Now().Sub(refreshTime).Seconds()
}

func (c *cacheResponseWriter) Header() http.Header {
	return c.header
}

func (c *cacheResponseWriter) Flush() {
	c.body = make([]byte, 0)
	c.header = make(map[string][]string)
}

func (c *cacheResponseWriter) Write(body []byte) (int, error) {
	if c.body == nil {
		c.Flush()
	}
	c.body = append(c.body, body...)
	return len(c.body), nil
}

func (c *cacheResponseWriter) WriteHeader(statusCode int) {
	c.statusCode = statusCode
}

type (
	Cachelifetime int
	apiCacheStore struct {
		sync.RWMutex
		caches            []*apiCache
		IgnoreCache       Cachelifetime
		ShortTermLifetime Cachelifetime
		MidTermLifetime   Cachelifetime
		LongTermLifetime  Cachelifetime
	}
)

var (
	GlobalApiCacheStore apiCacheStore
	ctx                 context.Context
)

func NewResponseCache(mainContext context.Context, config *config.Config) jobs.NamedFunction {
	GlobalApiCacheStore = apiCacheStore{
		caches:            make([]*apiCache, 0),
		ShortTermLifetime: Cachelifetime(config.ApiCacheConfig.ShortTermLifetime),
		MidTermLifetime:   Cachelifetime(config.ApiCacheConfig.MidTermLifetime),
		LongTermLifetime:  Cachelifetime(config.ApiCacheConfig.LongTermLifetime),
		IgnoreCache:       Cachelifetime(-1),
	}
	ctx = mainContext
	return jobs.Later("ResponseCacheDeleteExpiredJobs", func() {
		for {
			if ctx.Err() != nil {
				CacheLogger.Info("Shutdown background response cache population")
				return
			}
			GlobalApiCacheStore.DeleteExpired()
			jobs.Sleep(ctx, time.Minute)
		}
	})
}

func (store *apiCacheStore) Flush() {
	store.Lock()
	defer store.Unlock()
	store.caches = make([]*apiCache, 0)
}

func (store *apiCacheStore) DeleteExpired() {
	store.Lock()
	defer store.Unlock()
	for i := 0; i < len(store.caches); i++ {
		if store.caches[i].lastUsed.Add(ApiCacheLifetime).Add(time.Second * 30).Before(time.Now()) {
			store.caches = append(store.caches[:i], store.caches[i+1:]...)
			i--
		}
	}
}

func (c *apiCache) Expired() bool {
	return c.lastUsed.Add(ApiCacheLifetime).Before(time.Now())
}

func (store *apiCacheStore) Get(lifetime Cachelifetime, refreshFunc ApiCacheRefreshFunc, w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	api := store.Add(r.URL.Path+"/"+r.URL.RawQuery, lifetime)
	api.responseMutex.Lock()
	defer api.responseMutex.Unlock()
	if api.expired() {
		if api.response.body == nil || len(api.response.body) == 0 || api.expiredTime() >= 2*api.refreshInterval.Seconds() {
			api.response.Flush()
			refreshFunc(&api.response, r, params)
			api.lastRefreshed = time.Now()
		} else {
			go func(api *apiCache, refreshFunc ApiCacheRefreshFunc, r *http.Request, params httprouter.Params) {
				api.runnerMutex.Lock()
				defer api.runnerMutex.Unlock()
				if !api.lastRefreshed.Add(api.refreshInterval).Before(time.Now()) {
					return
				}
				var cWriter cacheResponseWriter
				cWriter.Flush()
				req, _ := http.NewRequestWithContext(ctx, r.Method, r.URL.String(), r.Body)
				refreshFunc(&cWriter, req, params)
				api.responseMutex.Lock()
				defer api.responseMutex.Unlock()
				if IsJSON(string(cWriter.body)) {
					api.lastRefreshed = time.Now()
					api.response = cWriter
				}
			}(api, refreshFunc, r, params)
		}
	}
	if api.response.statusCode != 0 {
		w.WriteHeader(api.response.statusCode)
	}
	for k, v := range api.response.header {
		for _, v1 := range v {
			w.Header().Set(k, v1)
		}
	}
	_, _ = w.Write(api.response.body)
}

func (store *apiCacheStore) Add(name string, lifetime Cachelifetime) *apiCache {
	store.Lock()
	defer store.Unlock()
	for _, c := range store.caches {
		if c.name == name {
			c.lastUsed = time.Now()
			return c
		}
	}
	var api *apiCache
	interval := time.Duration(lifetime) * time.Second
	if api == nil {
		api = &apiCache{
			lastUsed:        time.Now(),
			name:            name,
			responseMutex:   sync.RWMutex{},
			refreshInterval: interval,
			lastRefreshed:   time.Now().Add(-1 * interval).Add(-10 * time.Second),
			response:        cacheResponseWriter{},
			runnerMutex:     sync.Mutex{},
		}
	}
	if int(lifetime) > 0 && api.refreshInterval >= time.Duration(store.ShortTermLifetime)*time.Second {
		store.caches = append(store.caches, api)
	}
	return api
}

func IsJSON(str string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(str), &js) == nil
}
