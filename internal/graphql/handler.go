package graphql

import (
	"github.com/99designs/gqlgen/graphql/handler"
	"gitlab.com/thorchain/midgard/internal/usecase"

	"gitlab.com/thorchain/midgard/internal/graphql/generated"
	"gitlab.com/thorchain/midgard/internal/graphql/resolvers"
)

func NewHandler(uc *usecase.Usecase) *handler.Server {
	schema := generated.NewExecutableSchema(generated.Config{
		Resolvers:  resolvers.NewResolver(uc),
		Directives: generated.DirectiveRoot{},
		Complexity: generated.ComplexityRoot{},
	})

	return handler.NewDefaultServer(schema)
}
