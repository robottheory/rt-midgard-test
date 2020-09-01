package resolvers

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"time"

	"gitlab.com/thorchain/midgard/internal/graphql/models"
	"gitlab.com/thorchain/midgard/internal/graphql/qlink"
	"gitlab.com/thorchain/midgard/internal/timeseries/stat"
)

func (r *poolHistoryResolver) Swaps(ctx context.Context, obj *models.PoolHistory) (*models.PoolSwaps, error) {
	args, err := parseArgs(ctx)
	if err != nil {
		return nil, err
	}
	window := stat.Window{
		Start: time.Unix(int64(args.from), 0),
		End:   time.Unix(int64(args.until), 0),
	}

	swaps, err := stat.PoolSwapsLookup(args.poolID, window)
	poolSwap := &models.PoolSwaps{
		TotalCount: uint64(swaps.TxCount),
		BuyCount:   uint64(swaps.AssetE8Total),
		//SellCount :  ,
		//TotalVolume: ,
		BuyVolume: uint64(swaps.LiqFee),
		//SellVolume : ,
	}
	return poolSwap, nil
}

func (r *poolHistoryResolver) Fees(ctx context.Context, obj *models.PoolHistory) (*models.PoolFees, error) {
	args, err := parseArgs(ctx)
	if err != nil {
		return nil, err
	}
	window := stat.Window{
		Start: time.Unix(int64(args.from), 0),
		End:   time.Unix(int64(args.until), 0),
	}

	fees, err := stat.PoolFeesLookup(args.poolID, window)
	poolFees := &models.PoolFees{
		TotalFees: uint64(fees.AssetE8Total),
		// BuyFees      uint64
		// SellFees     uint64
		// MeanFees     uint64
		// MeanBuyFees  uint64
		// MeanSellFees uint64
	}
	return poolFees, nil
}

func (r *poolHistoryResolver) Slippage(ctx context.Context, obj *models.PoolHistory) (*models.PoolSlippage, error) {
	args, err := parseArgs(ctx)
	if err != nil {
		return nil, err
	}
	_ = stat.Window{
		Start: time.Unix(int64(args.from), 0),
		End:   time.Unix(int64(args.until), 0),
	}

	//slippage, err := stat.PoolSlippagesLookup(args.poolID, window)
	poolSlippage := &models.PoolSlippage{
		//TotalFees: uint64(fees.AssetE8Total),
		// BuyFees      uint64
		// SellFees     uint64
		// MeanFees     uint64
		// MeanBuyFees  uint64
		// MeanSellFees uint64
	}
	return poolSlippage, nil
}

// PoolHistory returns qlink.PoolHistoryResolver implementation.
func (r *Resolver) PoolHistory() qlink.PoolHistoryResolver { return &poolHistoryResolver{r} }

type poolHistoryResolver struct{ *Resolver }
