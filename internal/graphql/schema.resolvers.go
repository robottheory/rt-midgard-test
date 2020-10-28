package graphql

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"

	"gitlab.com/thorchain/midgard/chain/notinchain"
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

func (r *queryResolver) Stakers(ctx context.Context) ([]*model.Staker, error) {
	addrs, err := stakeAddrs(ctx, time.Time{})
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
	_, runeE8DepthPerPool, timestamp := getAssetAndRuneDepths()
	window := stat.Window{time.Unix(0, 0), timestamp}

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
	dailySwapsFromRune, err := swapsFromRuneLookup(ctx, stat.Window{Since: timestamp.Add(-24 * time.Hour), Until: timestamp})
	if err != nil {
		return nil, err
	}
	dailySwapsToRune, err := swapsToRuneLookup(ctx, stat.Window{Since: timestamp.Add(-24 * time.Hour), Until: timestamp})
	if err != nil {
		return nil, err
	}
	monthlySwapsFromRune, err := swapsFromRuneLookup(ctx, stat.Window{Since: timestamp.Add(-30 * 24 * time.Hour), Until: timestamp})
	if err != nil {
		return nil, err
	}
	monthlySwapsToRune, err := swapsToRuneLookup(ctx, stat.Window{Since: timestamp.Add(-30 * 24 * time.Hour), Until: timestamp})
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

// TODO(kashif) copy paste from v1.go.
// All these should be migrated to a common service layer
type sortedBonds []*int64

func (b sortedBonds) Len() int           { return len(b) }
func (b sortedBonds) Less(i, j int) bool { return *b[i] < *b[j] }
func (b sortedBonds) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }

func makeBondMetricStat(bonds sortedBonds) *model.BondMetricsStat {
	m := model.BondMetricsStat{
		TotalBond:   0,
		MinimumBond: 0,
		MaximumBond: 0,
		MedianBond:  0,
		AverageBond: 0,
	}
	if len(bonds) != 0 {
		for _, n := range bonds {
			m.TotalBond += *n
		}
		m.MinimumBond = *bonds[0]
		m.MaximumBond = *bonds[len(bonds)-1]
		m.AverageBond, _ = big.NewRat(m.TotalBond, int64(len(bonds))).Float64()
		m.MedianBond = *bonds[len(bonds)/2]
	}
	return &m
}

func (r *queryResolver) Network(ctx context.Context) (*model.Network, error) {
	_, runeE8DepthPerPool, _ := getAssetAndRuneDepths()

	var runeDepth int64
	for _, depth := range runeE8DepthPerPool {
		runeDepth += depth
	}

	activeNodes := make(map[string]struct{})
	standbyNodes := make(map[string]struct{})
	var activeBonds, standbyBonds sortedBonds
	nodes, err := cachedNodeAccountsLookup()
	if err != nil {
		return nil, err
	}

	for _, node := range nodes {
		switch node.Status {
		case "active":
			activeNodes[node.NodeAddr] = struct{}{}
			activeBonds = append(activeBonds, &node.Bond)
		case "standby":
			standbyNodes[node.NodeAddr] = struct{}{}
			standbyBonds = append(standbyBonds, &node.Bond)
		}
	}
	sort.Sort(activeBonds)
	sort.Sort(standbyBonds)

	activeNodeCount := int64(len(activeNodes))
	standbyNodeCount := int64(len(standbyNodes))
	result := &model.Network{
		ActiveBonds:      activeBonds,
		ActiveNodeCount:  activeNodeCount,
		TotalStaked:      runeDepth,
		StandbyBonds:     standbyBonds,
		StandbyNodeCount: standbyNodeCount,
		BondMetrics: &model.BondMetrics{
			Active:  makeBondMetricStat(activeBonds),
			Standby: makeBondMetricStat(standbyBonds),
		},
	}

	return result, nil
}

const ASSET_LIST_MAX = 10

func (r *queryResolver) Assets(ctx context.Context, query []*string) ([]*model.Asset, error) {
	if len(query) == 0 {
		return nil, errors.New("At least one asset is required in query")
	}
	if len(query) == 0 || len(query) > ASSET_LIST_MAX {
		return nil, errors.New(fmt.Sprintf("Maximum allowed assets in query is %v", ASSET_LIST_MAX))
	}

	assetE8DepthPerPool, runeE8DepthPerPool, timestamp := getAssetAndRuneDepths()
	window := stat.Window{time.Unix(0, 0), timestamp}

	result := make([]*model.Asset, 0)
	for _, asset := range query {
		stakes, err := poolStakesLookup(ctx, *asset, window)
		if err != nil {
			return nil, err
		}

		m := &model.Asset{
			Asset:   *asset,
			Created: stakes.First.String(),
		}

		if assetDepth := assetE8DepthPerPool[*asset]; assetDepth != 0 {
			m.Price = float64(runeE8DepthPerPool[*asset]) / float64(assetDepth)
		}

		// Ignore not found ones.
		if !stakes.First.IsZero() {
			result = append(result, m)
		}
	}

	return result, nil
}

func makeBucketSizeAndDurationWindow(from *int64, until *int64, interval *model.Interval) (time.Duration, stat.Window, error) {
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

	now := time.Now()
	durationWindow := stat.Window{
		Since: now.Add(-bucketSize),
		Until: now,
	}

	if from != nil && until != nil {
		if *from > *until {
			return bucketSize, durationWindow, fmt.Errorf("from %d cannot be greater than until %d", *from, *until)
		}
	}

	if until != nil {
		durationWindow.Until = time.Unix(*until, 0)
		durationWindow.Since = durationWindow.Until.Add(-bucketSize)
	}

	if from != nil {
		durationWindow.Since = time.Unix(*from, 0)
	}

	return bucketSize, durationWindow, nil
}

