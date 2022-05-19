package main

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"

	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/util/midlog"
)

type thorBalances struct {
	balances  map[string]Balance
	height    int64
	timestamp int64
}

type genesis struct {
	AppState struct {
		Bank struct {
			Balances []genesisBalance `json:"balances"`
		} `json:"bank"`
	} `json:"app_state"`
	InitialHeight string `json:"initial_height"`
}

type genesisBalance struct {
	Address string `json:"address"`
	Coins   []coin `json:"coins"`
}

type coin struct {
	Amount string `json:"amount"`
	Denom  string `json:"denom"`
}

func readThorBalances(thorGenesisPath string) thorBalances {
	f, err := os.Open(thorGenesisPath)
	if err != nil {
		midlog.FatalE(err, "Error reading genesis json")
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	var g genesis
	if err := dec.Decode(&g); err != nil {
		midlog.FatalE(err, "Error parsing genesis json")
	}
	height := parseInt64(g.InitialHeight) - 1
	return thorBalances{
		height:    height,
		timestamp: queryTimestampAtHeight(height),
		balances:  g.getBalances(),
	}
}

func queryTimestampAtHeight(height int64) (timestamp int64) {
	err := db.TheDB.QueryRow(
		`SELECT timestamp FROM block_log WHERE height = $1`,
		height).
		Scan(&timestamp)
	if err != nil {
		midlog.FatalE(err, "Error reading block timestamp from db")
	}

	return
}

func (g genesis) getBalances() map[string]Balance {
	balances := map[string]Balance{}
	for _, bal := range g.AppState.Bank.Balances {
		for _, coin := range bal.Coins {
			b := Balance{
				addr:     bal.Address,
				asset:    normalizeAsset(coin.Denom),
				amountE8: parseInt64(coin.Amount),
			}
			balances[b.key()] = b
		}
	}
	return balances
}

func normalizeAsset(asset string) string {
	switch asset {
	case "rune":
		return "THOR.RUNE"
	}
	return strings.ToUpper(asset)
}

func parseInt64(v string) int64 {
	res, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		midlog.ErrorE(err, "Cannot parse int64")
	}
	return res
}
