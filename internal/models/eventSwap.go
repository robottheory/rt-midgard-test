package models

import (
	"gitlab.com/thorchain/midgard/internal/clients/thorChain/types"
	"gitlab.com/thorchain/midgard/internal/common"
)

const (
	PriceTarget = "price_target"
	TradeSlip   = "trade_slip"
)

type EventSwap struct {
	Event
	Pool         common.Asset
	PriceTarget  int64
	TradeSlip    int64
	LiquidityFee int64
}

func NewSwapEvent(swap types.EventSwap, event types.Event) EventSwap {
	return EventSwap{
		Pool:         swap.Pool,
		PriceTarget:  swap.PriceTarget,
		TradeSlip:    swap.TradeSlip,
		LiquidityFee: swap.LiquidityFee,
		Event:        newEvent(event),
	}
}
