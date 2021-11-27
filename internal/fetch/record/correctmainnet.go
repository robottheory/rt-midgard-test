package record

import (
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
)

// This file contains many small independent corrections

const ChainIDMainnet202104 = "7D37DEF6E1BE23C912092069325C4A51E66B9EF7DDBDE004FF730CFABC0307B1"

func loadMainnet202104Corrections(chainID string) {
	if chainID == ChainIDMainnet202104 {
		loadMainnetCorrectionsWithdrawImpLoss()
		loadMainnetWithdrawForwardedAssetCorrections()
		loadMainnetWithdrawIncreasesUnits()
		loadMainnetcorrectGenesisNode()
		loadMainnetMissingWithdraws()
		registerArtificialPoolBallanceChanges(
			mainnetArtificialDepthChanges, "Midgard fix on mainnet")
		withdrawCoinKeptHeight = 1970000
		GlobalWithdrawCorrection = correctWithdawsMainnetFilter
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

//////////////////////// Missing Witdhdraws

type AdditionalWithdraw struct {
	Pool     string
	FromAddr string
	Reason   string
	RuneE8   int64
	AssetE8  int64
	Units    int64
}

func (w *AdditionalWithdraw) Record(d *Demux, meta *Metadata) {
	reason := []byte(w.Reason)
	chain := strings.Split(w.Pool, ".")[0]

	hashF := fnv.New32a()
	fmt.Fprint(hashF, w.Reason, w.Pool, w.FromAddr, w.RuneE8, w.AssetE8, w.Units)
	txID := strconv.Itoa(int(hashF.Sum32()))

	d.reuse.Unstake = Unstake{
		FromAddr:    []byte(w.FromAddr),
		Chain:       []byte(chain),
		Pool:        []byte(w.Pool),
		Asset:       []byte("THOR.RUNE"),
		ToAddr:      reason,
		Memo:        reason,
		Tx:          []byte(txID),
		EmitRuneE8:  w.RuneE8,
		EmitAssetE8: w.AssetE8,
		StakeUnits:  w.Units,
	}
	Recorder.OnUnstake(&d.reuse.Unstake, meta)
}

func addWithdraw(height int64, w AdditionalWithdraw) {
	AdditionalEvents.Add(height, w.Record)
}

func loadMainnetMissingWithdraws() {
	// A failed withdraw actually modified the pool, bug was corrected to not repeat again:
	// https://gitlab.com/thorchain/thornode/-/merge_requests/1643
	addWithdraw(63519, AdditionalWithdraw{
		Pool:     "BNB.BNB",
		FromAddr: "thor1tl9k7fjvye4hkvwdnl363g3f2xlpwwh7k7msaw",
		Reason:   "bug 1643 corrections fix for assymetric rune withdraw problem",
		RuneE8:   1999997,
		AssetE8:  0,
		Units:    1029728,
	})

	// TODO(muninn): find out reason for the divergence and document.
	// Discussion:
	// https://discord.com/channels/838986635756044328/902137599559335947
	addWithdraw(2360486, AdditionalWithdraw{
		Pool:     "BCH.BCH",
		FromAddr: "thor1nlkdr8wqaq0wtnatckj3fhem2hyzx65af8n3p7",
		Reason:   "midgard correction missing withdraw",
		RuneE8:   1934186,
		AssetE8:  29260,
		Units:    1424947,
	})
	addWithdraw(2501774, AdditionalWithdraw{
		Pool:     "BNB.BUSD-BD1",
		FromAddr: "thor1prlky34zkpr235lelpan8kj8yz30nawn2cuf8v",
		Reason:   "midgard correction missing withdraw",
		RuneE8:   1481876,
		AssetE8:  10299098,
		Units:    962674,
	})

	// On Pool suspension the withdraws had FromAddr=null and they were skipped by Midgard.
	// Later the pool was reactivated, so having correct units is important even at suspension.
	// There is a plan to fix ThorNode events:
	// https://gitlab.com/thorchain/thornode/-/issues/1164
	addWithdraw(2606240, AdditionalWithdraw{
		Pool:     "BNB.FTM-A64",
		FromAddr: "thor14sz7ca8kwhxmzslds923ucef22pm0dh28hhfve",
		Reason:   "midgard correction suspended pool withdraws missing",
		RuneE8:   0,
		AssetE8:  0,
		Units:    768586678,
	})
	addWithdraw(2606240, AdditionalWithdraw{
		Pool:     "BNB.FTM-A64",
		FromAddr: "thor1jhuy9ft2rgr4whvdks36sjxee5sxfyhratz453",
		Reason:   "midgard correction suspended pool withdraws missing",
		RuneE8:   0,
		AssetE8:  0,
		Units:    110698993,
	})
	addWithdraw(2606240, AdditionalWithdraw{
		Pool:     "BNB.FTM-A64",
		FromAddr: "thor19wcfdx2yk8wjze7l0cneynjvjyquprjwj063vh",
		Reason:   "midgard correction suspended pool withdraws missing",
		RuneE8:   0,
		AssetE8:  0,
		Units:    974165115,
	})
}

//////////////////////// Fix withdraw assets not forwarded.

// In the early blocks of the chain the asset sent in with the withdraw initiation
// was not forwarded back to the user. This was fixed for later blocks:
//  https://gitlab.com/thorchain/thornode/-/merge_requests/1635

func correctWithdawsForwardedAsset(withdraw *Unstake, meta *Metadata) KeepOrDiscard {
	withdraw.AssetE8 = 0
	return Keep
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

func correctWithdawsMainnetFilter(withdraw *Unstake, meta *Metadata) KeepOrDiscard {
	// In the beginning of the chain withdrawing pending liquidity emitted a
	// withdraw event with units=0.
	// This was later corrected, and pending_liquidity events are emitted instead.
	if withdraw.StakeUnits == 0 && meta.BlockHeight < 1000000 {
		return Discard
	}
	return Keep
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
	1483166: {
		{"ETH.YFI-0X0BC529C00C6401AEF6D220BE8C6EA1667F6AD93E", -18571915693442, 555358575},
		{"ETH.ETH", 42277574737435, -725548423675},
		{"ETH.SUSHI-0X6B3595068778DD592E39A122F4F5A5CF09C90FE2", -19092838139426, 3571961904132},
		{"ETH.HEGIC-0X584BC13C7D411C00C01A62E8019472DE68768430", -69694964523, 2098087620642},
		{"ETH.AAVE-0X7FC66500C84A76AD7E9C93437BFC5AC33E2DDAE9", -2474742542086, 20912604795},
		{"ETH.DODO-0X43DFC4159D86F3A37A5A4B3D4580B888AD7D4DDD", -38524681038, 83806675976},
		{"ETH.KYL-0X67B6D479C7BB412C54E03DCA8E1BC6740CE6B99C", -2029858716920, 35103388382444},
	},
	// TODO(muninn): document divergency reason
	2597851: {
		{"ETH.ALCX-0XDBDB4D16EDA451D0503B854CF79D55697F90C8DF", 0, -96688561785},
		{"ETH.SUSHI-0X6B3595068778DD592E39A122F4F5A5CF09C90FE2", 0, -5159511094095},
		{"ETH.USDT-0XDAC17F958D2EE523A2206206994597C13D831EC7", 0, -99023689717400},
		{"ETH.XRUNE-0X69FA0FEE221AD11012BAB0FDB45D444D3D2CE71C", 0, -2081880169421610},
		{"ETH.YFI-0X0BC529C00C6401AEF6D220BE8C6EA1667F6AD93E", 0, -727860649},
	},
	// TODO(muninn): document divergency reason
	// LTC
}
