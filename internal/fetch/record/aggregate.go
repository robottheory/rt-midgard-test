package record

// RunningTotals captures statistics in memory.
type runningTotals struct {
	// running totals
	// TODO(muninn): get rid of the pointers
	assetE8DepthPerPool map[string]*int64
	runeE8DepthPerPool  map[string]*int64
	synthE8DepthPerPool map[string]*int64
	unitsPerPool        map[string]*int64
}

func newRunningTotals() *runningTotals {
	return &runningTotals{
		assetE8DepthPerPool: make(map[string]*int64),
		runeE8DepthPerPool:  make(map[string]*int64),
		synthE8DepthPerPool: make(map[string]*int64),
		unitsPerPool:        make(map[string]*int64),
	}
}

func (t *runningTotals) CurrentDepths(pool []byte) (assetE8, runeE8, synthE8 int64) {
	if p, ok := t.assetE8DepthPerPool[string(pool)]; ok {
		assetE8 = *p
	}
	if p, ok := t.runeE8DepthPerPool[string(pool)]; ok {
		runeE8 = *p
	}
	if p, ok := t.synthE8DepthPerPool[string(pool)]; ok {
		synthE8 = *p
	}
	return
}

// AddPoolAssetE8Depth adjusts the quantity. Use a negative value to deduct.
func (t *runningTotals) AddPoolAssetE8Depth(pool []byte, assetE8 int64) {
	if p, ok := t.assetE8DepthPerPool[string(pool)]; ok {
		*p += assetE8
	} else {
		t.assetE8DepthPerPool[string(pool)] = &assetE8
	}
}

// AddPoolRuneE8Depth adjusts the quantity. Use a negative value to deduct.
func (t *runningTotals) AddPoolRuneE8Depth(pool []byte, runeE8 int64) {
	if p, ok := t.runeE8DepthPerPool[string(pool)]; ok {
		*p += runeE8
	} else {
		t.runeE8DepthPerPool[string(pool)] = &runeE8
	}
}

// AddPoolSynthE8Depth adjusts the quantity. Use a negative value to deduct.
func (t *runningTotals) AddPoolSynthE8Depth(pool []byte, synthE8 int64) {
	if p, ok := t.synthE8DepthPerPool[string(pool)]; ok {
		*p += synthE8
	} else {
		t.synthE8DepthPerPool[string(pool)] = &synthE8
	}
}

// AddPoolUnit adjusts the pool units of a pool
func (t *runningTotals) AddPoolUnit(pool []byte, unit int64) {
	if p, ok := t.unitsPerPool[string(pool)]; ok {
		*p += unit
	} else {
		t.unitsPerPool[string(pool)] = &unit
	}
}

func (t *runningTotals) SetAssetDepth(pool string, assetE8 int64) {
	v := assetE8
	t.assetE8DepthPerPool[pool] = &v
}

func (t *runningTotals) SetRuneDepth(pool string, runeE8 int64) {
	v := runeE8
	t.runeE8DepthPerPool[pool] = &v
}

func (t *runningTotals) SetSynthDepth(pool string, synthE8 int64) {
	v := synthE8
	t.synthE8DepthPerPool[pool] = &v
}

// Set units of a pool
func (t *runningTotals) SetPoolUnit(pool string, unit int64) {
	v := unit
	t.unitsPerPool[pool] = &v
}

// AssetE8DepthPerPool returns a snapshot copy.
func (t *runningTotals) AssetE8DepthPerPool() map[string]int64 {
	m := make(map[string]int64, len(t.assetE8DepthPerPool))
	for asset, p := range t.assetE8DepthPerPool {
		m[asset] = *p
	}
	return m
}

// RuneE8DepthPerPool returns a snapshot copy.
func (t *runningTotals) RuneE8DepthPerPool() map[string]int64 {
	m := make(map[string]int64, len(t.runeE8DepthPerPool))
	for asset, p := range t.runeE8DepthPerPool {
		m[asset] = *p
	}
	return m
}

// SynthE8DepthPerPool returns a snapshot copy.
func (t *runningTotals) SynthE8DepthPerPool() map[string]int64 {
	m := make(map[string]int64, len(t.synthE8DepthPerPool))
	for asset, p := range t.synthE8DepthPerPool {
		m[asset] = *p
	}
	return m
}

// UnitsPerPool returns pool units for all pools
func (t *runningTotals) UnitsPerPool() map[string]int64 {
	m := make(map[string]int64, len(t.unitsPerPool))
	for asset, p := range t.unitsPerPool {
		m[asset] = *p
	}
	return m
}
