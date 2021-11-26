package blockstore

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
	"gitlab.com/thorchain/midgard/internal/fetch/sync/chain"
	"gitlab.com/thorchain/midgard/internal/util/miderr"
)

const unfinishedFilename = "tmp"
const DefaultBlocksPerFile int64 = 10000
const DefaultCompressionLevel = 1 // 0 means no compression

var BLOCKSTORE_NOT_FOUND = errors.New("not found")

type BlockStore struct {
	lastFetchedHeight int64

	ctx                  context.Context
	folder               string
	unfinishedBlocksFile *os.File
	blockWriter          *zstd.Writer
	nextStartHeight      int64
	writeCursorHeight    int64
	blocksPerFile        int64
	compressionLevel     int
}

func NewBlockStore(ctx context.Context, folder string) *BlockStore {
	return NewCustomBlockStore(ctx, folder, DefaultBlocksPerFile, DefaultCompressionLevel)
}

// TODO(freki): return nil if Folder empty.
//     Also Make sure that public functions return sane results for null.
// TODO(freki): log if blockstore is created or not and what the latest height there is.
func NewCustomBlockStore(
	ctx context.Context, folder string, blocksPerFile int64, compressionLevel int) *BlockStore {
	b := &BlockStore{ctx: ctx}
	b.folder = folder
	b.blocksPerFile = blocksPerFile
	b.compressionLevel = compressionLevel
	b.lastFetchedHeight = b.findLastFetchedHeight()
	b.nextStartHeight = b.lastFetchedHeight + 1
	b.writeCursorHeight = b.nextStartHeight
	return b
}

func (b *BlockStore) LastFetchedHeight() int64 {
	return b.lastFetchedHeight
}

func (b *BlockStore) HasHeight(height int64) bool {
	return height <= b.lastFetchedHeight
}

func (b *BlockStore) SingleBlock(height int64) (*chain.Block, error) {
	return nil, miderr.InternalErr("Blockstore read not implemented")
}

// TODO(muninn): consider also modifying main and adding another job there and keep chain.go simpler
func (b *BlockStore) Batch(batch []chain.Block, height int64) error {
	// It can assume blocks are going to be asked in continous order.
	return miderr.InternalErr("Blockstore read not implemented")
}

func (b *BlockStore) findLastFetchedHeight() int64 {
	folder := b.folder
	dirEntry, err := os.ReadDir(folder)
	if err != nil {
		// TODO(freki): add error to the return value (miderr.InternalE)
		log.Warn().Err(err).Msgf("Cannot read folder %s", folder)
		return 0
	}

	for i := len(dirEntry) - 1; i >= 0; i-- {
		name := dirEntry[i].Name()
		if name != unfinishedFilename {
			lastHeight, err := strconv.ParseInt(name, 10, 64)
			if err != nil {
				// TODO(freki): add error to the return value (miderr.InternalE)
				log.Fatal().Err(err).Msgf("Cannot convert to int64: %s", name)
			}
			return lastHeight
		}
	}
	return 0
}

func (b *BlockStore) Dump(block *chain.Block) {
	if block.Height == b.nextStartHeight {
		b.unfinishedBlocksFile = b.createTemporaryFile()
		// TODO(freki): if compressionlevel == 0 keep original writer
		b.blockWriter = zstd.NewWriterLevel(b.unfinishedBlocksFile, b.compressionLevel)
	}
	bytes := b.marshal(block)
	if _, err := b.blockWriter.Write(bytes); err != nil {
		log.Fatal().Err(err).Msgf("Error writing to %s block %v", b.unfinishedBlocksFile.Name(), b)
	}
	if _, err := b.blockWriter.Write([]byte{'\n'}); err != nil {
		log.Fatal().Err(err).Msgf("Error writing to %s", b.unfinishedBlocksFile.Name())
	}
	b.writeCursorHeight = block.Height
	if block.Height == b.nextStartHeight+b.blocksPerFile-1 {
		if err := b.blockWriter.Close(); err != nil {
			log.Fatal().Err(err).Msgf("Error closing zstd stream")
		}
		b.createDumpFile()
		b.nextStartHeight = b.nextStartHeight + b.blocksPerFile
	}
}

func (b *BlockStore) Close() {
	path := filepath.Join(b.folder, unfinishedFilename)
	if err := os.Remove(path); err != nil {
		log.Fatal().Err(err).Msgf("Cannot remove %s", path)
	}
}

func (b *BlockStore) marshal(block *chain.Block) []byte {
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

func (b *BlockStore) Iterator(startHeight int64) Iterator {
	return Iterator{}
}

type Iterator struct {
}

func (it *Iterator) Next() (*chain.Block, error) {
	return nil, nil
}
