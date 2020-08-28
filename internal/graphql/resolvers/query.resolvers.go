package resolvers

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"
	"sort"

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
	fmt.Println(asset)

	status, err := stat.PoolStatusLookup(poolID)
	// pool, err := stat.PoolFeesLookup() //r.uc.GetPoolDetails(asset)
	// if err != nil {
	// 	return nil, err
	// }

	price := uint64(2)

	assetTaked := stat.PoolStoakedLookup(poolID)

	return &models.Pool{
		Asset:  asset.String(),
		Status: status, // pool.Status.String(),
		//Price:  price,  // uint64(pool.SellVolume),
		AssetStakedTotal: assetStaked,// uint64(pool.AssetStaked),
		// 	RuneStakedTotal:  // uint64(pool.RuneStaked),
		// 	PoolStakedTotal:  // pool.PoolStakedTotal,
		// 	AssetDepth:       uint64(pool.AssetDepth),
		// 	RuneDepth:        uint64(pool.RuneDepth),
		// 	PoolDepth:        pool.PoolDepth,
		// 	PoolUnits:        uint64(pool.Units),
		// 	CurrentAssetROI:  pool.AssetROI,
		// 	CurrentRuneROI:   pool.RuneROI,
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
