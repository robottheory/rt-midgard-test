package graphql

import (
	"context"

	"gitlab.com/thorchain/midgard/internal/graphql/models"
	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/internal/timeseries/stat"
)

// Data Source Stubs
var (
	lastBlock        = timeseries.LastBlock
	poolStakesLookup = stat.PoolStakesLookup
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
	_, timestamp, _ := lastBlock()
	stakes, err := poolStakesLookup(poolID, stat.Window{Until: timestamp})
	if err != nil {
		return nil, err
	}
	return &models.Pool{
		Asset:           poolID,
		PoolStakedTotal: uint64(stakes.AssetE8Total),
		RuneStakedTotal: uint64(stakes.RuneE8Total),
		PoolUnits:       uint64(stakes.StakeUnitsTotal),
	}, nil
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
