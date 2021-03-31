package record

// This file contains temporary hacks for when thornode is lacking events or sending extra events.

func FixMissingFees(d *Demux, meta *Metadata) {
	if meta.BlockHeight == 77000 {
		// On testnet (2021-03) we had a withdrawal of 100% where the fee event was not emmitted.
		// This creates the fake event to fix it.
		d.reuse.PoolBalanceChange = PoolBalanceChange{
			Asset:   []byte("ETH.THOR-0XA0B515C058F127A15DD3326F490EBF47D215588E"),
			RuneAmt: 61861488,
			RuneAdd: false,
			Reason:  "Fix: Fee event was not emmitted",
		}
		Recorder.OnPoolBalanceChange(&d.reuse.PoolBalanceChange, meta)
	}
}
