package timeseries

import (
	"bytes"
	"context"
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
	deadOutbound         = LinkedDeads("outbound")
	deadFee              = LinkedDeads("fee")
)

// Double Depth Counting
var (
	swapFeePerPoolAndAsset         = metrics.Must2LabelCounter("midgard_event_swap_fee_E8s_total", "pool", "asset")
	swapOutboundPerPoolAndAsset    = metrics.Must2LabelCounter("midgard_event_swap_outbound_E8s_total", "pool", "asset")
	unstakeOutboundPerPoolAndAsset = metrics.Must2LabelCounter("midgard_event_unstake_outbound_E8s_total", "pool", "asset")
)

func init() {
	metrics.MustHelp("midgard_chain_event_linked_found", "Number of events with a matching input transaction ID.")
	metrics.MustHelp("midgard_chain_event_linked_deads", "Number of events with an unknown input transaction ID.")
}

// LinkedEvents resolves relations inbetween events.
type linkedEvents struct {
	// enqueued for lookup
	outboundQ map[string][]event.Amount
	feeQ      map[string][]event.Fee
}

func newLinkedEvents() *linkedEvents {
	return &linkedEvents{
		outboundQ: make(map[string][]event.Amount),
		feeQ:      make(map[string][]event.Fee),
	}
}

func (l *linkedEvents) OnFee(e *event.Fee, meta *event.Metadata) {
	l.feeQ[string(e.Tx)] = append(l.feeQ[string(e.Tx)], *e)
}

func (l *linkedEvents) OnOutbound(e *event.Outbound, meta *event.Metadata) {
	l.outboundQ[string(e.InTx)] = append(l.outboundQ[string(e.InTx)],
		event.Amount{Asset: e.Asset, E8: e.AssetE8})
}

// ApplyOutboundQ reads (and clears) any enqueued outbound events, and feeds t
// with the outcome.
func (l *linkedEvents) ApplyOutboundQ(t *runningTotals, blockHeight int64, blockTimestamp time.Time) {
	if len(l.outboundQ) == 0 {
		return
	}

	defer func() {
		// expired or missing 😱 should be zero
		deadOutbound.Add(uint64(len(l.outboundQ)))
		for txID := range l.outboundQ {
			delete(l.outboundQ, txID)
		}
	}()

	l.matchSwapOutbounds(t, blockHeight, blockTimestamp)
	l.matchUnstakeOutbounds(t, blockHeight, blockTimestamp)
}

func (l *linkedEvents) matchSwapOutbounds(t *runningTotals, blockHeight int64, blockTimestamp time.Time) {
	if len(l.outboundQ) == 0 {
		return
	}

	txIDs := make([]string, 0, len(l.outboundQ))
	for s := range l.outboundQ {
		txIDs = append(txIDs, s)
	}

	// find matching swap events
	const q = "SELECT tx, pool, from_asset FROM swap_events WHERE tx = ANY($1) AND block_timestamp > $2"
	rows, err := DBQuery(context.Background(), q, txIDs, blockTimestamp.Add(-OutboundTimeout).UnixNano())
	if err != nil {
		log.Printf("block height %d swaps for outbounds lookup: %s", blockHeight, err)
		return
	}
	defer rows.Close()

	// Build a set of swaps with respective pools
	// double swaps have to swap events with the same tx id
	matchingSwaps := make(map[string][](struct{ pool, fromAsset []byte }))
	for rows.Next() {
		var txID, pool, fromAsset []byte
		if err := rows.Scan(&txID, &pool, &fromAsset); err != nil {
			log.Printf("block height %d swap for outbound resolve: %s", blockHeight, err)
			continue
		}
		matchingSwaps[string(txID)] = append(
			matchingSwaps[string(txID)],
			struct{ pool, fromAsset []byte }{pool, fromAsset})

		if err := rows.Err(); err != nil {
			log.Printf("block height %d swaps for outbounds resolve: %s", blockHeight, err)
			return
		}
	}

	for txID, swaps := range matchingSwaps {
		amounts := l.outboundQ[string(txID)]
		delete(l.outboundQ, string(txID))

		if len(swaps) == 1 {
			// Single swap (RUNE <=> Asset), outbound from pool to user
			pool := swaps[0].pool
			amount := amounts[0]

			if event.IsRune(amount.Asset) {
				swapOutboundToRune.Add(1)
				t.AddPoolRuneE8Depth(pool, -amount.E8)
				swapOutboundPerPoolAndAsset(string(pool), "rune").Add(uint64(amount.E8))
			} else {
				swapOutboundFromRune.Add(1)
				t.AddPoolAssetE8Depth(amount.Asset, -amount.E8)
				swapOutboundPerPoolAndAsset(string(pool), "pool").Add(uint64(amount.E8))
			}
		} else if len(swaps) == 2 {
			// Double swap (assetIn => assetOut)
			// RUNE outbound goes from poolIn to poolOut
			// AssetOut outbound goes from poolOut to user

			// Determine which assets are being swapped
			// fromAsset in RUNE is AssetOut pool (rune output from in pool is swaped to AssetOut
			// in poolOut that goes back to the user)
			// assetIn pool: swap.fromAsset: AssetIn, outboundAmount.asset: RUNE
			// assetOut pool: swap.fromAsset: RUNE, outboundAmount.asset: AssetOut
			var assetIn []byte
			var assetOut []byte
			if event.IsRune(swaps[0].fromAsset) && !event.IsRune(swaps[1].fromAsset) {
				assetIn = swaps[1].pool
				assetOut = swaps[0].pool
			} else if !event.IsRune(swaps[0].fromAsset) && event.IsRune(swaps[1].fromAsset) {
				assetIn = swaps[0].pool
				assetOut = swaps[1].pool
			} else {
				log.Printf(
					"block height %d, tx %s, invalid swap from_asset pair (one should be RUNE and the other different from RUNE, got %s, %s",
					blockHeight,
					string(txID),
					string(swaps[0].fromAsset),
					string(swaps[1].fromAsset))
				continue
			}

			for _, amount := range amounts {
				// If outbound amount is in RUNE, it belongs to poolIn
				// If outbound amount is in Asset, it belongs to poolOut
				if event.IsRune(amount.Asset) {
					swapOutboundToRune.Add(1)
					t.AddPoolRuneE8Depth(assetIn, -amount.E8)
					swapOutboundPerPoolAndAsset(string(assetIn), "rune").Add(uint64(amount.E8))
				} else {
					swapOutboundFromRune.Add(1)
					t.AddPoolAssetE8Depth(assetOut, -amount.E8)
					swapOutboundPerPoolAndAsset(string(assetOut), "pool").Add(uint64(amount.E8))
				}
			}
		} else {
			log.Printf("block height %d too many swaps(%d) for tx %s", blockHeight, len(swaps), string(txID))
		}
	}

}

