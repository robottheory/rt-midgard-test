package record

// Testnet started on 2021-04-10
const ChainIDTestnet202104 = "8371BCEB807EEC52AC6A23E2FFC300D18FD3938374D3F4FC78EEB5FE33F78AF7"

func loadTestnet202104Corrections(chainID string) {
	if chainID == ChainIDTestnet202104 {
		loadTestnetAdditionalEvents()
	}
}

func loadTestnetAdditionalEvents() {
	AdditionalEvents.Add(28795, func(d *Demux, meta *Metadata) {
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
	})
}
