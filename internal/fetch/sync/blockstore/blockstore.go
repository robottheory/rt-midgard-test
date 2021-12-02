package blockstore

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/DataDog/zstd"
	"github.com/rs/zerolog/log"
	"gitlab.com/thorchain/midgard/internal/fetch/sync/chain"
	"gitlab.com/thorchain/midgard/internal/util/miderr"
)

//TODO(freki) replace log.Fatal()-s to log.Warn()-s on write path

type resource string

const unfinishedResource resource = "tmp"
const DefaultBlocksPerFile = 10000
const DefaultCompressionLevel = 1 // 0 means no compression

func (r resource) path(blockStore *BlockStore) string {
	return filepath.Join(blockStore.folder, string(r))
}

func (r resource) isFinished() bool {
	return r != unfinishedResource
}

func (r resource) maxHeight() int64 {
	height, err := strconv.ParseInt(string(r), 10, 64)
	if err != nil {
		// TODO(freki): add error to the return value (miderr.InternalE)
		log.Fatal().Err(err).Msgf("Cannot convert to int64: %s", r)
	}
	return height
}

type BlockStore struct {
	ctx               context.Context
	unfinishedFile    *os.File
	blockWriter       *zstd.Writer
	nextStartHeight   int64
	writeCursorHeight int64

	folder            string
	blocksPerFile     int64
	compressionLevel  int
	lastFetchedHeight int64
}

func NewBlockStore(ctx context.Context, folder string) *BlockStore {
	return NewCustomBlockStore(ctx, folder, DefaultBlocksPerFile, DefaultCompressionLevel)
}

// TODO(freki): Make sure that public functions return sane results for null.
// TODO(freki): log if blockstore is created or not and what the latest height there is.
func NewCustomBlockStore(
	ctx context.Context, folder string, blocksPerFile int64, compressionLevel int) *BlockStore {
	if len(folder) == 0 {
		return nil
	}
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
	it := b.Iterator(height)
	res, err := it.Next()
	if err != nil {
		return nil, err
	}
	if err := it.cleanupCurrentResource(); err != nil {
		return nil, err
	}
	return res, nil
}

// TODO(muninn): consider also modifying main and adding another job there and keep chain.go simpler
func (b *BlockStore) Batch(batch []chain.Block, height int64) error {
	// It can assume blocks are going to be asked in continous order.
	return miderr.InternalErr("Blockstore read not implemented")
}

func (b *BlockStore) Dump(block *chain.Block) {
	if block.Height == b.nextStartHeight {
		b.unfinishedFile = b.createTemporaryFile()
		// TODO(freki): if compressionlevel == 0 keep original writer
		b.blockWriter = zstd.NewWriterLevel(b.unfinishedFile, b.compressionLevel)
	}
	bytes := b.marshal(block)
	if _, err := b.blockWriter.Write(bytes); err != nil {
		log.Fatal().Err(err).Msgf("Error writing to %s block %v", b.unfinishedFile.Name(), b)
	}
	if _, err := b.blockWriter.Write([]byte{'\n'}); err != nil {
		log.Fatal().Err(err).Msgf("Error writing to %s", b.unfinishedFile.Name())
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
	path := unfinishedResource.path(b)
	if err := os.Remove(path); err != nil {
		log.Fatal().Err(err).Msgf("Cannot remove %s", path)
	}
}

func (b *BlockStore) Iterator(startHeight int64) Iterator {
	return Iterator{blockStore: b, nextHeight: startHeight}
}

func (b *BlockStore) findLastFetchedHeight() int64 {
	resources, err := b.getFinishedResources()
	if err != nil || len(resources) == 0 {
		return 0
	}
	return resources[len(resources)-1].maxHeight()
}

//TODO(freki) add caching
func (b *BlockStore) getFinishedResources() ([]resource, error) {
	folder := b.folder
	dirEntries, err := os.ReadDir(folder)
	if err != nil {
		// TODO(freki): add error to the return value (miderr.InternalE)
		log.Warn().Err(err).Msgf("Cannot read folder %s", b.folder)
		return nil, err
	}
	var resources []resource
	for _, de := range dirEntries {
		r := resource(de.Name())
		if r.isFinished() {
			resources = append(resources, r)
		}
	}
	return resources, nil
}

func (b *BlockStore) marshal(block *chain.Block) []byte {
	out, err := json.Marshal(block)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed marshalling block %v", block)
	}
	return out
}

