package blockstore

import (
	"bufio"
	"context"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/DataDog/zstd"
	"github.com/rs/zerolog/log"
	"gitlab.com/thorchain/midgard/config"
	"gitlab.com/thorchain/midgard/internal/fetch/sync/chain"
	"gitlab.com/thorchain/midgard/internal/util/miderr"

	tmjson "github.com/tendermint/tendermint/libs/json"
)

//TODO(freki): replace log.Fatal()-s to log.Warn()-s on write path

type BlockStore struct {
	cfg               config.BlockStore
	ctx               context.Context
	unfinishedFile    *os.File
	blockWriter       io.WriteCloser
	nextStartHeight   int64
	writeCursorHeight int64
	lastFetchedHeight int64
}

// TODO(freki): Make sure that public functions return sane results for null.
// TODO(freki): log if blockstore is created or not and what the latest height there is.
func NewBlockStore(ctx context.Context, cfg config.BlockStore) *BlockStore {
	if len(cfg.LocalFolder) == 0 {
		log.Info().Msgf("BlockStore: not started, local folder not configured")
		return nil
	}
	b := &BlockStore{ctx: ctx, cfg: cfg}
	b.cleanUp()
	b.updateFromBucket()
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
		if err := b.createTemporaryFile(); err != nil {
			log.Fatal().Err(err)
		}
		// TODO(freki): if compressionlevel == 0 keep original writer
		b.blockWriter = zstd.NewWriterLevel(b.unfinishedFile, b.cfg.CompressionLevel)
	}
	bytes := b.marshal(block)
	if _, err := b.blockWriter.Write(bytes); err != nil {
		log.Fatal().Err(err).Msgf("Error writing to %s block %v", b.unfinishedFile.Name(), b)
	}
	if _, err := b.blockWriter.Write([]byte{'\n'}); err != nil {
		log.Fatal().Err(err).Msgf("Error writing to %s", b.unfinishedFile.Name())
	}
	b.writeCursorHeight = block.Height
	if block.Height == b.nextStartHeight+b.cfg.BlocksPerBatch-1 {
		if err := b.createDumpFile(b.resourcePathFromHeight(b.writeCursorHeight, withoutExtension)); err != nil {
			log.Fatal().Err(err)
		}
		b.nextStartHeight = b.nextStartHeight + b.cfg.BlocksPerBatch
	}
}

func (b *BlockStore) Close() {
	if err := b.createDumpFile(b.resourcePathFromHeight(b.writeCursorHeight, "."+unfinishedResource)); err != nil {
		log.Warn().Err(err)
	}
}

func (b *BlockStore) Iterator(startHeight int64) Iterator {
	return Iterator{blockStore: b, nextHeight: startHeight}
}

func (b *BlockStore) findLastFetchedHeight() int64 {
	resources, err := b.getLocalResources()
	if err != nil || len(resources) == 0 {
		return 0
	}
	height := resources[len(resources)-1].maxHeight()
	log.Info().Msgf("BlockStore: last fetched height %d", height)
	return height
}

func (b *BlockStore) getLocalResources() ([]resource, error) {
	folder := b.cfg.LocalFolder
	dirEntries, err := os.ReadDir(folder)
	if err != nil {
		return nil, miderr.InternalErrF("BlockStore: Error reading folder %s (%v)", b.cfg.LocalFolder, err)
	}
	var resources []resource
	for _, de := range dirEntries {
		r := resource{name: de.Name()}
		resources = append(resources, r)
	}
	return resources, nil
}

func (b *BlockStore) getLocalResourceNames() (map[string]bool, error) {
	localResources, err := b.getLocalResources()
	if err != nil {
		return nil, err
	}
	res := map[string]bool{}
	if localResources == nil {
		return res, nil
	}
	for _, r := range localResources {
		res[r.name] = true
	}
	return res, nil
}

func (b *BlockStore) marshal(block *chain.Block) []byte {
	out, err := tmjson.Marshal(block)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed marshalling block %v", block)
	}
	return out
}

func (b *BlockStore) createTemporaryFile() error {
	path := resource{name: unfinishedResource}.localPath(b)
	file, err := os.Create(path)
	if err != nil {
		return miderr.InternalErrF("BlockStore: Cannot open %s", path)
	}
	b.unfinishedFile = file
	return nil
}

func (b *BlockStore) createDumpFile(newName string) error {
	if b.unfinishedFile == nil {
		return nil
	}
	if err := b.blockWriter.Close(); err != nil {
		return miderr.InternalErrF("BlockStore: Error closing block writer: %v", err)
	}
	if _, err := os.Stat(newName); err == nil {
		return miderr.InternalErrF("BlockStore: Error renaming temporary file to already already existing: %s (%v)", newName, err)
	}
	oldName := b.unfinishedFile.Name()
	log.Info().Msgf("BlockStore: flushing %s and renaming to %s", oldName, newName)
	if b.blockWriter != b.unfinishedFile {
		if err := b.unfinishedFile.Close(); err != nil {
			return miderr.InternalErrF("BlockStore: Error closing %s (%v)", oldName, err)
		}
	}
	if err := os.Rename(oldName, newName); err != nil {
		return miderr.InternalErrF("BlockStore: Error renaming %s (%v)", oldName, err)
	}
	return nil
}

func (b *BlockStore) resourcePathFromHeight(height int64, ext string) string {
	return toResource(height).localPath(b) + ext
}

func (b *BlockStore) findResourcePathForHeight(h int64) (string, error) {
	resources, err := b.getLocalResources()
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
	return resources[lo].localPath(b), nil
}

func (b *BlockStore) cleanUp() {
	res, err := b.getLocalResources()
	if err != nil {
		log.Fatal().Err(err)
	}
	for _, r := range res {
		if _, err := r.toHeight(); err != nil {
			path := r.localPath(b)
			log.Info().Msgf("BlockStore: cleanup, removing %s", path)
			if err := os.Remove(path); err != nil {
				log.Fatal().Err(err).Msgf("BlockStore: Error cleaning up resource  %s\n", path)
			}
		}
	}
	b.unfinishedFile = nil
	b.blockWriter = nil
}

func (b *BlockStore) readResourceHashes() []resource {
	resources := []resource{}
	log.Info().Msgf("BlockStore: reading resource hashes from %s", b.getResourceHashesPath())
	f, err := os.Open(b.getResourceHashesPath())
	if err != nil {
		log.Warn().Err(err).Msgf("BlockStore: error reading resource hashes")
		return resources
	}
	defer f.Close()
	r := bufio.NewReader(f)
	seen := map[string]bool{}
	for {
		bytes, err := r.ReadBytes('\n')
		if err != nil {
			if err == io.EOF && len(bytes) == 0 {
				break
			}
			log.Warn().Err(err).Msgf("BlockStore: error reading resource hashes")
			break
		}
		entry := string(bytes)
		fields := strings.Fields(entry)
		if len(fields) != 2 {
			log.Warn().Msgf("BlockStore: invalid hash entry %s", entry)
			break
		}
		resource := resource{name: fields[1], hash: fields[0]}
		if seen[resource.name] {
			continue
		}
		seen[resource.name] = true
		resources = append(resources, resource)
	}
	if l := len(resources); l > 0 {
		log.Info().Msgf("BlockStore: last found hash %v", resources[l-1])
	} else {
		log.Info().Msgf("BlockStore: no hashes found")
	}
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].name < resources[j].name
	})
	return resources
}

func (b *BlockStore) getResourceHashesPath() string {
	// TODO(munnin): replace chain_id with configurable first hash id of the chain (chaos/stage)
	return "./resources/hashes/chain_id"
}
