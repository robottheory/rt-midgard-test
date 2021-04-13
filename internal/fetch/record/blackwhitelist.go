package record

import (
	"gitlab.com/thorchain/midgard/internal/db"
)

// This file contains temporary hacks for when thornode is lacking events or sending extra events.

func AddMissingEvents(d *Demux, meta *Metadata) {
	if db.ChainID() == "7D37DEF6E1BE23C912092069325C4A51E66B9EF7DDBDE004FF730CFABC0307B1" {
		if meta.BlockHeight == 12824 {
			// Genesis node bonded rune and became listed as Active without any events.
			d.reuse.UpdateNodeAccountStatus = UpdateNodeAccountStatus{
				NodeAddr: []byte("thor1xfqaqhk5r6x9hdwlvmye0w9agv8ynljacmxulf"),
				Former:   []byte("Ready"),
				Current:  []byte("Active"),
			}
			Recorder.OnUpdateNodeAccountStatus(&d.reuse.UpdateNodeAccountStatus, meta)
		}
	}
}
