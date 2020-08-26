// Package api provides the HTTP interface.
package api

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/pascaldekloe/metrics"
)

// Handler serves the entire API.
var Handler http.Handler

func init() {
	var router = httprouter.New()
	Handler = router

	router.HandlerFunc(http.MethodGet, "/metrics", metrics.ServeHTTP)

	router.HandlerFunc(http.MethodGet, "/v1/health", serveV1Health)
	router.HandlerFunc(http.MethodGet, "/v1/nodes", serveV1Nodes)
	router.HandlerFunc(http.MethodGet, "/v1/pools", serveV1Pools)
	router.HandlerFunc(http.MethodGet, "/v1/pools/:asset", serveV1PoolsAsset)
	router.HandlerFunc(http.MethodGet, "/v1/swagger.json", serveV1SwaggerJSON)
}
