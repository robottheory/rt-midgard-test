package resolvers

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"sort"
	"time"

	e "github.com/pkg/errors"
	"gitlab.com/thorchain/midgard/internal/common"
	"gitlab.com/thorchain/midgard/internal/graphql/generated"
	"gitlab.com/thorchain/midgard/internal/graphql/models"
	"gitlab.com/thorchain/midgard/internal/timeseries/stat"
)

func (r *queryResolver) Pool(ctx context.Context, poolID string) (*models.Pool, error) {
	asset, err := common.NewAsset(poolID)
	if err != nil {
		return nil, e.Wrap(err, "failed to create new asset")
	}

	status, err := stat.PoolStatusLookup(poolID)

	stake, err := stat.PoolStakesLookup(poolID, stat.Since(time.Time{}))
	if err != nil {
		return nil, e.Wrap(err, "failed to lookup pool stakes")
	}

	poolDetails, err := stat.GetPoolDetails(poolID, stat.Since(time.Time{}))
	if err != nil {
		return nil, e.Wrap(err, "failed to get pool details")
	}

	return &models.Pool{
		Asset:  asset.String(),
		Status: status, 
		//Price:          // uint64(pool.SellVolume),
		AssetStakedTotal: uint64(stake.AssetE8Total), 
		RuneStakedTotal:  uint64(stake.RuneE8Total),
		PoolStakedTotal:  uint64(stake.UnitsTotal),
		AssetDepth:       uint64(poolDetails.AssetDepth),
		RuneDepth:        uint64(poolDetails.RuneDepth),
		PoolDepth:        uint64(poolDetails.PoolDepth),
		// 	PoolUnits:        uint64(pool.Units),
		CurrentAssetROI: poolDetails.AssetROI,
		CurrentRuneROI:  poolDetails.RuneROI,
	}, nil
}

func (r *queryResolver) Pools(ctx context.Context, orderBy *models.PoolOrderAttribute, limit *int) ([]*models.Pool, error) {
	pools, err := stat.PoolsLookup()
	if err != nil {
		return nil, err
	}

	l := len(pools)
	res := make([]*models.Pool, l)
	for i := 0; i < l; i++ {
		poolID := pools[i]
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
