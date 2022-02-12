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

	localTrunks, err := b.getLocalTrunkNames()
	if err != nil {
		log.Warn().Err(err).Msgf("BlockStore: error updating from remote")
		return
	}
	acceptableHashVals := b.readTrunkHashes()
	n := float32(len(acceptableHashVals))
	for i, trunkHash := range acceptableHashVals {
		if ctx.Err() != nil {
			log.Info().Msg("BlockStore: fetch interrupted")
			break
		}
		if localTrunks[trunkHash.name] {
			continue
		}
		log.Info().Msgf("BlockStore:  [%.2f%%] fetching trunk: %v", 100*float32(i)/n, trunkHash.name)
		if err := b.fetchTrunk(trunkHash); err != nil {
			if err == io.EOF {
				log.Info().Msgf("BlockStore: trunk not found %v", trunkHash)
				break
			}
			log.Warn().Err(err).Msgf("BlockStore: error updating from remote")
			return
		}
	}

	log.Info().Msgf("BlockStore: updating from remote done")
}

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
