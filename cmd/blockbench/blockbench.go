package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"io"
	"os"
	"time"

	"github.com/DataDog/zstd"
	goccyjson "github.com/goccy/go-json"
	jsoniter "github.com/json-iterator/go"
	"github.com/mailru/easyjson"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	segmentiojson "github.com/segmentio/encoding/json"
	tmjson "github.com/tendermint/tendermint/libs/json"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	"gitlab.com/thorchain/midgard/cmd/blockbench/easy"
	"gitlab.com/thorchain/midgard/internal/fetch/sync/chain"
)

func decodeOrig(line []byte, bp *chain.Block) {
	err := tmjson.Unmarshal(line, bp)
	if err != nil {
		log.Fatal().Err(err).Msg("unmarshal")
	}
}

func decodeStdJSON(line []byte, bp *chain.Block) {
	err := json.Unmarshal(line, bp)
	if err != nil {
		log.Fatal().Err(err).Msg("unmarshal")
	}
}

func decodeGoccyJSON(line []byte, bp *chain.Block) {
	err := goccyjson.Unmarshal(line, bp)
	if err != nil {
		log.Fatal().Err(err).Msg("unmarshal")
	}
}

func decodeSegmentioJSON(line []byte, bp *chain.Block) {
	err := segmentiojson.Unmarshal(line, bp)
	if err != nil {
		log.Fatal().Err(err).Msg("unmarshal")
	}
}

func decodeEasyJSON(line []byte, bp *chain.Block) {
	var block easy.EasyBlock
	err := easyjson.Unmarshal(line, &block)
	if err != nil {
		log.Fatal().Err(err).Msg("unmarshal")
	}
	bp.Height = block.Height
	bp.Time = block.Time
	bp.Hash = block.Hash
	var results coretypes.ResultBlockResults
	results.Height = block.Results.Height
	results.TxsResults = block.Results.TxsResults
	results.BeginBlockEvents = block.Results.BeginBlockEvents
	results.EndBlockEvents = block.Results.EndBlockEvents
	results.ConsensusParamUpdates = block.Results.ConsensusParamUpdates
	bp.Results = &results
}

var jsoni = jsoniter.ConfigCompatibleWithStandardLibrary

func decodeIterJSON(line []byte, bp *chain.Block) {
	err := jsoni.Unmarshal(line, bp)
	if err != nil {
		log.Fatal().Err(err).Msg("unmarshal")
	}
}

func decodeGob(line []byte, bp *chain.Block) {
	raw := make([]byte, base64.RawStdEncoding.DecodedLen(len(line)))
	_, err := base64.RawStdEncoding.Decode(raw, line)
	if err != nil {
		log.Fatal().Err(err).Msg("base64 decode")
	}
	buf := bytes.NewBuffer(raw)
	dec := gob.NewDecoder(buf)
	err = dec.Decode(bp)
	if err != nil {
		log.Fatal().Err(err).Msg("gob decode")
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func encodeOrig(bp *chain.Block) []byte {
	res, err := tmjson.Marshal(bp)
	if err != nil {
		log.Fatal().Err(err).Msg("marshal")
	}
	return res
}

func encodeCleanJSON(bp *chain.Block) []byte {
	bp.Results.ValidatorUpdates = nil
	res, err := json.Marshal(bp)
	if err != nil {
		log.Fatal().Err(err).Msg("marshal")
	}
	return res
}

func encodeGob(bp *chain.Block) []byte {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(bp)
	if err != nil {
		log.Fatal().Err(err).Msg("gob encode")
	}
	res := make([]byte, base64.RawStdEncoding.EncodedLen(buf.Len()))
	base64.RawStdEncoding.Encode(res, buf.Bytes())
	return res
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})

	if len(os.Args) < 2 || 3 < len(os.Args) {
		log.Fatal().Msg("usage: go run ./cmd/blockbench infile [outfile]")
	}

	inFile, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal().Err(err).Msgf("open %s", os.Args[1])
	}
	defer inFile.Close()
	inp := bufio.NewReader(zstd.NewReader(inFile))

	var outp *zstd.Writer
	if len(os.Args) == 3 {
		outFile, err := os.Create(os.Args[2])
		if err != nil {
			log.Fatal().Err(err).Msgf("create %s", os.Args[2])
		}
		defer outFile.Close()
		outp = zstd.NewWriterLevel(outFile, 1)
	}

	t := time.Now()
	var count int
	var checksum int64

	for count = 0; count < 1000; count++ {
		line, err := inp.ReadBytes('\n')
		if err != nil {
			if err == io.EOF && len(line) == 0 {
				break
			}
			log.Fatal().Err(err).Msg("read")
		}

		var block chain.Block
		// decodeOrig(line, &block)
		// decodeStdJSON(line, &block)
		// decodeIterJSON(line, &block)
		// decodeGoccyJSON(line, &block)
		// decodeSegmentioJSON(line, &block)
		decodeEasyJSON(line, &block)
		// decodeGob(line, &block)

		checksum += block.Results.Height

		if outp != nil {
			outLine := encodeOrig(&block)
			// outLine := encodeCleanJSON(&block)
			// outLine := encodeGob(&block)
			outLine = append(outLine, '\n')
			if _, err := outp.Write(outLine); err != nil {
				log.Fatal().Err(err).Msg("write")
			}
		}
	}

	if outp != nil {
		outp.Close()
	}

	perBlock := float64(time.Since(t).Milliseconds()) / float64(count)
	log.Info().Int("count", count).Float64("ms", perBlock).Int64("checksum", checksum).Msg("Per block")

}
