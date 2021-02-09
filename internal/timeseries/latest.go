package timeseries

import "sync"

type PriceMap map[string]float64

func (p PriceMap) PoolExistst(pool string) bool {
	_, ok := p[pool]
	return ok
}

type LatestStates struct {
	sync.RWMutex
	prices PriceMap
}

var Latest LatestStates

func (latest *LatestStates) setLatestStates(track *blockTrack) {
	newPrices := PriceMap{}
	runeDepths := track.RuneE8DepthPerPool
	for pool, assetDepth := range track.AssetE8DepthPerPool {
		if assetDepth == 0 {
			continue
		}
		runeDepth, ok := runeDepths[pool]
		if !ok || runeDepth == 0 {
			continue
		}
		newPrices[pool] = float64(assetDepth) / float64(runeDepth)
	}
	latest.Lock()
	latest.prices = newPrices
	latest.Unlock()
}

func (latest *LatestStates) GetPrices() PriceMap {
	latest.RLock()
	defer latest.RUnlock()
	return latest.prices
}
