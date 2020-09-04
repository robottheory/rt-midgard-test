package timeseries

import (
	"testing"

	_ "github.com/lib/pq"

	"gitlab.com/thorchain/midgard/event"
)

func TestOnAdd(t *testing.T) {
	mustSetup(t)
	EventListener.OnAdd(&event.Add{}, &event.Metadata{})
}

func TestOnErrata(t *testing.T) {
	mustSetup(t)
	EventListener.OnErrata(&event.Errata{}, &event.Metadata{})
}

func TestOnFee(t *testing.T) {
	mustSetup(t)
	EventListener.OnFee(&event.Fee{}, &event.Metadata{})
}

func TestOnGas(t *testing.T) {
	mustSetup(t)
	EventListener.OnGas(&event.Gas{}, &event.Metadata{})
}

func TestOnNewNode(t *testing.T) {
	mustSetup(t)
	EventListener.OnNewNode(&event.NewNode{}, &event.Metadata{})
}

func TestOnOutbound(t *testing.T) {
	mustSetup(t)
	EventListener.OnOutbound(&event.Outbound{}, &event.Metadata{})
}

func TestOnRefund(t *testing.T) {
	mustSetup(t)
	EventListener.OnRefund(&event.Refund{}, &event.Metadata{})
}

func TestOnReserve(t *testing.T) {
	mustSetup(t)
	EventListener.OnReserve(&event.Reserve{}, &event.Metadata{})
}

func TestOnRewards(t *testing.T) {
	mustSetup(t)
	EventListener.OnRewards(&event.Rewards{Pool: []event.Amount{{}, {}}}, &event.Metadata{})
}

func TestOnSetIPAddress(t *testing.T) {
	mustSetup(t)
	EventListener.OnSetIPAddress(&event.SetIPAddress{}, &event.Metadata{})
}

func TestOnSetNodeKeys(t *testing.T) {
	mustSetup(t)
	EventListener.OnSetNodeKeys(&event.SetNodeKeys{}, &event.Metadata{})
}

func TestOnSetVersion(t *testing.T) {
	mustSetup(t)
	EventListener.OnSetVersion(&event.SetVersion{}, &event.Metadata{})
}

func TestOnSlash(t *testing.T) {
	mustSetup(t)
	EventListener.OnSlash(&event.Slash{Amounts: []event.Amount{{}, {}}}, &event.Metadata{})
}

func TestOnStake(t *testing.T) {
	mustSetup(t)
	EventListener.OnStake(&event.Stake{}, &event.Metadata{})
}

func TestOnSwap(t *testing.T) {
	mustSetup(t)
	EventListener.OnSwap(&event.Swap{}, &event.Metadata{})
}
