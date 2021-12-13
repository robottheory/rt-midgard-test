package blockstore

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/rs/zerolog/log"
)

type trunk struct {
	name string
	hash string
}

const unfinishedTrunk = "tmp"
const withoutExtension = ""

func (r trunk) localPath(blockStore *BlockStore) string {
	return filepath.Join(blockStore.cfg.Local, r.name)
}

func (r trunk) remotePath(b *BlockStore) string {
	return b.cfg.Remote + r.name + "?alt=media"
}

func (r trunk) maxHeight() int64 {
	height, err := r.toHeight()
	if err != nil {
		// TODO(freki): add error to the return value (miderr.InternalE)
		log.Fatal().Err(err).Msgf("Cannot convert to int64: %s", r)
	}
	return height
}

func (r trunk) toHeight() (int64, error) {
	return strconv.ParseInt(r.name, 10, 64)
}

func toTrunk(height int64) trunk {
	return trunk{name: fmt.Sprintf("/%012d", height)}
}
