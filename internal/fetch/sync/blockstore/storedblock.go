package blockstore

import (
	tmjson "github.com/tendermint/tendermint/libs/json"
	"gitlab.com/thorchain/midgard/internal/fetch/sync/chain"
)

// Wrapper for chain.Block that makes it possible to serialize with generic libraries
//
// Most serialization libraries (and the CBOR library that we use, in particular) cannot handle
// `chain.Block.Results.ValidatorUpdates`, as it contains an interface field. Fortunately,
// most of the time `ValidatorUpdates` is nil, so we don't have any issues. Whenever it's not
// nil, we serialize it separately, by its native tendermint library and store it separately
// like that.
type storedBlock struct {
	Block                      chain.Block
	SerializedValidatorUpdates []byte
}

func blockToStored(block *chain.Block) (*storedBlock, error) {
	sBlock := storedBlock{Block: *block}

	if sBlock.Block.Results.ValidatorUpdates != nil {
		serialized, err := tmjson.Marshal(sBlock.Block.Results.ValidatorUpdates)
		if err != nil {
			return nil, err
		}
		sBlock.SerializedValidatorUpdates = serialized

		copy0 := *sBlock.Block.Results
		copy0.ValidatorUpdates = nil
		sBlock.Block.Results = &copy0
	}
	return &sBlock, nil
}

// Note: mutates the supplied argument!
func storedToBlock(sBlock *storedBlock) (*chain.Block, error) {
	if sBlock.SerializedValidatorUpdates != nil {
		err := tmjson.Unmarshal(sBlock.SerializedValidatorUpdates,
			&sBlock.Block.Results.ValidatorUpdates)
		if err != nil {
			return nil, err
		}
	}
	return &sBlock.Block, nil
}
