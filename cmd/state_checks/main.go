package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/timeseries"
)

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: "2006-01-02 15:04:05", FullTimestamp: true})
	logrus.SetLevel(logrus.DebugLevel)

	// read configuration file
	// NOTE: Should use same as midgard in order to use the same DB and thornode endpoints
	var c Config
	if len(os.Args) != 2 {
		logrus.Fatal("configuration file must be included as the only argument")
	}
	c = *MustLoadConfigFile(os.Args[1])

	err := envconfig.Process("midgard", &c)
	if err != nil {
		logrus.Fatal("Failed to process config environment variables, ", err)
	}

	ctx := context.Background()

	db.Setup(&c.TimeScale)

	// Get Reference Height and Timestamps
	logrus.Info("Geting latest recorded height...")
	lastHeightRows, err := db.Query(ctx, "SELECT height, timestamp from block_log order by height desc limit 1")
	if err != nil {
		logrus.Fatal(err)
	}
	defer lastHeightRows.Close()

	var (
		lastHeight    int64
		lastTimestamp db.Nano
	)

	if lastHeightRows.Next() {
		err := lastHeightRows.Scan(&lastHeight, &lastTimestamp)
		if err != nil {
			logrus.Fatal(err)
		}
	}

	logrus.Infof("Will use height: %d, timestamp: %d for state check", lastHeight, lastTimestamp)

	// Query Midgard data
	logrus.Infof("Gettting Midgard data...")

	depthsQ := `
	SELECT pool, LAST(asset_E8, block_timestamp), LAST(rune_E8, block_timestamp)
	FROM block_pool_depths
	WHERE block_timestamp <= $1
	GROUP BY pool
	`
	depthsRows, err := db.Query(ctx, depthsQ, lastTimestamp)
	if err != nil {
		logrus.Fatal(err)
	}
	defer depthsRows.Close()

	poolsWithStatus, err := timeseries.GetPoolsStatuses(ctx, lastTimestamp)
	if err != nil {
		logrus.Fatal(err)
	}

	midgardPools := make(map[string]Pool)
	for depthsRows.Next() {
		var pool Pool

		err := depthsRows.Scan(&pool.Pool, &pool.AssetDepth, &pool.RuneDepth)
		if err != nil {
			logrus.Fatal(err)
		}
		pool.Status = poolsWithStatus[pool.Pool]
		if pool.Status == "" {
			pool.Status = timeseries.DefaultPoolStatus
		}
		midgardPools[pool.Pool] = pool
	}

	midgardActiveNodeCount, err := timeseries.ActiveNodeCount(ctx, lastTimestamp)

	// Query Thornode Data
	logrus.Infof("Getting ThorNode data...")

	var (
		thornodePools           []Pool
		thornodeNodes           []Node
		thornodeActiveNodeCount int64
	)
	queryThornode(c, "/pools", lastHeight, &thornodePools)
	queryThornode(c, "/nodes", lastHeight, &thornodeNodes)

	for _, node := range thornodeNodes {
		if node.Status == "active" {
			thornodeActiveNodeCount++
		}
	}

	// Run checks
	var errors []string

	// pools
	for _, thornodePool := range thornodePools {
		midgardPool, ok := midgardPools[thornodePool.Pool]
		prompt := fmt.Sprintf("[Pools:%s]:", thornodePool.Pool)
		delete(midgardPools, thornodePool.Pool)
		if !ok {
			errors = append(errors, fmt.Sprintf("%s Did not find pool in Midgard (Exists in Thornode)", prompt))
			continue
		}

		if midgardPool.RuneDepth != thornodePool.RuneDepth {
			errors = append(errors, fmt.Sprintf("%s RUNE Depth mismatch Thornode: %d, Midgard: %d", prompt, thornodePool.RuneDepth, midgardPool.RuneDepth))
		}

		if midgardPool.AssetDepth != thornodePool.AssetDepth {
			errors = append(errors, fmt.Sprintf("%s Asset Depth mismatch Thornode: %d, Midgard: %d", prompt, thornodePool.AssetDepth, midgardPool.AssetDepth))
		}

		if midgardPool.Status != strings.ToLower(thornodePool.Status) {
			errors = append(errors, fmt.Sprintf("%s Status mismatch Thornode: %s, Midgard: %s", prompt, strings.ToLower(thornodePool.Status), midgardPool.Status))
		}
	}

	for pool := range midgardPools {
		prompt := fmt.Sprintf("[Pools:%s]:", pool)
		errors = append(errors, fmt.Sprintf("%s Did not find pool in Thornode (Exists in Midgard)", prompt))
		continue
	}

	// Nodes
	if thornodeActiveNodeCount != midgardActiveNodeCount {
		errors = append(errors, fmt.Sprintf("[Nodes]: Active Node Count mismatch Thornode: %d, Midgard %d", thornodeActiveNodeCount, midgardActiveNodeCount))
	}

	if len(errors) > 0 {
		logrus.Warnf("%d ERRORS where found", len(errors))
		for _, err := range errors {
			fmt.Printf("\t- %s\n", err)
		}
	} else {
		logrus.Infof("All state checks OK")
	}
}

type Pool struct {
	RuneDepth  int64  `json:"balance_rune,string"`
	AssetDepth int64  `json:"balance_asset,string"`
	Status     string `json:"status"`
	Pool       string `json:"asset"`
}

type Node struct {
	Status string `json:"status"`
}

type Config struct {
	TimeScale db.Config `json:"timescale"`
	ThorChain struct {
		ThorNodeURL string `json:"thornode_url" split_words:"true"`
	}
}

func MustLoadConfigFile(path string) *Config {
	f, err := os.Open(path)
	if err != nil {
		logrus.Fatal("exit on configuration file unavailable: ", err)
	}
	defer f.Close()

	dec := json.NewDecoder(f)

	// prevent config not used due typos
	// dec.DisallowUnknownFields()

	var c Config
	if err := dec.Decode(&c); err != nil {
		logrus.Fatal("exit on malformed configuration: ", err)
	}
	return &c
}

func queryThornode(c Config, urlPath string, height int64, dest interface{}) {
	url := c.ThorChain.ThorNodeURL + urlPath + "?height=" + strconv.FormatInt(height, 10)
	logrus.Debug("Querying thornode: ", url)
	resp, err := http.Get(url)
	if err != nil {
		logrus.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	err = json.Unmarshal(body, dest)
	if err != nil {
		logrus.Fatal(err)
	}
}
