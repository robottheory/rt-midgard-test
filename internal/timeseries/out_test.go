package timeseries

import (
	"testing"

	_ "github.com/lib/pq"
	"github.com/pascaldekloe/sqltest"

	"gitlab.com/thorchain/midgard/event"
)

func TestOnAdd(t *testing.T) {
	DBExec = sqltest.NewTx(t).Exec
	EventListener.OnAdd(&event.Add{}, &event.Metadata{})
}

func TestOnErrata(t *testing.T) {
	DBExec = sqltest.NewTx(t).Exec
	EventListener.OnErrata(&event.Errata{}, &event.Metadata{})
}

func TestOnFee(t *testing.T) {
	DBExec = sqltest.NewTx(t).Exec
	EventListener.OnFee(&event.Fee{}, &event.Metadata{})
}

func TestOnGas(t *testing.T) {
	DBExec = sqltest.NewTx(t).Exec
	EventListener.OnGas(&event.Gas{}, &event.Metadata{})
}

func TestOnNewNode(t *testing.T) {
	DBExec = sqltest.NewTx(t).Exec
	EventListener.OnNewNode(&event.NewNode{}, &event.Metadata{})
}

func TestOnOutbound(t *testing.T) {
	DBExec = sqltest.NewTx(t).Exec
	EventListener.OnOutbound(&event.Outbound{}, &event.Metadata{})
}

func TestOnRefund(t *testing.T) {
	DBExec = sqltest.NewTx(t).Exec
	EventListener.OnRefund(&event.Refund{}, &event.Metadata{})
}

func TestOnReserve(t *testing.T) {
	DBExec = sqltest.NewTx(t).Exec
	EventListener.OnReserve(&event.Reserve{}, &event.Metadata{})
}

func TestOnRewards(t *testing.T) {
	DBExec = sqltest.NewTx(t).Exec
	EventListener.OnRewards(&event.Rewards{Pool: []event.Amount{{}, {}}}, &event.Metadata{})
}

func TestOnSetIPAddress(t *testing.T) {
	DBExec = sqltest.NewTx(t).Exec
	EventListener.OnSetIPAddress(&event.SetIPAddress{}, &event.Metadata{})
}

func TestOnSetNodeKeys(t *testing.T) {
	DBExec = sqltest.NewTx(t).Exec
	EventListener.OnSetNodeKeys(&event.SetNodeKeys{}, &event.Metadata{})
}

func TestOnSetVersion(t *testing.T) {
	DBExec = sqltest.NewTx(t).Exec
	EventListener.OnSetVersion(&event.SetVersion{}, &event.Metadata{})
}

func TestOnSlash(t *testing.T) {
	DBExec = sqltest.NewTx(t).Exec
	EventListener.OnSlash(&event.Slash{Amounts: []event.Amount{{}, {}}}, &event.Metadata{})
}

func TestOnStake(t *testing.T) {
	DBExec = sqltest.NewTx(t).Exec
	EventListener.OnStake(&event.Stake{}, &event.Metadata{})
}

func TestOnSwap(t *testing.T) {
	DBExec = sqltest.NewTx(t).Exec
	EventListener.OnSwap(&event.Swap{}, &event.Metadata{})
}
