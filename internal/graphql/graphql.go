// Package graphql provides the query interface.
package graphql

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	e "github.com/pkg/errors"
	"gitlab.com/thorchain/midgard/internal/graphql/models"
)

// Compile GraphQL schemas.
//go:generate go run github.com/99designs/gqlgen

// Server is the engine singleton.
var Server = handler.NewDefaultServer(NewExecutableSchema(Config{
	Resolvers:  new(Resolver),
	Directives: DirectiveRoot{},
	Complexity: ComplexityRoot{},
}))

type parentArgs struct {
	poolID   string
	from     int
	until    int
	interval models.Interval
}

func parseArgs(ctx context.Context) (args parentArgs, err error) {
	rctx := graphql.GetResolverContext(ctx)
	for k, v := range rctx.Parent.Args {
		switch k {
		case "poolId":
			args.poolID = *v.(*string)
		case "from":
			args.from = *v.(*int)
		case "until":
			args.until = *v.(*int)
		case "interval":
			args.interval = *v.(*models.Interval)
		default:
			err = e.Errorf("arg in query not defined %s", k)
			return
		}
	}
	return
}
