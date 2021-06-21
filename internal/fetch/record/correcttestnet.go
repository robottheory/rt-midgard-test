package record

// Testnet started on 2021-04-10
const ChainIDTestnet202104 = "8371BCEB807EEC52AC6A23E2FFC300D18FD3938374D3F4FC78EEB5FE33F78AF7"

// ThorNode state and events diverged on testnet. We apply all these changes to be in sync with
// Thornode. Unlike for mainnet these errors were not investigated in every case.
func loadTestnet202104Corrections(chainID string) {
	if chainID == ChainIDTestnet202104 {
		AdditionalEvents.Add(200000, func(d *Demux, meta *Metadata) {
			d.reuse.Pool = Pool{
				Asset:  []byte("."),
				Status: []byte("Suspended"),
			}
			Recorder.OnPool(&d.reuse.Pool, meta)
		})

		registerArtificialDeposits(testnetArtificialDeposits)

		registerArtificialPoolBallanceChanges(testnetArtificialDepthChanges, "Midgard fix on testnet")
	}
}

// Corrections are not on correct heights, the goal is to have a good end state only
var testnetArtificialDeposits = artificialUnitChanges{
	200000: {
		{"BCH.BCH", "tthor1ru2a946zpa4cyz9s93xuje2pwkswsqzn26jc6j", 15144770},
		{"BNB.BNB", "tbnb1egyrhwap09qqyvw4hyes9yvk4z7g2n629z47t0", 419534},
		{"BNB.BNB", "tthor13gym97tmw3axj3hpewdggy2cr288d3qffr8skg", 123839374},
		{"BNB.BNB", "tthor1puhn8fclwvmmzh7uj7546wnxz5h3zar8e66sc5", 73435168},
		{"BNB.BNB", "tthor1pw4duket6d22ha6ze6r5458qkw6dkd2qqxdeeu", 6308},
		{"BNB.BNB", "tthor1ruwh7nh8lsl3m5xn0rl404t5wjfgu4rmgz3e6w", 235630},
		{"BNB.BUSD-74E", "tthor1vhl7xq52gc7ejn8vrrtkjvw7hl98rnjmsyxhmd", 7259846},
		{"BTC.BTC", "tthor1vhl7xq52gc7ejn8vrrtkjvw7hl98rnjmsyxhmd", 745468934},
		{"BTC.BTC", "tthor1zg5pz4hgsctyclmu97ynaj3hmjvz9prw4679r0", 2484274},
		{"ETH.DAI-0XAD6D458402F60FD3BD25163575031ACDCE07538D", "tthor1ruwh7nh8lsl3m5xn0rl404t5wjfgu4rmgz3e6w", 1280974560},
		{"ETH.USDT-0XA3910454BF2CB59B8B3A401589A3BACC5CA42306", "tthor1ruwh7nh8lsl3m5xn0rl404t5wjfgu4rmgz3e6w", 14614499},
		{"LTC.LTC", "tthor102hv29wngdpr29z0z26p3wd69xfjgv0m3tq452", -1805353},
		{"LTC.LTC", "tthor1erl5a09ahua0umwcxp536cad7snerxt4eflyq0", 185308161},
		{"LTC.LTC", "tthor1ruwh7nh8lsl3m5xn0rl404t5wjfgu4rmgz3e6w", -254112},
	},
	300000: {
		{"BCH.BCH", "tthor19deprm338t90w35uyqztkcsaplwnemvqmctj8n", 19687020},
		{"BCH.BCH", "tthor1pw4duket6d22ha6ze6r5458qkw6dkd2qqxdeeu", 92387188},
		{"ETH.ETH", "tthor13z3lz8z39wwkyrsjymjlaup88nhhr9ttgenr7z", 19515363},
		{"ETH.ETH", "tthor1s8tcs532kxztwp3lczm70ftcktpmgar4l2x7hl", 54696770},
		{"ETH.ETH", "tthor1uggfwrrf94akz7xus42x7kzfvk9ureguewcdmh", 2471},
	},
	500000: {
		{"BNB.BNB", "tthor12sjrr7lz728a34dr97a4v05fmlqdyp8zah9q45", 13719297},
		{"BTC.BTC", "tthor139gj73hulcesq5fsz4txgmjumkmrt7w3e6t9wt", 30827},
		{"ETH.DAI-0XAD6D458402F60FD3BD25163575031ACDCE07538D", "tthor1zg5pz4hgsctyclmu97ynaj3hmjvz9prw4679r0", 20743587},
		{"ETH.ETH", "tthor1vhl7xq52gc7ejn8vrrtkjvw7hl98rnjmsyxhmd", 131680455},
		{"LTC.LTC", "tthor1tf5xd9eklal8l4z096dzhzcpurycgv42vzs0wn", 6458},
	},
}

var testnetArtificialDepthChanges = artificialPoolBallanceChanges{
	200000: {
		{"ETH.ETH", 19527886, -96693},
		{"LTC.LTC", -2985622, -93468},
	},
}
