package graphql

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/tendermint/tendermint/libs/math"
	"gitlab.com/thorchain/midgard/internal/graphql/generated"
	"gitlab.com/thorchain/midgard/internal/graphql/model"
	"gitlab.com/thorchain/midgard/internal/timeseries/stat"
)

func (r *poolResolver) Status(ctx context.Context, obj *model.Pool) (string, error) {
	_, _, timestamp := getAssetAndRuneDepths()
	return getPoolStatus(ctx, obj.Asset, timestamp)
}

func (r *poolResolver) Price(ctx context.Context, obj *model.Pool) (float64, error) {
	assetE8DepthPerPool, runeE8DepthPerPool, _ := getAssetAndRuneDepths()
	assetDepth := assetE8DepthPerPool[obj.Asset]
	runeDepth := runeE8DepthPerPool[obj.Asset]

	if assetDepth == 0 {
		return 0, nil
	}

	priceInRune := big.NewRat(runeDepth, assetDepth)
	f, _ := priceInRune.Float64()
	return f, nil
}

func (r *poolResolver) Units(ctx context.Context, obj *model.Pool) (int64, error) {
	_, _, timestamp := getAssetAndRuneDepths()
	window := stat.Window{Since: time.Unix(0, 0), Until: timestamp}
	stakes, err := poolStakesLookup(ctx, obj.Asset, window)
	if err != nil {
		return 0, err
	}
	unstakes, err := poolUnstakesLookup(ctx, obj.Asset, window)
	if err != nil {
		return 0, err
	}
	return stakes.StakeUnitsTotal - unstakes.StakeUnitsTotal, nil
}

func (r *poolResolver) Stakes(ctx context.Context, obj *model.Pool) (*model.PoolStakes, error) {
	assetE8DepthPerPool, runeE8DepthPerPool, timestamp := getAssetAndRuneDepths()
	window := stat.Window{Since: time.Unix(0, 0), Until: timestamp}
	stakes, err := poolStakesLookup(ctx, obj.Asset, window)
	if err != nil {
		return nil, err
	}
	unstakes, err := poolUnstakesLookup(ctx, obj.Asset, window)
	if err != nil {
		return nil, err
	}

	ps := &model.PoolStakes{
		AssetStaked: stakes.AssetE8Total,
		RuneStaked:  stakes.RuneE8Total - unstakes.RuneE8Total,
	}
	assetDepth := assetE8DepthPerPool[obj.Asset]
	runeDepth := runeE8DepthPerPool[obj.Asset]

	if assetDepth != 0 {
		priceInRune := big.NewRat(runeDepth, assetDepth)
		poolStakedTotal := big.NewRat(stakes.AssetE8Total-unstakes.AssetE8Total, 1)
		poolStakedTotal.Mul(poolStakedTotal, priceInRune)
		poolStakedTotal.Add(poolStakedTotal, big.NewRat(stakes.RuneE8Total-unstakes.RuneE8Total, 1))
		ps.PoolStaked = new(big.Int).Div(poolStakedTotal.Num(), poolStakedTotal.Denom()).Int64()
	}

	return ps, nil
}

func (r *poolResolver) Depth(ctx context.Context, obj *model.Pool) (*model.PoolDepth, error) {
	assetE8DepthPerPool, runeE8DepthPerPool, _ := getAssetAndRuneDepths()
	assetDepth := assetE8DepthPerPool[obj.Asset]
	runeDepth := runeE8DepthPerPool[obj.Asset]
	return &model.PoolDepth{
		AssetDepth: assetDepth,
		RuneDepth:  runeDepth,
		PoolDepth:  2 * runeDepth,
	}, nil
}

func (r *poolResolver) Roi(ctx context.Context, obj *model.Pool) (*model.Roi, error) {
	assetE8DepthPerPool, runeE8DepthPerPool, timestamp := getAssetAndRuneDepths()
	window := stat.Window{Since: time.Unix(0, 0), Until: timestamp}
	stakes, err := poolStakesLookup(ctx, obj.Asset, window)
	if err != nil {
		return nil, err
	}
	unstakes, err := poolUnstakesLookup(ctx, obj.Asset, window)
	if err != nil {
		return nil, err
	}

	var result = &model.Roi{}
	var assetROI, runeROI *big.Rat
	assetDepth := assetE8DepthPerPool[obj.Asset]
	runeDepth := runeE8DepthPerPool[obj.Asset]
	if staked := stakes.AssetE8Total - unstakes.AssetE8Total; staked != 0 {
		assetROI = big.NewRat(assetDepth-staked, staked)
		f, _ := assetROI.Float64()
		result.AssetRoi = f
	}
	if staked := stakes.RuneE8Total - unstakes.RuneE8Total; staked != 0 {
		runeROI = big.NewRat(runeDepth-staked, staked)
		f, _ := runeROI.Float64()
		result.RuneRoi = f
	}
	return result, nil
}

