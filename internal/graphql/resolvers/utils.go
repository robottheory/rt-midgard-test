package resolvers

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	e "github.com/pkg/errors"
	"gitlab.com/thorchain/midgard/internal/graphql/models"
)

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
