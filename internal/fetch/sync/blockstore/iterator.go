package blockstore

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/DataDog/zstd"
	tmjson "github.com/tendermint/tendermint/libs/json"
	"gitlab.com/thorchain/midgard/internal/fetch/sync/chain"
	"gitlab.com/thorchain/midgard/internal/util/miderr"
)

type Iterator struct {
	blockStore *BlockStore
	file       *os.File
	zstdReader io.ReadCloser
	reader     *bufio.Reader
	nextHeight int64
}

func (it *Iterator) Next() (*chain.Block, error) {
	if it.blockStore == nil {
		return nil, nil
	}
	if it.isNextResourceReached() {
		if err := it.cleanupCurrentResource(); err != nil {
			return nil, err
		}
		if err := it.openNextResource(); err != nil {
			if err == io.EOF {
				return nil, nil
			}
			return nil, err
		}
	}
	return it.unmarshalNextBlock()
}

func (it *Iterator) isNextResourceReached() bool {
	if it.file == nil {
		return true
	}
	return resource(filepath.Base(it.file.Name())).maxHeight() < it.nextHeight
}

func (it *Iterator) cleanupCurrentResource() error {
	if it.reader == nil {
		return nil
	}
	if err := it.zstdReader.Close(); err != nil {
		return err
	}
	if err := it.file.Close(); err != nil {
		return err
	}

	it.file = nil
	it.zstdReader = nil
	it.reader = nil

	return nil
}

func (it *Iterator) openNextResource() error {
	nextResourcePath, err := it.blockStore.findResourcePathForHeight(it.nextHeight)
	if err != nil {
		return err
	}
	f, err := os.Open(nextResourcePath)
	if err != nil {
		return miderr.InternalErrF("Unable to open resource %s: %v", nextResourcePath, err)
	}

	it.file = f
	it.zstdReader = zstd.NewReader(bufio.NewReader(it.file))
	it.reader = bufio.NewReader(it.zstdReader)
	return nil
}

func (it *Iterator) unmarshalNextBlock() (*chain.Block, error) {
	if it.reader == nil {
		return nil, io.EOF
	}
	prefix := []byte(fmt.Sprintf("{\"height\":\"%d\"", it.nextHeight))
	for {
		line, err := it.reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				return nil, err
			}
			if len(line) == 0 {
				return nil, miderr.InternalErrF("blockstore: reached end of file didn't find the block %d", it.nextHeight)
			}
		}
		if !bytes.HasPrefix(line, prefix) {
			continue
		}
		var block chain.Block
		if err := tmjson.Unmarshal(line, &block); err != nil {
			return nil, err
		}
		it.nextHeight++
		return &block, nil
	}
}
