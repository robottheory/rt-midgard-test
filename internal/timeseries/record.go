package timeseries

import (
	"bytes"
	"fmt"
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
	unstakeOutboundRune  = LinkedFound("outbound", "unstake", "rune")
	unstakeOutboundAsset = LinkedFound("outbound", "unstake", "asset")
	feeSwapRune          = LinkedFound("fee", "swap", "rune")
	feeSwapAsset         = LinkedFound("fee", "swap", "asset")
	deadOutbound         = LinkedDeads("outbound")
)

func init() {
	metrics.MustHelp("midgard_chain_event_linked_found", "Number of events with a matching input transaction ID.")
	metrics.MustHelp("midgard_chain_event_linked_deads", "Number of events with an unknown input transaction ID.")
}

// EventListener is a singleton implementation which MUST be invoked seqentially
// in order of appearance.
var EventListener event.Listener = recorder

// Recorder gets initialised by Setup.
var recorder = &eventRecorder{
	runningTotals: *newRunningTotals(),
	outbounds:     make(map[string][]event.Amount),
	fees:          make(map[string][]event.Amount),
	refunds:       make(map[string][]event.Amount),
}

type eventRecorder struct {
	runningTotals
	// see applyOutbounds and applyRefunds
	outbounds, fees, refunds map[string][]event.Amount
}

// ApplyOutbounds reads (and clears) .outbounds to gather information
// about the respective event.Outbound.InTx references.
func (r *eventRecorder) applyOutbounds(blockHeight int64, blockTimestamp time.Time) {
	if len(r.outbounds) == 0 {
		return
	}
	defer func() {
		// compiler optimised reset
		for txID := range r.outbounds {
			delete(recorder.outbounds, txID)
		}
	}()

	r.matchSwaps(blockHeight, blockTimestamp)
	r.matchUnstakes(blockHeight, blockTimestamp)

	deadOutbound.Add(uint64(len(recorder.outbounds)))
}

