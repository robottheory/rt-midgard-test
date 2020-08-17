package event

import (
	"log"

	"github.com/influxdata/influxdb-client-go"
	influxapi "github.com/influxdata/influxdb-client-go/api"

	"gitlab.com/thorchain/midgard/event"
)

// InfluxOut is the persistence client.
var InfluxOut influxapi.WriteAPI = influxdb2.NewClient("http://influxdb:8086", "").WriteAPI("thorchain", "midgard")

// TODO(pascaldekloe): Setup timeseries clients from main with proper error retries & abort.
func init() {
	go func() {
		for err := range InfluxOut.Errors() {
			log.Print("outbound InfluxDB error: ", err)
		}
	}()
}

// EventListener is a singleton implementation using InfluxOut.
var EventListener event.Listener = eventListener{}

type eventListener struct{}

func (_ eventListener) OnAdd(event *event.Add, meta *event.Metadata) {
	InfluxOut.WritePoint(influxdb2.NewPoint("add",
		map[string]string{
			"asset": string(event.Asset),
			"pool":  string(event.Pool),
		},
		map[string]interface{}{
			"tx":        event.TxID,
			"chain":     event.Chain,
			"from":      event.FromAddr,
			"to":        event.ToAddr,
			"amount_E8": event.AmountE8,
			"memo":      event.Memo,
		},
		meta.BlockTimestamp))
}

func (_ eventListener) OnFee(event *event.Fee, meta *event.Metadata) {
	InfluxOut.WritePoint(influxdb2.NewPoint("fee",
		map[string]string{
			"asset": string(event.Asset),
		},
		map[string]interface{}{
			"tx_in":       event.InTxID,
			"amount_E8":   event.AmountE8,
			"pool_deduct": event.PoolDeduct,
		},
		meta.BlockTimestamp))
}

func (_ eventListener) OnGas(event *event.Gas, meta *event.Metadata) {
	InfluxOut.WritePoint(influxdb2.NewPoint("gas",
		map[string]string{
			"asset": string(event.Asset),
		},
		map[string]interface{}{
			"amount_E8":      event.AmountE8,
			"rune_amount_E8": event.RuneAmountE8,
			"tx_count":       event.TxCount,
		},
		meta.BlockTimestamp))
}

func (_ eventListener) OnOutbound(event *event.Outbound, meta *event.Metadata) {
	InfluxOut.WritePoint(influxdb2.NewPoint("outbound",
		map[string]string{
			"asset": string(event.Asset),
		},
		map[string]interface{}{
			"tx":        event.TxID,
			"chain":     event.Chain,
			"from":      event.FromAddr,
			"to":        event.ToAddr,
			"amount_E8": event.AmountE8,
			"memo":      event.Memo,
			"tx_in":     event.InTxID,
		},
		meta.BlockTimestamp))
}

func (_ eventListener) OnPool(event *event.Pool, meta *event.Metadata) {
	InfluxOut.WritePoint(influxdb2.NewPoint("pool",
		map[string]string{
			"pool": string(event.Pool),
		},
		map[string]interface{}{
			"status": event.Status,
		},
		meta.BlockTimestamp))
}

func (_ eventListener) OnRefund(event *event.Refund, meta *event.Metadata) {
	InfluxOut.WritePoint(influxdb2.NewPoint("refund",
		map[string]string{
			"asset": string(event.Asset),
		},
		map[string]interface{}{
			"tx":        event.TxID,
			"chain":     event.Chain,
			"from":      event.FromAddr,
			"to":        event.ToAddr,
			"amount_E8": event.AmountE8,
			"memo":      event.Memo,
			"code":      event.Code,
			"reason":    event.Reason,
		},
		meta.BlockTimestamp))
}

func (_ eventListener) OnReserve(event *event.Reserve, meta *event.Metadata) {
	InfluxOut.WritePoint(influxdb2.NewPoint("reserve",
		map[string]string{
			"asset": string(event.Asset),
		},
		map[string]interface{}{
			"tx":                    event.TxID,
			"chain":                 event.Chain,
			"from":                  event.FromAddr,
			"to":                    event.ToAddr,
			"amount_E8":             event.AmountE8,
			"memo":                  event.Memo,
			"contributor_addr":      event.ContributorAddr,
			"contributor_amount_E8": event.ContributorAmountE8,
		},
		meta.BlockTimestamp))
}

func (_ eventListener) OnStake(event *event.Stake, meta *event.Metadata) {
	InfluxOut.WritePoint(influxdb2.NewPoint("stake",
		map[string]string{
			"pool": string(event.Pool),
		},
		map[string]interface{}{
			"stake_units":    event.StakeUnits,
			"rune_addr":      event.RuneAddr,
			"rune_amount_E8": event.RuneAmountE8,
			"rune_tx":        event.RuneTxID,
			"asset_tx":       event.AssetTxID,
		},
		meta.BlockTimestamp))
}

func (_ eventListener) OnSwap(event *event.Swap, meta *event.Metadata) {
	InfluxOut.WritePoint(influxdb2.NewPoint("swap",
		map[string]string{
			"asset": string(event.Asset),
			"pool":  string(event.Pool),
		},
		map[string]interface{}{
			"tx":                    event.TxID,
			"chain":                 event.Chain,
			"from":                  event.FromAddr,
			"to":                    event.ToAddr,
			"amount_E8":             event.AmountE8,
			"memo":                  event.Memo,
			"price_target":          event.PriceTarget,
			"trade_slip":            event.TradeSlip,
			"liquidity_fee":         event.LiquidityFee,
			"liquidity_fee_in_rune": event.LiquidityFeeInRune,
		},
		meta.BlockTimestamp))
}

func (_ eventListener) OnUnstake(event *event.Unstake, meta *event.Metadata) {
	InfluxOut.WritePoint(influxdb2.NewPoint("unstake",
		map[string]string{
			"asset": string(event.Asset),
			"pool":  string(event.Pool),
		},
		map[string]interface{}{
			"tx":           event.TxID,
			"chain":        event.Chain,
			"from":         event.FromAddr,
			"to":           event.ToAddr,
			"amount_E8":    event.AmountE8,
			"memo":         event.Memo,
			"stake_units":  event.StakeUnits,
			"basis_points": event.BasisPoints,
			"asymmetry":    event.Asymmetry,
		},
		meta.BlockTimestamp))
}
