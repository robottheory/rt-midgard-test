package blockstore

import (
	"fmt"
	"path/filepath"
	"strconv"

	"gitlab.com/thorchain/midgard/internal/util/miderr"
)

type trunk struct {
	name   string
	height int64
	hash   string
}

const (
	unfinishedTrunk  = "tmp"
	withoutExtension = ""
)

func NewTrunk(name string) (*trunk, error) {
	t := trunk{name: name}
	mh, err := t.maxHeight()
	t.height = mh
	return &t, err
}

func (r trunk) localPath(blockStore *BlockStore) string {
	return filepath.Join(blockStore.cfg.Local, r.name)
}

func (r trunk) remotePath(b *BlockStore) string {
	return b.cfg.Remote + r.name + "?alt=media"
}

func (r trunk) maxHeight() (int64, error) {
	height, err := r.toHeight()
	if err != nil {
		return 0, miderr.InternalErrF("BlockStore: cannot convert trunk %v", r)
	}
	return height, nil
}

func (r trunk) toHeight() (int64, error) {
	return strconv.ParseInt(r.name, 10, 64)
}

func toTrunk(height int64) trunk {
	return trunk{name: fmt.Sprintf("/%012d", height)}
}
