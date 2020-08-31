// Compile GraphQL schemas.
//go:generate go run github.com/99designs/gqlgen

package graphql

import (
	"github.com/99designs/gqlgen/graphql/handler"

	"gitlab.com/thorchain/midgard/internal/graphql/qlink"
	"gitlab.com/thorchain/midgard/internal/graphql/resolvers"
)

func NewHandler() *handler.Server {
	schema := qlink.NewExecutableSchema(qlink.Config{
		Resolvers:  resolvers.NewResolver(),
		Directives: qlink.DirectiveRoot{},
		Complexity: qlink.ComplexityRoot{},
	})

	return handler.NewDefaultServer(schema)
}
