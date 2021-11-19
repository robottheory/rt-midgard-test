package chain

import (
	"context"
	"errors"

	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	"gitlab.com/thorchain/midgard/config"
)

var BLOCKSTORE_NOT_FOUND = errors.New("not found")

type BlockStore struct {
	ctx context.Context
	cfg *config.Config
}

func (b *BlockStore) DebugFetchResults(height int64) *coretypes.ResultBlockResults {
	return nil
}

func (b *BlockStore) FetchBlock(block *Block, height int64) error {
	return BLOCKSTORE_NOT_FOUND
}

func (b *BlockStore) CatchUp(out chan<- Block, nextHeight int64) (height int64) {
	height = nextHeight
	return
}
