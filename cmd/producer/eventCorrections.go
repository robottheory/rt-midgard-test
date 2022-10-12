package main

import (
	"bytes"
	"fmt"
	"github.com/tendermint/tendermint/abci/types"
	"gitlab.com/thorchain/midgard/internal/fetch/record"
	"gitlab.com/thorchain/midgard/internal/util/kafka"
	"gitlab.com/thorchain/midgard/internal/util/midlog"
	"hash/fnv"
	"strconv"
	"strings"
)

func loadAllCorrections() {
	loadAddInsteadOfWithdrawal()
	loadImpLossEvents()
	loadCorrectGenesisNode()
	loadMainnetMissingWithdraws()
	loadWithdrawIncreasesUnits()
	loadArtificialBalanceChanges()
	loadForwardAssetEvents()
}

type ExtraEvents map[int64][]*types.Event

func (e ExtraEvents) Add(height int64, event types.Event) {
	if e[height] == nil {
		e[height] = make([]*types.Event, 0)
	}

	e[height] = append(e[height], &event)
}

type EventCorrector func(event *kafka.IndexedEvent) record.KeepOrDiscard

type CorrectEvents map[int64][]EventCorrector

func (e CorrectEvents) Add(height int64, ec EventCorrector) {
	if e[height] == nil {
		e[height] = make([]EventCorrector, 0)
	}

	e[height] = append(e[height], ec)
}

var (
	extraEvents   = make(ExtraEvents)
	correctEvents = make(CorrectEvents)
)

type artificialUnitChange struct {
	Pool  string
	Addr  string
	Units int64
}
type artificialUnitChanges map[int64][]artificialUnitChange

var addInsteadWithdrawMapMainnet202104 = artificialUnitChanges{
	// Sometimes when withdrawing the pool units of a member went up, not down:
	// https://gitlab.com/thorchain/thornode/-/issues/896
	84876:  {{"BTC.BTC", "thor1h7n7lakey4tah37226musffwjhhk558kaay6ur", 2029187601}},
	170826: {{"BNB.BNB", "thor1t5t5xg7muu3fl2lv6j9ck6hgy0970r08pvx0rz", 31262905}},
	// At a withdraw member units went up
	// TODO(muninn): document ThorNode bugfix for it.
	2677311: {{"LTC.LTC", "thor19jhhfjvnauryeq3r56e0llvndrz8xxcwgjlhzz", 8033289}},
}

func loadAddInsteadOfWithdrawal() {
	for k, v := range addInsteadWithdrawMapMainnet202104 {
		for _, change := range v {
			midlog.WarnF("Adding event %v", change)

			event := types.Event{}

			if change.Units >= 0 {
				event.Type = "add_liquidity"

				event.Attributes = []types.EventAttribute{
					{Key: []byte("pool"), Value: []byte(change.Pool)},
					{Key: []byte("liquidity_provider_units"), Value: []byte(fmt.Sprintf("%v", change.Units))},
				}

				if addressIsRune(change.Addr) {
					event.Attributes = append(event.Attributes, types.EventAttribute{Key: []byte("rune_address"), Value: []byte(change.Addr)})
				} else {
					event.Attributes = append(event.Attributes, types.EventAttribute{Key: []byte("asset_address"), Value: []byte(change.Addr)})
				}
			} else {
				event.Type = "withdraw"

				event.Attributes = []types.EventAttribute{
					{Key: []byte("pool"), Value: []byte(change.Pool)},
					{Key: []byte("coin"), Value: []byte(change.Pool)},
					{Key: []byte("from"), Value: []byte(change.Addr)},
					{Key: []byte("to"), Value: []byte(change.Addr)},
					{Key: []byte("id"), Value: []byte(change.Addr + strconv.Itoa(int(-change.Units)))},
					{Key: []byte("chain"), Value: []byte(strings.Split(change.Pool, ".")[0])},
					{Key: []byte("memo"), Value: []byte("Midgard Fix")},
					{Key: []byte("liquidity_provider_units"), Value: []byte(fmt.Sprintf("%v", -change.Units))},
				}

			}

			extraEvents.Add(k, event)
		}
	}
}

func addressIsRune(address string) bool {
	return strings.HasPrefix(address, "thor") || strings.HasPrefix(address, "tthor") || strings.HasPrefix(address, "sthor")
}

//////////////////////// Follow ThorNode bug on withdraw (units and rune was added to the pool)

