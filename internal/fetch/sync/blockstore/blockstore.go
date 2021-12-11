package blockstore

import (
	"context"
	"io"
	"os"

	"github.com/DataDog/zstd"
	"github.com/rs/zerolog/log"
	"gitlab.com/thorchain/midgard/internal/fetch/sync/chain"

	tmjson "github.com/tendermint/tendermint/libs/json"
)

//TODO(freki) replace log.Fatal()-s to log.Warn()-s on write path

const DefaultBlocksPerBatch = 10000
const DefaultCompressionLevel = 1 // 0 means no compression

type BlockStore struct {
	ctx               context.Context
	unfinishedFile    *os.File
	blockWriter       *zstd.Writer
	nextStartHeight   int64
	writeCursorHeight int64

	folder            string
	remoteBucket      string
	blocksPerBatch    int64
	compressionLevel  int
	lastFetchedHeight int64
}

func NewBlockStore(ctx context.Context, localFolder string, remoteBucket string) *BlockStore {
	return NewCustomBlockStore(ctx, localFolder, remoteBucket, DefaultBlocksPerBatch, DefaultCompressionLevel)
}

// TODO(freki): Make sure that public functions return sane results for null.
// TODO(freki): log if blockstore is created or not and what the latest height there is.
func NewCustomBlockStore(
	ctx context.Context, localFolder string, remoteBucket string, blocksPerBatch int64, compressionLevel int) *BlockStore {
	if len(localFolder) == 0 {
		return nil
	}
	b := &BlockStore{ctx: ctx}
	b.folder = localFolder
	b.remoteBucket = remoteBucket
	b.blocksPerBatch = blocksPerBatch
	b.compressionLevel = compressionLevel
	b.cleanUp()
	b.lastFetchedHeight = b.findLastFetchedHeight()
	b.nextStartHeight = b.lastFetchedHeight + 1
	b.writeCursorHeight = b.nextStartHeight
	return b
}

func (b *BlockStore) LastFetchedHeight() int64 {
	if b == nil {
		return 0
	}
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
	if block.Height == b.nextStartHeight+b.blocksPerBatch-1 {
		b.createDumpFile(withoutExtension)
		b.nextStartHeight = b.nextStartHeight + b.blocksPerBatch
	}
}

func (b *BlockStore) Close() {
	b.createDumpFile("." + string(unfinishedResource))
}

func (b *BlockStore) Iterator(startHeight int64) Iterator {
	return Iterator{blockStore: b, nextHeight: startHeight}
}

func (b *BlockStore) findLastFetchedHeight() int64 {
	resources, err := b.getResources()
	if err != nil || len(resources) == 0 {
		return 0
	}
	return resources[len(resources)-1].maxHeight()
}

func (b *BlockStore) getResources() ([]resource, error) {
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
		resources = append(resources, r)
	}
	return resources, nil
}

func (b *BlockStore) marshal(block *chain.Block) []byte {
	out, err := tmjson.Marshal(block)
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

func (b *BlockStore) createDumpFile(ext string) {
	if b.unfinishedFile == nil {
		return
	}
	if err := b.blockWriter.Close(); err != nil {
		log.Fatal().Err(err).Msgf("Error closing zstd stream")
	}
	newName := b.resourcePathFromHeight(b.writeCursorHeight, ext)
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

func (b *BlockStore) resourcePathFromHeight(height int64, ext string) string {
	return toResource(height).path(b) + ext
}

func (b *BlockStore) findResourcePathForHeight(h int64) (string, error) {
	resources, err := b.getResources()
	if err != nil || len(resources) == 0 {
		return "", err
	}
	lo, hi := 0, len(resources)-1
	if resources[hi].maxHeight() < h {
		return "", io.EOF
	}
	for lo < hi {
		mid := lo + (hi-lo)/2
		if resources[mid].maxHeight() < h {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return resources[lo].path(b), nil
}

func (b *BlockStore) cleanUp() {
	res, err := b.getResources()
	if err != nil {
		log.Fatal().Err(err)
	}
	for _, r := range res {
		if _, err := r.toHeight(); err != nil {
			path := r.path(b)
			log.Info().Msgf("BlockStore: cleanup, removing %s", path)
			if err := os.Remove(path); err != nil {
				log.Fatal().Err(err).Msgf("Error cleanin up resource  %s", path)
			}
		}
	}
}
