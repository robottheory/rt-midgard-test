package record

// This file contains many small independent corrections

const ChainIDMainnet202104 = "7D37DEF6E1BE23C912092069325C4A51E66B9EF7DDBDE004FF730CFABC0307B1"

func loadMainnet202104Corrections(chainID string) {
	if chainID == ChainIDMainnet202104 {
		loadMainnetCorrectionsWithdrawImpLoss()
		loadMainnetWithdrawForwardedAssetCorrections()
		loadMainnetWithdrawIncreasesUnits()
		loadMainnetcorrectGenesisNode()
		loadMainnetFailedWithdraw()
		registerArtificialPoolBallanceChanges(
			mainnetArtificialDepthChanges, "Midgard fix on mainnet")
		withdrawCoinKeptHeight = 1970000
	}
}

//////////////////////// Activate genesis node.

// Genesis node bonded rune and became listed as Active without any events.
func loadMainnetcorrectGenesisNode() {
	AdditionalEvents.Add(12824, func(d *Demux, meta *Metadata) {
		d.reuse.UpdateNodeAccountStatus = UpdateNodeAccountStatus{
			NodeAddr: []byte("thor1xfqaqhk5r6x9hdwlvmye0w9agv8ynljacmxulf"),
			Former:   []byte("Ready"),
			Current:  []byte("Active"),
		}
		Recorder.OnUpdateNodeAccountStatus(&d.reuse.UpdateNodeAccountStatus, meta)
	})
}

//////////////////////// Withdraw bug 1643

// A failed withdraw actually modified the pool, bug was corrected to not repeat again:
// https://gitlab.com/thorchain/thornode/-/merge_requests/1643
func loadMainnetFailedWithdraw() {
	AdditionalEvents.Add(63519, func(d *Demux, meta *Metadata) {
		reason := []byte("Midgard fix for assymetric rune withdraw problem")
		d.reuse.Unstake = Unstake{
			FromAddr:   []byte("thor1tl9k7fjvye4hkvwdnl363g3f2xlpwwh7k7msaw"),
			Chain:      []byte("BNB"),
			Pool:       []byte("BNB.BNB"),
			Asset:      []byte("THOR.RUNE"),
			ToAddr:     reason,
			Memo:       reason,
			Tx:         reason,
			EmitRuneE8: 1999997,
			StakeUnits: 1029728,
		}
		Recorder.OnUnstake(&d.reuse.Unstake, meta)
	})
}

//////////////////////// Fix withdraw assets not forwarded.

// In the early blocks of the chain the asset sent in with the withdraw initiation
// was not forwarded back to the user. This was fixed for later blocks:
//  https://gitlab.com/thorchain/thornode/-/merge_requests/1635

func correctWithdawsForwardedAsset(withdraw *Unstake, meta *Metadata) {
	withdraw.AssetE8 = 0
}

// generate block heights where this occured:
//   select FORMAT('    %s,', b.height)
//   from unstake_events as x join block_log as b on x.block_timestamp = b.timestamp
//   where asset_e8 != 0 and asset != 'THOR.RUNE' and b.height < 220000;
func loadMainnetWithdrawForwardedAssetCorrections() {
	var heightWithOldWithdraws []int64
	heightWithOldWithdraws = []int64{
		29113,
		110069,
	}
	for _, height := range heightWithOldWithdraws {
		WithdrawCorrections.Add(height, correctWithdawsForwardedAsset)
	}
}

//////////////////////// Follow ThorNode bug on withdraw (units and rune was added to the pool)

// https://gitlab.com/thorchain/thornode/-/issues/954
// ThorNode added units to a member after a withdraw instead of removing.
// The bug was corrected, but an arbitrage account hit this bug 13 times.
//
// The values were generated with cmd/statechecks
// The member address was identified with cmd/membercheck
func loadMainnetWithdrawIncreasesUnits() {
	type MissingAdd struct {
		AdditionalRune  int64
		AdditionalUnits int64
	}
	corrections := map[int64]MissingAdd{
		672275: {
			AdditionalRune:  2546508574,
			AdditionalUnits: 967149543,
		},
		674411: {
			AdditionalRune:  1831250392,
			AdditionalUnits: 704075160,
		},
		676855: {
			AdditionalRune:  1699886243,
			AdditionalUnits: 638080440,
		},
		681060: {
			AdditionalRune:  1101855537,
			AdditionalUnits: 435543069,
		},
		681061: {
			AdditionalRune:  1146177337,
			AdditionalUnits: 453014832,
		},
		681063: {
			AdditionalRune:  271977087,
			AdditionalUnits: 106952192,
		},
		681810: {
			AdditionalRune:  3830671893,
			AdditionalUnits: 1518717776,
		},
		681815: {
			AdditionalRune:  2749916233,
			AdditionalUnits: 1090492640,
		},
		681819: {
			AdditionalRune:  540182490,
			AdditionalUnits: 213215502,
		},
		682026: {
			AdditionalRune:  1108123249,
			AdditionalUnits: 443864231,
		},
		682028: {
			AdditionalRune:  394564637,
			AdditionalUnits: 158052776,
		},
		682031: {
			AdditionalRune:  1043031822,
			AdditionalUnits: 417766496,
		},
		682128: {
			AdditionalRune:  3453026237,
			AdditionalUnits: 1384445390,
		},
	}

	correct := func(d *Demux, meta *Metadata) {
		missingAdd := corrections[meta.BlockHeight]
		d.reuse.Stake = Stake{
			AddBase: AddBase{
				Pool:     []byte("ETH.ETH"),
				RuneAddr: []byte("thor1hyarrh5hslcg3q5pgvl6mp6gmw92c4tpzdsjqg"),
				RuneE8:   missingAdd.AdditionalRune,
			},
			StakeUnits: missingAdd.AdditionalUnits,
		}
		Recorder.OnStake(&d.reuse.Stake, meta)
	}
	for k := range corrections {
		AdditionalEvents.Add(k, correct)
	}
}

var mainnetArtificialDepthChanges = artificialPoolBallanceChanges{
	// Bug was fixed in ThorNode: https://gitlab.com/thorchain/thornode/-/merge_requests/1765
	1043090: {{"BCH.BCH", -1, 0}},
	// Fix for ETH chain attack was submitted to Thornode, but some needed events were not emitted:
	// https://gitlab.com/thorchain/thornode/-/merge_requests/1815/diffs?commit_id=9a7d8e4a1b0c25cb4d56c737c5fe826e7aa3e6f2
	1483166: {{"ETH.YFI-0X0BC529C00C6401AEF6D220BE8C6EA1667F6AD93E", -18571915693442, 555358575},
		{"ETH.ETH", 42277574737435, -725548423675},
		{"ETH.SUSHI-0X6B3595068778DD592E39A122F4F5A5CF09C90FE2", -19092838139426, 3571961904132},
		{"ETH.HEGIC-0X584BC13C7D411C00C01A62E8019472DE68768430", -69694964523, 2098087620642},
		{"ETH.AAVE-0X7FC66500C84A76AD7E9C93437BFC5AC33E2DDAE9", -2474742542086, 20912604795},
		{"ETH.DODO-0X43DFC4159D86F3A37A5A4B3D4580B888AD7D4DDD", -38524681038, 83806675976},
		{"ETH.KYL-0X67B6D479C7BB412C54E03DCA8E1BC6740CE6B99C", -2029858716920, 35103388382444},
	},
}
