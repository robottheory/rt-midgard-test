package chain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/DataDog/zstd"
	"github.com/rs/zerolog/log"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
)

const unfinishedFilename = "tmp"
const blocksPerFile int64 = 10000
const compressionLevel = 1

var BLOCKSTORE_NOT_FOUND = errors.New("not found")

type BlockStore struct {
	LastFetchedHeight int64

	ctx                  context.Context
	folder               string
	unfinishedBlocksFile *os.File
	blockWriter          *zstd.Writer
	nextStartHeight      int64
	writeCursorHeight    int64
}

func NewBlockStore(ctx context.Context, folder string) *BlockStore {
	b := &BlockStore{ctx: ctx}
	b.folder = folder
	b.LastFetchedHeight = b.findLastFetchedHeight()
	b.nextStartHeight = b.LastFetchedHeight + 1
	b.writeCursorHeight = b.nextStartHeight
	return b
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

func (b *BlockStore) findLastFetchedHeight() int64 {
	folder := b.folder
	dirEntry, err := os.ReadDir(folder)
	if err != nil {
		log.Warn().Err(err).Msgf("Cannot read folder %s", folder)
		return 0
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
	if block.Height == b.nextStartHeight {
		b.unfinishedBlocksFile = b.createTemporaryFile()
		// TODO(freki): if compressionlevel == 0 keep original writer
		b.blockWriter = zstd.NewWriterLevel(b.unfinishedBlocksFile, compressionLevel)
	}
	bytes := b.marshal(block)
	if _, err := b.blockWriter.Write(bytes); err != nil {
		log.Fatal().Err(err).Msgf("Error writing to %s block %v", b.unfinishedBlocksFile.Name(), b)
	}
	if _, err := b.blockWriter.Write([]byte{'\n'}); err != nil {
		log.Fatal().Err(err).Msgf("Error writing to %s", b.unfinishedBlocksFile.Name())
	}
	b.writeCursorHeight = block.Height
	if block.Height == b.nextStartHeight+blocksPerFile-1 {
		if err := b.blockWriter.Close(); err != nil {
			log.Fatal().Err(err).Msgf("Error closing zstd stream")
		}
		b.createDumpFile()
		b.nextStartHeight = b.nextStartHeight + blocksPerFile
	}
}

func (b *BlockStore) Close() {
	path := filepath.Join(b.folder, unfinishedFilename)
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
	fileName := filepath.Join(b.folder, unfinishedFilename)
	file, err := os.Create(fileName)
	if err != nil {
		log.Fatal().Err(err).Msgf("Cannot open %s", fileName)
	}
	return file
}

func (b *BlockStore) createDumpFile() {
	if b.unfinishedBlocksFile == nil {
		return
	}
	newName := fmt.Sprintf(b.folder+"/%012d", b.writeCursorHeight)
	if _, err := os.Stat(newName); err == nil {
		log.Fatal().Msgf("File already exists %s", newName)
	}
	oldName := b.unfinishedBlocksFile.Name()
	log.Info().Msgf("flush %s", oldName)
	if err := b.unfinishedBlocksFile.Close(); err != nil {
		log.Fatal().Err(err).Msgf("Error closing %s", oldName)
	}
	if err := os.Rename(oldName, newName); err != nil {
		log.Fatal().Err(err).Msgf("Error renaming %s", oldName)
	}
}
