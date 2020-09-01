package graphql

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"

	"gitlab.com/thorchain/midgard/internal/graphql/models"
)

func (r *queryResolver) Pool(ctx context.Context, poolID string) (*models.Pool, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) Pools(ctx context.Context, orderBy *models.PoolOrderAttribute, limit *int) ([]*models.Pool, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) PoolHistory(ctx context.Context, from *int, until *int, interval *models.Interval, poolID *string) (*models.PoolHistory, error) {
	panic(fmt.Errorf("not implemented"))
}

// Query returns QueryResolver implementation.
func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

type queryResolver struct{ *Resolver }
