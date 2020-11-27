package graphql

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"time"

	"gitlab.com/thorchain/midgard/internal/timeseries"

	"gitlab.com/thorchain/midgard/chain/notinchain"
	"gitlab.com/thorchain/midgard/internal/graphql/generated"
	"gitlab.com/thorchain/midgard/internal/graphql/model"
	"gitlab.com/thorchain/midgard/internal/timeseries/stat"
)

func (r *poolResolver) Status(ctx context.Context, obj *model.Pool) (string, error) {
	_, _, timestamp := timeseries.AssetAndRuneDepths()
	return getPoolStatus(ctx, obj.Asset, timestamp)
}

func (r *poolResolver) Price(ctx context.Context, obj *model.Pool) (float64, error) {
	assetE8DepthPerPool, runeE8DepthPerPool, _ := timeseries.AssetAndRuneDepths()
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
	_, _, timestamp := timeseries.AssetAndRuneDepths()
	window := stat.Window{From: time.Unix(0, 0), Until: timestamp}
	stakes, err := stat.PoolStakesLookup(ctx, obj.Asset, window)
	if err != nil {
		return 0, err
	}
	unstakes, err := stat.PoolUnstakesLookup(ctx, obj.Asset, window)
	if err != nil {
		return 0, err
	}
	return stakes.StakeUnitsTotal - unstakes.StakeUnitsTotal, nil
}

// TODO(donfrigo) add memoization layer to cache requests
// or find a way to only make the same query once every request
func (r *poolResolver) Volume24h(ctx context.Context, obj *model.Pool) (int64, error) {
	assetE8DepthPerPool, runeE8DepthPerPool, timestamp := timeseries.AssetAndRuneDepths()

	_, assetOk := assetE8DepthPerPool[obj.Asset]
	_, runeOk := runeE8DepthPerPool[obj.Asset]

	// TODO(acsaba): centralize the logic of checking pool existence.
	// TODO(acsaba): don't check pool existence at each graphql field.
	if !assetOk && !runeOk {
		return 0, errors.New("pool not found")
	}
	dailyVolume, err := stat.PoolsTotalVolume(ctx, []string{obj.Asset}, timestamp.Add(-24*time.Hour), timestamp)
	if err != nil {
		return 0, err
	}
	return dailyVolume[obj.Asset], err
}

func (r *poolResolver) PoolApy(ctx context.Context, obj *model.Pool) (float64, error) {
	_, runeE8DepthPerPool, timestamp := timeseries.AssetAndRuneDepths()
	runeDepth := runeE8DepthPerPool[obj.Asset]

	_, ok := runeE8DepthPerPool[obj.Asset]
	if !ok {
		return 0, errors.New("pool not found")
	}

	poolWeeklyRewards, err := timeseries.PoolsTotalIncome(ctx, []string{obj.Asset}, timestamp.Add(-1*time.Hour*24*7), timestamp)
	if err != nil {
		return 0, err
	}
	rewards := poolWeeklyRewards[obj.Asset]

	poolAPY := timeseries.GetPoolAPY(runeDepth, rewards)

	return poolAPY, nil
}

func (r *poolResolver) Stakes(ctx context.Context, obj *model.Pool) (*model.PoolStakes, error) {
	assetE8DepthPerPool, runeE8DepthPerPool, timestamp := timeseries.AssetAndRuneDepths()
	window := stat.Window{From: time.Unix(0, 0), Until: timestamp}
	stakes, err := stat.PoolStakesLookup(ctx, obj.Asset, window)
	if err != nil {
		return nil, err
	}
	unstakes, err := stat.PoolUnstakesLookup(ctx, obj.Asset, window)
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
	assetE8DepthPerPool, runeE8DepthPerPool, _ := timeseries.AssetAndRuneDepths()
	assetDepth := assetE8DepthPerPool[obj.Asset]
	runeDepth := runeE8DepthPerPool[obj.Asset]
	return &model.PoolDepth{
		AssetDepth: assetDepth,
		RuneDepth:  runeDepth,
		PoolDepth:  2 * runeDepth,
	}, nil
}

func (r *queryResolver) Pool(ctx context.Context, asset string) (*model.Pool, error) {
	result := &model.Pool{
		Asset: asset,
	}
	return result, nil
}

