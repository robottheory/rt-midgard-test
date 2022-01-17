package main

// Output:
//
// random_access
//     Count: 30
//     Average: 46.9ms
//     Histogram: 3ms: 3.333%, 10ms: 6.667%, 100ms: 93.333%, 300ms: 100.000%,

// sequencial_access
//     Count: 1000
//     Average: 1.31ms
//     Histogram: 3ms: 99.900%, 10ms: 100.000%,

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gitlab.com/thorchain/midgard/config"
	"gitlab.com/thorchain/midgard/internal/fetch/sync/blockstore"
	"gitlab.com/thorchain/midgard/internal/util/timer"
)

const (
	SequentialStartBlock = 990000
	SequentialCount      = 15000
	RandomCount          = 300
)

var blockStore *blockstore.BlockStore

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
	blockStore = blockstore.NewBlockStore(context.Background(), config.ReadConfig().BlockStore.Local)
	measureRandomAccess()
	measureSequentialAccess()
}

func measureSequentialAccess() {
	summary := timer.NewTimer("sequencial_access")
	it := blockStore.Iterator(SequentialStartBlock)
	for i := 0; i < SequentialCount; i++ {
		t := summary.One()
		_, err := it.Next()
		if err != nil {
			log.Fatal().Err(err).Msgf("Cannot read block")
		}
		t()
	}
	fmt.Println(summary.String())
}

func measureRandomAccess() {
	summary := timer.NewTimer("random_access")
	for i := 0; i < RandomCount; i++ {
		t := summary.One()
		height := rand.Int63n(blockStore.LastFetchedHeight())
		_, err := blockStore.SingleBlock(height)
		if err != nil {
			log.Fatal().Err(err).Msgf("Cannot read block")
		}
		t()
	}
	fmt.Println(summary.String())
}
