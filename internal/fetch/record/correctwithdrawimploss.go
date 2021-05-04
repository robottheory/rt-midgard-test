package record

// In 2021-04 ThorNode had two bug when withdrawing with impermanent loss.

// Sometimes when withdrawing the pool units of a member went up, not down:
// https://gitlab.com/thorchain/thornode/-/issues/896
type addInsteadWithdraw struct {
	Pool     string
	RuneAddr string
	Units    int64
}

var addInsteadWithdrawMapMainnet202104 = map[int64]addInsteadWithdraw{
	84876:  {"BTC.BTC", "thor1h7n7lakey4tah37226musffwjhhk558kaay6ur", 2029187601},
	170826: {"BNB.BNB", "thor1t5t5xg7muu3fl2lv6j9ck6hgy0970r08pvx0rz", 31262905},
}

func LoadCorrectionsWithdrawImpLoss(chainID string) AddEventsFuncMap {
	ret := AddEventsFuncMap{}
	if chainID == ChainIDMainnet202104 {
		f := func(d *Demux, meta *Metadata) {
			add, ok := addInsteadWithdrawMapMainnet202104[meta.BlockHeight]
			if ok {
				d.reuse.Stake = Stake{
					Pool:       []byte(add.Pool),
					RuneAddr:   []byte(add.RuneAddr),
					StakeUnits: add.Units,
				}
				Recorder.OnStake(&d.reuse.Stake, meta)
			}
		}
		for k := range addInsteadWithdrawMapMainnet202104 {
			ret[k] = f
		}
	}
	return ret
}
