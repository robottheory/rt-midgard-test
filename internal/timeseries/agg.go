package timeseries

// RunningTotals captures statistics in memory.
type runningTotals struct {
	// running totals
	assetE8PerPool map[string]*int64
	runeE8PerPool  map[string]*int64
}

func newRunningTotals() *runningTotals {
	return &runningTotals{
		assetE8PerPool: make(map[string]*int64),
		runeE8PerPool:  make(map[string]*int64),
	}
}

// AddPoolAssetE8 adjusts the stake quantities. Use negative values to deduct.
func (t *runningTotals) AddPoolAssetE8(pool []byte, assetE8 int64) {
	if p, ok := t.assetE8PerPool[string(pool)]; ok {
		*p += assetE8
	} else {
		t.assetE8PerPool[string(pool)] = &assetE8
	}
}

// AddPoolRuneE8 adjusts the stake quantities. Use negative values to deduct.
func (t *runningTotals) AddPoolRuneE8(pool []byte, runeE8 int64) {
	if p, ok := t.runeE8PerPool[string(pool)]; ok {
		*p += runeE8
	} else {
		t.runeE8PerPool[string(pool)] = &runeE8
	}
}

// AssetE8PerPool returns a snapshot copy.
func (t *runningTotals) AssetE8PerPool() map[string]int64 {
	m := make(map[string]int64, len(t.assetE8PerPool))
	for asset, p := range t.assetE8PerPool {
		m[asset] = *p
	}
	return m
}

// RuneE8PerPool returns a snapshot copy.
func (t *runningTotals) RuneE8PerPool() map[string]int64 {
	m := make(map[string]int64, len(t.runeE8PerPool))
	for asset, p := range t.runeE8PerPool {
		m[asset] = *p
	}
	return m
}