func (r *eventRecorder) matchSwaps(blockHeight int64, blockTimestamp time.Time) {
	if len(r.outbounds) == 0 {
		return
	}
	txIDs := make([]string, 0, len(r.outbounds))
	for s := range r.outbounds {
		txIDs = append(txIDs, s)
	}

	// filter outbounds for swap events
	const q = "SELECT tx, pool FROM swap_events WHERE tx = ANY($1) AND block_timestamp > $2"
	rows, err := DBQuery(q, txIDs, blockTimestamp.Add(-OutboundTimeout).UnixNano())
	if err != nil {
		log.Printf("block height %d swap outbounds lookup: %s", blockHeight, err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var txID, pool []byte
		if err := rows.Scan(&txID, &pool); err != nil {
			log.Printf("block height %d swap outbound resolve: %s", blockHeight, err)
			continue
		}
		amounts, ok := recorder.outbounds[string(txID)]
		if !ok {
			continue // already done (double-swap)
		}
		delete(recorder.outbounds, string(txID))

		for _, a := range amounts {
			// There's no clean way to distinguish between RUNE and
			// pool asset based solely on event data/relations.
			if event.IsRune(a.Asset) {
				swapOutboundToRune.Add(1)
				r.AddPoolRuneE8Depth(pool, -a.E8)
			} else {
				swapOutboundFromRune.Add(1)
				r.AddPoolAssetE8Depth(a.Asset, -a.E8)
			}
		}
	}
	if err := rows.Err(); err != nil {
		log.Printf("block height %d swap outbounds resolve: %s", blockHeight, err)
		return
	}
}

func (r *eventRecorder) matchUnstakes(blockHeight int64, blockTimestamp time.Time) {
	if len(r.outbounds) == 0 {
		return
	}
	txIDs := make([]string, 0, len(r.outbounds))
	for s := range r.outbounds {
		txIDs = append(txIDs, s)
	}

	// filter outbounds for unstake events
	const q = "SELECT tx, pool FROM unstake_events WHERE tx = ANY($1) AND block_timestamp > $2"
	rows, err := DBQuery(q, txIDs, blockTimestamp.Add(-OutboundTimeout).UnixNano())
	if err != nil {
		log.Printf("block height %d unstake outbounds lookup: %s", blockHeight, err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var txID, pool []byte
		if err := rows.Scan(&txID, &pool); err != nil {
			log.Printf("block height %d unstake outbound resolve: %s", blockHeight, err)
			continue
		}
		amounts := recorder.outbounds[string(txID)]
		delete(recorder.outbounds, string(txID))
		for _, a := range amounts {
			if bytes.Equal(a.Asset, pool) {
				unstakeOutboundAsset.Add(1)
				r.AddPoolRuneE8Depth(pool, -a.E8)
			} else {
				unstakeOutboundRune.Add(1)
				r.AddPoolAssetE8Depth(pool, -a.E8)
			}
		}
	}
	if err := rows.Err(); err != nil {
		log.Printf("block height %d unstake outbounds resolve: %s", blockHeight, err)
		return
	}
}

func (r *eventRecorder) applyFees(blockHeight int64, blockTimestamp time.Time) {
	if len(r.fees) == 0 {
		return
	}
	defer func() {
		// reset
		for txID := range r.fees {
			delete(recorder.fees, txID)
		}
	}()

	txIDs := make([]string, 0, len(r.fees))
	for s := range r.outbounds {
		txIDs = append(txIDs, s)
	}

	// filter swaps for fee events
	const q = "SELECT tx, pool FROM swap_events WHERE tx = ANY($1) AND block_timestamp > $2"
	rows, err := DBQuery(q, txIDs, blockTimestamp.Add(-OutboundTimeout).UnixNano())
	if err != nil {
		log.Printf("block height %d fee outbounds lookup: %s", blockHeight, err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var txID, pool []byte
		if err := rows.Scan(&txID, &pool); err != nil {
			log.Printf("block height %d fee outbound resolve: %s", blockHeight, err)
			continue
		}
		amounts := recorder.fees[string(txID)]
		delete(recorder.fees, string(txID))
		for _, a := range amounts {
			if bytes.Equal(a.Asset, pool) {
				feeSwapAsset.Add(1)
				r.AddPoolAssetE8Depth(pool, a.E8)
			} else {
				feeSwapRune.Add(1)
				r.AddPoolRuneE8Depth(pool, a.E8)
			}
		}
	}
	if err := rows.Err(); err != nil {
		log.Printf("block height %d fee outbounds resolve: %s", blockHeight, err)
		return
	}
}

func (r *eventRecorder) applyRefunds(blockHeight int64, blockTimestamp time.Time) {
	if len(r.refunds) == 0 {
		return
	}
	defer func() {
		// reset
		for txID := range r.refunds {
			delete(recorder.refunds, txID)
		}
	}()

	// BUG(pascaldekloe): Refunds are not taken into account with calculation.
}

func (r *eventRecorder) OnAdd(e *event.Add, meta *event.Metadata) {
	const q = `INSERT INTO add_events (tx, chain, from_addr, to_addr, asset, asset_E8, memo, rune_E8, pool, block_timestamp)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := DBExec(q, e.Tx, e.Chain, e.FromAddr, e.ToAddr, e.Asset, e.AssetE8, e.Memo, e.RuneE8, e.Pool, meta.BlockTimestamp.UnixNano())
	if err != nil {
		log.Printf("add event from height %d lost on %s", meta.BlockHeight, err)
		return
	}

	r.AddPoolAssetE8Depth(e.Pool, e.AssetE8)
	r.AddPoolRuneE8Depth(e.Pool, e.RuneE8)
}

func (_ *eventRecorder) OnBond(e *event.Bond, meta *event.Metadata) {
	const q = `INSERT INTO bond_events (tx, chain, from_addr, to_addr, asset, asset_E8, memo, bound_type, E8, block_timestamp)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := DBExec(q, e.Tx, e.Chain, e.FromAddr, e.ToAddr, e.Asset, e.AssetE8, e.Memo, e.BoundType, e.E8, meta.BlockTimestamp.UnixNano())
	if err != nil {
		log.Printf("bond event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (r *eventRecorder) OnErrata(e *event.Errata, meta *event.Metadata) {
	const q = `INSERT INTO errata_events (in_tx, asset, asset_E8, rune_E8, block_timestamp)
VALUES ($1, $2, $3, $4, $5)`
	_, err := DBExec(q, e.InTx, e.Asset, e.AssetE8, e.RuneE8, meta.BlockTimestamp.UnixNano())
	if err != nil {
		log.Printf("errata event from height %d lost on %s", meta.BlockHeight, err)
		return
	}

	r.AddPoolAssetE8Depth(e.Asset, e.AssetE8)
	r.AddPoolRuneE8Depth(e.Asset, e.RuneE8)
}

func (r *eventRecorder) OnFee(e *event.Fee, meta *event.Metadata) {
	const q = `INSERT INTO fee_events (tx, asset, asset_E8, pool_deduct, block_timestamp)
VALUES ($1, $2, $3, $4, $5)`
	_, err := DBExec(q, e.Tx, e.Asset, e.AssetE8, e.PoolDeduct, meta.BlockTimestamp.UnixNano())
	if err != nil {
		log.Printf("fee event from height %d lost on %s", meta.BlockHeight, err)
	}

	r.fees[string(e.Tx)] = append(r.fees[string(e.Tx)],
		event.Amount{Asset: e.Asset, E8: e.AssetE8})
}

func (r *eventRecorder) OnGas(e *event.Gas, meta *event.Metadata) {
	const q = `INSERT INTO gas_events (asset, asset_E8, rune_E8, tx_count, block_timestamp)
VALUES ($1, $2, $3, $4, $5)`
	_, err := DBExec(q, e.Asset, e.AssetE8, e.RuneE8, e.TxCount, meta.BlockTimestamp.UnixNano())
	if err != nil {
		log.Printf("gas event from height %d lost on %s", meta.BlockHeight, err)
		return
	}

	r.AddPoolAssetE8Depth(e.Asset, e.AssetE8)
	r.AddPoolRuneE8Depth(e.Asset, e.RuneE8)
}

func (_ *eventRecorder) OnNewNode(e *event.NewNode, meta *event.Metadata) {
	const q = `INSERT INTO new_node_events (node_addr, block_timestamp)
VALUES ($1, $2)`
	_, err := DBExec(q, e.NodeAddr, meta.BlockTimestamp.UnixNano())
	if err != nil {
		log.Printf("new_node event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (r *eventRecorder) OnOutbound(e *event.Outbound, meta *event.Metadata) {
	const q = `INSERT INTO outbound_events (tx, chain, from_addr, to_addr, asset, asset_E8, memo, in_tx, block_timestamp)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := DBExec(q, e.Tx, e.Chain, e.FromAddr, e.ToAddr, e.Asset, e.AssetE8, e.Memo, e.InTx, meta.BlockTimestamp.UnixNano())
	if err != nil {
		log.Printf("outound event from height %d lost on %s", meta.BlockHeight, err)
	}

	r.outbounds[string(e.InTx)] = append(r.outbounds[string(e.InTx)],
		event.Amount{Asset: e.Asset, E8: e.AssetE8})
}

func (_ *eventRecorder) OnPool(e *event.Pool, meta *event.Metadata) {
	const q = `INSERT INTO pool_events (asset, status, block_timestamp)
VALUES ($1, $2, $3)`
	_, err := DBExec(q, e.Asset, e.Status, meta.BlockTimestamp.UnixNano())
	if err != nil {
		log.Printf("pool event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (r *eventRecorder) OnRefund(e *event.Refund, meta *event.Metadata) {
	const q = `INSERT INTO refund_events (tx, chain, from_addr, to_addr, asset, asset_E8, asset_2nd, asset_2nd_E8, memo, code, reason, block_timestamp)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	_, err := DBExec(q, e.Tx, e.Chain, e.FromAddr, e.ToAddr, e.Asset, e.AssetE8, e.Asset2nd, e.Asset2ndE8, e.Memo, e.Code, e.Reason, meta.BlockTimestamp.UnixNano())
	if err != nil {
		log.Printf("refund event from height %d lost on %s", meta.BlockHeight, err)
	}

	if e.Asset2nd == nil {
		r.refunds[string(e.Tx)] = append(r.refunds[string(e.Tx)],
			event.Amount{Asset: e.Asset, E8: e.AssetE8})
	} else {
		r.refunds[string(e.Tx)] = append(r.refunds[string(e.Tx)],
			event.Amount{Asset: e.Asset, E8: e.AssetE8},
			event.Amount{Asset: e.Asset2nd, E8: e.Asset2ndE8})
	}
}

func (_ *eventRecorder) OnReserve(e *event.Reserve, meta *event.Metadata) {
	const q = `INSERT INTO reserve_events (tx, chain, from_addr, to_addr, asset, asset_E8, memo, addr, E8, block_timestamp)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := DBExec(q, e.Tx, e.Chain, e.FromAddr, e.ToAddr, e.Asset, e.AssetE8, e.Memo, e.Addr, e.E8, meta.BlockTimestamp.UnixNano())
	if err != nil {
		log.Printf("reserve event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (r *eventRecorder) OnRewards(e *event.Rewards, meta *event.Metadata) {
	blockTimestamp := meta.BlockTimestamp.UnixNano()
	const q = "INSERT INTO rewards_events (bond_E8, block_timestamp) VALUES ($1, $2)"
	_, err := DBExec(q, e.BondE8, blockTimestamp)
	if err != nil {
		log.Printf("reserve event from height %d lost on %s", meta.BlockHeight, err)
		return
	}

	if len(e.Pool) == 0 {
		return
	}

	// make batch insert work 🥴
	buf := bytes.NewBufferString("INSERT INTO rewards_pools (asset, asset_E8, block_timestamp) VALUES ")
	args := make([]interface{}, len(e.Pool)*3)
	for i, p := range e.Pool {
		fmt.Fprintf(buf, "($%d, $%d, $%d),", i*3+1, i*3+2, i*3+3)
		args[i*3], args[i*3+1], args[i*3+2] = p.Asset, p.E8, blockTimestamp
	}
	buf.Truncate(buf.Len() - 1) // last comma
	if _, err := DBExec(buf.String(), args...); err != nil {
		log.Printf("reserve event pools from height %d lost on %s", meta.BlockHeight, err)
	}

	for _, a := range e.Pool {
		r.AddPoolRuneE8Depth(a.Asset, a.E8)
	}
}

func (_ *eventRecorder) OnSetIPAddress(e *event.SetIPAddress, meta *event.Metadata) {
	const q = `INSERT INTO set_ip_address_events (node_addr, ip_addr, block_timestamp)
VALUES ($1, $2, $3)`
	_, err := DBExec(q, e.NodeAddr, e.IPAddr, meta.BlockTimestamp.UnixNano())
	if err != nil {
		log.Printf("set_ip_address event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (_ *eventRecorder) OnSetNodeKeys(e *event.SetNodeKeys, meta *event.Metadata) {
	const q = `INSERT INTO set_node_keys_events (node_addr, secp256k1, ed25519, validator_consensus, block_timestamp)
VALUES ($1, $2, $3, $4, $5)`
	_, err := DBExec(q, e.NodeAddr, e.Secp256k1, e.Ed25519, e.ValidatorConsensus, meta.BlockTimestamp.UnixNano())
	if err != nil {
		log.Printf("set_node_keys event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (_ *eventRecorder) OnSetVersion(e *event.SetVersion, meta *event.Metadata) {
	const q = `INSERT INTO set_version_events (node_addr, version, block_timestamp)
VALUES ($1, $2, $3)`
	_, err := DBExec(q, e.NodeAddr, e.Version, meta.BlockTimestamp.UnixNano())
	if err != nil {
		log.Printf("set_version event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (_ *eventRecorder) OnSlash(e *event.Slash, meta *event.Metadata) {
	if len(e.Amounts) == 0 {
		log.Printf("slash event on pool %q ignored: zero amounts", e.Pool)
	}
	for _, a := range e.Amounts {
		const q = "INSERT INTO slash_amounts (pool, asset, asset_E8, block_timestamp) VALUES ($1, $2, $3, $4)"
		_, err := DBExec(q, e.Pool, a.Asset, a.E8, meta.BlockTimestamp.UnixNano())
		if err != nil {
			log.Printf("slash amount from height %d lost on %s", meta.BlockHeight, err)
		}
	}
}

func (r *eventRecorder) OnStake(e *event.Stake, meta *event.Metadata) {
	const q = `INSERT INTO stake_events (pool, asset_tx, asset_chain, asset_E8, rune_tx, rune_addr, rune_E8, stake_units, block_timestamp)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := DBExec(q, e.Pool, e.AssetTx, e.AssetChain, e.AssetE8, e.RuneTx, e.RuneAddr, e.RuneE8, e.StakeUnits, meta.BlockTimestamp.UnixNano())
	if err != nil {
		log.Printf("stake event from height %d lost on %s", meta.BlockHeight, err)
		return
	}

	r.AddPoolAssetE8Depth(e.Pool, e.AssetE8)
	r.AddPoolRuneE8Depth(e.Pool, e.RuneE8)
}

func (r *eventRecorder) OnSwap(e *event.Swap, meta *event.Metadata) {
	const q = `INSERT INTO swap_events (tx, chain, from_addr, to_addr, from_asset, from_E8, memo, pool, to_E8_min, trade_slip_BP, liq_fee_E8, liq_fee_in_rune_E8, block_timestamp)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`
	_, err := DBExec(q, e.Tx, e.Chain, e.FromAddr, e.ToAddr, e.FromAsset, e.FromE8, e.Memo, e.Pool, e.ToE8Min, e.TradeSlipBP, e.LiqFeeE8, e.LiqFeeInRuneE8, meta.BlockTimestamp.UnixNano())
	if err != nil {
		log.Printf("swap event from height %d lost on %s", meta.BlockHeight, err)
		return
	}

	if e.ToRune() {
		// Swap adds pool asset.
		r.AddPoolAssetE8Depth(e.Pool, e.FromE8)
		// Swap deducts RUNE from pool with an event.Outbound.
	} else {
		// Swap adds RUNE to pool.
		r.AddPoolRuneE8Depth(e.Pool, e.FromE8)
		// Swap deducts pool asset with an event.Outbound.
	}
}

func (r *eventRecorder) OnUnstake(e *event.Unstake, meta *event.Metadata) {
	const q = `INSERT INTO unstake_events (tx, chain, from_addr, to_addr, asset, asset_E8, memo, pool, stake_units, basis_points, asymmetry, block_timestamp)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	_, err := DBExec(q, e.Tx, e.Chain, e.FromAddr, e.ToAddr, e.Asset, e.AssetE8, e.Memo, e.Pool, e.StakeUnits, e.BasisPoints, e.Asymmetry, meta.BlockTimestamp.UnixNano())
	if err != nil {
		log.Printf("unstake event from height %d lost on %s", meta.BlockHeight, err)
		return
	}
}
