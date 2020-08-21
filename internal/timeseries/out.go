package timeseries

import (
	"database/sql"
	"log"

	"gitlab.com/thorchain/midgard/event"
)

// TODO(pascaldekloe): Log with chain id [height] (from Metadata?).

// DBExec is used to write the data.
var DBExec func(query string, args ...interface{}) (sql.Result, error)

// EventListener is a singleton implementation using InfluxOut.
var EventListener event.Listener = eventListener{}

type eventListener struct{}

func (_ eventListener) OnAdd(e *event.Add, meta *event.Metadata) {
	const q = `INSERT INTO add_events (tx, chain, from_addr, to_addr, asset, asset_E8, memo, rune_E8, pool, block_timestamp)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := DBExec(q, e.Tx, e.Chain, e.FromAddr, e.ToAddr, e.Asset, e.AssetE8, e.Memo, e.RuneE8, e.Pool, meta.BlockTimestamp)
	if err != nil {
		log.Print("add event lost on ", err)
	}
}

func (_ eventListener) OnBond(e *event.Bond, meta *event.Metadata) {
	const q = `INSERT INTO bond_events (tx, chain, from_addr, to_addr, asset, asset_E8, memo, bound_type, E8, block_timestamp)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := DBExec(q, e.Tx, e.Chain, e.FromAddr, e.ToAddr, e.Asset, e.AssetE8, e.Memo, e.BoundType, e.E8, meta.BlockTimestamp)
	if err != nil {
		log.Print("bond event lost on ", err)
	}
}

func (_ eventListener) OnErrata(e *event.Errata, meta *event.Metadata) {
	log.Printf("GOT errata event as %#v", e)
}

func (_ eventListener) OnFee(e *event.Fee, meta *event.Metadata) {
	const q = `INSERT INTO fee_events (tx, asset, asset_E8, pool_deduct, block_timestamp)
VALUES ($1, $2, $3, $4, $5)`
	_, err := DBExec(q, e.Tx, e.Asset, e.AssetE8, e.PoolDeduct, meta.BlockTimestamp)
	if err != nil {
		log.Print("fee event lost on ", err)
	}
}

func (_ eventListener) OnGas(e *event.Gas, meta *event.Metadata) {
	const q = `INSERT INTO gas_events (asset, asset_E8, rune_E8, tx_count, block_timestamp)
VALUES ($1, $2, $3, $4, $5)`
	_, err := DBExec(q, e.Asset, e.AssetE8, e.RuneE8, e.TxCount, meta.BlockTimestamp)
	if err != nil {
		log.Print("gas event lost on ", err)
	}
}

func (_ eventListener) OnOutbound(e *event.Outbound, meta *event.Metadata) {
	const q = `INSERT INTO outbound_events (tx, chain, from_addr, to_addr, asset, asset_E8, memo, in_tx, block_timestamp)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := DBExec(q, e.Tx, e.Chain, e.FromAddr, e.ToAddr, e.Asset, e.AssetE8, e.Memo, e.InTx, meta.BlockTimestamp)
	if err != nil {
		log.Print("outound event lost on ", err)
	}
}

func (_ eventListener) OnPool(e *event.Pool, meta *event.Metadata) {
	const q = `INSERT INTO pool_events (asset, status, block_timestamp)
VALUES ($1, $2, $3)`
	_, err := DBExec(q, e.Asset, e.Status, meta.BlockTimestamp)
	if err != nil {
		log.Print("pool event lost on ", err)
	}
}

func (_ eventListener) OnRefund(e *event.Refund, meta *event.Metadata) {
	const q = `INSERT INTO refund_events (tx, chain, from_addr, to_addr, asset, asset_E8, memo, code, reason, block_timestamp)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := DBExec(q, e.Tx, e.Chain, e.FromAddr, e.ToAddr, e.Asset, e.AssetE8, e.Memo, e.Code, e.Reason, meta.BlockTimestamp)
	if err != nil {
		log.Print("refund event lost on ", err)
	}
}

func (_ eventListener) OnReserve(e *event.Reserve, meta *event.Metadata) {
	const q = `INSERT INTO reserve_events (tx, chain, from_addr, to_addr, asset, asset_E8, memo, addr, E8, block_timestamp)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := DBExec(q, e.Tx, e.Chain, e.FromAddr, e.ToAddr, e.Asset, e.AssetE8, e.Memo, e.Addr, e.E8, meta.BlockTimestamp)
	if err != nil {
		log.Print("reserve event lost on ", err)
	}
}

func (_ eventListener) OnRewards(e *event.Rewards, meta *event.Metadata) {
	log.Printf("GOT rewards event as %#v", e)
}

func (_ eventListener) OnSlash(e *event.Slash, meta *event.Metadata) {
	log.Printf("GOT slash event as %#v", e)
}

func (_ eventListener) OnStake(e *event.Stake, meta *event.Metadata) {
	const q = `INSERT INTO stake_events (pool, asset_tx, asset_chain, asset_E8, rune_tx, rune_addr, rune_E8, stake_units, block_timestamp)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := DBExec(q, e.Pool, e.AssetTx, e.AssetChain, e.AssetE8, e.RuneTx, e.RuneAddr, e.RuneE8, e.StakeUnits, meta.BlockTimestamp)
	if err != nil {
		log.Print("stake event lost on ", err)
	}
}

func (_ eventListener) OnSwap(e *event.Swap, meta *event.Metadata) {
	const q = `INSERT INTO swap_events (tx, chain, from_addr, to_addr, asset, asset_E8, memo, pool, price_target, trade_slip, liq_fee, liq_fee_in_rune, block_timestamp)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`
	_, err := DBExec(q, e.Tx, e.Chain, e.FromAddr, e.ToAddr, e.Asset, e.AssetE8, e.Memo, e.Pool, e.PriceTarget, e.TradeSlip, e.LiqFee, e.LiqFeeInRune, meta.BlockTimestamp)
	if err != nil {
		log.Print("swap event lost on ", err)
	}
}

func (_ eventListener) OnUnstake(e *event.Unstake, meta *event.Metadata) {
	const q = `INSERT INTO unstake_events (tx, chain, from_addr, to_addr, asset, asset_E8, memo, pool, stake_units, basis_points, asymmetry, block_timestamp)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	_, err := DBExec(q, e.Tx, e.Chain, e.FromAddr, e.ToAddr, e.Asset, e.AssetE8, e.Memo, e.Pool, e.StakeUnits, e.BasisPoints, e.Asymmetry, meta.BlockTimestamp)
	if err != nil {
		log.Print("unstake event lost on ", err)
	}
}
