package record

import (
	"bytes"
	"fmt"
	"strings"

	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/util/miderr"
)

// Empty prevents the SQL driver from writing NULL values.
var empty = []byte{}

// Recorder gets initialised by Setup.
var Recorder = &eventRecorder{
	runningTotals: *newRunningTotals(),
}

type eventRecorder struct {
	runningTotals
}

func (r *eventRecorder) OnActiveVault(e *ActiveVault, meta *Metadata) {
	const q = `INSERT INTO active_vault_events (add_asgard_addr, block_timestamp)
VALUES ($1, $2)`
	_, err := db.Exec(q, e.AddAsgardAddr, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("ActiveVault event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (r *eventRecorder) OnAdd(e *Add, meta *Metadata) {
	const q = `INSERT INTO add_events (tx, chain, from_addr, to_addr, asset, asset_E8, memo, rune_E8, pool, block_timestamp)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := db.Exec(q, e.Tx, e.Chain, e.FromAddr, e.ToAddr, e.Asset, e.AssetE8, e.Memo, e.RuneE8, e.Pool, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("add event from height %d lost on %s", meta.BlockHeight, err)
		return
	}

	r.AddPoolAssetE8Depth(e.Pool, e.AssetE8)
	r.AddPoolRuneE8Depth(e.Pool, e.RuneE8)
}

func (r *eventRecorder) OnAsgardFundYggdrasil(e *AsgardFundYggdrasil, meta *Metadata) {
	const q = `INSERT INTO asgard_fund_yggdrasil_events (tx, asset, asset_E8, vault_key, block_timestamp)
VALUES ($1, $2, $3, $4, $5)`
	_, err := db.Exec(q, e.Tx, e.Asset, e.AssetE8, e.VaultKey, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("asgard_fund_yggdrasil event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (_ *eventRecorder) OnBond(e *Bond, meta *Metadata) {
	const q = `INSERT INTO bond_events (tx, chain, from_addr, to_addr, asset, asset_E8, memo, bond_type, E8, block_timestamp)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := db.Exec(q, e.Tx, e.Chain, e.FromAddr, e.ToAddr, e.Asset, e.AssetE8, e.Memo, e.BondType, e.E8, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("bond event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (r *eventRecorder) OnErrata(e *Errata, meta *Metadata) {
	const q = `INSERT INTO errata_events (in_tx, asset, asset_E8, rune_E8, block_timestamp)
VALUES ($1, $2, $3, $4, $5)`
	_, err := db.Exec(q, e.InTx, e.Asset, e.AssetE8, e.RuneE8, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("errata event from height %d lost on %s", meta.BlockHeight, err)
		return
	}

	r.AddPoolAssetE8Depth(e.Asset, e.AssetE8)
	r.AddPoolRuneE8Depth(e.Asset, e.RuneE8)
}

func (r *eventRecorder) OnFee(e *Fee, meta *Metadata) {
	const q = `INSERT INTO fee_events (tx, asset, asset_E8, pool_deduct, block_timestamp)
VALUES ($1, $2, $3, $4, $5)`
	_, err := db.Exec(q, e.Tx, e.Asset, e.AssetE8, e.PoolDeduct, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("fee event from height %d lost on %s", meta.BlockHeight, err)
	}

	// NOTE: Fee applies to an outbound transaction amount and
	// is then sent to the reserve, so the pool is not involved in principle.
	// However, when the outbound amount is not RUNE,
	// the asset amount correspoinding to the fee is left in its pool,
	// and the RUNE equivalent is deducted from the pool's RUNE and sent to the reserve
	coinType := GetCoinType(e.Asset)
	pool := GetNativeAsset(e.Asset)
	if !IsRune(e.Asset) {
		if coinType == AssetNative {
			r.AddPoolAssetE8Depth(pool, e.AssetE8)
		}
		if coinType == AssetSynth {
			r.AddPoolSynthE8Depth(pool, -e.AssetE8)
		}
		r.AddPoolRuneE8Depth(pool, -e.PoolDeduct)
	}
}

func (r *eventRecorder) OnGas(e *Gas, meta *Metadata) {
	const q = `INSERT INTO gas_events (asset, asset_E8, rune_E8, tx_count, block_timestamp)
VALUES ($1, $2, $3, $4, $5)`
	_, err := db.Exec(q, e.Asset, e.AssetE8, e.RuneE8, e.TxCount, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("gas event from height %d lost on %s", meta.BlockHeight, err)
		return
	}

	r.AddPoolAssetE8Depth(e.Asset, -e.AssetE8)
	r.AddPoolRuneE8Depth(e.Asset, e.RuneE8)
}

func (r *eventRecorder) OnInactiveVault(e *InactiveVault, meta *Metadata) {
	const q = `INSERT INTO inactive_vault_events (add_asgard_addr, block_timestamp)
VALUES ($1, $2)`
	_, err := db.Exec(q, e.AddAsgardAddr, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("InactiveVault event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (_ *eventRecorder) OnMessage(e *Message, meta *Metadata) {
	if e.FromAddr == nil {
		e.FromAddr = empty
	}
	if e.Action == nil {
		e.Action = empty
	}
	const q = `INSERT INTO message_events (from_addr, action, block_timestamp)
VALUES ($1, $2, $3)`
	_, err := db.Exec(q, e.FromAddr, e.Action, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("message event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (_ *eventRecorder) OnNewNode(e *NewNode, meta *Metadata) {
	const q = `INSERT INTO new_node_events (node_addr, block_timestamp)
VALUES ($1, $2)`
	_, err := db.Exec(q, e.NodeAddr, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("new_node event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (r *eventRecorder) OnOutbound(e *Outbound, meta *Metadata) {
	const q = `INSERT INTO outbound_events (tx, chain, from_addr, to_addr, asset, asset_E8, memo, in_tx, block_timestamp)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := db.Exec(q, e.Tx, e.Chain, e.FromAddr, e.ToAddr, e.Asset, e.AssetE8, e.Memo, e.InTx, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("outound event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (r *eventRecorder) OnPool(e *Pool, meta *Metadata) {
	const q = `INSERT INTO pool_events (asset, status, block_timestamp)
VALUES ($1, $2, $3)`
	_, err := db.Exec(q, e.Asset, e.Status, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("pool event from height %d lost on %s", meta.BlockHeight, err)
	}
	if strings.ToLower(string(e.Status)) == "suspended" {
		pool := string(e.Asset)
		r.SetAssetDepth(pool, 0)
		r.SetRuneDepth(pool, 0)
	}
}

func (r *eventRecorder) OnRefund(e *Refund, meta *Metadata) {
	const q = `INSERT INTO refund_events (tx, chain, from_addr, to_addr, asset, asset_E8, asset_2nd, asset_2nd_E8, memo, code, reason, block_timestamp)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	_, err := db.Exec(q, e.Tx, e.Chain, e.FromAddr, e.ToAddr, e.Asset, e.AssetE8, e.Asset2nd, e.Asset2ndE8, e.Memo, e.Code, e.Reason, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("refund event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (_ *eventRecorder) OnReserve(e *Reserve, meta *Metadata) {
	const q = `INSERT INTO reserve_events (tx, chain, from_addr, to_addr, asset, asset_E8, memo, addr, E8, block_timestamp)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := db.Exec(q, e.Tx, e.Chain, e.FromAddr, e.ToAddr, e.Asset, e.AssetE8, e.Memo, e.Addr, e.E8, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("reserve event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (r *eventRecorder) OnRewards(e *Rewards, meta *Metadata) {
	blockTimestamp := meta.BlockTimestamp.UnixNano()
	const q = "INSERT INTO rewards_events (bond_E8, block_timestamp) VALUES ($1, $2)"
	_, err := db.Exec(q, e.BondE8, blockTimestamp)
	if err != nil {
		miderr.Printf("reserve event from height %d lost on %s", meta.BlockHeight, err)
		return
	}

	if len(e.PerPool) == 0 {
		return
	}

	// make batch insert work 🥴
	buf := bytes.NewBufferString("INSERT INTO rewards_event_entries (pool, rune_E8, block_timestamp) VALUES ")
	args := make([]interface{}, len(e.PerPool)*3)
	for i, p := range e.PerPool {
		fmt.Fprintf(buf, "($%d, $%d, $%d),", i*3+1, i*3+2, i*3+3)
		args[i*3], args[i*3+1], args[i*3+2] = p.Asset, p.E8, blockTimestamp
	}
	buf.Truncate(buf.Len() - 1) // last comma
	if _, err := db.Exec(buf.String(), args...); err != nil {
		miderr.Printf("reserve event pools from height %d lost on %s", meta.BlockHeight, err)
		return
	}

	for _, a := range e.PerPool {
		r.AddPoolRuneE8Depth(a.Asset, a.E8)
	}
}

func (_ *eventRecorder) OnSetIPAddress(e *SetIPAddress, meta *Metadata) {
	const q = `INSERT INTO set_ip_address_events (node_addr, ip_addr, block_timestamp)
VALUES ($1, $2, $3)`
	_, err := db.Exec(q, e.NodeAddr, e.IPAddr, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("set_ip_address event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (_ *eventRecorder) OnSetMimir(e *SetMimir, meta *Metadata) {
	const q = `INSERT INTO set_mimir_events (key, value, block_timestamp)
VALUES ($1, $2, $3)`
	_, err := db.Exec(q, e.Key, e.Value, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("set_mimir event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (_ *eventRecorder) OnSetNodeKeys(e *SetNodeKeys, meta *Metadata) {
	const q = `INSERT INTO set_node_keys_events (node_addr, secp256k1, ed25519, validator_consensus, block_timestamp)
VALUES ($1, $2, $3, $4, $5)`
	_, err := db.Exec(q, e.NodeAddr, string(e.Secp256k1), string(e.Ed25519), e.ValidatorConsensus, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("set_node_keys event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (_ *eventRecorder) OnSetVersion(e *SetVersion, meta *Metadata) {
	const q = `INSERT INTO set_version_events (node_addr, version, block_timestamp)
VALUES ($1, $2, $3)`
	_, err := db.Exec(q, e.NodeAddr, e.Version, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("set_version event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (r *eventRecorder) OnSlash(e *Slash, meta *Metadata) {
	if len(e.Amounts) == 0 {
		miderr.Printf("slash event on pool %q ignored: zero amounts", e.Pool)
	}
	for _, a := range e.Amounts {
		const q = "INSERT INTO slash_amounts (pool, asset, asset_E8, block_timestamp) VALUES ($1, $2, $3, $4)"
		_, err := db.Exec(q, e.Pool, a.Asset, a.E8, meta.BlockTimestamp.UnixNano())
		if err != nil {
			miderr.Printf("slash amount from height %d lost on %s", meta.BlockHeight, err)
		}
		coinType := GetCoinType(a.Asset)
		switch coinType {
		case Rune:
			r.AddPoolRuneE8Depth(e.Pool, a.E8)
		case AssetNative:
			r.AddPoolAssetE8Depth(e.Pool, a.E8)
		default:
			miderr.Printf("Unhandeled slash coin type: %s", a.Asset)
		}
	}
}

func (r *eventRecorder) OnPendingLiquidity(e *PendingLiquidity, meta *Metadata) {
	const q = `
		INSERT INTO pending_liquidity_events
			(pool,
				asset_tx, asset_chain, asset_addr, asset_E8,
				rune_tx, rune_addr, rune_E8,
				pending_type, block_timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := db.Exec(q, e.Pool,
		e.AssetTx, e.AssetChain, e.AssetAddr, e.AssetE8,
		e.RuneTx, e.RuneAddr, e.RuneE8,
		e.PendingType, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("pending_liquidity event from height %d lost on %s", meta.BlockHeight, err)
		return
	}
}

func (r *eventRecorder) OnStake(e *Stake, meta *Metadata) {
	const q = `INSERT INTO stake_events (pool, asset_tx, asset_chain, asset_addr, asset_E8, rune_tx, rune_addr, rune_E8, stake_units, block_timestamp)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := db.Exec(q, e.Pool, e.AssetTx, e.AssetChain, e.AssetAddr, e.AssetE8, e.RuneTx, e.RuneAddr, e.RuneE8, e.StakeUnits, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("stake event from height %d lost on %s", meta.BlockHeight, err)
		return
	}

	r.AddPoolAssetE8Depth(e.Pool, e.AssetE8)
	r.AddPoolRuneE8Depth(e.Pool, e.RuneE8)
}

func (r *eventRecorder) OnSwap(e *Swap, meta *Metadata) {
	fromCoin := GetCoinType(e.FromAsset)
	toCoin := GetCoinType(e.ToAsset)
	if fromCoin == UnknownCoin {
		miderr.Printf(
			"swap event from height %d lost - unknown from Coin %s",
			meta.BlockHeight, e.FromAsset)
		return
	}
	if toCoin == UnknownCoin {
		miderr.Printf(
			"swap event from height %d lost - unknown to Coin %s",
			meta.BlockHeight, e.ToAsset)
		return
	}
	if (fromCoin == Rune) == (toCoin == Rune) {
		miderr.Printf(
			"swap event from height %d lost - exactly one side should be Rune. fromCoin: %s toCoin: %s",
			meta.BlockHeight, e.FromAsset, e.ToAsset)
		return
	}
	const q = `INSERT INTO swap_events (tx, chain, from_addr, to_addr, from_asset, from_E8, to_asset, to_E8, memo, pool, to_E8_min, swap_slip_BP, liq_fee_E8, liq_fee_in_rune_E8, block_timestamp)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`
	_, err := db.Exec(q, e.Tx, e.Chain, e.FromAddr, e.ToAddr, e.FromAsset, e.FromE8, e.ToAsset, e.ToE8, e.Memo, e.Pool, e.ToE8Min, e.SwapSlipBP, e.LiqFeeE8, e.LiqFeeInRuneE8, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("swap event from height %d lost on %s", meta.BlockHeight, err)
		return
	}

	if toCoin == Rune {
		// Swap adds pool asset in exchange of RUNE.
		if fromCoin == AssetNative {
			r.AddPoolAssetE8Depth(e.Pool, e.FromE8)
		}
		// Swap burns synths in exchange of RUNE.
		if fromCoin == AssetSynth {
			r.AddPoolSynthE8Depth(e.Pool, -e.FromE8)
		}
		r.AddPoolRuneE8Depth(e.Pool, -e.ToE8)
	} else {
		// Swap adds RUNE to pool in exchange of asset.
		r.AddPoolRuneE8Depth(e.Pool, e.FromE8)
		if toCoin == AssetNative {
			r.AddPoolAssetE8Depth(e.Pool, -e.ToE8)
		}
		// Swap mints synths in exchange of RUNE.
		if toCoin == AssetSynth {
			r.AddPoolSynthE8Depth(e.Pool, e.ToE8)
		}
	}
}

func (_ *eventRecorder) OnTransfer(e *Transfer, meta *Metadata) {
	const q = `INSERT INTO transfer_events (from_addr, to_addr, asset, amount_E8, block_timestamp)
VALUES ($1, $2, $3, $4, $5)`
	_, err := db.Exec(q, e.FromAddr, e.ToAddr, e.Asset, e.AmountE8, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("transfer event from height %d lost on %s", meta.BlockHeight, err)
		return
	}
}

func (r *eventRecorder) OnUnstake(e *Unstake, meta *Metadata) {
	const q = `INSERT INTO unstake_events (tx, chain, from_addr, to_addr, asset, asset_E8, emit_asset_E8, emit_rune_E8, memo, pool, stake_units, basis_points, asymmetry, imp_loss_protection_E8, block_timestamp)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`
	_, err := db.Exec(q, e.Tx, e.Chain, e.FromAddr, e.ToAddr, e.Asset, e.AssetE8, e.EmitAssetE8, e.EmitRuneE8, e.Memo, e.Pool, e.StakeUnits, e.BasisPoints, e.Asymmetry, e.ImpLossProtectionE8, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("unstake event from height %d lost on %s", meta.BlockHeight, err)
	}
	// Rune/Asset withdrawn from pool
	r.AddPoolAssetE8Depth(e.Pool, -e.EmitAssetE8)
	r.AddPoolRuneE8Depth(e.Pool, -e.EmitRuneE8)

	// Rune added to pool from reserve as impermanent loss protection
	r.AddPoolRuneE8Depth(e.Pool, e.ImpLossProtectionE8)

	coinType := GetCoinType(e.Asset)
	if coinType == AssetNative && e.AssetE8 != 0 {
		// In order to initiate a withdraw the user needs to send a transaction with a memo.
		// On many asset chains one can't send 0 value, therefore they often send a small amount
		// of asset (e.g. 1e-8). Pools don't keep this amount, it's forwarded back and included in
		// the EmitAssetE8.
		// Therefore pool depth decreases with less then the EmitAssetE8, we correct it here.
		// Note: for Rune there is no minimum amount, if some rune is sent it's kept as donation.

		// TODO(muninn): clarify what to do with out of pool assets
		//   and replace this hack with final solution
		if string(e.Pool) == string(e.Asset) {
			r.AddPoolAssetE8Depth(e.Pool, e.AssetE8)
		}
	}
}

func (_ *eventRecorder) OnUpdateNodeAccountStatus(e *UpdateNodeAccountStatus, meta *Metadata) {
	if e.Former == nil {
		e.Former = empty
	}
	const q = `INSERT INTO update_node_account_status_events (node_addr, former, current, block_timestamp)
VALUES ($1, $2, $3, $4)`
	_, err := db.Exec(q, e.NodeAddr, e.Former, e.Current, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("UpdateNodeAccountStatus event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (_ *eventRecorder) OnValidatorRequestLeave(e *ValidatorRequestLeave, meta *Metadata) {
	const q = `INSERT INTO validator_request_leave_events (tx, from_addr, node_addr, block_timestamp)
VALUES ($1, $2, $3, $4)`
	_, err := db.Exec(q, e.Tx, e.FromAddr, e.NodeAddr, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("validator_request_leave event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (r *eventRecorder) OnPoolBalanceChange(e *PoolBalanceChange, meta *Metadata) {
	const q = `
		INSERT INTO pool_balance_change_events
			(asset, rune_amt, rune_add, asset_amt, asset_add, reason, block_timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := db.Exec(q,
		e.Asset, e.RuneAmt, e.RuneAdd, e.AssetAmt, e.AssetAdd, e.Reason,
		meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("pool_balance_change event from height %d lost on %s", meta.BlockHeight, err)
	}

	assetAmount := e.AssetAmt
	if assetAmount != 0 {
		if !e.AssetAdd {
			assetAmount *= -1
		}
		r.AddPoolAssetE8Depth(e.Asset, assetAmount)
	}
	runeAmount := e.RuneAmt
	if runeAmount != 0 {
		if !e.RuneAdd {
			runeAmount *= -1
		}
		r.AddPoolRuneE8Depth(e.Asset, runeAmount)
	}
}

func (r *eventRecorder) OnTHORNameChange(e *THORNameChange, meta *Metadata) {
	const q = `
		INSERT INTO thorname_change_events
			(name, chain, address, registration_fee_e8, fund_amount_e8, expire, owner, block_timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := db.Exec(q,
		e.Name, e.Chain, e.Address, e.RegistrationFeeE8, e.FundAmountE8, e.ExpireHeight, e.Owner,
		meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("thorname event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (r *eventRecorder) OnSwitch(e *Switch, meta *Metadata) {
	const q = `
		INSERT INTO switch_events
			(from_addr, to_addr, burn_asset, burn_E8, block_timestamp)
		VALUES ($1, $2, $3, $4, $5)`
	_, err := db.Exec(q,
		e.FromAddr, e.ToAddr, e.BurnAsset, e.BurnE8, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("switch event from height %d lost on %s", meta.BlockHeight, err)
	}
}
