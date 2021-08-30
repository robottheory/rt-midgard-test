package api

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

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

var GlobalApiCacheStore apiCacheStore

func init() {
	GlobalApiCacheStore = apiCacheStore{
		caches: make([]*apiCache, 0),
	}

	// Remove old cache from memory
	go func() {
		for {
			GlobalApiCacheStore.DeleteExpired()
			time.Sleep(time.Minute)
		}
	}()
}

func (store *apiCacheStore) Flush() {
	store.Lock()
	defer store.Unlock()
	store.caches = make([]*apiCache, 0)
}

func (store *apiCacheStore) DeleteExpired() {
	store.Lock()
	defer store.Unlock()
	for i, c := range store.caches {
		if c.lastUsed.Add(ApiCacheLifetime).Add(time.Second * 30).Before(time.Now()) {
			store.caches = append(store.caches[:i], store.caches[i+1:]...)
		}
	}
}

func (c *apiCache) Expired() bool {
	return c.lastUsed.Add(ApiCacheLifetime).Before(time.Now())
}

func (store *apiCacheStore) Get(refreshInterval time.Duration, refreshFunc ApiCacheRefreshFunc, w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	fmt.Println(r.URL.Query().Encode())
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
				var cWriter cacheResponseWriter
				cWriter.Flush()
				fmt.Println(r.URL.String())
				req, _ := http.NewRequestWithContext(context.Background(), r.Method, r.URL.String(), r.Body)
				refreshFunc(&cWriter, req, params)
				api.responseMutex.Lock()
				defer api.responseMutex.Unlock()
				api.lastRefreshed = time.Now()
				api.response = cWriter
			}(api, refreshFunc, r, params)
		}
	}
	_, _ = w.Write(api.response.body)
	w.WriteHeader(api.response.statusCode)
	for k, v := range api.response.header {
		for _, v1 := range v {
			w.Header().Set(k, v1)
		}
	}
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
		}
	}
	store.caches = append(store.caches, api)
	return api
}
