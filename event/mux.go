package event

import (
	"errors"
	"time"

	"github.com/pascaldekloe/metrics"

	abci "github.com/tendermint/tendermint/abci/types"

	"gitlab.com/thorchain/midgard/internal/fetch/chain"
	"gitlab.com/thorchain/midgard/internal/util/miderr"
	"gitlab.com/thorchain/midgard/internal/util/timer"
)

// Package Metrics
var (
	blockProcTimer = timer.NewNano("block_write_process")
	EventProcTime  = metrics.Must1LabelHistogram("midgard_chain_event_process_seconds", "type", 0.001, 0.01, 0.1)

	EventTotal            = metrics.Must1LabelCounter("midgard_chain_events_total", "group")
	DeliverTxEventsTotal  = EventTotal("deliver_tx")
	BeginBlockEventsTotal = EventTotal("begin_block")
	EndBlockEventsTotal   = EventTotal("end_block")
	IgnoresTotal          = metrics.MustCounter("midgard_chain_event_ignores_total", "Number of known types not in use seen.")
	UnknownsTotal         = metrics.MustCounter("midgard_chain_event_unknowns_total", "Number of unknown types discarded.")

	AttrPerEvent = metrics.MustHistogram("midgard_chain_event_attrs", "Number of attributes per event.", 0, 1, 7, 21, 144)

	PoolRewardsTotal = metrics.MustCounter("midgard_pool_rewards_total", "Number of asset amounts on rewards events seen.")
)

// Metadata has metadata for a block (from the chain).
type Metadata struct {
	BlockHeight    int64     // Tendermint sequence identifier
	BlockTimestamp time.Time // official acceptance moment
}

// Listener defines an event callback.
type Listener interface {
	OnActiveVault(*ActiveVault, *Metadata)
	OnAdd(*Add, *Metadata)
	OnAsgardFundYggdrasil(*AsgardFundYggdrasil, *Metadata)
	OnBond(*Bond, *Metadata)
	OnErrata(*Errata, *Metadata)
	OnFee(*Fee, *Metadata)
	OnGas(*Gas, *Metadata)
	OnInactiveVault(*InactiveVault, *Metadata)
	OnMessage(*Message, *Metadata)
	OnNewNode(*NewNode, *Metadata)
	OnOutbound(*Outbound, *Metadata)
	OnPool(*Pool, *Metadata)
	OnRefund(*Refund, *Metadata)
	OnReserve(*Reserve, *Metadata)
	OnRewards(*Rewards, *Metadata)
	OnSetIPAddress(*SetIPAddress, *Metadata)
	OnSetMimir(*SetMimir, *Metadata)
	OnSetNodeKeys(*SetNodeKeys, *Metadata)
	OnSetVersion(*SetVersion, *Metadata)
	OnSlash(*Slash, *Metadata)
	OnStake(*Stake, *Metadata)
	OnSwap(*Swap, *Metadata)
	OnTransfer(*Transfer, *Metadata)
	OnUnstake(*Unstake, *Metadata)
	OnUpdateNodeAccountStatus(*UpdateNodeAccountStatus, *Metadata)
	OnValidatorRequestLeave(*ValidatorRequestLeave, *Metadata)
}

// Demux is a demultiplexer for events from the blockchain.
type Demux struct {
	// Listener is the output destination.
	// Implementations MAY NOT retain any of the events provided.
	Listener

	// prevent memory allocation
	reuse struct {
		ActiveVault
		Add
		AsgardFundYggdrasil
		Bond
		Errata
		Fee
		Gas
		InactiveVault
		Message
		NewNode
		Outbound
		Pool
		Refund
		Reserve
		Rewards
		SetIPAddress
		SetMimir
		SetNodeKeys
		SetVersion
		Slash
		Stake
		Swap
		Transfer
		Unstake
		UpdateNodeAccountStatus
		ValidatorRequestLeave
	}
}