func (b *BlockStore) createTemporaryFile() *os.File {
	path := unfinishedResource.path(b)
	file, err := os.Create(path)
	if err != nil {
		log.Fatal().Err(err).Msgf("Cannot open %s", path)
	}
	return file
}

func (b *BlockStore) createDumpFile() {
	if b.unfinishedFile == nil {
		return
	}
	newName := b.resourcePathFromHeight(b.writeCursorHeight)
	if _, err := os.Stat(newName); err == nil {
		log.Fatal().Msgf("File already exists %s", newName)
	}
	oldName := b.unfinishedFile.Name()
	log.Info().Msgf("BlockStore flushing %s and renaming to %s", oldName, newName)
	if err := b.unfinishedFile.Close(); err != nil {
		log.Fatal().Err(err).Msgf("Error closing %s", oldName)
	}
	if err := os.Rename(oldName, newName); err != nil {
		log.Fatal().Err(err).Msgf("Error renaming %s", oldName)
	}
}

func (b *BlockStore) resourcePathFromHeight(height int64) string {
	return toResource(height).path(b)
}

func toResource(height int64) resource {
	return resource(fmt.Sprintf("/%012d", height))
}

func (b *BlockStore) findResourcePathForHeight(h int64) string {
	resources, err := b.getFinishedResources()
	if err != nil || len(resources) == 0 {
		return ""
	}
	lo, hi := 0, len(resources)-1
	if resources[hi].maxHeight() < h {
		return ""
	}
	for lo < hi {
		mid := lo + (hi-lo)/2
		if resources[mid].maxHeight() < h {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return resources[lo].path(b)
}

type Iterator struct {
	blockStore *BlockStore
	file       *os.File
	zstdReader io.ReadCloser
	reader     *bufio.Reader
	nextHeight int64
}

func (it *Iterator) Next() (*chain.Block, error) {
	if it.isNextResourceReached() {
		if err := it.cleanupCurrentResource(); err != nil {
			return nil, err
		}
		if err := it.openNextResource(); err != nil {
			return nil, err
		}
	}
	return it.unmarshalNextBlock()
}

func (it *Iterator) isNextResourceReached() bool {
	if it.file == nil {
		return true
	}
	return resource(filepath.Base(it.file.Name())).maxHeight() < it.nextHeight
}

func (it *Iterator) cleanupCurrentResource() error {
	if it.reader == nil {
		return nil
	}
	if err := it.zstdReader.Close(); err != nil {
		return err
	}
	if err := it.file.Close(); err != nil {
		return err
	}

	it.file = nil
	it.zstdReader = nil
	it.reader = nil

	return nil
}

func (it *Iterator) openNextResource() error {
	nextResourcePath := it.blockStore.findResourcePathForHeight(it.nextHeight)
	f, err := os.Open(nextResourcePath)
	if err != nil {
		return miderr.InternalErrF("Unable to open resource %s: %v", nextResourcePath, err)
	}

	it.file = f
	it.zstdReader = zstd.NewReader(bufio.NewReader(it.file))
	it.reader = bufio.NewReader(it.zstdReader)
	return nil
}

func (it *Iterator) unmarshalNextBlock() (*chain.Block, error) {
	if it.reader == nil {
		return nil, io.EOF
	}
	prefix := []byte(fmt.Sprintf("{\"height\":%d", it.nextHeight))
	for {
		line, err := it.reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				return nil, err
			}
			if len(line) == 0 {
				return nil, miderr.InternalErrF("blockstore: reached end of file didn't find the block %d", it.nextHeight)
			}
		}
		if !bytes.HasPrefix(line, prefix) {
			continue
		}
		var block chain.Block
		if err := json.Unmarshal(line, &block); err != nil {
			return nil, err
		}
		it.nextHeight++
		return &block, nil
	}
}
