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
)

//TODO(freki): replace log.Fatal()-s to log.Warn()-s on write path

type BlockStore struct {
	cfg               config.BlockStore
	chainId           string
	currentFile       *os.File
	blockWriter       io.WriteCloser
	writeCursorHeight int64
	lastFetchedHeight int64
}

// If chainId != "" then blocks until missing chunks are downloaded from remote repository to local
// folder. During the download the hashes of the remote chunks are checked.
//
// TODO(freki): Make sure that public functions return sane results for null.
// TODO(freki): Log if blockstore is created or not and what the latest height there is.
// TODO(freki): Read acceptable hash values for this specific chainId
func NewBlockStore(ctx context.Context, cfg config.BlockStore, chainId string) *BlockStore {
	if len(cfg.Local) == 0 {
		log.Info().Msgf("BlockStore: not started, local folder not configured")
		return nil
	}
	b := &BlockStore{cfg: cfg, chainId: chainId}
	b.cleanUp()
	if b.chainId != "" {
		b.updateFromRemote(ctx)
	}
	b.lastFetchedHeight = b.findLastFetchedHeight()
	b.writeCursorHeight = b.lastFetchedHeight + 1
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
	if err := it.cleanupCurrentChunk(); err != nil {
		return nil, err
	}
	return res, nil
}

func (b *BlockStore) DumpBlock(block *chain.Block) {
	if b.currentFile == nil {
		err := b.startNewFile()
		if err != nil {
			log.Fatal().Err(err).Msgf(
				"BlockStore: Couldn't create temporary file. Did you create the folder %s ?",
				b.cfg.Local)
		}

		// TODO(freki): if compressionlevel == 0 keep original writer
		b.blockWriter = zstd.NewWriterLevel(b.currentFile, b.cfg.CompressionLevel)
	}

	if b.currentFile == nil {
		log.Fatal().Msgf("BlockStore: currentFile is nil")
	}

	err := writeBlockAsGobLine(block, b.blockWriter)
	if err != nil {
		log.Fatal().Err(err).Msgf("BlockStore: error writing to %s, block height %d",
			b.currentFile.Name(),
			block.Height)
	}

	b.writeCursorHeight = block.Height
	if block.Height%b.cfg.BlocksPerChunk == 0 {
		log.Info().Msgf("BlockStore: creating dump file for height %d", b.writeCursorHeight)

		err = b.finalizeChunk(b.chunkPathFromHeight(b.writeCursorHeight, withoutExtension))
		if err != nil {
			log.Fatal().Err(err).Msg("BlockStore: error creating file")
		}
	}
}

func (b *BlockStore) Close() {
	err := b.finalizeChunk(b.chunkPathFromHeight(b.writeCursorHeight, "."+currentChunkName))
	if err != nil {
		log.Error().Err(err).Msg("BlockStore: error closing")
	}
}

func (b *BlockStore) Iterator(startHeight int64) Iterator {
	return Iterator{blockStore: b, nextHeight: startHeight}
}

