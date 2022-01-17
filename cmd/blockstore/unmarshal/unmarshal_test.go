package main

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	tmjson "github.com/tendermint/tendermint/libs/json"
	"gitlab.com/thorchain/midgard/internal/fetch/sync/chain"
)

/*
json: cannot unmarshal object into Go struct field PublicKey.results.validator_updates.pub_key.Sum of type crypto.isPublicKey_Sum, cannot read block
	9793, 23114, 38596, 52998, 89720, 250282, 297804, 341006, 384208, 427410, 470613, 513816, 557019, 600222, 695988, 739193, 782396, 825600, 869523, 912727, 956651, 999859, 1043064, 1086268, 1130913, 1175558, 1218763, 1261969, 1307335, 1354149, 1445597, 1445627
*/

func TestUnmarshal_without_type_info(t *testing.T) {
	bytes, err := ioutil.ReadFile("block_9793_without_type_info.json")
	assert.Nil(t, err)
	assert.NotNil(t, bytes)
	var block chain.Block
	err = json.Unmarshal(bytes, &block)
	assert.NotNil(t, err)
}

func TestUnmarshal_with_type_info(t *testing.T) {
	bytes, err := ioutil.ReadFile("block_9793_with_type_info.json")
	assert.Nil(t, err)
	assert.NotNil(t, bytes)
	var block chain.Block
	err = tmjson.Unmarshal(bytes, &block)
	assert.Nil(t, err)
}
