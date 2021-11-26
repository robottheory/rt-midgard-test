package sync

import (
	"gitlab.com/thorchain/midgard/internal/fetch/sync/blockstore"
	"gitlab.com/thorchain/midgard/internal/fetch/sync/chain"
	"gitlab.com/thorchain/midgard/internal/util/miderr"
)

type Iterator struct {
	s           *Sync
	height      int64
	finalHeight int64
	chainIt     *chain.Iterator
	bStoreIt    *blockstore.Iterator
}

func NewIterator(s *Sync, startHeight, finalHeight int64) Iterator {
	if CheckBlockStoreBlocks {
		return Iterator{}
	}
	bStoreIt := s.blockStore.Iterator(startHeight)
	ret := Iterator{
		s:           s,
		height:      startHeight,
		finalHeight: finalHeight,
		chainIt:     nil,
		bStoreIt:    &bStoreIt,
	}
	return ret
}

func (i *Iterator) Next() (*chain.Block, error) {
	if CheckBlockStoreBlocks {
		return nil, miderr.InternalErr("Unimplemented")
	}
	if i.finalHeight < i.height {
		return nil, nil
	}

	if i.bStoreIt != nil {
		ret, err := i.bStoreIt.Next()
		if err == nil && ret != nil {
			if ret.Height != i.height {
				return nil, miderr.InternalErrF(
					"BlockStore height not incremented by one. Actual: %d Expected: %d", ret.Height, i.height)
			}
			i.height++
			return ret, nil
		} else {
			if err != nil {
				logger.Error().Err(err).Msg("Blockstore error, switching over to chain")
			} else {
				logger.Info().Msg("Reached blockstore end, switching over to chain")
			}
			i.bStoreIt = nil
			cIt := i.s.chainClient.Iterator(i.height, i.finalHeight)
			i.chainIt = &cIt
		}
	}

	if i.chainIt == nil {
		return nil, miderr.InternalErr("Programming error no iterator present")
	}
	ret, err := i.chainIt.Next()
	if err != nil {
		return nil, err
	}
	if ret != nil {
		if ret.Height != i.height {
			return nil, miderr.InternalErrF(
				"Chain height not incremented by one. Actual: %d Expected: %d", ret.Height, i.height)
		}
		i.height++
		return ret, nil
	}
	return nil, miderr.InternalErr("Programming error, chain returned no result and no error")
}
