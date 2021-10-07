package record

// Testnet started on 2021-07-10
const ChainIDTestnet202107 = "D6E12364E25D460C8D1155ADAD7CB827EE5D8D0B54B9609C928BF9EE9E23AC4C"

// ThorNode state and events diverged on testnet. We apply all these changes to be in sync with
// Thornode.
func loadTestnet202107Corrections(chainID string) {
	if chainID == ChainIDTestnet202107 {
		loadTestnetUnnecesaryFee()
		loadTestnetMissingWithdraw()
		loadTestnetWithdrawImpLossNotReported()
		withdrawCoinKeptHeight = 907000
	}
}

//////////////////////// Withdraw impermanent loss not reported

// These withdraw events had impermanent loss, but the events didn't report them.
// Bug was fixed:
//
// https://gitlab.com/thorchain/thornode/-/issues/1092
// PR to fix it : https://gitlab.com/thorchain/thornode/-/merge_requests/1903
func loadTestnetWithdrawImpLossNotReported() {
	impLossMissing := map[int64]int64{
		695829: 4369620487,
		696073: 4586529689,
		845393: 15057810864,
	}
	correctF := func(withdraw *Unstake, meta *Metadata) {
		if string(withdraw.Pool) != "BNB.BNB" {
			return
		}
		actualImpLoss, ok := impLossMissing[meta.BlockHeight]
		if ok {
			withdraw.ImpLossProtectionE8 = actualImpLoss
		}
	}

	for k := range impLossMissing {
		WithdrawCorrections.Add(k, correctF)
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
		{906653, "ETH.USDT-0XA3910454BF2CB59B8B3A401589A3BACC5CA42306", 5599700, 213193445},
		{906664, "ETH.USDT-0XA3910454BF2CB59B8B3A401589A3BACC5CA42306", 559900, 21316679},
		{906668, "ETH.USDT-0XA3910454BF2CB59B8B3A401589A3BACC5CA42306", 559900, 21316679},
		{906671, "ETH.USDT-0XA3910454BF2CB59B8B3A401589A3BACC5CA42306", 111900, 4260290},
		{906785, "ETH.USDT-0XA3910454BF2CB59B8B3A401589A3BACC5CA42306", 11199500, 426390698},
		{906882, "ETH.USDT-0XA3910454BF2CB59B8B3A401589A3BACC5CA42306", 11199500, 426390698},
		{906905, "ETH.USDT-0XA3910454BF2CB59B8B3A401589A3BACC5CA42306", 11199500, 426390698},
		{906975, "ETH.USDT-0XA3910454BF2CB59B8B3A401589A3BACC5CA42306", 11199500, 426390698},
		{906982, "ETH.USDT-0XA3910454BF2CB59B8B3A401589A3BACC5CA42306", 11199500, 426390698},
		{907080, "ETH.USDT-0XA3910454BF2CB59B8B3A401589A3BACC5CA42306", 11199500, 426347299},
		{926021, "BNB.BNB", 3547, 46548884},
		{943161, "BNB.BNB", 5427, 68148367},
		{926022, "BTC.BTC", 1294, 87464131},
		{1032380, "BTC.BTC", 1381, 83497811},
		{1072116, "BTC.BTC", 13306, 761951453},
		{1072176, "BTC.BTC", 13306, 761951453},
		{1148134, "BTC.BTC", 28691, 1588144091},
		{1086116, "ETH.ETH", 805258, 389359858},
		{1086150, "ETH.ETH", 805258, 389359858},
		{1086157, "ETH.ETH", 805258, 389359858},
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

// There was a reorg on the ETH chain and the add_liquidity was undone.
// There was an errata event which corrected the depths but the LPs units
// were not removed in the errata event (units were rewoked in ThorNode) .
//
// Tracking for the ThorNode fix:
// https://gitlab.com/thorchain/thornode/-/issues/1087
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