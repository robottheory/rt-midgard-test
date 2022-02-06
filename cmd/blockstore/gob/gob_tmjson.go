package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"

	tmjson "github.com/tendermint/tendermint/libs/json"

	"gitlab.com/thorchain/midgard/internal/fetch/sync/blockstore"
)

func main() {
	rd := bufio.NewReader(os.Stdin)
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
		fmt.Println(string(bytes))
	}
}

func nextLine(rd *bufio.Reader) []byte {
	bytes, err := rd.ReadBytes('\n')
	if len(bytes) == 0 {
		if err != nil {
			if err != io.EOF {
				log.Fatal(err)
			}
			os.Exit(0)
		}
	}
	return bytes
}
