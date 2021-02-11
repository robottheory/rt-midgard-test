package timeseries

import (
	"sync"

	"gitlab.com/thorchain/midgard/internal/db"
)

type DepthPair struct {
	AssetDepth int64
	RuneDepth  int64
}

func AssetPrice(assetDepth, runeDepth int64) float64 {
	if assetDepth == 0 {
		return 0
	}
	return float64(runeDepth) / float64(assetDepth)
}

func (p DepthPair) Price() float64 {
	return AssetPrice(p.AssetDepth, p.RuneDepth)
}

type DepthMap map[string]DepthPair

type BlockState struct {
	Height    int64
	Timestamp db.Nano
	Pools     DepthMap
}

func (s BlockState) PoolExists(pool string) bool {
	_, ok := s.Pools[pool]
	return ok
}

// Returns nil if pool doesn't exist
func (s BlockState) PoolInfo(pool string) *DepthPair {
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

func (latest *LatestState) setLatestStates(track *blockTrack) {
	newState := BlockState{
		Height:    track.Height,
		Timestamp: db.TimeToNano(track.Timestamp),
		Pools:     DepthMap{}}

	runeDepths := track.RuneE8DepthPerPool
	for pool, assetDepth := range track.AssetE8DepthPerPool {
		runeDepth, ok := runeDepths[pool]
		if !ok {
			continue
		}
		newState.Pools[pool] = DepthPair{AssetDepth: assetDepth, RuneDepth: runeDepth}
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
	return Latest.state.PoolExists(pool)
}
