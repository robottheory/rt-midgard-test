package blockstore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"

	"github.com/rs/zerolog/log"
	"gitlab.com/thorchain/midgard/internal/util/miderr"
)

func (b *BlockStore) updateFromRemote(ctx context.Context) {
	if b.cfg.Remote == "" {
		return
	}

	defer b.cleanUp()

	localChunks, err := b.getLocalChunkNames()
	if err != nil {
		log.Warn().Err(err).Msgf("BlockStore: error updating from remote")
		return
	}
	acceptableHashVals := b.readChunkHashes()
	n := float32(len(acceptableHashVals))
	for i, chunkHash := range acceptableHashVals {
		if ctx.Err() != nil {
			log.Info().Msg("BlockStore: fetch interrupted")
			break
		}
		if localChunks[chunkHash.name] {
			continue
		}
		log.Info().Msgf("BlockStore:  [%.2f%%] fetching chunk: %v", 100*float32(i)/n, chunkHash.name)
		if err := b.fetchChunk(chunkHash); err != nil {
			if err == io.EOF {
				log.Info().Msgf("BlockStore: chunk not found %v", chunkHash)
				break
			}
			log.Warn().Err(err).Msgf("BlockStore: error updating from remote")
			return
		}
	}

	log.Info().Msgf("BlockStore: updating from remote done")
}

func (b *BlockStore) fetchChunk(aChunk *chunk) error {
	log.Info().Msgf("BlockStore: fetching chunk %v", aChunk)
	resp, err := http.Get(aChunk.remotePath(b))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return io.EOF
	}
	defer resp.Body.Close()
	if err := b.createTemporaryFile(); err != nil {
		return err
	}
	b.blockWriter = b.unfinishedFile
	sha256 := sha256.New()
	if _, err := io.Copy(io.MultiWriter(b.unfinishedFile, sha256), resp.Body); err != nil {
		return err
	}
	if actualHash := hex.EncodeToString(sha256.Sum(nil)); aChunk.hash != actualHash {
		return miderr.InternalErrF("BlockStore: Chunk hash mismatch, expected %v, received %v", aChunk, actualHash)
	}
	if err := b.createDumpFile(aChunk.localPath(b)); err != nil {
		return err
	}

	return nil
}
