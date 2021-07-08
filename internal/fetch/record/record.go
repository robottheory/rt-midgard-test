package record

import (
	"strings"

	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/util/miderr"
)

func AddressIsRune(address string) bool {
	return strings.HasPrefix(address, "thor") || strings.HasPrefix(address, "tthor")
}

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
	q := []string{"add_asgard_addr", "block_timestamp"}
	err := db.Inserter.Insert("active_vault_events", q, e.AddAsgardAddr, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("ActiveVault event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (r *eventRecorder) OnAdd(e *Add, meta *Metadata) {
	q := []string{"tx", "chain", "from_addr", "to_addr", "asset", "asset_e8", "memo", "rune_e8", "pool", "block_timestamp"}
	err := db.Inserter.Insert("add_events", q, e.Tx, e.Chain, e.FromAddr, e.ToAddr, e.Asset, e.AssetE8, e.Memo, e.RuneE8, e.Pool, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("add event from height %d lost on %s", meta.BlockHeight, err)
		return
	}

	r.AddPoolAssetE8Depth(e.Pool, e.AssetE8)
	r.AddPoolRuneE8Depth(e.Pool, e.RuneE8)
}

func (r *eventRecorder) OnAsgardFundYggdrasil(e *AsgardFundYggdrasil, meta *Metadata) {
	q := []string{"tx", "asset", "asset_e8", "vault_key", "block_timestamp"}
	err := db.Inserter.Insert("asgard_fund_yggdrasil_events", q, e.Tx, e.Asset, e.AssetE8, e.VaultKey, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("asgard_fund_yggdrasil event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (_ *eventRecorder) OnBond(e *Bond, meta *Metadata) {
	q := []string{"tx", "chain", "from_addr", "to_addr", "asset", "asset_e8", "memo", "bond_type", "e8", "block_timestamp"}
	err := db.Inserter.Insert("bond_events", q, e.Tx, e.Chain, e.FromAddr, e.ToAddr, e.Asset, e.AssetE8, e.Memo, e.BondType, e.E8, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("bond event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (r *eventRecorder) OnErrata(e *Errata, meta *Metadata) {
	q := []string{"in_tx", "asset", "asset_e8", "rune_e8", "block_timestamp"}
	err := db.Inserter.Insert("errata_events", q, e.InTx, e.Asset, e.AssetE8, e.RuneE8, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("errata event from height %d lost on %s", meta.BlockHeight, err)
		return
	}

	r.AddPoolAssetE8Depth(e.Asset, e.AssetE8)
	r.AddPoolRuneE8Depth(e.Asset, e.RuneE8)
}

func (r *eventRecorder) OnFee(e *Fee, meta *Metadata) {
	q := []string{"tx", "asset", "asset_e8", "pool_deduct", "block_timestamp"}
	err := db.Inserter.Insert("fee_events", q, e.Tx, e.Asset, e.AssetE8, e.PoolDeduct, meta.BlockTimestamp.UnixNano())
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
	q := []string{"asset", "asset_e8", "rune_e8", "tx_count", "block_timestamp"}
	err := db.Inserter.Insert("gas_events", q, e.Asset, e.AssetE8, e.RuneE8, e.TxCount, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("gas event from height %d lost on %s", meta.BlockHeight, err)
		return
	}

	r.AddPoolAssetE8Depth(e.Asset, -e.AssetE8)
	r.AddPoolRuneE8Depth(e.Asset, e.RuneE8)
}

func (r *eventRecorder) OnInactiveVault(e *InactiveVault, meta *Metadata) {
	q := []string{"add_asgard_addr", "block_timestamp"}
	err := db.Inserter.Insert("inactive_vault_events", q, e.AddAsgardAddr, meta.BlockTimestamp.UnixNano())
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
	q := []string{"from_addr", "action", "block_timestamp"}
	err := db.Inserter.Insert("message_events", q, e.FromAddr, e.Action, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("message event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (_ *eventRecorder) OnNewNode(e *NewNode, meta *Metadata) {
	q := []string{"node_addr", "block_timestamp"}
	err := db.Inserter.Insert("new_node_events", q, e.NodeAddr, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("new_node event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (r *eventRecorder) OnOutbound(e *Outbound, meta *Metadata) {
	q := []string{"tx", "chain", "from_addr", "to_addr", "asset", "asset_e8", "memo", "in_tx", "block_timestamp"}
	err := db.Inserter.Insert("outbound_events", q, e.Tx, e.Chain, e.FromAddr, e.ToAddr, e.Asset, e.AssetE8, e.Memo, e.InTx, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("outound event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (r *eventRecorder) OnPool(e *Pool, meta *Metadata) {
	q := []string{"asset", "status", "block_timestamp"}
	err := db.Inserter.Insert("pool_events", q, e.Asset, e.Status, meta.BlockTimestamp.UnixNano())
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
	q := []string{"tx", "chain", "from_addr", "to_addr", "asset", "asset_e8", "asset_2nd", "asset_2nd_e8", "memo", "code", "reason", "block_timestamp"}
	err := db.Inserter.Insert("refund_events", q, e.Tx, e.Chain, e.FromAddr, e.ToAddr, e.Asset, e.AssetE8, e.Asset2nd, e.Asset2ndE8, e.Memo, e.Code, e.Reason, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("refund event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (_ *eventRecorder) OnReserve(e *Reserve, meta *Metadata) {
	q := []string{"tx", "chain", "from_addr", "to_addr", "asset", "asset_e8", "memo", "addr", "e8", "block_timestamp"}
	err := db.Inserter.Insert("reserve_events", q, e.Tx, e.Chain, e.FromAddr, e.ToAddr, e.Asset, e.AssetE8, e.Memo, e.Addr, e.E8, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("reserve event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (r *eventRecorder) OnRewards(e *Rewards, meta *Metadata) {
	blockTimestamp := meta.BlockTimestamp.UnixNano()
	q := []string{"bond_e8", "block_timestamp"}
	err := db.Inserter.Insert("rewards_events", q, e.BondE8, blockTimestamp)
	if err != nil {
		miderr.Printf("reserve event from height %d lost on %s", meta.BlockHeight, err)
		return
	}

	if len(e.PerPool) == 0 {
		return
	}

	q2 := []string{"pool", "rune_e8", "block_timestamp"}
	for _, p := range e.PerPool {
		err := db.Inserter.Insert("rewards_event_entries", q2, p.Asset, p.E8, blockTimestamp)
		if err != nil {
			miderr.Printf("reserve event pools from height %d lost on %s", meta.BlockHeight, err)
			return
		}
	}

	for _, a := range e.PerPool {
		r.AddPoolRuneE8Depth(a.Asset, a.E8)
	}
}

func (_ *eventRecorder) OnSetIPAddress(e *SetIPAddress, meta *Metadata) {
	q := []string{"node_addr", "ip_addr", "block_timestamp"}
	err := db.Inserter.Insert("set_ip_address_events", q, e.NodeAddr, e.IPAddr, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("set_ip_address event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (_ *eventRecorder) OnSetMimir(e *SetMimir, meta *Metadata) {
	q := []string{"key", "value", "block_timestamp"}
	err := db.Inserter.Insert("set_mimir_events", q, e.Key, e.Value, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("set_mimir event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (_ *eventRecorder) OnSetNodeKeys(e *SetNodeKeys, meta *Metadata) {
	q := []string{"node_addr", "secp256k1", "ed25519", "validator_consensus", "block_timestamp"}
	err := db.Inserter.Insert("set_node_keys_events", q, e.NodeAddr, string(e.Secp256k1), string(e.Ed25519), e.ValidatorConsensus, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("set_node_keys event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (_ *eventRecorder) OnSetVersion(e *SetVersion, meta *Metadata) {
	q := []string{"node_addr", "version", "block_timestamp"}
	err := db.Inserter.Insert("set_version_events", q, e.NodeAddr, e.Version, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("set_version event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (r *eventRecorder) OnSlash(e *Slash, meta *Metadata) {
	if len(e.Amounts) == 0 {
		miderr.Printf("slash event on pool %q ignored: zero amounts", e.Pool)
	}
	for _, a := range e.Amounts {
		q := []string{"pool", "asset", "asset_e8", "block_timestamp"}
		err := db.Inserter.Insert("slash_amounts", q, e.Pool, a.Asset, a.E8, meta.BlockTimestamp.UnixNano())
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
	q := []string{"pool", "asset_tx", "asset_chain", "asset_addr", "asset_e8", "rune_tx", "rune_addr", "rune_e8", "pending_type", "block_timestamp"}
	err := db.Inserter.Insert("pending_liquidity_events", q, e.Pool,
		e.AssetTx, e.AssetChain, e.AssetAddr, e.AssetE8,
		e.RuneTx, e.RuneAddr, e.RuneE8,
		e.PendingType, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("pending_liquidity event from height %d lost on %s", meta.BlockHeight, err)
		return
	}
}

func (r *eventRecorder) OnStake(e *Stake, meta *Metadata) {
	q := []string{"pool", "asset_tx", "asset_chain", "asset_addr", "asset_e8", "stake_units", "rune_tx", "rune_addr", "rune_e8", "block_timestamp"}
	err := db.Inserter.Insert("stake_events", q, e.Pool, e.AssetTx, e.AssetChain, e.AssetAddr, e.AssetE8, e.StakeUnits, e.RuneTx, e.RuneAddr, e.RuneE8, meta.BlockTimestamp.UnixNano())
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
	q := []string{"tx", "chain", "from_addr", "to_addr", "from_asset", "from_e8", "to_asset", "to_e8", "memo", "pool", "to_e8_min", "swap_slip_bp", "liq_fee_e8", "liq_fee_in_rune_e8", "block_timestamp"}
	err := db.Inserter.Insert("swap_events", q, e.Tx, e.Chain, e.FromAddr, e.ToAddr, e.FromAsset, e.FromE8, e.ToAsset, e.ToE8, e.Memo, e.Pool, e.ToE8Min, e.SwapSlipBP, e.LiqFeeE8, e.LiqFeeInRuneE8, meta.BlockTimestamp.UnixNano())
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
	q := []string{"from_addr", "to_addr", "asset", "amount_e8", "block_timestamp"}
	err := db.Inserter.Insert("transfer_events", q, e.FromAddr, e.ToAddr, e.Asset, e.AmountE8, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("transfer event from height %d lost on %s", meta.BlockHeight, err)
		return
	}
}

func (r *eventRecorder) OnUnstake(e *Unstake, meta *Metadata) {
	q := []string{"tx", "chain", "from_addr", "to_addr", "asset", "asset_e8", "emit_asset_e8", "emit_rune_e8", "memo", "pool", "stake_units", "basis_points", "asymmetry", "imp_loss_protection_e8", "block_timestamp"}
	err := db.Inserter.Insert("unstake_events", q, e.Tx, e.Chain, e.FromAddr, e.ToAddr, e.Asset, e.AssetE8, e.EmitAssetE8, e.EmitRuneE8, e.Memo, e.Pool, e.StakeUnits, e.BasisPoints, e.Asymmetry, e.ImpLossProtectionE8, meta.BlockTimestamp.UnixNano())
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
	q := []string{"node_addr", "former", "current", "block_timestamp"}
	err := db.Inserter.Insert("update_node_account_status_events", q, e.NodeAddr, e.Former, e.Current, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("UpdateNodeAccountStatus event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (_ *eventRecorder) OnValidatorRequestLeave(e *ValidatorRequestLeave, meta *Metadata) {
	q := []string{"tx", "from_addr", "node_addr", "block_timestamp"}
	err := db.Inserter.Insert("validator_request_leave_events", q, e.Tx, e.FromAddr, e.NodeAddr, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("validator_request_leave event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (r *eventRecorder) OnPoolBalanceChange(e *PoolBalanceChange, meta *Metadata) {
	q := []string{"asset", "rune_amt", "rune_add", "asset_amt", "asset_add", "reason", "block_timestamp"}
	err := db.Inserter.Insert("pool_balance_change_events", q,
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
	q := []string{"name", "chain", "address", "registration_fee_e8", "fund_amount_e8", "expire", "owner", "block_timestamp"}
	err := db.Inserter.Insert("thorname_change_events", q,
		e.Name, e.Chain, e.Address, e.RegistrationFeeE8, e.FundAmountE8, e.ExpireHeight, e.Owner,
		meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("thorname event from height %d lost on %s", meta.BlockHeight, err)
	}
}

func (r *eventRecorder) OnSwitch(e *Switch, meta *Metadata) {
	q := []string{"tx", "from_addr", "to_addr", "burn_asset", "burn_e8", "block_timestamp"}
	err := db.Inserter.Insert("switch_events", q,
		e.Tx, e.FromAddr, e.ToAddr, e.BurnAsset, e.BurnE8, meta.BlockTimestamp.UnixNano())
	if err != nil {
		miderr.Printf("switch event from height %d lost on %s", meta.BlockHeight, err)
	}
}
