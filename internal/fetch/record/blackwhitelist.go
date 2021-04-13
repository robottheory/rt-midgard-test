package record

import (
	"gitlab.com/thorchain/midgard/internal/db"
)

// This file contains temporary hacks for when thornode is lacking events or sending extra events.

func AddMissingEvents(d *Demux, meta *Metadata) {
	switch db.ChainID() {
	case "7D37DEF6E1BE23C912092069325C4A51E66B9EF7DDBDE004FF730CFABC0307B1":
		// Chaosnet started on 2021-04-10
		if meta.BlockHeight == 12824 {
			// Genesis node bonded rune and became listed as Active without any events.
			d.reuse.UpdateNodeAccountStatus = UpdateNodeAccountStatus{
				NodeAddr: []byte("thor1xfqaqhk5r6x9hdwlvmye0w9agv8ynljacmxulf"),
				Former:   []byte("Ready"),
				Current:  []byte("Active"),
			}
			Recorder.OnUpdateNodeAccountStatus(&d.reuse.UpdateNodeAccountStatus, meta)
		}
	case "8371BCEB807EEC52AC6A23E2FFC300D18FD3938374D3F4FC78EEB5FE33F78AF7":
		// Testnet started on 2021-04-10
		if meta.BlockHeight == 28795 {
			// Withdraw id 57BD5B26B0D78CD4A0340F8ECA2356B23B029157E43DE99EF03114CC15577C8A
			// failed, still pool balances were changed.
			// Fix for future was submitted on Thornode:
			// https://gitlab.com/thorchain/thornode/-/merge_requests/1634
			d.reuse.PoolBalanceChange = PoolBalanceChange{
				Asset:    []byte("LTC.LTC"),
				RuneAmt:  1985607,
				RuneAdd:  false,
				AssetAmt: 93468,
				AssetAdd: false,
				Reason:   "Midgard fix: Reserve didn't have rune for gas",
			}
			Recorder.OnPoolBalanceChange(&d.reuse.PoolBalanceChange, meta)
		}
	default:
	}

}
