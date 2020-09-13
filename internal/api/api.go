// Package api provides the HTTP interface.
package api

import (
	"fmt"
	"io"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/pascaldekloe/metrics"
	thunder "github.com/samsarahq/thunder/graphql"

	"gitlab.com/thorchain/midgard/internal/graphql"
)

// Handler serves the entire API.
var Handler http.Handler

func init() {
	var router = httprouter.New()
	Handler = router

	// apply some navigation pointers
	router.HandleMethodNotAllowed = true
	router.HandleOPTIONS = true
	router.HandlerFunc(http.MethodGet, "/", serveRoot)

	router.HandlerFunc(http.MethodGet, "/metrics", metrics.ServeHTTP)

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

	// version 2 with GraphQL
	router.HandlerFunc(http.MethodGet, "/v2", serveV2)
	router.Handler(http.MethodPost, "/v2/graphql", thunder.HTTPHandler(graphql.Schema))
}

func serveRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain;charset=UTF-8")
	io.WriteString(w, `# THORChain Midgard

Welcome to the HTTP interface.
`)
}

func serveV2(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html;charset=UTF-8")
	fmt.Fprintf(w, `<html>
  <head>
    <title>Midgard GraphiQL — THORChain</title>
    <link href="https://unpkg.com/graphiql/graphiql.min.css" rel="stylesheet" />
  </head>
  <body style="margin: 0;">
    <div id="graphiql" style="height: 100vh;"></div>

    <script crossorigin src="https://unpkg.com/react/umd/react.production.min.js"></script>
    <script crossorigin src="https://unpkg.com/react-dom/umd/react-dom.production.min.js"></script>
    <script crossorigin src="https://unpkg.com/graphiql/graphiql.min.js"></script>

    <script>
      const graphQLFetcher = graphQLParams =>
        fetch('http://%s/v2/graphql', {
          method: 'post',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(graphQLParams),
        })
          .then(response => response.json())
          .catch(() => response.text());
      ReactDOM.render(
        React.createElement(GraphiQL, { fetcher: graphQLFetcher }),
        document.getElementById('graphiql'),
      );
    </script>
  </body>
</html>
`, r.Host)
}

// CORS returns a Handler which applies CORS on h.
func CORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		h.ServeHTTP(w, r)
	})
}
