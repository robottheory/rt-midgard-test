package main

import (
	"context"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gitlab.com/thorchain/midgard/config"
	"gitlab.com/thorchain/midgard/internal/fetch/sync/blockstore"
)

var blockStore *blockstore.BlockStore

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
	blockStore = blockstore.NewBlockStore(context.Background(), config.ReadConfig().BlockStoreFolder)
	measureRandomAccess()
	measureSequentialAccess()
}

/*
height:000001599993,t:0000003.00,min:0000000.00,max:0000306.00,avg:0000000.28
height:000001599994,t:0000003.00,min:0000000.00,max:0000306.00,avg:0000000.28
height:000001599995,t:0000003.00,min:0000000.00,max:0000306.00,avg:0000000.28
height:000001599996,t:0000003.00,min:0000000.00,max:0000306.00,avg:0000000.28
height:000001599997,t:0000002.00,min:0000000.00,max:0000306.00,avg:0000000.28
height:000001599998,t:0000003.00,min:0000000.00,max:0000306.00,avg:0000000.28
height:000001599999,t:0000003.00,min:0000000.00,max:0000306.00,avg:0000000.28

height:000001600000,t:0000004.00,min:0000000.00,max:0000306.00,avg:0000000.28
height:000001600001,t:0000006.00,min:0000000.00,max:0000306.00,avg:0000000.28
height:000001600002,t:0000002.00,min:0000000.00,max:0000306.00,avg:0000000.28
height:000001600003,t:0000003.00,min:0000000.00,max:0000306.00,avg:0000000.28
height:000001600004,t:0000004.00,min:0000000.00,max:0000306.00,avg:0000000.28
height:000001600005,t:0000003.00,min:0000000.00,max:0000306.00,avg:0000000.28
height:000001600006,t:0000003.00,min:0000000.00,max:0000306.00,avg:0000000.28
height:000001600007,t:0000003.00,min:0000000.00,max:0000306.00,avg:0000000.28
height:000001600008,t:0000003.00,min:0000000.00,max:0000306.00,avg:0000000.28
height:000001600009,t:0000006.00,min:0000000.00,max:0000306.00,avg:0000000.28
*/
func measureSequentialAccess() {
	height := int64(1800429) // start of block files without issue: 1445628
	it := blockStore.Iterator(height)
	min, max, avg := math.MaxFloat32, 0., 0.
	for {
		timer := Measure()
		b, err := it.Next()
		if err != nil {
			log.Fatal().Err(err).Msgf("Error %v, cannot read block %d\n", err, height)
			break
		}
		t := float64(timer())
		min, max, avg = math.Min(min, t), math.Max(max, t), avg+t
		fmt.Printf("height:%012d,t:%010.2f,min:%010.2f,max:%010.2f,avg:%010.2f\n", b.Height, t, min, max, avg/float64(height))
		if err == io.EOF {
			break
		}
		height++
	}
}

/*
height:000001539410,t:0000205.00,min:0000205.00,max:0000205.00,avg:0000205.00
height:000002633551,t:0000116.00,min:0000116.00,max:0000205.00,avg:0000160.50
height:000000205821,t:0000047.00,min:0000047.00,max:0000205.00,avg:0000122.67
height:000001550051,t:0000005.00,min:0000005.00,max:0000205.00,avg:0000093.25
height:000000493937,t:0000061.00,min:0000005.00,max:0000205.00,avg:0000086.80
height:000001827320,t:0000136.00,min:0000005.00,max:0000205.00,avg:0000095.00
height:000002749758,t:0000311.00,min:0000005.00,max:0000311.00,avg:0000125.86
height:000000436148,t:0000068.00,min:0000005.00,max:0000311.00,avg:0000118.62
height:000001217216,t:0000194.00,min:0000005.00,max:0000311.00,avg:0000127.00
height:000000519449,t:0000146.00,min:0000005.00,max:0000311.00,avg:0000128.90
height:000002238084,t:0000275.00,min:0000005.00,max:0000311.00,avg:0000142.18
height:000002269287,t:0000480.00,min:0000005.00,max:0000480.00,avg:0000170.33
height:000002451574,t:0000034.00,min:0000005.00,max:0000480.00,avg:0000159.85
height:000002038836,t:0000174.00,min:0000005.00,max:0000480.00,avg:0000160.86
height:000001125515,t:0000158.00,min:0000005.00,max:0000480.00,avg:0000160.67
height:000000242873,t:0000026.00,min:0000005.00,max:0000480.00,avg:0000152.25
height:000002304968,t:0000147.00,min:0000005.00,max:0000480.00,avg:0000151.94
height:000000924091,t:0000095.00,min:0000005.00,max:0000480.00,avg:0000148.78
height:000001150790,t:0000030.00,min:0000005.00,max:0000480.00,avg:0000142.53
height:000002393331,t:0000093.00,min:0000005.00,max:0000480.00,avg:0000140.05
height:000002318273,t:0000226.00,min:0000005.00,max:0000480.00,avg:0000144.14
*/
func measureRandomAccess() {
	min, max, avg, cnt := math.MaxFloat32, 0., 0., 0.
	for i := 0; i < 100; i++ {
		cnt++
		timer := Measure()
		height := rand.Int63n(blockStore.LastFetchedHeight())
		b, err := blockStore.SingleBlock(height)
		if err != nil {
			log.Fatal().Err(err).Msgf("Error %v, cannot read block %d\n", err, height)
			break
		}
		t := float64(timer())
		min, max, avg = math.Min(min, t), math.Max(max, t), avg+t
		fmt.Printf("height:%012d,t:%010.2f,min:%010.2f,max:%010.2f,avg:%010.2f\n", b.Height, t, min, max, avg/cnt)
		if err == io.EOF {
			break
		}
	}
}

// TODO(muninn): use timers for measureing
func Measure() func() (milli int) {
	start := time.Now()
	return func() int {
		return int(time.Since(start).Milliseconds())
	}
}
