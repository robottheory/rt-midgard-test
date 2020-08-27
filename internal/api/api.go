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

	router.HandlerFunc(http.MethodGet, "/v1/pools", servePools)
	router.HandlerFunc(http.MethodGet, "/v1/pools/:asset", servePoolsAsset)

	graphqlHandler := graphql.NewHandler(nil)
	router.Handler(http.MethodGet, graphqlHandler)
}