func (l *linkedEvents) matchUnstakeOutbounds(t *runningTotals, blockHeight int64, blockTimestamp time.Time) {
	if len(l.outboundQ) == 0 {
		return
	}

	txIDs := make([]string, 0, len(l.outboundQ))
	for s := range l.outboundQ {
		txIDs = append(txIDs, s)
	}

	// find matching unstake events
	const q = "SELECT tx, pool FROM unstake_events WHERE tx = ANY($1) AND block_timestamp > $2"
	rows, err := DBQuery(context.Background(), q, txIDs, blockTimestamp.Add(-OutboundTimeout).UnixNano())
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
				unstakeOutboundPerPoolAndAsset(string(pool), "pool").Add(uint64(a.E8))
			} else {
				if !event.IsRune(a.Asset) {
					log.Printf("block height %d unstake outbound asset %q for pool %q assumed RUNE", blockHeight, a.Asset, pool)
				}
				unstakeOutboundRune.Add(1)
				t.AddPoolRuneE8Depth(pool, -a.E8)
				unstakeOutboundPerPoolAndAsset(string(pool), "rune").Add(uint64(a.E8))
			}
		}
	}
	if err := rows.Err(); err != nil {
		log.Printf("block height %d unstakes for outbounds resolve: %s", blockHeight, err)
		return
	}
}

// ApplyFeeQ reads (and clears) any enqueued fee events, and feeds t with
// the outcome.
func (l *linkedEvents) ApplyFeeQ(t *runningTotals, blockHeight int64, blockTimestamp time.Time) {
	if len(l.feeQ) == 0 {
		return
	}

	defer func() {
		// expired or missing 😱 should be zero
		deadFee.Add(uint64(len(l.feeQ)))
		for txID := range l.feeQ {
			delete(l.feeQ, txID)
		}
	}()

	l.matchRefundFees(t, blockHeight, blockTimestamp)
	l.matchSwapFees(t, blockHeight, blockTimestamp)
	l.matchUnstakeFees(t, blockHeight, blockTimestamp)
}

