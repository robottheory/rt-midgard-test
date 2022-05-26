package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/DataDog/zstd"
	tmjson "github.com/tendermint/tendermint/libs/json"

	"gitlab.com/thorchain/midgard/internal/fetch/sync/blockstore"
)

func main() {
	if len(os.Args) == 1 {
		fmt.Println("Usage: [file ...]")
	}
	for _, fn := range os.Args[1:] {
		f, err := os.Open(fn)
		if err != nil {
			log.Fatal(err)
		}
		printChunkFile(bufio.NewReader(zstd.NewReader(f)))
	}
}

func printChunkFile(rd *bufio.Reader) {
	for {
		bytes := nextLine(rd)
		block, err := blockstore.GobLineToBlock(bytes)
		if err != nil {
			log.Fatal(err)
		}
		bytes, err = tmjson.Marshal(block)
		if err != nil {
			log.Fatal(err)
		}
		os.Stdout.Write(bytes)
		os.Stdout.Write([]byte{'\n'})
	}
}

func nextLine(rd *bufio.Reader) []byte {
	bytes, err := rd.ReadBytes('\n')
	if err != nil {
		if err == io.EOF {
			if len(bytes) == 0 {
				os.Exit(0)
			}
			log.Println("premature end")
		}
		log.Fatal(err)
	}
	return bytes
}