func (b *BlockStore) findLastFetchedHeight() int64 {
	chunks, err := b.getLocalChunks()
	if err != nil || len(chunks) == 0 {
		return 0
	}
	height := chunks[len(chunks)-1].height
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

func (b *BlockStore) getLocalChunks() ([]*chunk, error) {
	dirEntries, err := b.getLocalDirEntries()
	if err != nil {
		log.Fatal().Err(err).Msg("BlockStore: error listing directory")
	}
	var chunks []*chunk
	for _, de := range dirEntries {
		r, err := NewChunk(de.Name())
		if err != nil {
			log.Fatal().Err(err).Msgf("BlockStore: error reading chunk %s", r.name)
		}
		chunks = append(chunks, r)
	}
	return chunks, nil
}

func (b *BlockStore) getLocalChunkNames() (map[string]bool, error) {
	localChunks, err := b.getLocalChunks()
	if err != nil {
		return nil, err
	}
	res := map[string]bool{}
	if localChunks == nil {
		return res, nil
	}
	for _, r := range localChunks {
		res[r.name] = true
	}
	return res, nil
}

func (b *BlockStore) startNewFile() error {
	path := chunk{name: currentChunkName}.localPath(b)
	file, err := os.Create(path)
	if err != nil {
		return miderr.InternalErrF("BlockStore: Cannot open %s", path)
	}
	b.currentFile = file
	return nil
}

func (b *BlockStore) finalizeChunk(newName string) error {
	if b.currentFile == nil {
		return miderr.InternalErrF("BlockStore: currentFile is nil, cannot dump it")
	}
	if err := b.blockWriter.Close(); err != nil {
		return miderr.InternalErrF("BlockStore: error closing block writer: %v", err)
	}
	if _, err := os.Stat(newName); err == nil {
		return miderr.InternalErrF("BlockStore: error renaming temporary file to already existing: %s (%v)", newName, err)
	}
	oldName := b.currentFile.Name()
	if b.blockWriter != b.currentFile {
		if err := b.currentFile.Close(); err != nil {
			return miderr.InternalErrF("BlockStore: error closing %s (%v)", oldName, err)
		}
	}
	if err := os.Rename(oldName, newName); err != nil {
		return miderr.InternalErrF("BlockStore: error renaming %s (%v)", oldName, err)
	}
	b.currentFile = nil
	return nil
}

func (b *BlockStore) chunkPathFromHeight(height int64, ext string) string {
	return toChunk(height).localPath(b) + ext
}

func (b *BlockStore) findChunkPathForHeight(h int64) (string, error) {
	chunks, err := b.getLocalChunks()
	if err != nil || len(chunks) == 0 {
		return "", err
	}
	lo, hi := 0, len(chunks)-1
	if chunks[hi].height < h {
		return "", io.EOF
	}
	for lo < hi {
		mid := lo + (hi-lo)/2
		if chunks[mid].height < h {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return chunks[lo].localPath(b), nil
}

func (b *BlockStore) cleanUp() {
	dirEntries, err := b.getLocalDirEntries()
	if err != nil {
		log.Fatal().Err(err).Msg("BlockStore: error listing directory")
	}
	for _, de := range dirEntries {
		r, err := NewChunk(de.Name())
		if err != nil {
			path := r.localPath(b)
			log.Info().Msgf("BlockStore: cleanup, removing %s", path)
			if err := os.Remove(path); err != nil {
				log.Fatal().Err(err).Msgf("BlockStore: error cleaning up chunk  %s", path)
			}
		}
	}
	b.currentFile = nil
	b.blockWriter = nil
}

func (b *BlockStore) readChunkHashes() []*chunk {
	chunks := []*chunk{}
	log.Info().Msgf("BlockStore: reading chunk hashes from %s", b.getChunkHashesPath())
	f, err := os.Open(b.getChunkHashesPath())
	if err != nil {
		log.Error().Err(err).Msgf("BlockStore: error reading chunk hashes")
		return chunks
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
			log.Error().Err(err).Msgf("BlockStore: error reading chunk hashes")
			break
		}
		entry := string(bytes)
		fields := strings.Fields(entry)
		if len(fields) != 2 {
			log.Error().Msgf("BlockStore: invalid hash entry %s", entry)
			break
		}
		chunk, err := NewChunk(fields[1])
		if err != nil {
			log.Error().Err(err).Msgf("BlockStore: error parsing %s", entry)
			break
		}
		chunk.hash = fields[0]
		if len(chunk.hash) == 0 {
			log.Error().Err(err).Msgf("BlockStore: invalid hash entry %s", entry)
			break
		}
		if seen[chunk.name] {
			continue
		}
		seen[chunk.name] = true
		chunks = append(chunks, chunk)
	}
	if l := len(chunks); l > 0 {
		log.Info().Msgf("BlockStore: last found hash %v", chunks[l-1])
	} else {
		log.Info().Msgf("BlockStore: no hashes found")
	}
	sort.Slice(chunks, func(i, j int) bool {
		return chunks[i].name < chunks[j].name
	})
	return chunks
}

func (b *BlockStore) getChunkHashesPath() string {
	return "./resources/hashes/" + b.chainId
}