// https://gitlab.com/thorchain/thornode/-/issues/954
// ThorNode added units to a member after a withdraw instead of removing.
// The bug was corrected, but an arbitrage account hit this bug 13 times.
//
// The values were generated with cmd/statechecks
// The member address was identified with cmd/membercheck
func loadWithdrawIncreasesUnits() {
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

	for k, v := range corrections {
		event := types.Event{}
		event.Type = "add_liquidity"

		event.Attributes = []types.EventAttribute{
			{Key: []byte("pool"), Value: []byte("ETH.ETH")},
			{Key: []byte("rune_address"), Value: []byte("thor1hyarrh5hslcg3q5pgvl6mp6gmw92c4tpzdsjqg")},
			{Key: []byte("rune_amount"), Value: []byte(fmt.Sprintf("%v", v.AdditionalRune))},
			{Key: []byte("liquidity_provider_units"), Value: []byte(fmt.Sprintf("%v", v.AdditionalUnits))},
		}

		if extraEvents[k] == nil {
			extraEvents[k] = make([]*types.Event, 0)
		}
		extraEvents[k] = append(extraEvents[k], &event)
	}
}

//////////////////////// Activate genesis node.

// Genesis node bonded rune and became listed as Active without any events.
func loadCorrectGenesisNode() {
	//var k int64 = 12824
	//updateNodeAccountStatus := record.UpdateNodeAccountStatus{
	//	NodeAddr: []byte("thor1xfqaqhk5r6x9hdwlvmye0w9agv8ynljacmxulf"),
	//	Former:   []byte("Ready"),
	//	Current:  []byte("Active"),
	//}
	//if extraNodeStatusEvents[k] == nil {
	//	extraNodeStatusEvents[k] = make([]record.UpdateNodeAccountStatus, 0)
	//}
	//extraNodeStatusEvents[k] = append(extraNodeStatusEvents[k], updateNodeAccountStatus)
}

func addWithdraw(height int64, w record.AdditionalWithdraw) {
	reason := []byte(w.Reason)
	chain := strings.Split(w.Pool, ".")[0]

	hashF := fnv.New32a()
	fmt.Fprint(hashF, w.Reason, w.Pool, w.FromAddr, w.RuneE8, w.AssetE8, w.Units)
	txID := strconv.Itoa(int(hashF.Sum32()))

	event := types.Event{}

	event.Type = "withdraw"
	event.Attributes = []types.EventAttribute{
		{Key: []byte("pool"), Value: []byte(w.Pool)},
		{Key: []byte("coin"), Value: []byte("0 THOR.RUNE")},
		{Key: []byte("from"), Value: []byte(w.FromAddr)},
		{Key: []byte("to"), Value: reason},
		{Key: []byte("emit_rune"), Value: []byte(fmt.Sprintf("%v", w.RuneE8))},
		{Key: []byte("emit_asset"), Value: []byte(fmt.Sprintf("%v", w.AssetE8))},
		{Key: []byte("liquidity_provider_units"), Value: []byte(fmt.Sprintf("%v", w.Units))},
		{Key: []byte("id"), Value: []byte(txID)},
		{Key: []byte("chain"), Value: []byte(chain)},
		{Key: []byte("memo"), Value: reason},
	}

	extraEvents.Add(height, event)
}

