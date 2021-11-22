package chain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/rs/zerolog/log"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	"gitlab.com/thorchain/midgard/config"
)

const unfinishedFilename = "tmp"
const blocksPerFile int64 = 20000

var BLOCKSTORE_NOT_FOUND = errors.New("not found")

type BlockStore struct {
	ctx             context.Context
	cfg             *config.Config
	unfinishedFile  *os.File
	fileStartHeight int64
	heightCursor    int64
}

func NewBlockStore(ctx context.Context, cfg *config.Config) *BlockStore {
	bs := &BlockStore{ctx: ctx, cfg: cfg}
	bs.fileStartHeight = bs.LastFetchedHeight() + 1
	bs.heightCursor = bs.fileStartHeight
	return bs
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

func (b *BlockStore) LastFetchedHeight() int64 {
	folder := b.cfg.BlockStoreFolder
	dirEntry, err := os.ReadDir(folder)
	if err != nil {
		log.Fatal().Err(err).Msgf("Cannot read folder %s", folder)
	}

	for i := len(dirEntry) - 1; i >= 0; i-- {
		name := dirEntry[i].Name()
		if name != unfinishedFilename {
			lastHeight, err := strconv.ParseInt(name, 10, 64)
			if err != nil {
				log.Fatal().Err(err).Msgf("Cannot convert to int64: %s", name)
			}
			return lastHeight
		}
	}
	return 0
}

func (b *BlockStore) Dump(block *Block) {
	if block.Height == b.fileStartHeight {
		b.unfinishedFile = b.createTemporaryFile()
	}
	bytes := b.marshal(block)
	if _, err := b.unfinishedFile.Write(bytes); err != nil {
		log.Fatal().Err(err).Msgf("Error writing to %s block %v", b.unfinishedFile.Name(), b)
	}
	if _, err := b.unfinishedFile.Write([]byte{'\n'}); err != nil {
		log.Fatal().Err(err).Msgf("Error writing to %s", b.unfinishedFile.Name())
	}
	b.heightCursor = block.Height
	if block.Height == b.fileStartHeight+blocksPerFile-1 {
		b.createDumpFile()
		b.fileStartHeight = b.fileStartHeight + blocksPerFile
	}
}

func (b *BlockStore) Close() {
	path := filepath.Join(b.cfg.BlockStoreFolder, unfinishedFilename)
	if err := os.Remove(path); err != nil {
		log.Fatal().Err(err).Msgf("Cannot remove %s", path)
	}
}

func (b *BlockStore) marshal(block *Block) []byte {
	out, err := json.Marshal(block)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed marshalling block %v", block)
	}
	return out
}

func (b *BlockStore) createTemporaryFile() *os.File {
	fileName := filepath.Join(b.cfg.BlockStoreFolder, unfinishedFilename)
	file, err := os.Create(fileName)
	if err != nil {
		log.Fatal().Err(err).Msgf("Cannot open %s", fileName)
	}
	return file
}

func (b *BlockStore) createDumpFile() {
	if b.unfinishedFile == nil {
		return
	}
	newName := fmt.Sprintf(b.cfg.BlockStoreFolder+"/%012d", b.heightCursor)
	if _, err := os.Stat(newName); err == nil {
		log.Fatal().Msgf("File already exists %s", newName)
	}
	oldName := b.unfinishedFile.Name()
	log.Info().Msgf("flush %s", oldName)
	if err := b.unfinishedFile.Close(); err != nil {
		log.Fatal().Err(err).Msgf("Error closing %s", oldName)
	}
	if err := os.Rename(oldName, newName); err != nil {
		log.Fatal().Err(err).Msgf("Error renaming %s", oldName)
	}
}
