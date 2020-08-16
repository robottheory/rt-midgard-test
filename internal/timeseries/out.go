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

// BUG(pascaldekloe): Overwrites with the same meta.BlockTimestamp.

func (l eventListener) OnAdd(event *event.Add, meta *event.Metadata) {
	InfluxOut.WritePoint(influxdb2.NewPoint("add",
		map[string]string{
			"chain": string(event.Chain),
			"asset": string(event.CoinAsset),
		},
		map[string]interface{}{
			"tx":     event.TxID,
			"from":   event.FromAddr,
			"to":     event.ToAddr,
			"amount": event.CoinAmount,
			"memo":   event.Memo,
			"pool":   event.Pool,
		},
		meta.BlockTimestamp))
}

func (l eventListener) OnFee(event *event.Fee, meta *event.Metadata) {
	InfluxOut.WritePoint(influxdb2.NewPoint("fee",
		map[string]string{
			"asset": string(event.CoinAsset),
		},
		map[string]interface{}{
			"tx_in":       event.InTxID,
			"amount":      event.CoinAmount,
			"pool_deduct": event.PoolDeduct,
		},
		meta.BlockTimestamp))
}

func (l eventListener) OnMessage(event *event.Message, meta *event.Metadata) {
	InfluxOut.WritePoint(influxdb2.NewPoint("message",
		map[string]string{},
		map[string]interface{}{
			"action": event.Action,
		},
		meta.BlockTimestamp))
}

func (l eventListener) OnOutbound(event *event.Outbound, meta *event.Metadata) {
	InfluxOut.WritePoint(influxdb2.NewPoint("outbound",
		map[string]string{
			"asset": string(event.CoinAsset),
			"chain": string(event.Chain),
		},
		map[string]interface{}{
			"tx_in":  event.InTxID,
			"tx":     event.TxID,
			"from":   event.FromAddr,
			"to":     event.ToAddr,
			"amount": event.CoinAmount,
			"memo":   event.Memo,
		},
		meta.BlockTimestamp))
}

func (l eventListener) OnPool(event *event.Pool, meta *event.Metadata) {
	InfluxOut.WritePoint(influxdb2.NewPoint("pool",
		map[string]string{
			"asset": string(event.CoinAsset),
			"chain": string(event.Chain),
		},
		map[string]interface{}{
			"tx":     event.TxID,
			"from":   event.FromAddr,
			"to":     event.ToAddr,
			"amount": event.CoinAmount,
			"memo":   event.Memo,
			"pool":   event.Pool,
			"status": event.Status,
		},
		meta.BlockTimestamp))
}

func (l eventListener) OnRefund(event *event.Refund, meta *event.Metadata) {
	InfluxOut.WritePoint(influxdb2.NewPoint("refund",
		map[string]string{
			"asset": string(event.CoinAsset),
			"chain": string(event.Chain),
		},
		map[string]interface{}{
			"tx":     event.TxID,
			"from":   event.FromAddr,
			"to":     event.ToAddr,
			"amount": event.CoinAmount,
			"memo":   event.Memo,
			"code":   event.Code,
			"reason": event.Reason,
		},
		meta.BlockTimestamp))
}

func (l eventListener) OnReserve(event *event.Reserve, meta *event.Metadata) {
	InfluxOut.WritePoint(influxdb2.NewPoint("reserve",
		map[string]string{
			"asset": string(event.CoinAsset),
			"chain": string(event.Chain),
		},
		map[string]interface{}{
			"tx":             event.TxID,
			"from":           event.FromAddr,
			"to":             event.ToAddr,
			"amount":         event.CoinAmount,
			"memo":           event.Memo,
			"contrib_addr":   event.ContribAddr,
			"reserve_amount": event.Amount,
		},
		meta.BlockTimestamp))
}

func (l eventListener) OnStake(event *event.Stake, meta *event.Metadata) {
	// TODO(pascaldekloe): Are TxIDsPerChain needed?
	InfluxOut.WritePoint(influxdb2.NewPoint("stake",
		map[string]string{
			"chain": string(event.Chain),
		},
		map[string]interface{}{
			"memo":        event.Memo,
			"pool":        event.Pool,
			"stake_units": event.StakeUnits,
			"rune_addr":   event.RuneAddr,
			"rune_amount": event.RuneAmount,
		},
		meta.BlockTimestamp))
}

func (l eventListener) OnSwap(event *event.Swap, meta *event.Metadata) {
	InfluxOut.WritePoint(influxdb2.NewPoint("swap",
		map[string]string{
			"asset": string(event.CoinAsset),
			"chain": string(event.Chain),
		},
		map[string]interface{}{
			"tx":                    event.TxID,
			"from":                  event.FromAddr,
			"to":                    event.ToAddr,
			"amount":                event.CoinAmount,
			"memo":                  event.Memo,
			"pool":                  event.Pool,
			"price_target":          event.PriceTarget,
			"trade_slip":            event.TradeSlip,
			"liquidity_fee":         event.LiquidityFee,
			"liquidity_fee_in_rune": event.LiquidityFeeInRune,
		},
		meta.BlockTimestamp))
}

func (l eventListener) OnUnstake(event *event.Unstake, meta *event.Metadata) {
	// TODO(pascaldekloe): Are TxIDsPerChain needed?
	InfluxOut.WritePoint(influxdb2.NewPoint("unstake",
		map[string]string{
			"asset": string(event.CoinAsset),
			"chain": string(event.Chain),
		},
		map[string]interface{}{
			"tx":           event.TxID,
			"from":         event.FromAddr,
			"to":           event.ToAddr,
			"amount":       event.CoinAmount,
			"memo":         event.Memo,
			"pool":         event.Pool,
			"stake_units":  event.StakeUnits,
			"basis_points": event.BasisPoints,
			"asymmetry":    event.Asymmetry,
		},
		meta.BlockTimestamp))
}