// Block invokes Listener for each transaction event in block.
func (d *Demux) Block(block chain.Block) {
	defer blockProcTimer.One()()

	m := Metadata{
		BlockHeight:    block.Height,
		BlockTimestamp: block.Time,
	}

	// “The BeginBlock ABCI message is sent from the underlying Tendermint
	// engine when a block proposal created by the correct proposer is
	// received, before DeliverTx is run for each transaction in the block.
	// It allows developers to have logic be executed at the beginning of
	// each block.”
	// — https://docs.cosmos.network/master/core/baseapp.html#beginblock
	BeginBlockEventsTotal.Add(uint64(len(block.Results.BeginBlockEvents)))
	for eventIndex, event := range block.Results.BeginBlockEvents {
		if err := d.event(event, &m); err != nil {
			miderr.Printf("block height %d begin event %d type %q skipped: %s",
				block.Height, eventIndex, event.Type, err)
		}
	}

	for txIndex, tx := range block.Results.TxsResults {
		DeliverTxEventsTotal.Add(uint64(len(tx.Events)))
		for eventIndex, event := range tx.Events {
			if err := d.event(event, &m); err != nil {
				miderr.Printf("block height %d tx %d event %d type %q skipped: %s",
					block.Height, txIndex, eventIndex, event.Type, err)
			}
		}
	}

	// “The EndBlock ABCI message is sent from the underlying Tendermint
	// engine after DeliverTx as been run for each transaction in the block.
	// It allows developers to have logic be executed at the end of each
	// block.”
	// — https://docs.cosmos.network/master/core/baseapp.html#endblock
	EndBlockEventsTotal.Add(uint64(len(block.Results.EndBlockEvents)))
	for eventIndex, event := range block.Results.EndBlockEvents {
		if err := d.event(event, &m); err != nil {
			miderr.Printf("block height %d end event %d type %q skipped: %s",
				block.Height, eventIndex, event.Type, err)
		}
	}
}

var errEventType = errors.New("unknown event type")