func loadMainnetMissingWithdraws() {
	// A failed withdraw actually modified the pool, bug was corrected to not repeat again:
	// https://gitlab.com/thorchain/thornode/-/merge_requests/1643
	addWithdraw(63519, record.AdditionalWithdraw{
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
	addWithdraw(2360486, record.AdditionalWithdraw{
		Pool:     "BCH.BCH",
		FromAddr: "thor1nlkdr8wqaq0wtnatckj3fhem2hyzx65af8n3p7",
		Reason:   "midgard correction missing withdraw",
		RuneE8:   1934186,
		AssetE8:  29260,
		Units:    1424947,
	})
	addWithdraw(2501774, record.AdditionalWithdraw{
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
	addWithdraw(2606240, record.AdditionalWithdraw{
		Pool:     "BNB.FTM-A64",
		FromAddr: "thor14sz7ca8kwhxmzslds923ucef22pm0dh28hhfve",
		Reason:   "midgard correction suspended pool withdraws missing",
		RuneE8:   0,
		AssetE8:  0,
		Units:    768586678,
	})
	addWithdraw(2606240, record.AdditionalWithdraw{
		Pool:     "BNB.FTM-A64",
		FromAddr: "thor1jhuy9ft2rgr4whvdks36sjxee5sxfyhratz453",
		Reason:   "midgard correction suspended pool withdraws missing",
		RuneE8:   0,
		AssetE8:  0,
		Units:    110698993,
	})
	addWithdraw(2606240, record.AdditionalWithdraw{
		Pool:     "BNB.FTM-A64",
		FromAddr: "thor19wcfdx2yk8wjze7l0cneynjvjyquprjwj063vh",
		Reason:   "midgard correction suspended pool withdraws missing",
		RuneE8:   0,
		AssetE8:  0,
		Units:    974165115,
	})
	addWithdraw(1166400, record.AdditionalWithdraw{
		Pool:     "ETH.WBTC-0X2260FAC5E5542A773AA44FBCFEDF7C193BC2C599",
		FromAddr: "thor1g6pnmnyeg48yc3lg796plt0uw50qpp7hgz477u",
		Reason:   "midgard correction suspended pool withdraws missing",
		RuneE8:   0,
		AssetE8:  0,
		Units:    2228000000,
	})
}

/////////////// Artificial pool balance changes to fix ThorNode/Midgard depth divergences.

type artificialPoolBalanceChange struct {
	Pool  string
	Rune  int64
	Asset int64
}

func absAndSign(x int64) (abs int64, pos bool) {
	if 0 <= x {
		return x, true
	} else {
		return -x, false
	}
}

type artificialPoolBalanceChanges map[int64][]artificialPoolBalanceChange

var mainnetArtificialDepthChanges = artificialPoolBalanceChanges{
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

func loadArtificialBalanceChanges() {
	for k, v := range mainnetArtificialDepthChanges {
		for _, i := range v {
			event := types.Event{}
			event.Type = "pool_balance_change"

			runeAmt, runeAdd := absAndSign(i.Rune)
			assetAmt, assetAdd := absAndSign(i.Asset)

			event.Attributes = []types.EventAttribute{
				{Key: []byte("asset"), Value: []byte(i.Pool)},
				{Key: []byte("rune_amt"), Value: []byte(fmt.Sprintf("%v", runeAmt))},
				{Key: []byte("rune_add"), Value: []byte(strconv.FormatBool(runeAdd))},
				{Key: []byte("asset_amt"), Value: []byte(fmt.Sprintf("%v", assetAmt))},
				{Key: []byte("asset_add"), Value: []byte(strconv.FormatBool(assetAdd))},
				{Key: []byte("reason"), Value: []byte("Fix in Midgard")},
			}

			extraEvents.Add(k, event)
		}
	}

}

// In 2021-04 ThorNode had two bugs when withdrawing with impermanent loss.
// All the constants were generated by querying the real values from Thornode with:
// $ go run ./cmd/onetime/fetchunits [config.json]

// https://gitlab.com/thorchain/thornode/-/issues/912
// There was a bug in thornode, it withdraw units were more then the actually removed pool units,
// but the impermanent loss protection units were also added to them.
type withdrawUnitCorrection struct {
	TX          string
	ActualUnits int64
}

var withdrawUnitCorrectionsMainnet202104 = map[int64]withdrawUnitCorrection{
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

func loadImpLossEvents() {
	for k, v := range withdrawUnitCorrectionsMainnet202104 {
		copiedID := strings.Clone(v.TX)
		copiedAU := v.ActualUnits
		correctEvents.Add(k, func(event *kafka.IndexedEvent) record.KeepOrDiscard {
			if event.Event.Type != "withdraw" {
				return record.Keep
			}

			found := false
			for _, attribute := range event.Event.Attributes {
				if string(attribute.Key) == "id" && string(attribute.Value) == copiedID {
					found = true
					break
				}
			}

			if found {
				for i, attribute := range event.Event.Attributes {
					if string(attribute.Key) == "liquidity_provider_units" {
						midlog.WarnF("Correcting event in block %v", event.Height)
						attribute.Value = []byte(fmt.Sprintf("%v", copiedAU))
						event.Event.Attributes[i] = attribute
						break
					}
				}
			}

			return record.Keep
		})
	}
}

//////////////////////// Fix withdraw assets not forwarded.

// In the early blocks of the chain the asset sent in with the withdraw initiation
// was not forwarded back to the user. This was fixed for later blocks:
//  https://gitlab.com/thorchain/thornode/-/merge_requests/1635

// generate block heights where this occured:
//   select FORMAT('    %s,', b.height)
//   from unstake_events as x join block_log as b on x.block_timestamp = b.timestamp
//   where asset_e8 != 0 and asset != 'THOR.RUNE' and b.height < 220000;

func loadForwardAssetEvents() {
	heightWithOldWithdraws := []int64{
		29113,
		110069,
	}

	for _, h := range heightWithOldWithdraws {
		correctEvents.Add(h, func(event *kafka.IndexedEvent) record.KeepOrDiscard {
			if event.Event.Type != "withdraw" {
				return record.Keep
			}

			for j, attribute := range event.Event.Attributes {
				if string(attribute.Key) == "coin" {
					i := bytes.IndexByte(attribute.Value, ' ')
					asset := string(attribute.Value[i+1:])
					attribute.Value = []byte(fmt.Sprintf("0 %s", asset))
					event.Event.Attributes[j] = attribute
					break
				}
			}

			return record.Keep
		})
	}
}

func mainnetFilter(event *kafka.IndexedEvent) record.KeepOrDiscard {
	// In the beginning of the chain withdrawing pending liquidity emitted a
	// withdraw event with units=0.
	// This was later corrected, and pending_liquidity events are emitted instead.
	if event.Event.Type != "withdraw" || event.Height >= 1000000 {
		return record.Keep
	}

	for _, attribute := range event.Event.Attributes {
		if string(attribute.Key) == "liquidity_provider_units" {
			stakeUnits, _ := strconv.ParseInt(string(attribute.Value), 10, 64)
			if stakeUnits == 0 {
				return record.Discard
			}
			break
		}
	}

	return record.Keep
}
