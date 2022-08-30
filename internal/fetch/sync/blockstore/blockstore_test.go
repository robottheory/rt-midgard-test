package blockstore

import (
	"os"
	"testing"

	"crypto/md5"

	"github.com/stretchr/testify/require"
)

func TestGetChunkHashesPath(t *testing.T) {

	files := make(map[string]string)

	files["thorchain"] = "4\n\xfb&\x95\x8eC\xa7\xb1V\xfc\xe3@\x14\xc7W"
	files["thorchain-stagenet-v1"] = "\xa2\xbf\xc5+a\x8b\xfb\xfa\x91\xdah\\\xe92\x95e"
	files["thorchain-testnet-v0"] = "\x17q\x1bi\xf5\xa4\xcfJ8\x1cD\xfe\xf6\xf4X\xb1"

	for key, value := range files {

		data, err := os.ReadFile("../../../../resources/hashes/" + key)
		require.Equal(t, err, nil)

		hash := md5.Sum([]byte(data))
		require.Equal(t, string(hash[:]), value)
	}

}
