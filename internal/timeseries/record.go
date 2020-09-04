package timeseries

import (
	"database/sql"
	"log"

	"gitlab.com/thorchain/midgard/event"
)

// EventListener is a singleton implementation which MUST be invoked seqentially
// in order of appearance.
var EventListener event.Listener = recorder

// Recorder gets initialised by Setup.
var recorder = new(eventRecorder)

type eventRecorder struct {
	runningTotals
	// applied on CommitBlock, including reset
	outboundTxIDs, refundTxIDs []string
}

func (l *eventRecorder) OnAdd(e *event.Add, meta *event.Metadata) {
	l.AddPoolAssetE8(e.Pool, e.AssetE8)
	l.AddPoolRuneE8(e.Pool, e.RuneE8)

	const q = `INSERT INTO add_events (tx, chain, from_addr, to_addr, asset, asset_E8, memo, rune_E8, pool, block_timestamp)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := DBExec(q, e.Tx, e.Chain, e.FromAddr, e.ToAddr, e.Asset, e.AssetE8, e.Memo, e.RuneE8, e.Pool, meta.BlockTimestamp.UnixNano())
	if err != nil {
		log.Printf("add event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (_ *eventRecorder) OnBond(e *event.Bond, meta *event.Metadata) {
	const q = `INSERT INTO bond_events (tx, chain, from_addr, to_addr, asset, asset_E8, memo, bound_type, E8, block_timestamp)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := DBExec(q, e.Tx, e.Chain, e.FromAddr, e.ToAddr, e.Asset, e.AssetE8, e.Memo, e.BoundType, e.E8, meta.BlockTimestamp.UnixNano())
	if err != nil {
		log.Printf("bond event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (l *eventRecorder) OnErrata(e *event.Errata, meta *event.Metadata) {
	l.AddPoolAssetE8(e.Asset, e.AssetE8)
	l.AddPoolRuneE8(e.Asset, e.RuneE8)

	const q = `INSERT INTO errata_events (in_tx, asset, asset_E8, rune_E8, block_timestamp)
VALUES ($1, $2, $3, $4, $5)`
	_, err := DBExec(q, e.InTx, e.Asset, e.AssetE8, e.RuneE8, meta.BlockTimestamp.UnixNano())
	if err != nil {
		log.Printf("errata event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (_ *eventRecorder) OnFee(e *event.Fee, meta *event.Metadata) {
	const q = `INSERT INTO fee_events (tx, asset, asset_E8, pool_deduct, block_timestamp)
VALUES ($1, $2, $3, $4, $5)`
	_, err := DBExec(q, e.Tx, e.Asset, e.AssetE8, e.PoolDeduct, meta.BlockTimestamp.UnixNano())
	if err != nil {
		log.Printf("fee event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (_ *eventRecorder) OnGas(e *event.Gas, meta *event.Metadata) {
	const q = `INSERT INTO gas_events (asset, asset_E8, rune_E8, tx_count, block_timestamp)
VALUES ($1, $2, $3, $4, $5)`
	_, err := DBExec(q, e.Asset, e.AssetE8, e.RuneE8, e.TxCount, meta.BlockTimestamp.UnixNano())
	if err != nil {
		log.Printf("gas event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (_ *eventRecorder) OnNewNode(e *event.NewNode, meta *event.Metadata) {
	const q = `INSERT INTO new_node_events (node_addr, block_timestamp)
VALUES ($1, $2)`
	_, err := DBExec(q, e.NodeAddr, meta.BlockTimestamp.UnixNano())
	if err != nil {
		log.Printf("new_node event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (l *eventRecorder) OnOutbound(e *event.Outbound, meta *event.Metadata) {
	l.outboundTxIDs = append(l.outboundTxIDs, string(e.Tx))

	var txVal interface{} = e.Tx
	if e.Tx == nil {
		txVal = sql.NullString{} // maybe just use null
	}

	const q = `INSERT INTO outbound_events (tx, chain, from_addr, to_addr, asset, asset_E8, memo, in_tx, block_timestamp)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := DBExec(q, txVal, e.Chain, e.FromAddr, e.ToAddr, e.Asset, e.AssetE8, e.Memo, e.InTx, meta.BlockTimestamp.UnixNano())
	if err != nil {
		log.Printf("outound event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (_ *eventRecorder) OnPool(e *event.Pool, meta *event.Metadata) {
	const q = `INSERT INTO pool_events (asset, status, block_timestamp)
VALUES ($1, $2, $3)`
	_, err := DBExec(q, e.Asset, e.Status, meta.BlockTimestamp.UnixNano())
	if err != nil {
		log.Printf("pool event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (l *eventRecorder) OnRefund(e *event.Refund, meta *event.Metadata) {
	l.refundTxIDs = append(l.refundTxIDs, string(e.Tx))

	const q = `INSERT INTO refund_events (tx, chain, from_addr, to_addr, asset, asset_E8, asset_2nd, asset_2nd_E8, memo, code, reason, block_timestamp)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	_, err := DBExec(q, e.Tx, e.Chain, e.FromAddr, e.ToAddr, e.Asset, e.AssetE8, e.Asset2nd, e.Asset2ndE8, e.Memo, e.Code, e.Reason, meta.BlockTimestamp.UnixNano())
	if err != nil {
		log.Printf("refund event from height %d lost on %s", meta.BlockHeight, err)
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

func (l *eventRecorder) OnRewards(e *event.Rewards, meta *event.Metadata) {
	const q = "INSERT INTO rewards_events (bond_E8, block_timestamp) VALUES ($1, $2)"
	_, err := DBExec(q, e.BondE8, meta.BlockTimestamp.UnixNano())
	if err != nil {
		log.Printf("reserve event from height %d lost on %s", meta.BlockHeight, err)
	}
	for _, a := range e.Pool {
		const q = "INSERT INTO rewards_pools (asset, asset_E8, block_timestamp) VALUES ($1, $2, $3)"
		_, err := DBExec(q, a.Asset, a.E8, meta.BlockTimestamp.UnixNano())
		if err != nil {
			log.Printf("reserve event pool from height %d lost on %s", meta.BlockHeight, err)
		}
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

func (l *eventRecorder) OnStake(e *event.Stake, meta *event.Metadata) {
	l.AddPoolAssetE8(e.Pool, e.AssetE8)
	l.AddPoolRuneE8(e.Pool, e.RuneE8)

	const q = `INSERT INTO stake_events (pool, asset_tx, asset_chain, asset_E8, rune_tx, rune_addr, rune_E8, stake_units, block_timestamp)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := DBExec(q, e.Pool, e.AssetTx, e.AssetChain, e.AssetE8, e.RuneTx, e.RuneAddr, e.RuneE8, e.StakeUnits, meta.BlockTimestamp.UnixNano())
	if err != nil {
		log.Printf("stake event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (l *eventRecorder) OnSwap(e *event.Swap, meta *event.Metadata) {
	if e.ToRune() {
		// Swap adds pool asset.
		l.AddPoolAssetE8(e.Pool, e.FromE8)
		// Swap deducts RUNE from pool with an event.Outbound.
	} else {
		// Swap adds RUNE to pool.
		l.AddPoolRuneE8(e.Pool, e.FromE8)
		// Swap deducts pool asset with an event.Outbound.
	}

	const q = `INSERT INTO swap_events (tx, chain, from_addr, to_addr, from_asset, from_E8, memo, pool, to_E8_min, trade_slip_BP, liq_fee_E8, liq_fee_in_rune_E8, block_timestamp)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`
	_, err := DBExec(q, e.Tx, e.Chain, e.FromAddr, e.ToAddr, e.FromAsset, e.FromE8, e.Memo, e.Pool, e.ToE8Min, e.TradeSlipBP, e.LiqFeeE8, e.LiqFeeInRuneE8, meta.BlockTimestamp.UnixNano())
	if err != nil {
		log.Printf("swap event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (l *eventRecorder) OnUnstake(e *event.Unstake, meta *event.Metadata) {
	if event.IsRune(e.Asset) {
		l.AddPoolRuneE8(e.Pool, -e.AssetE8)
	} else {
		l.AddPoolAssetE8(e.Pool, -e.AssetE8)
	}

	const q = `INSERT INTO unstake_events (tx, chain, from_addr, to_addr, asset, asset_E8, memo, pool, stake_units, basis_points, asymmetry, block_timestamp)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	_, err := DBExec(q, e.Tx, e.Chain, e.FromAddr, e.ToAddr, e.Asset, e.AssetE8, e.Memo, e.Pool, e.StakeUnits, e.BasisPoints, e.Asymmetry, meta.BlockTimestamp.UnixNano())
	if err != nil {
		log.Printf("unstake event from height %d lost on %s", meta.BlockHeight, err)
	}
}