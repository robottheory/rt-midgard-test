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
	defer b.cleanUp()
	if err := b.fetchMissingTrunks(ctx); err != nil {
		log.Warn().Err(err).Msgf("BlockStore: error updating from remote")
		return
	}
	log.Info().Msgf("BlockStore: updating from remote done")
}

func (b *BlockStore) fetchMissingTrunks(ctx context.Context) error {
	localTrunks, err := b.getLocalTrunkNames()
	if err != nil {
		return err
	}
	for _, trunkHash := range b.readTrunkHashes() {
		if ctx.Err() != nil {
			log.Info().Msg("BlockStore: fetch interrupted")
			break
		}
		if localTrunks[trunkHash.name] {
			continue
		}
		if err := b.fetchTrunk(trunkHash); err != nil {
			if err == io.EOF {
				log.Info().Msgf("BlockStore: trunk not found %v", trunkHash)
				break
			}
			return err
		}
	}
	return nil
}

// TODO(freki): progress bar
func (b *BlockStore) fetchTrunk(aTrunk *trunk) error {
	log.Info().Msgf("BlockStore: fetching trunk %v", aTrunk)
	resp, err := http.Get(aTrunk.remotePath(b))
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
	if actualHash := hex.EncodeToString(sha256.Sum(nil)); aTrunk.hash != actualHash {
		return miderr.InternalErrF("BlockStore: Trunk hash mismatch, expected %v, received %v", aTrunk, actualHash)
	}
	if err := b.createDumpFile(aTrunk.localPath(b)); err != nil {
		return err
	}

	return nil
}
