// Package api provides the HTTP interface.
package api

import (
	"net/http"

	"gitlab.com/thorchain/midgard/internal/graphql"

	"github.com/julienschmidt/httprouter"
	"github.com/pascaldekloe/metrics"
)

// Handler serves the entire API.
var Handler http.Handler

func init() {
	var router = httprouter.New()
	Handler = router

	router.HandlerFunc(http.MethodGet, "/metrics", metrics.ServeHTTP)

	// version 1
	router.HandlerFunc(http.MethodGet, "/v1/health", serveV1Health)
	router.HandlerFunc(http.MethodGet, "/v1/nodes", serveV1Nodes)
	router.HandlerFunc(http.MethodGet, "/v1/pools", serveV1Pools)
	router.HandlerFunc(http.MethodGet, "/v1/pools/:asset", serveV1PoolsAsset)
	router.HandlerFunc(http.MethodGet, "/v1/stakers", serveV1Stakers)
	router.HandlerFunc(http.MethodGet, "/v1/swagger.json", serveV1SwaggerJSON)

	// version 2 with GraphQL
	router.Handler(http.MethodGet, "/v2/graphql", graphql.Server)
	router.Handler(http.MethodPost, "/v2/graphql", graphql.Server)
}

// CORS returns a Handler which applies CORS on h.
func CORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		h.ServeHTTP(w, r)
	})
}
