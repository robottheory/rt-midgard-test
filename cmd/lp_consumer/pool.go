package main

import (
	"encoding/json"
	"fmt"

	"gitlab.com/thorchain/midgard/internal/fetch/record"
	"gitlab.com/thorchain/midgard/internal/util/kafka"
	"gitlab.com/thorchain/midgard/internal/util/midlog"
)

const withdrawCoinKeptHeight = 1970000

type pool struct {
	AddCount      int64
	WithdrawCount int64
	SwapCount     int64

	AssetE8Depth     int64
	RuneE8Depth      int64
	SynthE8Depth     int64
	AssetInRuneTotal int64

	StakeUnits      int64
	RewardPerUnit   float64
	RuneFeePerUnit  float64
	AssetFeePerUnit float64

	// Keyed by partition
	LastEventIndexes map[int32]kafka.EventIdx
}

func NewPool() *pool {
	p := new(pool)
	p.LastEventIndexes = make(map[int32]kafka.EventIdx)

	return p
}

func (p *pool) Encode(value interface{}) ([]byte, error) {
	if _, isPool := value.(*pool); !isPool {
		return nil, fmt.Errorf("Codec requires value *pool, got %T", value)
	}
	return json.Marshal(value)
}

func (p *pool) Decode(data []byte) (interface{}, error) {
	var (
		c   pool
		err error
	)
	err = json.Unmarshal(data, &c)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshaling pool: %v", err)
	}
	return &c, nil
}

func (p *pool) AddLiquidity(stake record.Stake) int64 {
	var assetInRune int64

	p.AddCount++
	p.AssetE8Depth += stake.AssetE8
	p.RuneE8Depth += stake.RuneE8
	p.StakeUnits += stake.StakeUnits

	if p.AssetE8Depth != 0 {
		assetInRune = int64(float64(stake.AssetE8)*(float64(p.RuneE8Depth)/float64(p.AssetE8Depth)) + 0.5)
	}

	p.AssetInRuneTotal += assetInRune

	return assetInRune
}

func (p *pool) WithdrawLiquidity(idx kafka.EventIdx, unstake record.Unstake) int64 {
	var assetInRune int64

	// Calculated prior to withdrawal
	if p.AssetE8Depth != 0 {
		assetInRune = int64(float64(unstake.EmitAssetE8)*(float64(p.RuneE8Depth)/float64(p.AssetE8Depth)) + 0.5)
	}

	p.WithdrawCount++
	p.AssetE8Depth -= unstake.EmitAssetE8
	p.RuneE8Depth -= unstake.EmitRuneE8
	p.StakeUnits -= unstake.StakeUnits

	p.RuneE8Depth += unstake.ImpLossProtectionE8

	p.AssetInRuneTotal -= assetInRune

	// TODO: Move this to the event producer
	if idx.Height < withdrawCoinKeptHeight {
		if unstake.AssetE8 != 0 && string(unstake.Pool) == string(unstake.Asset) {
			p.AssetE8Depth += unstake.AssetE8
		}
	}

	return assetInRune
}

func (p *pool) Swap(swap record.Swap) {
	fromCoin := record.GetCoinType(swap.FromAsset)
	toCoin := record.GetCoinType(swap.ToAsset)

	if fromCoin == record.UnknownCoin || toCoin == record.UnknownCoin {
		midlog.Warn("Unknown coin in swap, skipping")
		return
	}

	p.SwapCount++
	var direction record.SwapDirection
	switch {
	case fromCoin == record.Rune && toCoin == record.AssetNative:
		direction = record.RuneToAsset
	case fromCoin == record.AssetNative && toCoin == record.Rune:
		direction = record.AssetToRune
	case fromCoin == record.Rune && toCoin == record.AssetSynth:
		direction = record.RuneToSynth
	case fromCoin == record.AssetSynth && toCoin == record.Rune:
		direction = record.SynthToRune
	}

	if direction%2 == 1 {
		p.RuneFeePerUnit += float64(swap.LiqFeeE8) / float64(p.StakeUnits)
	} else if direction%2 == 0 {
		p.AssetFeePerUnit += float64(swap.LiqFeeE8) / float64(p.StakeUnits)
	}

	if toCoin == record.Rune {
		// Swap adds pool asset in exchange of RUNE.
		if fromCoin == record.AssetNative {
			p.AssetE8Depth += swap.FromE8
		}
		// Swap burns synths in exchange of RUNE.
		if fromCoin == record.AssetSynth {
			p.SynthE8Depth -= swap.FromE8
		}
		p.RuneE8Depth -= swap.ToE8
	} else {
		// Swap adds RUNE to pool in exchange of asset.
		p.RuneE8Depth += swap.FromE8
		if toCoin == record.AssetNative {
			p.AssetE8Depth -= swap.ToE8
		}
		// Swap mints synths in exchange of RUNE.
		if toCoin == record.AssetSynth {
			p.SynthE8Depth += swap.ToE8
		}
	}
}

func (p *pool) Donate(add record.Add) {
	p.AssetE8Depth += add.AssetE8
	p.RuneE8Depth += add.RuneE8
}

func (p *pool) Slash(slash record.Slash) {
	for _, a := range slash.Amounts {
		coinType := record.GetCoinType(a.Asset)
		switch coinType {
		case record.Rune:
			p.RuneE8Depth += a.E8
		case record.AssetNative:
			p.AssetE8Depth += a.E8
		}
	}
}

func (p *pool) PoolBalChange(poolBalChange record.PoolBalanceChange) {
	assetAmount := poolBalChange.AssetAmt
	if assetAmount != 0 {
		if !poolBalChange.AssetAdd {
			assetAmount *= -1
		}
		p.AssetE8Depth += assetAmount
	}
	runeAmount := poolBalChange.RuneAmt
	if runeAmount != 0 {
		if !poolBalChange.RuneAdd {
			runeAmount *= -1
		}
		p.RuneE8Depth += runeAmount
	}
}

func (p *pool) Errata(errata record.Errata) {
	p.AssetE8Depth += errata.AssetE8
	p.RuneE8Depth += errata.RuneE8
}

func (p *pool) Fee(fee record.Fee) {
	coinType := record.GetCoinType(fee.Asset)

	if !record.IsRune(fee.Asset) {
		if coinType == record.AssetNative {
			p.AssetE8Depth += fee.AssetE8
		}
		if coinType == record.AssetSynth {
			p.SynthE8Depth += -fee.AssetE8
		}
		p.RuneE8Depth += -fee.PoolDeduct
	}
}

func (p *pool) Gas(gas record.Gas) {
	p.AssetE8Depth += -gas.AssetE8
	p.RuneE8Depth += gas.RuneE8
}

// Rewards expects that this event contains only one pool
// Events that contain more than one pool should be split into multiple events
func (p *pool) Rewards(rewards record.Rewards) {
	if len(rewards.PerPool) != 1 {
		midlog.ErrorF("Received rewards event with wrong number of pools, expect 1 got %v",
			len(rewards.PerPool))
		return
	}

	amt := rewards.PerPool[0]
	p.RuneE8Depth += amt.E8
	if p.StakeUnits != 0 {
		p.RewardPerUnit += float64(amt.E8) / float64(p.StakeUnits)
	}
}
