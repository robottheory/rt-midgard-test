// Package api provides the HTTP interface.
package api

import (
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/julienschmidt/httprouter"
	"github.com/pascaldekloe/metrics"

	"gitlab.com/thorchain/midgard/internal/graphql"
	"gitlab.com/thorchain/midgard/internal/graphql/generated"
)

// Handler serves the entire API.
var Handler http.Handler

func InitHandler(nodeURL string, proxiedWhitelistedEndpoints []string) {
	var router = httprouter.New()
	Handler = router

	// apply some navigation pointers
	router.HandleMethodNotAllowed = true
	router.HandleOPTIONS = true
	router.HandlerFunc(http.MethodGet, "/", serveRoot)

	router.HandlerFunc(http.MethodGet, "/metrics", metrics.ServeHTTP)

	for _, endpoint := range proxiedWhitelistedEndpoints {
		midgardPath := "/v1/thorchain/" + endpoint
		router.HandlerFunc(http.MethodGet, midgardPath, proxiedEndpointHandlerFunc(nodeURL))
	}

	// version 1
	router.HandlerFunc(http.MethodGet, "/v1/assets", serveV1Assets)
	router.HandlerFunc(http.MethodGet, "/v1/health", serveV1Health)
	router.HandlerFunc(http.MethodGet, "/v1/network", serveV1Network)
	router.HandlerFunc(http.MethodGet, "/v1/nodes", serveV1Nodes)
	router.HandlerFunc(http.MethodGet, "/v1/pools", serveV1Pools)
	router.HandlerFunc(http.MethodGet, "/v1/pools/:asset", serveV1PoolsAsset)
	router.HandlerFunc(http.MethodGet, "/v1/stakers", serveV1Stakers)
	router.HandlerFunc(http.MethodGet, "/v1/stakers/:addr", serveV1StakersAddr)
	router.HandlerFunc(http.MethodGet, "/v1/stats", serveV1Stats)
	router.HandlerFunc(http.MethodGet, "/v1/swagger.json", serveV1SwaggerJSON)
	router.HandlerFunc(http.MethodGet, "/v1/tx", serveV1Tx)

	// version 2 with GraphQL
	router.HandlerFunc(http.MethodGet, "/v2/graphql", playground.Handler("Midgard Playground", "/v2"))
	router.Handle(http.MethodPost, "/v2", serverV2())
}

func serverV2() httprouter.Handle {
	h := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &graphql.Resolver{}}))
	return func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
		h.ServeHTTP(w, req)
	}
}

func serveRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain;charset=UTF-8")
	io.WriteString(w, `# THORChain Midgard

Welcome to the HTTP interface.
`)
}

func proxiedEndpointHandlerFunc(nodeURL string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// NOTE(elfedy): url may come with or without leading slash, so make sure we handle this
		// regardless
		// Path is the same without leading v1 (or /v1)
		targetPath := strings.ReplaceAll(r.URL.Path, "v1/thorchain", "")
		targetPath = strings.ReplaceAll(targetPath, "//", "/")
		targetPath = strings.TrimPrefix(targetPath, "/")
		url, err := url.Parse(nodeURL + "/" + targetPath)
		if err != nil {
			http.NotFound(w, r)
		}

		proxy := httputil.NewSingleHostReverseProxy(url)
		proxy.Director = func(req *http.Request) {
			req.Header.Add("X-Forwarded-Host", req.Host)
			req.Header.Add("X-Origin-Host", url.Host)
			req.URL.Scheme = url.Scheme
			req.URL.Host = url.Host
			req.URL.Path = url.Path
		}
		proxy.ServeHTTP(w, r)
	}
}

// CORS returns a Handler which applies CORS on h.
func CORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		h.ServeHTTP(w, r)
	})
}