// Block notifies Listener for the transaction event.
// Errors do not include the event type in the message.
func (d *Demux) event(event abci.Event, meta *Metadata) error {
	defer EventProcTime(event.Type).AddSince(time.Now())

	attrs := event.Attributes
	AttrPerEvent.Add(float64(len(attrs)))

	switch event.Type {
	case "ActiveVault":
		if err := d.reuse.ActiveVault.LoadTendermint(attrs); err != nil {
			return err
		}
		d.Listener.OnActiveVault(&d.reuse.ActiveVault, meta)
	case "donate":
		// TODO(acsaba): rename add to donate
		if err := d.reuse.Add.LoadTendermint(attrs); err != nil {
			return err
		}
		d.Listener.OnAdd(&d.reuse.Add, meta)
	case "asgard_fund_yggdrasil":
		if err := d.reuse.AsgardFundYggdrasil.LoadTendermint(attrs); err != nil {
			return err
		}
		d.Listener.OnAsgardFundYggdrasil(&d.reuse.AsgardFundYggdrasil, meta)
	case "bond":
		if err := d.reuse.Bond.LoadTendermint(attrs); err != nil {
			return err
		}
		d.Listener.OnBond(&d.reuse.Bond, meta)
	case "errata":
		if err := d.reuse.Errata.LoadTendermint(attrs); err != nil {
			return err
		}
		d.Listener.OnErrata(&d.reuse.Errata, meta)
	case "fee":
		if err := d.reuse.Fee.LoadTendermint(attrs); err != nil {
			return err
		}
		d.Listener.OnFee(&d.reuse.Fee, meta)
	case "InactiveVault":
		if err := d.reuse.InactiveVault.LoadTendermint(attrs); err != nil {
			return err
		}
		d.Listener.OnInactiveVault(&d.reuse.InactiveVault, meta)
	case "gas":
		if err := d.reuse.Gas.LoadTendermint(attrs); err != nil {
			return err
		}
		d.Listener.OnGas(&d.reuse.Gas, meta)
	case "message":
		if err := d.reuse.Message.LoadTendermint(attrs); err != nil {
			return err
		}
		d.Listener.OnMessage(&d.reuse.Message, meta)
	case "new_node":
		if err := d.reuse.NewNode.LoadTendermint(attrs); err != nil {
			return err
		}
		d.Listener.OnNewNode(&d.reuse.NewNode, meta)
	case "outbound":
		if err := d.reuse.Outbound.LoadTendermint(attrs); err != nil {
			return err
		}
		d.Listener.OnOutbound(&d.reuse.Outbound, meta)
	case "pool":
		if err := d.reuse.Pool.LoadTendermint(attrs); err != nil {
			return err
		}
		d.Listener.OnPool(&d.reuse.Pool, meta)
	case "refund":
		if err := d.reuse.Refund.LoadTendermint(attrs); err != nil {
			return err
		}
		d.Listener.OnRefund(&d.reuse.Refund, meta)
	case "reserve":
		if err := d.reuse.Reserve.LoadTendermint(attrs); err != nil {
			return err
		}
		d.Listener.OnReserve(&d.reuse.Reserve, meta)
	case "rewards":
		if err := d.reuse.Rewards.LoadTendermint(attrs); err != nil {
			return err
		}
		PoolRewardsTotal.Add(uint64(len(d.reuse.Rewards.PerPool)))
		d.Listener.OnRewards(&d.reuse.Rewards, meta)
	case "set_ip_address":
		if err := d.reuse.SetIPAddress.LoadTendermint(attrs); err != nil {
			return err
		}
		d.Listener.OnSetIPAddress(&d.reuse.SetIPAddress, meta)
	case "set_mimir":
		if err := d.reuse.SetMimir.LoadTendermint(attrs); err != nil {
			return err
		}
		d.Listener.OnSetMimir(&d.reuse.SetMimir, meta)
	case "set_node_keys":
		if err := d.reuse.SetNodeKeys.LoadTendermint(attrs); err != nil {
			return err
		}
		d.Listener.OnSetNodeKeys(&d.reuse.SetNodeKeys, meta)
	case "set_version":
		if err := d.reuse.SetVersion.LoadTendermint(attrs); err != nil {
			return err
		}
		d.Listener.OnSetVersion(&d.reuse.SetVersion, meta)
	case "slash":
		if err := d.reuse.Slash.LoadTendermint(attrs); err != nil {
			return err
		}
		d.Listener.OnSlash(&d.reuse.Slash, meta)
	case "add_liquidity":
		if err := d.reuse.Stake.LoadTendermint(attrs); err != nil {
			return err
		}
		d.Listener.OnStake(&d.reuse.Stake, meta)
	case "swap":
		if err := d.reuse.Swap.LoadTendermint(attrs); err != nil {
			return err
		}
		d.Listener.OnSwap(&d.reuse.Swap, meta)
	case "transfer":
		if err := d.reuse.Transfer.LoadTendermint(attrs); err != nil {
			return err
		}
		d.Listener.OnTransfer(&d.reuse.Transfer, meta)
	case "withdraw":
		// TODO(acsaba): rename unstake->withdraw.
		if err := d.reuse.Unstake.LoadTendermint(attrs); err != nil {
			return err
		}
		d.Listener.OnUnstake(&d.reuse.Unstake, meta)
	case "UpdateNodeAccountStatus":
		if err := d.reuse.UpdateNodeAccountStatus.LoadTendermint(attrs); err != nil {
			return err
		}
		d.Listener.OnUpdateNodeAccountStatus(&d.reuse.UpdateNodeAccountStatus, meta)
	case "validator_request_leave":
		if err := d.reuse.ValidatorRequestLeave.LoadTendermint(attrs); err != nil {
			return err
		}
		d.Listener.OnValidatorRequestLeave(&d.reuse.ValidatorRequestLeave, meta)
	case "tss_keygen", "tss_keysign", "slash_points":
		// TODO(acsaba): decide if we want to store these events.
	default:
		miderr.Printf("Unkown event type: %s", event.Type)
		UnknownsTotal.Add(1)
		return errEventType
	}
	return nil
}