func (r *queryResolver) Pool(ctx context.Context, asset string) (*model.Pool, error) {
	result := &model.Pool{
		Asset: asset,
	}
	return result, nil
}

func (r *queryResolver) Pools(ctx context.Context, limit *int) ([]*model.Pool, error) {
	pools, err := getPools(ctx, time.Time{})
	if err != nil {
		return nil, err
	}

	var result []*model.Pool

	for _, p := range pools {
		result = append(result, &model.Pool{
			Asset: p,
		})
	}

	return result, nil
}

func (r *queryResolver) SwapHistory(ctx context.Context, asset string, from *int64, until *int64, interval *model.Interval) (*model.PoolSwapHistory, error) {
	bucketSize := 24 * time.Hour
	if interval != nil {
		switch *interval {
		case model.IntervalDay:
			bucketSize = 24 * time.Hour
			break
		case model.IntervalWeek:
			bucketSize = 7 * 24 * time.Hour
		case model.IntervalMonth:
			bucketSize = 30 * 24 * time.Hour
			break
		}
	}

	if from != nil && until != nil {
		if *from > *until {
			return nil, fmt.Errorf("from %v cannot be greater than until %v", from, until)
		}
	}
	sinceT := time.Now().Add(-24 * time.Hour)
	if from != nil {
		sinceT = time.Unix(*from, 0)
	}
	untilT := time.Now()
	if until != nil {
		untilT = time.Unix(*until, 0)

		//Update since if only until is provided
		if from == nil {
			sinceT = untilT.Add(-24 * time.Hour)
		}
	}
	durationWindow := stat.Window{Since: sinceT, Until: untilT}

	fromRune, err := poolSwapsFromRuneBucketsLookup(ctx, asset, bucketSize, durationWindow)
	if err != nil {
		return nil, err
	}

	toRune, err := poolSwapsToRuneBucketsLookup(ctx, asset, bucketSize, durationWindow)
	if err != nil {
		return nil, err
	}

	var (
		metaFirst time.Time = time.Now()
		metaLast  time.Time

		MetaToRuneTxCount       int64
		MetaToRuneFeesInRune    int64
		MetaToRuneVolumesInRune int64

		MetaToAssetTxCount       int64
		MetaToAssetFeesInRune    int64
		MetaToAssetVolumesInRune int64

		MetaCombTxCount       int64
		MetaCombFeesInRune    int64
		MetaCombVolumesInRune int64
	)
	result := &model.PoolSwapHistory{
		Intervals: []*model.PoolSwapHistoryBucket{},
	}

	//Looping both as sometimes the length of fromRune and toRune are different
	for i := 0; i < math.MaxInt(len(fromRune), len(toRune)); i++ {
		first := time.Now()
		var last time.Time
		ps := model.PoolSwapHistoryBucket{}

		var combTxCount int64
		var combFeesInRune int64
		var combVolumesInRune int64

		if i < len(fromRune) {
			fr := fromRune[i]

			//Setting first and last timestamp
			if fr.First.Before(first) {
				first = fr.First
			}
			if fr.Last.After(last) {
				last = fr.Last
			}

			ps.ToAsset = &model.SwapStats{
				Count:        &fr.TxCount,
				FeesInRune:   &fr.LiqFeeInRuneE8Total,
				VolumeInRune: &fr.RuneE8Total,
			}

			combTxCount += fr.TxCount
			combFeesInRune += fr.LiqFeeInRuneE8Total
			combVolumesInRune += fr.RuneE8Total

			//Also update to meta
			MetaToAssetTxCount += fr.TxCount
			MetaToAssetFeesInRune = fr.LiqFeeInRuneE8Total
			MetaToAssetVolumesInRune = fr.RuneE8Total

			MetaCombTxCount += fr.TxCount
			MetaCombFeesInRune += fr.LiqFeeInRuneE8Total
			MetaCombVolumesInRune += fr.RuneE8Total
		}

		if i < len(toRune) {
			tr := toRune[i]

			//Setting first and last timestamp
			if tr.First.Before(first) {
				first = tr.First
			}
			if tr.Last.After(last) {
				last = tr.Last
			}

			ps.ToRune = &model.SwapStats{
				Count:        &tr.TxCount,
				FeesInRune:   &tr.LiqFeeInRuneE8Total,
				VolumeInRune: &tr.RuneE8Total,
			}

			combTxCount += tr.TxCount
			combFeesInRune += tr.LiqFeeInRuneE8Total
			combVolumesInRune += tr.RuneE8Total

			//Also update to meta
			MetaToRuneTxCount += tr.TxCount
			MetaToRuneFeesInRune = tr.LiqFeeInRuneE8Total
			MetaToRuneVolumesInRune = tr.RuneE8Total

			MetaCombTxCount += tr.TxCount
			MetaCombFeesInRune += tr.LiqFeeInRuneE8Total
			MetaCombVolumesInRune += tr.RuneE8Total
		}

		ps.Combined = &model.SwapStats{
			Count:        &combTxCount,
			FeesInRune:   &combFeesInRune,
			VolumeInRune: &combVolumesInRune,
		}

		firstUnix := first.Unix()
		lastUnix := last.Unix()
		ps.First = &firstUnix
		ps.Last = &lastUnix

		//Setting first and last for overall meta
		if first.Before(metaFirst) {
			metaFirst = first
		}
		if last.After(metaLast) {
			metaLast = last
		}

		metaFirstUnix := metaFirst.Unix()
		metaLastUnix := metaLast.Unix()

		result.Meta = &model.PoolSwapHistoryBucket{
			First: &metaFirstUnix,
			Last:  &metaLastUnix,
			ToRune: &model.SwapStats{
				Count:        &MetaToRuneTxCount,
				FeesInRune:   &MetaToRuneFeesInRune,
				VolumeInRune: &MetaToRuneVolumesInRune,
			},
			ToAsset: &model.SwapStats{
				Count:        &MetaToAssetTxCount,
				FeesInRune:   &MetaToAssetFeesInRune,
				VolumeInRune: &MetaToAssetVolumesInRune,
			},
			Combined: &model.SwapStats{
				Count:        &MetaCombTxCount,
				FeesInRune:   &MetaCombFeesInRune,
				VolumeInRune: &MetaCombVolumesInRune,
			},
		}

		result.Intervals = append(result.Intervals, &ps)
	}

	return result, nil
}

