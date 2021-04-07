package record

// This file contains temporary hacks for when thornode is lacking events or sending extra events.

var failedGasEvents = map[int64]struct {
	pool        string
	runeDeficit int64
}{
	10219: {"ETH.ETH", 148293600},
	10578: {"ETH.ETH", 266594777},
	10940: {"ETH.ETH", 227716531},
	11299: {"ETH.ETH", 255412561},
	11660: {"ETH.ETH", 104486359},
	11802: {"BTC.BTC", 17585330},
	13476: {"BTC.BTC", 40488201},
}

// In the beginning of the chain there wasn't enough rune to compensate for gas.
// The rune in these gas events didn't arrive to the pools.
func FixFailedGasEvents(d *Demux, meta *Metadata) {
	failedGas, ok := failedGasEvents[meta.BlockHeight]
	if ok {
		d.reuse.PoolBalanceChange = PoolBalanceChange{
			Asset:   []byte(failedGas.pool),
			RuneAmt: failedGas.runeDeficit,
			RuneAdd: false,
			Reason:  "Midgard fix: Reserve didn't have rune for gas",
		}
		Recorder.OnPoolBalanceChange(&d.reuse.PoolBalanceChange, meta)
	}
}
