package record

// Testnet started on 2021-07-10
const ChainIDTestnet202107 = "D6E12364E25D460C8D1155ADAD7CB827EE5D8D0B54B9609C928BF9EE9E23AC4C"

// ThorNode state and events diverged on testnet. We apply all these changes to be in sync with
// Thornode.
func loadTestnet202107Corrections(chainID string) {
	if chainID == ChainIDTestnet202107 {
		loadTestnetUnnecesaryFee()
		loadTestnetMissingWithdraw()
	}
}

//////////////////////// Unnecesary fee event

// Seems like fee were emitted by mistake on a failed withdraw.
// https://discord.com/channels/838986635756044328/839002638653325333/879631885557452830
// TODO(muninn):  document fix on ThorNode side when it happens.
func loadTestnetUnnecesaryFee() {
	type BadFee struct {
		height     int64
		asset      string
		assetE8    int64
		poolDeduct int64
	}
	blacklist := []BadFee{
		{641194, "BNB.BUSD-74E", 9614624991, 475698862},
		{641213, "BNB.BUSD-74E", 9614624991, 475698862},
		{695087, "BNB.BUSD-74E", 9528536939, 1032690566},
	}

	for _, badFee := range blacklist {
		persistentBadFee := badFee
		FeeAcceptFuncs.Add(persistentBadFee.height, func(fee *Fee, meta *Metadata) bool {
			if string(fee.Asset) == persistentBadFee.asset &&
				fee.AssetE8 == persistentBadFee.assetE8 &&
				fee.PoolDeduct == persistentBadFee.poolDeduct {
				return false
			}
			return true
		})
	}
}

//////////////////////// Missing withdraw

// LP lost all it's unit's without a withdraw event.
// https://discord.com/channels/838986635756044328/839002638653325333/880026757925851146
// TODO(muninn): document fix on ThorNode side when it happens.
func loadTestnetMissingWithdraw() {
	AdditionalEvents.Add(152868, func(d *Demux, meta *Metadata) {
		reason := []byte("Midgard fix missing withdraw")
		d.reuse.Unstake = Unstake{
			FromAddr:   []byte("0xc092365acc5b3a39b5f709b168cdd8746a76d99b"),
			Chain:      []byte("ETH"),
			Pool:       []byte("ETH.ETH"),
			Asset:      []byte("THOR.RUNE"),
			ToAddr:     reason,
			Memo:       reason,
			Tx:         reason,
			EmitRuneE8: 0,
			StakeUnits: 17848470045,
		}
		Recorder.OnUnstake(&d.reuse.Unstake, meta)
	})
}
