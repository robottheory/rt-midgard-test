// Regroups a bunch of json blockstore files
// into new slices of blockPerBatch size files
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/DataDog/zstd"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gitlab.com/thorchain/midgard/internal/fetch/sync/blockstore"
	"gitlab.com/thorchain/midgard/internal/fetch/sync/chain"
)

const blocksPerBatch = blockstore.DefaultBlocksPerBatch
const compressionLevel = blockstore.DefaultCompressionLevel

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})

	folder := os.Args[1]
	dirEntry, err := os.ReadDir(folder)
	if err != nil {
		log.Fatal().Err(err).Msgf("Cannot read folder %s", folder)
	}

	// TODO(freki): Consider replacing implementation: Create two Blockstore, one for reading
	//     one for writing and pipe it over
	count := 0
	tmpFile, writer := openTmpFile()
	for _, de := range dirEntry {
		r, inFile := openInFile(folder, de)
		var lastIn []byte
		for {
			inBytes, errRead := r.ReadBytes('\n')
			if errRead != nil {
				if errRead == io.EOF && len(inBytes) == 0 {
					break
				}
				log.Fatal().Err(errRead).Msgf("Error reading from %s", inFile.Name())
			}
			if _, err := writer.Write(inBytes); err != nil {
				log.Fatal().Err(err).Msgf("Error writing to %s", tmpFile.Name())
			}
			lastIn = inBytes
			count++
			if count%blocksPerBatch == 0 {
				writer.Close()
				oldPath := tmpFile.Name()
				newPath := outPath(lastIn, inFile.Name())
				tmpFile.Close()
				if os.Rename(oldPath, newPath) != nil {
					log.Fatal().Err(err).Msgf("Error renaming %s to %s", oldPath, newPath)
				}
				tmpFile, writer = openTmpFile()
			}
		}
		inFile.Close()
	}
}

func outPath(b []byte, inFileName string) string {
	var block chain.Block
	if err := json.Unmarshal(b, &block); err != nil {
		log.Fatal().Err(err).Msgf("Error while unmarshalling from %s", inFileName)
	}
	return filepath.Join("/tmp", fmt.Sprintf("%012d", block.Height))
}

func openInFile(folder string, dirEntry os.DirEntry) (*bufio.Reader, *os.File) {
	name := dirEntry.Name()
	inFileName := filepath.Join(folder, name)
	inFile, err := os.Open(inFileName)
	if err != nil {
		log.Fatal().Err(err).Msgf("Unable to open file %s", inFileName)
	}
	r := bufio.NewReader(inFile)
	return r, inFile
}

func openTmpFile() (*os.File, *zstd.Writer) {
	outFileName := filepath.Join("/", "tmp", "tmp")
	outFile, err := os.Create(outFileName)
	if err != nil {
		log.Fatal().Err(err).Msgf("Unable to create file %s", outFileName)
	}
	writer := zstd.NewWriterLevel(outFile, compressionLevel)
	return outFile, writer
}
