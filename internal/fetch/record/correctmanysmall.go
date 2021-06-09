package record

// This file contains many small independent corrections

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
func loadWithdrawForwardedAssetCorrections(chainID string) {
	var heightWithOldWithdraws []int64
	if chainID == ChainIDMainnet202104 {
		heightWithOldWithdraws = []int64{
			29113,
			110069,
		}
	}
	for _, height := range heightWithOldWithdraws {
		WithdrawCorrections.Add(height, correctWithdawsForwardedAsset)
	}
}

//////////////////////// Follow ThorNode bug on withdraw (units and rune was added to the pool)

// https://gitlab.com/thorchain/thornode/-/issues/954
// ThorNode added units to a member after a withdraw instead of removing.
// The bug was corrected, but that single has to be accounted for in Midgard.

func loadWithdrawIncreasesUnits(chainID string) {
	if chainID == ChainIDMainnet202104 {
		AdditionalEvents.Add(672275, func(d *Demux, meta *Metadata) {
			d.reuse.Stake = Stake{
				AddBase: AddBase{
					Pool:     []byte("ETH.ETH"),
					RuneAddr: []byte("thor1hyarrh5hslcg3q5pgvl6mp6gmw92c4tpzdsjqg"),
					RuneE8:   2546508574,
				},
				StakeUnits: 967149543,
			}
			Recorder.OnStake(&d.reuse.Stake, meta)
		})
	}
}
