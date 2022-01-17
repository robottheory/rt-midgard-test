package blockstore

import (
	"bufio"
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

// TODO(freki): replace log.Fatal()-s to log.Warn()-s on write path

type BlockStore struct {
	cfg               config.BlockStore
	unfinishedFile    *os.File
	blockWriter       io.WriteCloser
	nextStartHeight   int64
	writeCursorHeight int64
	lastFetchedHeight int64
}

// TODO(freki): Make sure that public functions return sane results for null.
// TODO(freki): log if blockstore is created or not and what the latest height there is.
func NewBlockStore(cfg config.BlockStore) *BlockStore {
	if len(cfg.Local) == 0 {
		log.Info().Msgf("BlockStore: not started, local folder not configured")
		return nil
	}
	b := &BlockStore{cfg: cfg}
	b.cleanUp()
	RunWithInterruptSupport(b.updateFromRemote)
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
	if err := it.cleanupCurrentTrunk(); err != nil {
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
		log.Fatal().Err(err).Msgf("BlockStore: error writing to %s block %v", b.unfinishedFile.Name(), b)
	}
	if _, err := b.blockWriter.Write([]byte{'\n'}); err != nil {
		log.Fatal().Err(err).Msgf("BlockStore: error writing to %s", b.unfinishedFile.Name())
	}
	b.writeCursorHeight = block.Height
	if block.Height == b.nextStartHeight+b.cfg.BlocksPerTrunk-1 {
		if err := b.createDumpFile(b.trunkPathFromHeight(b.writeCursorHeight, withoutExtension)); err != nil {
			log.Fatal().Err(err)
		}
		b.nextStartHeight = b.nextStartHeight + b.cfg.BlocksPerTrunk
	}
}

func (b *BlockStore) Close() {
	if err := b.createDumpFile(b.trunkPathFromHeight(b.writeCursorHeight, "."+unfinishedTrunk)); err != nil {
		log.Warn().Err(err)
	}
}

func (b *BlockStore) Iterator(startHeight int64) Iterator {
	return Iterator{blockStore: b, nextHeight: startHeight}
}

func (b *BlockStore) findLastFetchedHeight() int64 {
	trunks, err := b.getLocalTrunks()
	if err != nil || len(trunks) == 0 {
		return 0
	}
	height := trunks[len(trunks)-1].height
	log.Info().Msgf("BlockStore: last fetched height %d", height)
	return height
}

func (b *BlockStore) getLocalDirEntries() ([]os.DirEntry, error) {
	folder := b.cfg.Local
	dirEntries, err := os.ReadDir(folder)
	if err != nil {
		return nil, miderr.InternalErrF("BlockStore: error reading folder %s (%v)", b.cfg.Local, err)
	}
	return dirEntries, nil
}

func (b *BlockStore) getLocalTrunks() ([]*trunk, error) {
	dirEntries, err := b.getLocalDirEntries()
	if err != nil {
		log.Fatal().Err(err)
	}
	var trunks []*trunk
	for _, de := range dirEntries {
		r, err := NewTrunk(de.Name())
		if err != nil {
			log.Fatal().Err(err).Msgf("BlockStore: error reading trunk %s", r.name)
		}
		trunks = append(trunks, r)
	}
	return trunks, nil
}

func (b *BlockStore) getLocalTrunkNames() (map[string]bool, error) {
	localTrunks, err := b.getLocalTrunks()
	if err != nil {
		return nil, err
	}
	res := map[string]bool{}
	if localTrunks == nil {
		return res, nil
	}
	for _, r := range localTrunks {
		res[r.name] = true
	}
	return res, nil
}

func (b *BlockStore) marshal(block *chain.Block) []byte {
	out, err := tmjson.Marshal(block)
	if err != nil {
		log.Fatal().Err(err).Msgf("BlockStore: failed marshalling block %v", block)
	}
	return out
}

func (b *BlockStore) createTemporaryFile() error {
	path := trunk{name: unfinishedTrunk}.localPath(b)
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
		return miderr.InternalErrF("BlockStore: error closing block writer: %v", err)
	}
	if _, err := os.Stat(newName); err == nil {
		return miderr.InternalErrF("BlockStore: error renaming temporary file to already existing: %s (%v)", newName, err)
	}
	oldName := b.unfinishedFile.Name()
	log.Info().Msgf("BlockStore: flushing %s and renaming to %s", oldName, newName)
	if b.blockWriter != b.unfinishedFile {
		if err := b.unfinishedFile.Close(); err != nil {
			return miderr.InternalErrF("BlockStore: error closing %s (%v)", oldName, err)
		}
	}
	if err := os.Rename(oldName, newName); err != nil {
		return miderr.InternalErrF("BlockStore: error renaming %s (%v)", oldName, err)
	}
	return nil
}

func (b *BlockStore) trunkPathFromHeight(height int64, ext string) string {
	return toTrunk(height).localPath(b) + ext
}

func (b *BlockStore) findTrunkPathForHeight(h int64) (string, error) {
	trunks, err := b.getLocalTrunks()
	if err != nil || len(trunks) == 0 {
		return "", err
	}
	lo, hi := 0, len(trunks)-1
	if trunks[hi].height < h {
		return "", io.EOF
	}
	for lo < hi {
		mid := lo + (hi-lo)/2
		if trunks[mid].height < h {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return trunks[lo].localPath(b), nil
}

func (b *BlockStore) cleanUp() {
	dirEntries, err := b.getLocalDirEntries()
	if err != nil {
		log.Fatal().Err(err)
	}
	for _, de := range dirEntries {
		r, err := NewTrunk(de.Name())
		if err != nil {
			path := r.localPath(b)
			log.Info().Msgf("BlockStore: cleanup, removing %s", path)
			if err := os.Remove(path); err != nil {
				log.Fatal().Err(err).Msgf("BlockStore: error cleaning up trunk  %s", path)
			}
		}
	}
	b.unfinishedFile = nil
	b.blockWriter = nil
}

func (b *BlockStore) readTrunkHashes() []*trunk {
	trunks := []*trunk{}
	log.Info().Msgf("BlockStore: reading trunk hashes from %s", b.getTrunkHashesPath())
	f, err := os.Open(b.getTrunkHashesPath())
	if err != nil {
		log.Warn().Err(err).Msgf("BlockStore: error reading trunk hashes")
		return trunks
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
			log.Warn().Err(err).Msgf("BlockStore: error reading trunk hashes")
			break
		}
		entry := string(bytes)
		fields := strings.Fields(entry)
		if len(fields) != 2 {
			log.Warn().Msgf("BlockStore: invalid hash entry %s", entry)
			break
		}
		trunk, err := NewTrunk(fields[1])
		if err != nil {
			log.Warn().Err(err).Msgf("BlockStore: error parsing %s", entry)
			break
		}
		trunk.hash = fields[0]
		if len(trunk.hash) == 0 {
			log.Warn().Err(err).Msgf("BlockStore: invalid hash entry %s", entry)
			break
		}
		if seen[trunk.name] {
			continue
		}
		seen[trunk.name] = true
		trunks = append(trunks, trunk)
	}
	if l := len(trunks); l > 0 {
		log.Info().Msgf("BlockStore: last found hash %v", trunks[l-1])
	} else {
		log.Info().Msgf("BlockStore: no hashes found")
	}
	sort.Slice(trunks, func(i, j int) bool {
		return trunks[i].name < trunks[j].name
	})
	return trunks
}

func (b *BlockStore) getTrunkHashesPath() string {
	// TODO(munnin): replace chain_id with configurable first hash id of the chain (chaos/stage)
	return "./resources/hashes/chain_id"
}