func (l *linkedEvents) matchRefundFees(t *runningTotals, blockHeight int64, blockTimestamp time.Time) {
	if len(l.feeQ) == 0 {
		return
	}

	txIDs := make([]string, 0, len(l.feeQ))
	for s := range l.feeQ {
		txIDs = append(txIDs, s)
	}

	// find matching refund events
	const q = `SELECT tx FROM refund_events WHERE block_timestamp > $2 AND tx = ANY($1)`
	rows, err := DBQuery(context.Background(), q, txIDs, blockTimestamp.Add(-OutboundTimeout).UnixNano())
	if err != nil {
		log.Printf("block height %d swaps for fees lookup: %s", blockHeight, err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var txID []byte
		if err := rows.Scan(&txID); err != nil {
			log.Printf("block height %d swap for fee resolve: %s", blockHeight, err)
			continue
		}

		fees := l.feeQ[string(txID)]
		// Iterate for all the tx fees. If fee is not in RUNE, add fee to asset pool
		// (Because the refund related Tx never did count as going into the pool) and deduct
		// pool deduct (That amount goes from pool to the reserve).
		for _, fee := range fees {
			if !event.IsRune(fee.Asset) {
				t.AddPoolAssetE8Depth(fee.Asset, fee.AssetE8)
				t.AddPoolRuneE8Depth(fee.Asset, -fee.PoolDeduct)
			}
		}

		delete(l.feeQ, string(txID))

		if err := rows.Err(); err != nil {
			log.Printf("block height %d swaps for fees resolve: %s", blockHeight, err)
			return
		}
	}
}

func (l *linkedEvents) matchSwapFees(t *runningTotals, blockHeight int64, blockTimestamp time.Time) {
	if len(l.feeQ) == 0 {
		return
	}

	txIDs := make([]string, 0, len(l.feeQ))
	for s := range l.feeQ {
		txIDs = append(txIDs, s)
	}

	// find matching swap events
	const q = `SELECT tx, pool FROM swap_events WHERE block_timestamp > $2 AND tx = ANY($1)`
	rows, err := DBQuery(context.Background(), q, txIDs, blockTimestamp.Add(-OutboundTimeout).UnixNano())
	if err != nil {
		log.Printf("block height %d swaps for fees lookup: %s", blockHeight, err)
		return
	}
	defer rows.Close()
	// Build a set of swaps with respective pools
	// double swaps have the same tx id
	matchingSwaps := make(map[string]([][]byte))
	for rows.Next() {
		var txID, pool []byte
		if err := rows.Scan(&txID, &pool); err != nil {
			log.Printf("block height %d swap for outbound resolve: %s", blockHeight, err)
			continue
		}
		matchingSwaps[string(txID)] = append(matchingSwaps[string(txID)], pool)

		if err := rows.Err(); err != nil {
			log.Printf("block height %d swaps for outbounds resolve: %s", blockHeight, err)
			return
		}
	}

	for txID, swaps := range matchingSwaps {
		fees := l.feeQ[string(txID)]
		delete(l.feeQ, string(txID))
		for _, fee := range fees {
			if !event.IsRune(fee.Asset) {
				// Remove pool deduct from pool, which goes into pool reserve.
				// Asset fee should be counted on outbounds
				t.AddPoolRuneE8Depth(fee.Asset, -fee.PoolDeduct)
				swapFeePerPoolAndAsset(string(fee.Asset), "pool").Add(uint64(fee.AssetE8))
			} else if len(swaps) == 1 {
				// For single swaps where outbound is RUNE, the network fee is charged by leaving
				// that amount on the pool's RUNE, then that needs to go from the pool to the
				// pool reserve
				t.AddPoolRuneE8Depth(swaps[0], -fee.AssetE8)
				swapFeePerPoolAndAsset(string(fee.Asset), "rune").Add(uint64(fee.AssetE8))
			}
		}

		if err := rows.Err(); err != nil {
			log.Printf("block height %d swaps for fees resolve: %s", blockHeight, err)
			return
		}
	}
}

func (l *linkedEvents) matchUnstakeFees(t *runningTotals, blockHeight int64, blockTimestamp time.Time) {
	if len(l.feeQ) == 0 {
		return
	}

	txIDs := make([]string, 0, len(l.feeQ))
	for s := range l.feeQ {
		txIDs = append(txIDs, s)
	}

	// find matching unstake events
	const q = `SELECT tx, pool FROM unstake_events WHERE block_timestamp > $2 AND tx = ANY($1)`
	rows, err := DBQuery(context.Background(), q, txIDs, blockTimestamp.Add(-OutboundTimeout).UnixNano())
	if err != nil {
		log.Printf("block height %d unstakes for fees lookup: %s", blockHeight, err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var txID, pool []byte
		if err := rows.Scan(&txID, &pool); err != nil {
			log.Printf("block height %d unstake for fee resolve: %s", blockHeight, err)
			continue
		}

		fees := l.feeQ[string(txID)]
		delete(l.feeQ, string(txID))

		for _, fee := range fees {
			if !event.IsRune(fee.Asset) {
				// Remove pool deduct from pool, which goes into pool reserve.
				// Asset fee should be counted on outbounds
				t.AddPoolRuneE8Depth(fee.Asset, -fee.PoolDeduct)
			} else {
				// Remove RUNE fee from pool, as it goes from pool into the pool reserve.
				t.AddPoolRuneE8Depth(pool, -fee.AssetE8)
			}
		}

		if err := rows.Err(); err != nil {
			log.Printf("block height %d unstakes for fees resolve: %s", blockHeight, err)
			return
		}
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