func (r *queryResolver) Pools(ctx context.Context, limit *int) ([]*model.Pool, error) {
	pools, err := getPools(ctx)
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

func (r *queryResolver) Stakers(ctx context.Context) ([]*model.Staker, error) {
	addrs, err := timeseries.MemberAddrs(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*model.Staker, len(addrs))
	for i, a := range addrs {
		result[i] = &model.Staker{
			Address: a,
			// TODO(kashif) other fields require subquery.
			// Not implemented yet.
		}
	}

	return result, nil
}
func (r *queryResolver) Staker(ctx context.Context, address string) (*model.Staker, error) {
	pools, err := allPoolStakesAddrLookup(ctx, address, stat.Window{Until: time.Now()})
	if err != nil {
		return nil, err
	}

	var runeE8Total int64
	assets := make([]*string, len(pools))
	for i := range pools {
		assets[i] = &pools[i].Asset
		runeE8Total += pools[i].RuneE8Total
	}

	// TODO(kashif) extra fields aren't supported yet as
	// it is still not available for v1.
	result := &model.Staker{
		PoolsArray:  assets,
		TotalStaked: runeE8Total,
		Address:     address,
	}

	return result, nil
}

func (r *queryResolver) Node(ctx context.Context, address string) (*model.Node, error) {
	node, err := cachedNodeAccountLookup(address)
	if err != nil {
		return nil, err
	}

	result := &model.Node{
		PublicKeys: &model.PublicKeys{
			Secp256k1: node.PublicKeys.Secp256k1,
			Ed25519:   node.PublicKeys.Ed25519,
		},
		Address:          node.NodeAddr,
		Status:           node.Status,
		Bond:             node.Bond,
		RequestedToLeave: node.RequestedToLeave,
		ForcedToLeave:    node.ForcedToLeave,
		LeaveHeight:      node.LeaveHeight,
		IPAddress:        node.IpAddress,
		Version:          node.Version,
		SlashPoints:      node.SlashPoints,
		Jail: &model.JailInfo{
			NodeAddr:      node.Jail.NodeAddr,
			ReleaseHeight: node.Jail.ReleaseHeight,
			Reason:        node.Jail.Reason,
		},
		CurrentAward: node.CurrentAward,
	}

	return result, nil
}

func (r *queryResolver) Nodes(ctx context.Context, status *model.NodeStatus) ([]*model.Node, error) {
	nodes, err := cachedNodeAccountsLookup()
	if err != nil {
		return nil, err
	}

	//Filter by status
	filteredNodes := []*notinchain.NodeAccount{}

	if status != nil {
		for _, n := range nodes {
			if n.Status == strings.ToLower(status.String()) {
				filteredNodes = append(filteredNodes, n)
			}
		}
		nodes = filteredNodes
	}

	result := make([]*model.Node, 0, len(nodes))
	for _, e := range nodes {
		result = append(result, &model.Node{
			PublicKeys: &model.PublicKeys{
				Secp256k1: e.PublicKeys.Secp256k1,
				Ed25519:   e.PublicKeys.Ed25519,
			},
			Address:          e.NodeAddr,
			Status:           e.Status,
			Bond:             e.Bond,
			RequestedToLeave: e.RequestedToLeave,
			ForcedToLeave:    e.ForcedToLeave,
			LeaveHeight:      e.LeaveHeight,
			IPAddress:        e.IpAddress,
			Version:          e.Version,
			SlashPoints:      e.SlashPoints,
			Jail: &model.JailInfo{
				NodeAddr:      e.Jail.NodeAddr,
				ReleaseHeight: e.Jail.ReleaseHeight,
				Reason:        e.Jail.Reason,
			},
			CurrentAward: e.CurrentAward,
		})
	}

	return result, nil
}

// TODO(kashif) This applies to ALL the stuff here. Ideally we
// should have a common service layer to handle all the business logic.
// So v1 and v2 can both call into the same common one.
func (r *queryResolver) Stats(ctx context.Context) (*model.Stats, error) {
	_, runeE8DepthPerPool, timestamp := timeseries.AssetAndRuneDepths()
	window := stat.Window{From: time.Unix(0, 0), Until: timestamp}

	stakes, err := stakesLookup(ctx, window)
	if err != nil {
		return nil, err
	}
	unstakes, err := unstakesLookup(ctx, window)
	if err != nil {
		return nil, err
	}
	swapsFromRune, err := swapsFromRuneLookup(ctx, window)
	if err != nil {
		return nil, err
	}
	swapsToRune, err := swapsToRuneLookup(ctx, window)
	if err != nil {
		return nil, err
	}
	dailySwapsFromRune, err := swapsFromRuneLookup(ctx, stat.Window{From: timestamp.Add(-24 * time.Hour), Until: timestamp})
	if err != nil {
		return nil, err
	}
	dailySwapsToRune, err := swapsToRuneLookup(ctx, stat.Window{From: timestamp.Add(-24 * time.Hour), Until: timestamp})
	if err != nil {
		return nil, err
	}
	monthlySwapsFromRune, err := swapsFromRuneLookup(ctx, stat.Window{From: timestamp.Add(-30 * 24 * time.Hour), Until: timestamp})
	if err != nil {
		return nil, err
	}
	monthlySwapsToRune, err := swapsToRuneLookup(ctx, stat.Window{From: timestamp.Add(-30 * 24 * time.Hour), Until: timestamp})
	if err != nil {
		return nil, err
	}

	var runeDepth int64
	for _, depth := range runeE8DepthPerPool {
		runeDepth += depth
	}

	result := &model.Stats{
		DailyActiveUsers:   dailySwapsFromRune.RuneAddrCount + dailySwapsToRune.RuneAddrCount,
		DailyTx:            dailySwapsFromRune.TxCount + dailySwapsToRune.TxCount,
		MonthlyActiveUsers: monthlySwapsFromRune.RuneAddrCount + monthlySwapsToRune.RuneAddrCount,
		MonthlyTx:          monthlySwapsFromRune.TxCount + monthlySwapsToRune.TxCount,
		// PoolCount:          0, //TODO(kashif)
		TotalAssetBuys:  swapsFromRune.TxCount,
		TotalAssetSells: swapsToRune.TxCount,
		TotalDepth:      runeDepth,
		// TotalEarned:        0, //TODO(kashif)
		TotalStakeTx: stakes.TxCount + unstakes.TxCount,
		TotalStaked:  stakes.RuneE8Total - unstakes.RuneE8Total,
		TotalTx:      swapsFromRune.TxCount + swapsToRune.TxCount + stakes.TxCount + unstakes.TxCount,
		TotalUsers:   swapsFromRune.RuneAddrCount + swapsToRune.RuneAddrCount,
		TotalVolume:  swapsFromRune.RuneE8Total + swapsToRune.RuneE8Total,
		// TotalVolume24hr:    0, //TODO(kashif)
		TotalWithdrawTx: unstakes.RuneE8Total,
	}

	return result, nil
}

// TODO(donfrigo) investigate if caching is possible for this endpoint as well
func (r *queryResolver) Network(ctx context.Context) (*model.Network, error) {
	networkData, err := timeseries.GetNetworkData(ctx)
	if err != nil {
		return nil, err
	}

	return &networkData, nil
}

// Modifies incoming parameters.
func setupDefaultParameters(from *int64, until *int64, interval *model.Interval) stat.Window {
	// If from is not provided, we go back one week
	if from == nil {
		fromPointer := int64(0)
		from = &fromPointer
	}
	if until == nil {
		untilPointer := int64(0)
		until = &untilPointer
	}
	if interval == nil {
		intervalPointer := model.IntervalDay
		interval = &intervalPointer
	}

	return stat.Window{
		// TODO(acsaba): check if timezones matter.
		From:  time.Unix(*from, 0),
		Until: time.Unix(*until, 0),
	}
}

// Bucketing logic under timeseries uses another enum than the public facing one.
var toStatInterval = map[model.Interval]stat.Interval{
	model.IntervalMinute5: stat.Min5,
	model.IntervalHour:    stat.Hour,
	model.IntervalDay:     stat.Day,
	model.IntervalMonth:   stat.Month,
	model.IntervalQuarter: stat.Quarter,
	model.IntervalYear:    stat.Year,
}

func (r *queryResolver) VolumeHistory(ctx context.Context, pool string, from *int64, until *int64, interval *model.Interval) (*model.PoolVolumeHistory, error) {
	window := setupDefaultParameters(from, until, interval)

	poolSwaps, err := stat.GetPoolSwaps(ctx, pool, window, toStatInterval[*interval])
	if err != nil {
		return nil, err
	}

	result, err := createPoolVolumeHistory(poolSwaps)
	if err != nil {
		return nil, err
	}

	return result, nil
}

type volumeMetaData struct {
	time time.Time

	ToRuneTxCount       int64
	ToRuneFeesInRune    int64
	ToRuneVolumesInRune int64

	ToAssetTxCount       int64
	ToAssetFeesInRune    int64
	ToAssetVolumesInRune int64

	CombTxCount       int64
	CombFeesInRune    int64
	CombVolumesInRune int64
}

func createPoolVolumeHistory(poolSwaps []stat.PoolSwaps) (*model.PoolVolumeHistory, error) {
	meta := &volumeMetaData{}

	result := &model.PoolVolumeHistory{
		Intervals: []*model.PoolVolumeHistoryBucket{},
	}

	for _, poolSwaps := range poolSwaps {
		combinedStats := &model.VolumeStats{}
		ps := model.PoolVolumeHistoryBucket{}

		fr := poolSwaps.FromRune
		tr := poolSwaps.ToRune

		ps.ToAsset = &fr
		ps.ToRune = &tr
		ps.Time = poolSwaps.TruncatedTime.Unix()
		ps.Combined = &model.VolumeStats{
			Count:        fr.Count + tr.Count,
			VolumeInRune: fr.VolumeInRune + tr.VolumeInRune,
			FeesInRune:   fr.FeesInRune + tr.FeesInRune,
		}

		updateCombinedStats(combinedStats, poolSwaps)
		updateSwapMetadata(meta, poolSwaps)

		result.Meta = &model.PoolVolumeHistoryMeta{
			ToRune: &model.VolumeStats{
				Count:        meta.ToRuneTxCount,
				FeesInRune:   meta.ToRuneFeesInRune,
				VolumeInRune: meta.ToRuneVolumesInRune,
			},
			ToAsset: &model.VolumeStats{
				Count:        meta.ToAssetTxCount,
				FeesInRune:   meta.ToAssetFeesInRune,
				VolumeInRune: meta.ToAssetVolumesInRune,
			},
			Combined: &model.VolumeStats{
				Count:        meta.CombTxCount,
				FeesInRune:   meta.CombFeesInRune,
				VolumeInRune: meta.CombVolumesInRune,
			},
		}

		result.Intervals = append(result.Intervals, &ps)
	}

	inv := result.Intervals
	if len(inv) > 0 {
		result.Meta.First = inv[0].Time
		result.Meta.Last = inv[len(inv)-1].Time
	}

	return result, nil
}

func updateSwapMetadata(meta *volumeMetaData, ps stat.PoolSwaps) {
	fromRune := ps.FromRune
	toRune := ps.ToRune

	meta.ToAssetTxCount += fromRune.Count
	meta.ToAssetFeesInRune += fromRune.FeesInRune
	meta.ToAssetVolumesInRune += fromRune.VolumeInRune

	meta.ToRuneTxCount += toRune.Count
	meta.ToRuneFeesInRune += toRune.FeesInRune
	meta.ToRuneVolumesInRune += toRune.VolumeInRune

	meta.CombTxCount += fromRune.Count + toRune.Count
	meta.CombFeesInRune += fromRune.FeesInRune + toRune.FeesInRune
	meta.CombVolumesInRune += fromRune.VolumeInRune + toRune.VolumeInRune
}

func updateCombinedStats(stats *model.VolumeStats, ps stat.PoolSwaps) {
	fromRune := ps.FromRune
	toRune := ps.ToRune

	stats.Count += fromRune.Count + toRune.Count
	stats.FeesInRune += fromRune.FeesInRune + toRune.FeesInRune
	stats.VolumeInRune += fromRune.VolumeInRune + toRune.VolumeInRune
}

func (r *queryResolver) PoolHistory(ctx context.Context, pool string, from *int64, until *int64, interval *model.Interval) (*model.PoolHistoryDetails, error) {
	window := setupDefaultParameters(from, until, interval)

	depthsArr, err := stat.PoolDepthBucketsLookup(ctx, pool, toStatInterval[*interval], window)
	if err != nil {
		return nil, err
	}

	meta := model.PoolHistoryMeta{}
	if len(depthsArr) > 0 {
		first := depthsArr[0]
		last := depthsArr[len(depthsArr)-1]

		// Array is ORDERED by time. (see depth.go)
		meta.First = first.Time
		meta.RuneFirst = first.Rune
		meta.AssetFirst = first.Asset
		meta.PriceFirst = first.Price
		meta.Last = last.Time
		meta.RuneLast = last.Rune
		meta.AssetLast = last.Asset
		meta.PriceLast = last.Price
	}

	result := &model.PoolHistoryDetails{
		Meta:      &meta,
		Intervals: depthsArr,
	}

	return result, nil
}

func (r *queryResolver) StakeHistory(ctx context.Context, pool string, from *int64, until *int64, interval *model.Interval) (*model.PoolStakeHistory, error) {
	window := setupDefaultParameters(from, until, interval)

	stakesArr, err := stat.GetPoolStakes(ctx, pool, window, toStatInterval[*interval])
	if err != nil {
		return nil, err
	}
	var intervals []*model.PoolStakeHistoryBucket
	meta := &model.PoolStakeHistoryMeta{}

	for i, s := range stakesArr {
		ps := model.PoolStakeHistoryBucket{
			Time:        s.Time.Unix(),
			Count:       s.TxCount,
			RuneVolume:  s.RuneE8Total,
			AssetVolume: s.AssetE8Total,
			Units:       s.StakeUnitsTotal,
		}

		meta.Count += s.TxCount
		meta.RuneVolume += s.RuneE8Total
		meta.AssetVolume += s.AssetE8Total
		meta.Units += s.StakeUnitsTotal

		if i == 0 {
			meta.First = s.Time.Unix()
		}
		if len(stakesArr)-1 == i {
			meta.Last = s.Time.Unix()
		}
		intervals = append(intervals, &ps)
	}

	result := &model.PoolStakeHistory{
		Meta:      meta,
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
