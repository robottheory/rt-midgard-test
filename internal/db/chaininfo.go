package db

import (
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"gitlab.com/thorchain/midgard/config"
)

type ChainInfo struct {
	Description         string
	ChainId             string
	EarliestBlockHeight int64
	EarliestBlockTime   time.Time
	latestBlockHeight   int64
	HardForkHeight      *int64
	RootChain           *ChainInfo
}

var Chain ChainInfo

func LiveChainInfoFrom(
	earliestBlockHash []byte,
	earliestBlockHeight int64,
	earliestBlockTime time.Time,
	latestBlockHeight int64,
	forkInfos ...config.ForkInfo,
) ChainInfo {
	ci := ChainInfo{
		Description:         "live",
		ChainId:             PrintableHash(earliestBlockHash),
		EarliestBlockHeight: earliestBlockHeight,
		EarliestBlockTime:   earliestBlockTime,
		latestBlockHeight:   latestBlockHeight}
	if len(forkInfos) == 0 && earliestBlockHeight != 1 {
		log.Fatal().Msgf("ForkInfo: forked chain %s without configured fork info", ci.ChainId)
	}
	if len(forkInfos) != 0 {
		forkInfoMap := forkInfoMap(forkInfos...)
		if fi, found := forkInfoMap[ci.ChainId]; found {
			for {
				nfi, found := forkInfoMap[fi.ParentChainId]
				if !found {
					break
				}
				fi = nfi
			}
			hfh := fi.HardForkHeight
			if fi.ChainId == ci.ChainId {
				if hfh != 0 {
					ci.HardForkHeight = &hfh
				}
				ci.RootChain = &ci
			} else {
				if hfh == 0 {
					log.Fatal().Msgf("ForkInfo: hard fork height not defined for root chain id %s", fi.ChainId)
				}
				if fi.EarliestBlockHeight == 0 {
					log.Fatal().Msgf("ForkInfo: earliest block height not defined for root chain id %s", fi.ChainId)
				}
				if len(fi.EarliestBlockTime) == 0 {
					log.Fatal().Msgf("ForkInfo: earliest block time not defined for root chain id %s", fi.ChainId)
				}
				ci.RootChain = &ChainInfo{
					Description:         "forkinfo",
					ChainId:             fi.ChainId,
					EarliestBlockHeight: fi.EarliestBlockHeight,
					EarliestBlockTime:   StrToTime(fi.EarliestBlockTime),
					latestBlockHeight:   fi.HardForkHeight,
					HardForkHeight:      &hfh}
			}
		}
	}
	return ci
}

func forkInfoMap(forkInfos ...config.ForkInfo) map[string]config.ForkInfo {
	fiMap := map[string]config.ForkInfo{}
	for _, fi := range forkInfos {
		if _, found := fiMap[fi.ChainId]; found {
			log.Fatal().Msgf("ForkInfo: invalid configuration, duplicate chain id %s", fi.ChainId)
		}
		fiMap[fi.ChainId] = fi
	}
	return fiMap
}

func (ci ChainInfo) AssertStartMatch(oi ChainInfo) {
	if ci.ChainId != oi.ChainId {
		log.Fatal().
			Str(ci.Description+" chain id", ci.ChainId).
			Str(oi.Description+" chain id", oi.ChainId).
			Msg(ci.Description + " and " + oi.Description + " chain id mismatch. Choose correct DB instance or wipe the DB Manually.")
	}
	if ci.EarliestBlockHeight != oi.EarliestBlockHeight {
		log.Fatal().
			Int64(ci.Description+" earliest block height", ci.EarliestBlockHeight).
			Int64(oi.Description+" earliest block height", oi.EarliestBlockHeight).
			Msg(ci.Description + " and " + oi.Description + " earliest block height mismatch. Choose correct DB instance or wipe the DB Manually.")
	}
	if ci.EarliestBlockTime != oi.EarliestBlockTime {
		log.Fatal().
			Int64(ci.Description+" earliest timestamp", int64(TimeToNano(ci.EarliestBlockTime))).
			Int64(oi.Description+" earliest timestamp", int64(TimeToNano(oi.EarliestBlockTime))).
			Msg(ci.Description + " and " + oi.Description + " earliest timestamp mismatch. Choose correct DB instance or wipe the DB Manually.")
	}
}

func (ci ChainInfo) AssertHasHeight(height int64) {
	if height < ci.StartHeight() || ci.EndHeight() < height {
		log.Fatal().
			Err(errors.New("height outiside of chain range")).
			Str("chainId", ci.ChainId).
			Int64("height", height).
			Int64("chain.startHeight", ci.StartHeight()).
			Int64("chain.endHeight", ci.EndHeight()).
			Msg("assertion failed")
	}
}

func (ci ChainInfo) StartHeight() int64 {
	return ci.EarliestBlockHeight
}

func (ci ChainInfo) EndHeight() int64 {
	if ci.HardForkHeight == nil {
		return ci.latestBlockHeight
	}
	if *ci.HardForkHeight < ci.latestBlockHeight {
		return *ci.HardForkHeight
	}
	return ci.latestBlockHeight
}

func PrintableHash(hash []byte) string {
	return strings.ToUpper(hex.EncodeToString(hash))
}

func SetChain(chain ChainInfo) {
	Chain = chain
	if Chain.RootChain == nil {
		Chain.RootChain = &Chain
	}
	if len(Chain.RootChain.ChainId) == 0 {
		log.Fatal().Msg("Root chain id not set")
	}
	if Chain.RootChain.EarliestBlockHeight < 1 {
		log.Fatal().Msg("Root chain earliest block height not set")
	}
	log.Info().Msgf("Chain set to: RootChainId[%s],ActualChainId[%s]", Chain.RootChain.ChainId, Chain.ChainId)
	FirstBlock.Set(Chain.RootChain.EarliestBlockHeight, TimeToNano(Chain.RootChain.EarliestBlockTime))
}
