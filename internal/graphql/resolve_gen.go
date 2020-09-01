package graphql

// THIS CODE IS A STARTING POINT ONLY. IT WILL NOT BE UPDATED WITH SCHEMA CHANGES.

import (
	"context"

	"gitlab.com/thorchain/midgard/internal/graphql/models"
)

type Resolver struct{}

func (r *poolHistoryResolver) Swaps(ctx context.Context, obj *models.PoolHistory) (*models.PoolSwaps, error) {
	panic("not implemented")
}

func (r *poolHistoryResolver) Fees(ctx context.Context, obj *models.PoolHistory) (*models.PoolFees, error) {
	panic("not implemented")
}

func (r *poolHistoryResolver) Slippage(ctx context.Context, obj *models.PoolHistory) (*models.PoolSlippage, error) {
	panic("not implemented")
}

func (r *queryResolver) Pool(ctx context.Context, poolID string) (*models.Pool, error) {
	panic("not implemented")
}

func (r *queryResolver) Pools(ctx context.Context, orderBy *models.PoolOrderAttribute, limit *int) ([]*models.Pool, error) {
	panic("not implemented")
}

func (r *queryResolver) PoolHistory(ctx context.Context, from *int, until *int, interval *models.Interval, poolID *string) (*models.PoolHistory, error) {
	panic("not implemented")
}

// PoolHistory returns PoolHistoryResolver implementation.
func (r *Resolver) PoolHistory() PoolHistoryResolver { return &poolHistoryResolver{r} }

// Query returns QueryResolver implementation.
func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

type poolHistoryResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
