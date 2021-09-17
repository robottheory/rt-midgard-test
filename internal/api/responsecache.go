package api

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

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

type apiCacheStore struct {
	sync.RWMutex
	caches []*apiCache
}

var (
	GlobalApiCacheStore apiCacheStore
	ctx                 context.Context
)

func NewResponseCache(mainContext context.Context) *jobs.Job {
	GlobalApiCacheStore = apiCacheStore{
		caches: make([]*apiCache, 0),
	}
	ctx = mainContext
	job := jobs.Start("ResponseCacheDeleteExpiredJobs", func() {
		for {
			if ctx.Err() != nil {
				CacheLogger.Info().Msgf("Shutdown background response cache population")
				return
			}
			GlobalApiCacheStore.DeleteExpired()
			jobs.Sleep(ctx, time.Minute)
		}
	})
	return &job
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

func (store *apiCacheStore) Get(refreshInterval time.Duration, refreshFunc ApiCacheRefreshFunc, w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	api := store.Add(r.URL.Path+"/"+r.URL.RawQuery, refreshInterval)
	api.responseMutex.Lock()
	defer api.responseMutex.Unlock()
	if api.lastRefreshed.Add(api.refreshInterval).Before(time.Now()) {
		if api.response.body == nil || len(api.response.body) == 0 {
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

func (store *apiCacheStore) Add(name string, refreshInterval time.Duration) *apiCache {
	store.Lock()
	defer store.Unlock()
	for _, c := range store.caches {
		if c.name == name {
			c.lastUsed = time.Now()
			return c
		}
	}
	var api *apiCache
	if api == nil {
		api = &apiCache{
			lastUsed:        time.Now(),
			name:            name,
			responseMutex:   sync.RWMutex{},
			refreshInterval: refreshInterval,
			lastRefreshed:   time.Now().Add(-10 * refreshInterval),
			response:        cacheResponseWriter{},
			runnerMutex:     sync.Mutex{},
		}
	}
	if api.refreshInterval > time.Second*5 {
		store.caches = append(store.caches, api)
	}
	return api
}

func IsJSON(str string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(str), &js) == nil
}
