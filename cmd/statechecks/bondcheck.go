package main

import (
	"context"
	"log"
	"strconv"

	"github.com/sirupsen/logrus"
	"gitlab.com/thorchain/midgard/internal/db"
)

func getThorNodeBonds(thorNodeUrl string, height int64) map[string]int64 {
	ret := map[string]int64{}
	var nodes []Node
	queryThorNode(thorNodeUrl, "/nodes", height, &nodes)
	for _, node := range nodes {
		bond, err := strconv.ParseInt(node.Bond, 10, 64)
		if err != nil {
			log.Fatal(err)
		}
		ret[node.BondAddress] = bond
	}
	return ret
}

func bondDifferences(thorNodeBonds map[string]int64, midgardBonds map[string]int64) bool {
	different := false
	seen := map[string]bool{}
	for addr, tbond := range thorNodeBonds {
		seen[addr] = true
		mbond, ok := midgardBonds[addr]
		if !ok {
			logrus.Infof(
				"Bond only in Thornode: %d -- %s\n",
				tbond, addr)
			different = true
		} else if tbond != mbond {
			logrus.Infof(
				"Bond mismatch, Thornode: %d Midgard: %d MidgardExcess: %.2f%% -- %s\n",
				tbond, mbond, 100*float64(mbond-tbond)/float64(tbond),
				addr)
			different = true
		} else {
			logrus.Debugf(
				"Bonds match: %d -- %s\n",
				tbond, addr)
		}
	}
	for addr, mbond := range midgardBonds {
		if !seen[addr] {
			logrus.Infof(
				"Bond only in Midgard: %d -- %s\n",
				mbond, addr)
			different = true
		}
	}

	return different
}

func BondDetails(ctx context.Context, thorNodeUrl string) {
	// TODO(huginn): Change to binary search if this becomes excessive.
	// Currently, this make a thor node query for every block that has a bond event.

	logrus.Info("======== Scanning for bond differences")
	var lastHeight int64 = -1

	midgardBonds := map[string]int64{}

	bondEventsQ := `
		SELECT
			COALESCE(from_addr, to_addr),
			bond_type,
			E8,
			height
		FROM bond_events
		JOIN block_log ON block_timestamp = timestamp
		ORDER BY block_timestamp ASC
	`
	rows, err := db.Query(ctx, bondEventsQ)
	if err != nil {
		logrus.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var bondAddress string
		var bond_type string
		var amount, height int64
		err := rows.Scan(&bondAddress, &bond_type, &amount, &height)
		if err != nil {
			logrus.Fatal(err)
		}

		if height != lastHeight {
			if lastHeight != -1 {
				if bondDifferences(getThorNodeBonds(thorNodeUrl, lastHeight), midgardBonds) {
					logrus.Infof("Divergence detected at block height: %d", lastHeight)
					return
				}
			}
			lastHeight = height
		}

		if bond_type == "bond_returned" || bond_type == "bond_cost" {
			amount *= -1
		}
		midgardBonds[bondAddress] += amount
	}
}
