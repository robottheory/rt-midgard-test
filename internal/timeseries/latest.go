package timeseries

import (
	"sync"

	"gitlab.com/thorchain/midgard/config"
	"gitlab.com/thorchain/midgard/internal/db"
)

type PoolDepths struct {
	AssetDepth int64
	RuneDepth  int64
	SynthDepth int64
	PoolUnit   int64
}

type PoolOHLCV struct {
	AssetDepth int64
	RuneDepth  int64
	SynthDepth int64
	PoolUnit   int64
	Liquidity  int64
	FirstPrice float64
	LastPrice  float64
	MaxPrice   float64
	MinPrice   float64
	StartDate  int64
	EndDate    int64
	MinDate    int64
	MaxDate    int64
	FirstDate  int64
	LastDate   int64
	Volume     int64
}

func AssetPrice(assetDepth, runeDepth int64) float64 {
	if assetDepth == 0 {
		return 0
	}
	return float64(runeDepth) / float64(assetDepth)
}

func (p PoolDepths) AssetPrice() float64 {
	return AssetPrice(p.AssetDepth, p.RuneDepth)
}

// When a pool becomes suspended all the funds are burned.
// We use this as a detection of pools which no longer exist.
func (p PoolDepths) ExistsNow() bool {
	return 0 < p.AssetDepth && 0 < p.RuneDepth
}

type DepthMap map[string]PoolDepths

type OHLCVMap map[string]PoolOHLCV

type BlockState struct {
	Height    int64
	Timestamp db.Nano
	Pools     DepthMap
}

func (s BlockState) PoolExists(pool string) bool {
	_, ok := s.Pools[pool]
	return ok
}

func (s BlockState) NextSecond() db.Second {
	return s.Timestamp.ToSecond() + 1
}

// Returns nil if pool doesn't exist
func (s BlockState) PoolInfo(pool string) *PoolDepths {
	info, ok := s.Pools[pool]
	if !ok {
		return nil
	}
	return &info
}

type LatestState struct {
	sync.RWMutex
	state BlockState
}

var Latest LatestState

func ResetLatestStateForTest() {
	Latest = LatestState{}
}

func UpdateUsdPools() {
	if config.Global.UsdPools != nil {
		usdPoolWhitelist = config.Global.UsdPools
	}
}

func (latest *LatestState) setLatestStates(track *blockTrack) {
	newState := BlockState{
		Height:    track.Height,
		Timestamp: db.TimeToNano(track.Timestamp),
		Pools:     DepthMap{},
	}

	runeDepths := track.RuneE8DepthPerPool
	synthDepths := track.SynthE8DepthPerPool
	poolUnits := track.UnitsPerPool
	for pool, assetDepth := range track.AssetE8DepthPerPool {
		runeDepth, ok := runeDepths[pool]
		if !ok {
			continue
		}
		synthDepth, ok := synthDepths[pool]
		if !ok {
			synthDepth = 0
		}
		poolUnit, ok := poolUnits[pool]
		if !ok {
			poolUnit = 0
		}
		newState.Pools[pool] = PoolDepths{AssetDepth: assetDepth, RuneDepth: runeDepth, SynthDepth: synthDepth, PoolUnit: poolUnit}
	}
	latest.Lock()
	latest.state = newState
	latest.Unlock()
}

func (latest *LatestState) GetState() BlockState {
	latest.RLock()
	defer latest.RUnlock()
	return latest.state
}

func PoolExists(pool string) bool {
	return Latest.GetState().PoolExists(pool)
}

func PoolExistsNow(pool string) bool {
	depths, ok := Latest.GetState().Pools[pool]
	return ok && depths.ExistsNow()
}
