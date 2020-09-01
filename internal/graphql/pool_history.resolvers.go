package graphql

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"

	"gitlab.com/thorchain/midgard/internal/graphql/models"
)

func (r *poolHistoryResolver) Swaps(ctx context.Context, obj *models.PoolHistory) (*models.PoolSwaps, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *poolHistoryResolver) Fees(ctx context.Context, obj *models.PoolHistory) (*models.PoolFees, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *poolHistoryResolver) Slippage(ctx context.Context, obj *models.PoolHistory) (*models.PoolSlippage, error) {
	panic(fmt.Errorf("not implemented"))
}

// PoolHistory returns PoolHistoryResolver implementation.
func (r *Resolver) PoolHistory() PoolHistoryResolver { return &poolHistoryResolver{r} }

type poolHistoryResolver struct{ *Resolver }
