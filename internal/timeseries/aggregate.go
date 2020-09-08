package timeseries

import (
	"bytes"
	"log"
	"time"

	"github.com/pascaldekloe/metrics"

	"gitlab.com/thorchain/midgard/event"
)

var (
	LinkedFound = metrics.Must3LabelCounter("midgard_chain_event_linked_found", "type", "ref_type", "class")
	LinkedDeads = metrics.Must1LabelCounter("midgard_chain_event_linked_deads", "type")

	swapOutboundFromRune = LinkedFound("outbound", "swap", "from_rune")
	swapOutboundToRune   = LinkedFound("outbound", "swap", "to_rune")
	unstakeOutboundAsset = LinkedFound("outbound", "unstake", "asset")
	unstakeOutboundRune  = LinkedFound("outbound", "unstake", "rune")
	swapFeeAsset         = LinkedFound("fee", "swap", "asset")
	swapFeeRune          = LinkedFound("fee", "swap", "rune")
	deadOutbound         = LinkedDeads("outbound")
	deadFee              = LinkedDeads("fee")
)

func init() {
	metrics.MustHelp("midgard_chain_event_linked_found", "Number of events with a matching input transaction ID.")
	metrics.MustHelp("midgard_chain_event_linked_deads", "Number of events with an unknown input transaction ID.")
}

// LinkedEvents resolves relations inbetween events.
type linkedEvents struct {
	// enqueued for lookup
	outboundQ, feeQ map[string][]event.Amount
}

func newLinkedEvents() *linkedEvents {
	return &linkedEvents{
		outboundQ: make(map[string][]event.Amount),
		feeQ:      make(map[string][]event.Amount),
	}
}

// ApplyOutboundQ reads (and clears) any enqueued outbound events to feed t.
func (l *linkedEvents) ApplyOutboundQ(t *runningTotals, blockHeight int64, blockTimestamp time.Time) {
	if len(l.outboundQ) == 0 {
		return
	}
	defer func() {
		deadOutbound.Add(uint64(len(l.outboundQ)))
		for txID := range l.outboundQ {
			delete(l.outboundQ, txID)
		}
	}()

	l.matchSwaps(t, blockHeight, blockTimestamp)
	if len(l.outboundQ) == 0 {
		return
	}
	l.matchUnstakes(t, blockHeight, blockTimestamp)
}

func (l *linkedEvents) matchSwaps(t *runningTotals, blockHeight int64, blockTimestamp time.Time) {
	txIDs := make([]string, 0, len(l.outboundQ))
	for s := range l.outboundQ {
		txIDs = append(txIDs, s)
	}

	// find matching swap events
	const q = "SELECT tx, pool FROM swap_events WHERE tx = ANY($1) AND block_timestamp > $2"
	rows, err := DBQuery(q, txIDs, blockTimestamp.Add(-OutboundTimeout).UnixNano())
	if err != nil {
		log.Printf("block height %d swaps for outbounds lookup: %s", blockHeight, err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var txID, pool []byte
		if err := rows.Scan(&txID, &pool); err != nil {
			log.Printf("block height %d swap for outbound resolve: %s", blockHeight, err)
			continue
		}

		amounts := l.outboundQ[string(txID)]
		delete(l.outboundQ, string(txID))

		for _, a := range amounts {
			// There's no clean way to distinguish between RUNE and
			// pool asset based solely on event data/relations.
			if event.IsRune(a.Asset) {
				swapOutboundToRune.Add(1)
				t.AddPoolRuneE8Depth(pool, -a.E8)
			} else {
				swapOutboundFromRune.Add(1)
				t.AddPoolAssetE8Depth(a.Asset, -a.E8)
			}
		}
	}
	if err := rows.Err(); err != nil {
		log.Printf("block height %d swaps for outbounds resolve: %s", blockHeight, err)
		return
	}
}

func (l *linkedEvents) matchUnstakes(t *runningTotals, blockHeight int64, blockTimestamp time.Time) {
	txIDs := make([]string, 0, len(l.outboundQ))
	for s := range l.outboundQ {
		txIDs = append(txIDs, s)
	}

	// find matching unstake events
	const q = "SELECT tx, pool FROM unstake_events WHERE tx = ANY($1) AND block_timestamp > $2"
	rows, err := DBQuery(q, txIDs, blockTimestamp.Add(-OutboundTimeout).UnixNano())
	if err != nil {
		log.Printf("block height %d unstakes for outbounds lookup: %s", blockHeight, err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var txID, pool []byte
		if err := rows.Scan(&txID, &pool); err != nil {
			log.Printf("block height %d unstake for outbound resolve: %s", blockHeight, err)
			continue
		}

		amounts := l.outboundQ[string(txID)]
		delete(l.outboundQ, string(txID))

		for _, a := range amounts {
			if bytes.Equal(a.Asset, pool) {
				unstakeOutboundAsset.Add(1)
				t.AddPoolAssetE8Depth(pool, -a.E8)
			} else {
				if !event.IsRune(a.Asset) {
					log.Printf("block height %d unstake outbound asset %q for pool %q assumed RUNE", blockHeight, a.Asset, pool)
				}
				unstakeOutboundRune.Add(1)
				t.AddPoolRuneE8Depth(pool, -a.E8)
			}
		}
	}
	if err := rows.Err(); err != nil {
		log.Printf("block height %d unstakes for outbounds resolve: %s", blockHeight, err)
		return
	}
}

// ApplyFeeQ reads (and clears) any enqueued fee events to feed t.
func (l *linkedEvents) ApplyFeeQ(t *runningTotals, blockHeight int64, blockTimestamp time.Time) {
	if len(l.feeQ) == 0 {
		return
	}
	defer func() {
		deadFee.Add(uint64(len(l.feeQ)))
		for txID := range l.feeQ {
			delete(l.feeQ, txID)
		}
	}()

	txIDs := make([]string, 0, len(l.feeQ))
	for s := range l.feeQ {
		txIDs = append(txIDs, s)
	}

	// find matching swap events
	const q = "SELECT tx, pool FROM swap_events WHERE tx = ANY($1) AND block_timestamp > $2"
	rows, err := DBQuery(q, txIDs, blockTimestamp.Add(-OutboundTimeout).UnixNano())
	if err != nil {
		log.Printf("block height %d swaps for fees lookup: %s", blockHeight, err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var txID, pool []byte
		if err := rows.Scan(&txID, &pool); err != nil {
			log.Printf("block height %d swap for fee resolve: %s", blockHeight, err)
			continue
		}

		amounts := l.feeQ[string(txID)]
		delete(l.feeQ, string(txID))

		for _, a := range amounts {
			// There's no clean way to distinguish between RUNE and
			// pool asset based solely on event data/relations.
			if event.IsRune(a.Asset) {
				swapFeeRune.Add(1)
				t.AddPoolRuneE8Depth(pool, a.E8)
			} else {
				swapFeeAsset.Add(1)
				t.AddPoolAssetE8Depth(pool, a.E8)
			}
		}
	}
	if err := rows.Err(); err != nil {
		log.Printf("block height %d swaps for fees resolve: %s", blockHeight, err)
		return
	}
}

// RunningTotals captures statistics in memory.
type runningTotals struct {
	// running totals
	assetE8DepthPerPool map[string]*int64
	runeE8DepthPerPool  map[string]*int64
}

func newRunningTotals() *runningTotals {
	return &runningTotals{
		assetE8DepthPerPool: make(map[string]*int64),
		runeE8DepthPerPool:  make(map[string]*int64),
	}
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
