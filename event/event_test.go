package event

import "testing"

var GoldenAssets = []struct{ Asset, Chain, Ticker, ID string }{
	{"BTC.BTC", "BTC", "BTC", ""},
	{"ETH.ETH", "ETH", "ETH", ""},
	{"ETH.USDT-0xdac17f958d2ee523a2206206994597c13d831ec7", "ETH", "USDT", "0xdac17f958d2ee523a2206206994597c13d831ec7"},
	{"BNB.BNB", "BNB", "BNB", ""},
	{"BNB.RUNE-B1A", "BNB", "RUNE", "B1A"},
	{"THOR.RUNE", "THOR", "RUNE", ""},
	{"", "", "", ""},
	{".", "", "", ""},
	{"-", "", "", ""},
	{".-", "", "", ""},
	{"1.", "1", "", ""},
	{"2.-", "2", "", ""},
	{"A", "", "A", ""},
	{".B", "", "B", ""},
	{"C-", "", "C", ""},
	{".D-", "", "D", ""},
	{"-a", "", "", "a"},
	{".-b", "", "", "b"},
}

func TestParseAsset(t *testing.T) {
	for _, gold := range GoldenAssets {
		chain, ticker, ID := ParseAsset([]byte(gold.Asset))
		if string(chain) != gold.Chain || string(ticker) != gold.Ticker || string(ID) != gold.ID {
			t.Errorf("%q got [%q %q %q], want [%q %q %q]", gold.Asset, chain, ticker, ID, gold.Chain, gold.Ticker, gold.ID)
		}
	}
}
