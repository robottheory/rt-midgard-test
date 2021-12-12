package blockstore

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/rs/zerolog/log"
)

type resource struct {
	name string
	hash string
}

const unfinishedResource = "tmp"
const withoutExtension = ""

func (r resource) localPath(blockStore *BlockStore) string {
	return filepath.Join(blockStore.cfg.LocalFolder, r.name)
}

func (r resource) remotePath(b *BlockStore) string {
	return b.cfg.RemoteBucket + r.name + "?alt=media"
}

func (r resource) maxHeight() int64 {
	height, err := r.toHeight()
	if err != nil {
		// TODO(freki): add error to the return value (miderr.InternalE)
		log.Fatal().Err(err).Msgf("Cannot convert to int64: %s", r)
	}
	return height
}

func (r resource) toHeight() (int64, error) {
	return strconv.ParseInt(r.name, 10, 64)
}

func toResource(height int64) resource {
	return resource{name: fmt.Sprintf("/%012d", height)}
}
