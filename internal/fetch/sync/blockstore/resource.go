package blockstore

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

type resource string

const unfinishedResource = "tmp"
const withoutExtension = ""

func (r resource) localPath(blockStore *BlockStore) string {
	return filepath.Join(blockStore.cfg.LocalFolder, string(r))
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
	return strconv.ParseInt(string(r), 10, 64)
}

func toResource(height int64) resource {
	return resource(fmt.Sprintf("/%012d", height))
}

func (r resource) name() resource {
	return r[strings.LastIndex(string(r), "/")+1:]
}
