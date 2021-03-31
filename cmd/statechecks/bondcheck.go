package main

// This is unused for now because bond reconstruction is not correct in it's current form.

import (
	"context"
	"log"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"gitlab.com/thorchain/midgard/internal/db"
)

func getThorNodeBonds(thorNodeUrl string) map[string]int64 {
	ret := map[string]int64{}
	var nodes []Node
	queryThorNode(thorNodeUrl, "/nodes", -1, &nodes)
	for _, node := range nodes {
		bond, err := strconv.ParseInt(node.Bond, 10, 64)
		if err != nil {
			log.Fatal(err)
		}
		ret[node.Address] = bond
	}
	return ret
}

func getMidgardBonds(ctx context.Context) map[string]int64 {
	ret := map[string]int64{}

	bondPaidQ := `
		SELECT
			memo,
			asset_E8,
			bond_type
		FROM bond_events
	`
	bondPaidRows, err := db.Query(ctx, bondPaidQ)
	if err != nil {
		logrus.Fatal(err)
	}
	defer bondPaidRows.Close()

	for bondPaidRows.Next() {
		var memo, bond_type string
		var assetE8 int64
		err := bondPaidRows.Scan(&memo, &assetE8, &bond_type)
		if err != nil {
			logrus.Fatal(err)
		}
		if bond_type == "bond_returned" {
			assetE8 *= -1
		}
		nodeAddress := strings.Split(memo, ":")[1]
		ret[nodeAddress] += assetE8
	}

	return ret
}

//nolint
func BondDetails(ctx context.Context, thorNodeUrl string) {
	logrus.Info("======== Checking bond differences per node")
	thorNodeBonds := getThorNodeBonds(thorNodeUrl)
	midgardBonds := getMidgardBonds(ctx)
	seen := map[string]bool{}
	for addr, tbond := range thorNodeBonds {
		seen[addr] = true
		mbond := midgardBonds[addr]
		if tbond != mbond {
			logrus.Infof(
				"Bonded mismatch Thornode: %d Midgard %d MidgardExcess %.2f%% -- %s\n",
				tbond, mbond, 100*float64(mbond-tbond)/float64(tbond),
				addr)
		}
	}
	for addr, mbond := range midgardBonds {
		if !seen[addr] {
			logrus.Infof(
				"Bond only in Midgard: %d -- %s\n",
				mbond, addr)

		}
	}
}
