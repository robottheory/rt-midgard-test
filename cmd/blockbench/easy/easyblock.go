package easy

import (
	"time"

	abci "github.com/tendermint/tendermint/abci/types"
)

type EasyResults struct {
	Height           int64                     `json:"height"`
	TxsResults       []*abci.ResponseDeliverTx `json:"txs_results"`
	BeginBlockEvents []abci.Event              `json:"begin_block_events"`
	EndBlockEvents   []abci.Event              `json:"end_block_events"`
	// ValidatorUpdates      []abci.ValidatorUpdate    `json:"validator_updates"`
	ConsensusParamUpdates *abci.ConsensusParams `json:"consensus_param_updates"`
}

type EasyBlock struct {
	Height  int64        `json:"height"`
	Time    time.Time    `json:"time"`
	Hash    []byte       `json:"hash"`
	Results *EasyResults `json:"results"`
}
