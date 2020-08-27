package resolvers

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"sort"

	e "github.com/pkg/errors"
	"gitlab.com/thorchain/midgard/internal/common"
	"gitlab.com/thorchain/midgard/internal/graphql/generated"
	"gitlab.com/thorchain/midgard/internal/graphql/models"
)

func (r *queryResolver) Pool(ctx context.Context, poolID string) (*models.Pool, error) {
	asset, err := common.NewAsset(poolID)
	if err != nil {
		return nil, e.Wrap(err, "failed to create new asset")
	}

	pool, err := r.uc.GetPoolDetails(asset)
	if err != nil {
		return nil, err
	}

	return &models.Pool{
		Asset:            asset.String(),
		Status:           pool.Status.String(),
		Price:            uint64(pool.SellVolume),
		AssetStakedTotal: uint64(pool.AssetStaked),
		RuneStakedTotal:  uint64(pool.RuneStaked),
		PoolStakedTotal:  pool.PoolStakedTotal,
		AssetDepth:       uint64(pool.AssetDepth),
		RuneDepth:        uint64(pool.RuneDepth),
		PoolDepth:        pool.PoolDepth,
		PoolUnits:        uint64(pool.Units),
		CurrentAssetROI:  pool.AssetROI,
		CurrentRuneROI:   pool.RuneROI,
	}, nil
}

func (r *queryResolver) Pools(ctx context.Context, orderBy *models.PoolOrderAttribute, limit *int) ([]*models.Pool, error) {
	pools, err := r.uc.GetPools()
	if err != nil {
		return nil, err
	}

	l := len(pools)
	res := make([]*models.Pool, l)
	for i := 0; i < l; i++ {
		poolID := pools[i].String()
		res[i], err = r.Pool(ctx, poolID)
		if err != nil {
			return nil, err
		}
	}

	if orderBy != nil {
		switch *orderBy {
		case "DEPTH":
			sort.Slice(res, func(i, j int) bool {
				return res[i].AssetDepth > res[j].AssetDepth
			})
		case "VOLUME":
			sort.Slice(res, func(i, j int) bool {
				return res[i].PoolDepth > res[j].PoolDepth
			})
		}

	}
	if limit != nil && *limit < l {
		l = *limit
	}
	return res[:l], nil
}

func (r *queryResolver) PoolHistory(ctx context.Context, from *int, until *int, interval *models.Interval) (*models.PoolHistory, error) {
	return nil, e.New("not implemented")
}

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type queryResolver struct{ *Resolver }