func (r *queryResolver) StakeHistory(ctx context.Context, asset string, from *int64, until *int64, interval *model.Interval) (*model.PoolStakeHistory, error) {
	bucketSize := 24 * time.Hour
	if interval != nil {
		switch *interval {
		case model.IntervalDay:
			bucketSize = 24 * time.Hour
			break
		case model.IntervalWeek:
			bucketSize = 7 * 24 * time.Hour
		case model.IntervalMonth:
			bucketSize = 30 * 24 * time.Hour
			break
		}
	}

	if from != nil && until != nil {
		if *from > *until {
			return nil, fmt.Errorf("from %v cannot be greater than until %v", from, until)
		}
	}
	sinceT := time.Now().Add(-24 * time.Hour)
	if from != nil {
		sinceT = time.Unix(*from, 0)
	}
	untilT := time.Now()
	if until != nil {
		untilT = time.Unix(*until, 0)

		//Update since if only until is provided
		if from == nil {
			sinceT = untilT.Add(-24 * time.Hour)
		}
	}
	durationWindow := stat.Window{Since: sinceT, Until: untilT}

	stakesArr, err := poolStakesBucketsLookup(ctx, asset, bucketSize, durationWindow)
	if err != nil {
		return nil, err
	}
	var intervals []*model.PoolStakeHistoryBucket

	for _, s := range stakesArr {
		first := s.First.Unix()
		last := s.Last.Unix()
		ps := model.PoolStakeHistoryBucket{
			First:         &first,
			Last:          &last,
			Count:         &s.TxCount,
			VolumeInRune:  &s.RuneE8Total,
			VolumeInAsset: &s.AssetE8Total,
			Units:         &s.StakeUnitsTotal,
		}

		intervals = append(intervals, &ps)
	}

	result := &model.PoolStakeHistory{
		Intervals: intervals,
	}

	return result, nil
}

// Pool returns generated.PoolResolver implementation.
func (r *Resolver) Pool() generated.PoolResolver { return &poolResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type poolResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
