package resolvers

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"errors"

	"gitlab.com/thorchain/midgard/internal/graphql/models"
	"gitlab.com/thorchain/midgard/internal/graphql/qlink"
)

func (r *poolHistoryResolver) Swaps(ctx context.Context, obj *models.PoolHistory) (*models.PoolSwaps, error) {
	return nil, errors.New("not implemented")
}

func (r *poolHistoryResolver) Fees(ctx context.Context, obj *models.PoolHistory) (*models.PoolFees, error) {
	return nil, errors.New("not implemented")
}

func (r *poolHistoryResolver) Slippage(ctx context.Context, obj *models.PoolHistory) (*models.PoolSlippage, error) {
	return nil, errors.New("not implemented")
}

// PoolHistory returns qlink.PoolHistoryResolver implementation.
func (r *Resolver) PoolHistory() qlink.PoolHistoryResolver { return &poolHistoryResolver{r} }

type poolHistoryResolver struct{ *Resolver }