func (r *queryResolver) VolumeHistory(ctx context.Context, pool string, from *int64, until *int64, interval *model.PoolVolumeInterval) (*model.PoolVolumeHistory, error) {
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
		intervalPointer := model.PoolVolumeIntervalDay
		interval = &intervalPointer
	}

	window := stat.Window{
		Since: time.Unix(*from, 0),
		Until: time.Unix(*until, 0),
	}

	// fromRune stores conversion from Rune to Asset -> selling Rune
	fromRune, err := stat.PoolSwapsLookup(ctx, pool, *interval, window, false)
	if err != nil {
		return nil, err
	}

	// fromAsset stores conversion from Asset to Rune -> buying Rune
	fromAsset, err := stat.PoolSwapsLookup(ctx, pool, *interval, window, true)
	if err != nil {
		return nil, err
	}

	result, err := mergeSwaps(fromRune, fromAsset)
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

func mergeSwaps(fromRune, fromAsset []stat.PoolSwaps) (*model.PoolVolumeHistory, error) {
	meta := &volumeMetaData{}

	result := &model.PoolVolumeHistory{
		Intervals: []*model.PoolVolumeHistoryBucket{},
	}

	mergedPoolSwaps, err := stat.MergeSwaps(fromRune, fromAsset)
	if err != nil {
		return nil, err
	}

	for _, poolSwaps := range mergedPoolSwaps {
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

func (r *queryResolver) PriceHistory(ctx context.Context, asset string, from *int64, until *int64, interval *model.Interval) (*model.PoolPriceHistory, error) {
	bucketSize, durationWindow, err := makeBucketSizeAndDurationWindow(from, until, interval)
	if err != nil {
		return nil, err
	}

	depthsArr, err := stat.PoolDepthBucketsLookup(ctx, asset, bucketSize, durationWindow)
	if err != nil {
		return nil, err
	}
	var intervals []*model.PoolPriceHistoryBucket
	meta := model.PoolPriceHistoryBucket{}

	if len(depthsArr) > 0 {
		first := depthsArr[0]
		last := depthsArr[len(depthsArr)-1]

		// Array is ORDERED by time. (see depth.go)
		meta.First = first.First.Unix()
		meta.PriceFirst = first.PriceFirst
		meta.Last = last.Last.Unix()
		meta.PriceLast = last.PriceLast
	}

	for _, s := range depthsArr {
		first := s.First.Unix()
		last := s.Last.Unix()
		ps := model.PoolPriceHistoryBucket{
			First:      first,
			Last:       last,
			PriceFirst: s.PriceFirst,
			PriceLast:  s.PriceLast,
		}

		intervals = append(intervals, &ps)
	}

	result := &model.PoolPriceHistory{
		Meta:      &meta,
		Intervals: intervals,
	}

	return result, nil
}

func (r *queryResolver) DepthHistory(ctx context.Context, asset string, from *int64, until *int64, interval *model.Interval) (*model.PoolDepthHistory, error) {
	bucketSize, durationWindow, err := makeBucketSizeAndDurationWindow(from, until, interval)
	if err != nil {
		return nil, err
	}

	depthsArr, err := stat.PoolDepthBucketsLookup(ctx, asset, bucketSize, durationWindow)
	if err != nil {
		return nil, err
	}

	intervals := []*model.PoolDepthHistoryBucket{}

	meta := model.PoolDepthHistoryBucket{}
	if len(depthsArr) > 0 {
		first := depthsArr[0]
		last := depthsArr[len(depthsArr)-1]

		// Array is ORDERED by time. (see depth.go)
		meta.First = first.First.Unix()
		meta.RuneFirst = first.RuneFirst
		meta.AssetFirst = first.AssetFirst
		meta.PriceFirst = first.PriceFirst
		meta.Last = last.Last.Unix()
		meta.RuneLast = last.RuneLast
		meta.AssetLast = last.AssetLast
		meta.PriceLast = last.PriceLast
	}

	for _, s := range depthsArr {
		first := s.First.Unix()
		last := s.Last.Unix()

		ps := model.PoolDepthHistoryBucket{
			First:      first,
			Last:       last,
			RuneFirst:  s.RuneFirst,
			RuneLast:   s.RuneLast,
			AssetFirst: s.AssetFirst,
			AssetLast:  s.AssetLast,
			PriceFirst: s.PriceFirst,
			PriceLast:  s.PriceLast,
		}

		intervals = append(intervals, &ps)

	}

	result := &model.PoolDepthHistory{
		Meta:      &meta,
		Intervals: intervals,
	}

	return result, nil
}

func (r *queryResolver) StakeHistory(ctx context.Context, asset string, from *int64, until *int64, interval *model.Interval) (*model.PoolStakeHistory, error) {
	bucketSize, durationWindow, err := makeBucketSizeAndDurationWindow(from, until, interval)
	if err != nil {
		return nil, err
	}

	stakesArr, err := poolStakesBucketsLookup(ctx, asset, bucketSize, durationWindow)
	if err != nil {
		return nil, err
	}
	var intervals []*model.PoolStakeHistoryBucket

	for _, s := range stakesArr {
		first := s.First.Unix()
		last := s.Last.Unix()
		ps := model.PoolStakeHistoryBucket{
			First:         first,
			Last:          last,
			Count:         s.TxCount,
			VolumeInRune:  s.RuneE8Total,
			VolumeInAsset: s.AssetE8Total,
			Units:         s.StakeUnitsTotal,
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
