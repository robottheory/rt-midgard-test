package record

import (
	"gitlab.com/thorchain/midgard/internal/db"
)

// This file contains temporary hacks for when thornode is lacking events or sending extra events.

// TODO(muninn): move into separate directory and split into different files
// TODO(muninn): refactor and make correct abstractions
// TODO(muninn): make it more efficient

// https://gitlab.com/thorchain/thornode/-/issues/912
// There was a bug in thornode, it withdraw units were more then the actually removed pool units,
// but the impermanent loss protection units were also added to them.
type WithdrawCorrection struct {
	TX          string
	ActualUnits int64
}

var WithdrawCorrections = map[int64]WithdrawCorrection{
	47832:  {"4338F014E1FAC05C2248ECE0A36061D92CC76ADF13CCA773272AD70E00B56154", 9066450465},
	79082:  {"A1B155BD4F57DDF91200733EE2552C9E0E828E632F0D91EF69BCAF3D74D8D512", 169807962},
	81055:  {"7613CEC05CA9B3A4BEF864F22E51EA29EB377EF4EC00885F91377F6D74D1DA4D", 2267292958},
	81462:  {"5E02AE1FE7A777BC6CBE8F4FC2DAFC9F8A6464BAAC58697202EAE1A2271D91D2", 8002689544},
	84221:  {"8885C9AC8A26002DA29090D6173D6A1C340AC6BD96837146BDA4ED059EF0760F", 288123877},
	85406:  {"E6907237BFFDFD5F733E5B422D4BC3106A8BCF933A7547843E458580C625D5D5", 609672362},
	88797:  {"F552E27BC9774E546CA4024B8274C758FC6433F3A38B0DB16137196F55E58C73", 2208373135},
	89415:  {"2E48177404B36CE893240A5B0CFF3FA501CE914BBA1F7D3FFEFC75D44110ADCF", 767266632},
	90002:  {"4D41DA864AE89E8B4CC315360F145E33501B2C1534A5757C1104606C967AB54F", 19621520713},
	100196: {"C94BD47100E0C9983845735A3FA0C6C511713CB4486CBB3777F8DA386011A0C0", 8280457915},
	105465: {"E86DCD9FDD898A3F7781D049EE0442DCC69ACBC2FBB110125A501AF7CF3003D7", 911047010},
	109333: {"C1BD2175944D490D56755B37D1EB88385F9BF7A34EF609418A332526859C6EE2", 406716426},
	110069: {"8D5BBF31ABCB8297AB2804186D6AAA1B479E79B1CB0A0C1B2586F0F89225C28B", 13600885317},
	112985: {"DAC7FCA92A9B42B82BFBE9C03C756A1AFBEF178CF8D2F6F2E044407A6696D581", 117224625},
	128842: {"34B820F7158C3AB690C2DCF088356D1A70E6721551C2159C96729CE9FA97B698", 93675000000},
	128845: {"0754C907993E389BA7947CB775D456BB829E12B3D7EEB676413E749BB847068B", 146382616748},
	131366: {"8EEB3FBAA095F46E12207257C3CB0771BDB55C3EB2322F86FD75594ECC015AD1", 45078869167},
	138590: {"7058BA9B3FF1173D620773458F84C5EA247EBB38C74C505E1FB8069CDB8A6E27", 14950765467},
	147789: {"8CDA8459400D97CC436F1D19B6E42A4CEDDD21F2A231D1F9D4438B43A7750136", 4873515514},
	147798: {"EAF6064BD7CB29389917BF4FF0D499D8E99890D9B561D8FF63F610092FADA4A3", 814479987},
	151691: {"6A7A7C3A7A65F4704151DB1972EAFA6A237B03BA82D46721E761F3063753C42C", 345151887},
	153980: {"85A19DA310282D35A6C51F4C34F921D27F2DF090535790F0C533FE61EA980CD7", 1115323168},
	163137: {"0BA388B1BCF76C04B81D885ECB99E0E98A295778234FF9A88E9CA8ED69706DF4", 3086810573},
	166532: {"156CCFBC66F775C7FDF9D3E18F071C6CEC2ADFAB4F7F435094AA516ECD1C698A", 8288025767},
	257485: {"E6B6FBC73BFD62BC36F0E236BC065FCC18D328832908C240399E2DF2E2CB6565", 9702125229},
	260113: {"A6788765BCFBEC33F0F4585CB736105D4005AB81FEC30113231CF1D41F843AEA", 272714488439},
	260114: {"1296D15627331C78CA5BC7CEE014C98273C5B08D358FA451C8039B42EAD61054", 128877756350},
	260115: {"F6B4EDB5555CC4FAF513729D16F2D906DEC5C950DC95530F26ABFDC7ECD5DBCE", 75139724801},
	260116: {"86679B5EE155F2997251108713C96AE0AC91444BFD0883A99D0611A255F0F2D7", 41517402427},
	260119: {"04B6F0AEDFA9ABD9DD949541C2B7762DC2EA62026ACB39C8992482355318FB8C", 29065838793},
	265159: {"2BABF243911BA2CFF2551143131985515C4873C9D6C87E44027E0F7F14E29792", 18962634918},
	269611: {"CCE905915CEC65FD6FDC48E31E43E65FCD73ABDCF90A4419EFDFE7E43B63DDD0", 156734300},
	271635: {"02BA91CF8F6FF3E35A1C7F0F1991BB2A2E200B78B3CF7A77DAF77E66067B205F", 83041261241},
	271741: {"4B95FDF07545DF8BDD9B05982F013166E0BAB8B54F419548DEB0D3EE2E5F454E", 1539766365},
	277262: {"E0E67CF364BFDD9B312C1899C60582F720A44F1A8023333F7849E0AAD0B9E4DB", 9402258},
	292069: {"70558EA306ADA6C6705A4C15AA60BB06D9000F75F9C2FA85153027F0AC131357", 10046673124},
}

