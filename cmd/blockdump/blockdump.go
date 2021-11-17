// Tool for dumping to a json structure the blocks received from ThorNode.
//
// The Output path is configured with the "block_store_folder" configuration parameter
// Each output file contains exactly blocksPerFile number of block events (block batch)
// sent from ThorNode
// Partially fetched block batches are stored in a temporary file.
//
// Each block batch file is named after the last contained block height (padded with zeros to 12 width)
//
// The tool is restartable, and will resume the dump from the last successfully fetched block
// batch (unfinished block batches are discarded)
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/pascaldekloe/metrics/gostat"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gitlab.com/thorchain/midgard/config"
	"gitlab.com/thorchain/midgard/internal/fetch/chain"
	"gitlab.com/thorchain/midgard/internal/fetch/sync"
	"gitlab.com/thorchain/midgard/internal/util/jobs"
	"gitlab.com/thorchain/midgard/internal/util/miderr"
)

const unfinishedFilename = "tmp"

const blocksPerFile int64 = 20000

func main() {
	// TODO(muninn) refactor main into utility functions, use them from here
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
	log.Info().Msgf("Daemon launch as %s", strings.Join(os.Args, " "))

	signals := make(chan os.Signal, 10)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	// include Go runtime metrics
	gostat.CaptureEvery(5 * time.Second)

	c := config.ReadConfig()

	miderr.SetFailOnError(true)

	mainContext, mainCancel := context.WithCancel(context.Background())

	lastFetchedHeight := lastFetchedFileName(&c)

	blocks, fetchJob, _ := sync.StartBlockFetch(mainContext, &c, lastFetchedHeight, func() {
		log.Info().Msg("In sync")
	})

	blockStoreJob := jobs.Start("BlockStore", func() {
		var file *os.File
		heightMarker := lastFetchedHeight + 1
		heightCursor := heightMarker
		for {
			if mainContext.Err() != nil {
				log.Info().Msgf("Error: shutdown process")
				cleanup(&c)
				return
			}
			select {
			case <-mainContext.Done():
				log.Info().Msgf("Done: shutdown process")
				cleanup(&c)
				return
			case block := <-blocks:
				if block.Height == heightMarker {
					file = createTemporaryFile(&c)
				}
				b := marshal(block)
				if _, err := file.Write(b); err != nil {
					log.Fatal().Err(err).Msgf("Error writing to %s block %v", file.Name(), b)
				}
				if _, err := file.Write([]byte{'\n'}); err != nil {
					log.Fatal().Err(err).Msgf("Error writing to %s", file.Name())
				}
				heightCursor = block.Height
				if block.Height == heightMarker+blocksPerFile-1 {
					createDumpFile(file, &c, heightCursor)
					heightMarker = heightMarker + blocksPerFile
				}
			}
		}
	})

	signal := <-signals
	timeout := c.ShutdownTimeout.WithDefault(5 * time.Second)
	log.Info().Msgf("Shutting down services initiated with timeout in %s", timeout)
	mainCancel()
	finishCTX, finishCancel := context.WithTimeout(context.Background(), timeout)
	defer finishCancel()

	jobs.WaitAll(finishCTX,
		fetchJob,
		&blockStoreJob,
	)

	log.Fatal().Msgf("Exit on signal %s", signal)

}

func createTemporaryFile(config *config.Config) *os.File {
	fileName := filepath.Join(config.BlockStoreFolder, unfinishedFilename)
	file, err := os.Create(fileName)
	if err != nil {
		log.Fatal().Err(err).Msgf("Cannot open %s", fileName)
	}
	return file
}

func createDumpFile(file *os.File, config *config.Config, height int64) {
	if file == nil {
		return
	}
	newName := fmt.Sprintf(config.BlockStoreFolder+"/%012d", height)
	if _, err := os.Stat(newName); err == nil {
		log.Fatal().Msgf("File already exists %s", newName)
	}
	oldName := file.Name()
	log.Info().Msgf("flush %s", oldName)
	if err := file.Close(); err != nil {
		log.Fatal().Err(err).Msgf("Error closing %s", oldName)
	}
	if err := os.Rename(oldName, newName); err != nil {
		log.Fatal().Err(err).Msgf("Error renaming %s", oldName)
	}
}

func marshal(block chain.Block) []byte {
	out, err := json.Marshal(block)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed marshalling block %v", block)
	}
	return out
}

func lastFetchedFileName(config *config.Config) int64 {
	folder := config.BlockStoreFolder
	dirEntry, err := os.ReadDir(folder)
	if err != nil {
		log.Fatal().Err(err).Msgf("Cannot read folder %s", folder)
	}

	for i := len(dirEntry) - 1; i >= 0; i-- {
		name := dirEntry[i].Name()
		if name != unfinishedFilename {
			lastHeight, err := strconv.ParseInt(name, 10, 64)
			if err != nil {
				log.Fatal().Err(err).Msgf("Cannot convert to int64: %s", name)
			}
			return lastHeight
		}
	}
	return 0
}

func cleanup(config *config.Config) {
	path := filepath.Join(config.BlockStoreFolder, unfinishedFilename)
	if err := os.Remove(path); err != nil {
		log.Fatal().Err(err).Msgf("Cannot remove %s", path)
	}
}
