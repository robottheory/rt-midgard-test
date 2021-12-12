package blockstore

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"

	"github.com/rs/zerolog/log"
	"gitlab.com/thorchain/midgard/internal/util/miderr"
)

func (b *BlockStore) updateFromBucket() {
	defer b.cleanUp()
	if err := b.fetchMissingResources(); err != nil {
		log.Warn().Err(err).Msgf("BlockStore: error updating from remote bucket")
		return
	}
	log.Info().Msgf("BlockStore: updating from remote bucket done")
}

func (b *BlockStore) fetchMissingResources() error {
	localResources, err := b.getLocalResourceNames()
	if err != nil {
		return err
	}
	for _, resourceHash := range b.readResourceHashes() {
		if err := b.ctx.Err(); err != nil { // TODO(freki): this won't activate on interrupt
			return err
		}
		if localResources[resourceHash.name] {
			continue
		}
		if err := b.fetchResource(resourceHash); err != nil {
			return err
		}
	}
	return nil
}

// TODO(freki): progress bar
func (b *BlockStore) fetchResource(resource resource) error {
	log.Info().Msgf("BlockStore: fetching resource: %s", resource)
	resp, err := http.Get(resource.remotePath(b))
	if err != nil {
		return err
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
	if resource.hash != hex.EncodeToString(sha256.Sum(nil)) {
		return miderr.InternalErrF("BlockStore: Hash mismatch for resource: %s", resource)
	}
	if err := b.createDumpFile(resource.localPath(b)); err != nil {
		return err
	}

	return nil
}