func FixWithdawUnits(withdraw *Unstake, meta *Metadata) {
	if db.ChainID() == "7D37DEF6E1BE23C912092069325C4A51E66B9EF7DDBDE004FF730CFABC0307B1" {
		correction, ok := WithdrawCorrections[meta.BlockHeight]
		if ok {
			if correction.TX == string(withdraw.Tx) {
				withdraw.StakeUnits = correction.ActualUnits
			}
		}
	}
}

func AddMissingEvents(d *Demux, meta *Metadata) {
	switch db.ChainID() {
	case "7D37DEF6E1BE23C912092069325C4A51E66B9EF7DDBDE004FF730CFABC0307B1":
		// Chaosnet started on 2021-04-10
		switch meta.BlockHeight {
		case 12824:
			// Genesis node bonded rune and became listed as Active without any events.
			d.reuse.UpdateNodeAccountStatus = UpdateNodeAccountStatus{
				NodeAddr: []byte("thor1xfqaqhk5r6x9hdwlvmye0w9agv8ynljacmxulf"),
				Former:   []byte("Ready"),
				Current:  []byte("Active"),
			}
			Recorder.OnUpdateNodeAccountStatus(&d.reuse.UpdateNodeAccountStatus, meta)
		case 63519:
			// Fix was applied in https://gitlab.com/thorchain/thornode/-/merge_requests/1643
			//
			// TODO(acsaba): add PR/issue id as context for this, update reason.
			// TODO(muninn): clarify with core team about
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
		case 84876:
			// TODO(muninn): move into it's own function
			// TODO(muninn): publish script used to generate these
			// There was a bug with impermanent loss and withdraws.
			// Members withdrew, but their pool units went up.
			// https://gitlab.com/thorchain/thornode/-/issues/896
			d.reuse.Stake = Stake{
				Pool:       []byte("BTC.BTC"),
				RuneAddr:   []byte("thor1h7n7lakey4tah37226musffwjhhk558kaay6ur"),
				StakeUnits: 2029187601,
			}
			Recorder.OnStake(&d.reuse.Stake, meta)
		case 170826:
			d.reuse.Stake = Stake{
				Pool:       []byte("BNB.BNB"),
				RuneAddr:   []byte("thor1t5t5xg7muu3fl2lv6j9ck6hgy0970r08pvx0rz"),
				StakeUnits: 31262905,
			}
			Recorder.OnStake(&d.reuse.Stake, meta)
		case 226753:
			// TODO(muninn): figure out what happened, relevant events:

			// unstake_events [tx: 0020F08F14B50992D92C391C2EEF93AFABDEE36B73F35B76664FD6F9AFD746DD,
			// chain: BNB, from_addr: bnb1zl2qg3r6mzd488nk8j9lxkan6pq2w0546lputm,
			// to_addr: bnb1n9esxuw8ca7ts8l6w66kdh800s09msvul6vlse, asset: BNB.BNB, asset_e8: 1,
			// emit_asset_e8: 74463250, emit_rune_e8: 0, memo: WITHDRAW:BNB.BNB:10000,
			// pool: BNB.BNB, stake_units: 1540819843, basis_points: 10000, asymmetry: 0,
			// imp_loss_protection_e8: 0, block_timestamp: 1619303098489715057]
			//
			// fee_events [tx: 0020F08F14B50992D92C391C2EEF93AFABDEE36B73F35B76664FD6F9AFD746DD,
			// asset: BNB.BNB, asset_e8: 22500, pool_deduct: 973297,
			// block_timestamp: 1619303098489715057]
			d.reuse.PoolBalanceChange = PoolBalanceChange{
				Asset:    []byte("BNB.BNB"),
				AssetAmt: 1,
				AssetAdd: true,
				Reason:   "Midgard fix: TODO figure out what happened",
			}
			Recorder.OnPoolBalanceChange(&d.reuse.PoolBalanceChange, meta)
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
