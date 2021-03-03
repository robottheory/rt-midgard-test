package main

// Automated check. To manually check values go here:
// https://testnet.thornode.thorchain.info/thorchain/pools
// https://testnet.midgard.thorchain.info/v2/pools

import (
	"bytes"
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

type Pool struct {
	Pool       string `json:"asset"`
	AssetDepth int64  `json:"balance_asset,string"`
	RuneDepth  int64  `json:"balance_rune,string"`
	Status     string `json:"status"`
	Timestamp  db.Nano
}

func (pool Pool) String() string {
	return fmt.Sprintf("%s [Asset: %d, Rune: %d]", pool.Pool, pool.AssetDepth, pool.RuneDepth)
}

type State struct {
	Pools           map[string]Pool
	ActiveNodeCount int64
}

type Node struct {
	Status string `json:"status"`
}

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: "2006-01-02 15:04:05", FullTimestamp: true})
	logrus.SetLevel(logrus.InfoLevel)
	// logrus.SetLevel(logrus.DebugLevel)

	// TODO(acsaba): move config parsing into a separate lib.
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

	lastHeight, lastTimestamp := getLastBlockFromDB(ctx)
	logrus.Infof("Latest height: %d, timestamp: %d", lastHeight, lastTimestamp)

	midgardState := getMidgardState(ctx, lastHeight, lastTimestamp)

	thornodeState := getThornodeState(ctx, c.ThorChain.ThorNodeURL, lastHeight, lastTimestamp)

	poolsWithMismatchingDepths := compareStates(midgardState, thornodeState)

	// binarySearch(ctx, c.ThorChain.ThorNodeURL, "ETH.ETH", 1, lastHeight)
	for _, pool := range poolsWithMismatchingDepths {
		binarySearch(ctx, c.ThorChain.ThorNodeURL, pool, 1, lastHeight)
	}
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

func getLastBlockFromDB(ctx context.Context) (lastHeight int64, lastTimestamp db.Nano) {
	logrus.Info("Geting latest recorded height...")
	lastHeightRows, err := db.Query(ctx, "SELECT height, timestamp from block_log order by height desc limit 1")
	if err != nil {
		logrus.Fatal(err)
	}
	defer lastHeightRows.Close()

	if lastHeightRows.Next() {
		err := lastHeightRows.Scan(&lastHeight, &lastTimestamp)
		if err != nil {
			logrus.Fatal(err)
		}
	}
	return
}

func getMidgardState(ctx context.Context, height int64, timestamp db.Nano) (state State) {
	logrus.Debug("Gettting Midgard data at height: ", height, ", timestamp: ", timestamp)

	depthsQ := `
	SELECT pool, LAST(asset_E8, block_timestamp), LAST(rune_E8, block_timestamp)
	FROM block_pool_depths
	WHERE block_timestamp <= $1
	GROUP BY pool
	`
	depthsRows, err := db.Query(ctx, depthsQ, timestamp)
	if err != nil {
		logrus.Fatal(err)
	}
	defer depthsRows.Close()

	poolsWithStatus, err := timeseries.GetPoolsStatuses(ctx, timestamp)
	if err != nil {
		logrus.Fatal(err)
	}

	state.Pools = map[string]Pool{}
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
		state.Pools[pool.Pool] = pool
	}

	state.ActiveNodeCount, err = timeseries.ActiveNodeCount(ctx, timestamp)
	return
}

func queryThorNode(thorNodeUrl string, urlPath string, height int64, dest interface{}) {
	url := thorNodeUrl + urlPath + "?height=" + strconv.FormatInt(height, 10)
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

func getThornodeState(ctx context.Context, thorNodeUrl string, height int64, timestamp db.Nano) (state State) {
	logrus.Debug("Getting ThorNode data...")

	var (
		pools []Pool
		nodes []Node
	)

	queryThorNode(thorNodeUrl, "/pools", height, &pools)
	state.Pools = map[string]Pool{}
	for _, pool := range pools {
		state.Pools[pool.Pool] = pool
	}

	queryThorNode(thorNodeUrl, "/nodes", height, &nodes)
	for _, node := range nodes {
		if node.Status == "active" {
			state.ActiveNodeCount++
		}
	}
	return
}

func compareStates(midgardState, thornodeState State) (poolsWithMismatchingDepths []string) {
	mismatchingPools := map[string]bool{}

	errors := bytes.Buffer{}

	for _, thornodePool := range thornodeState.Pools {
		midgardPool, ok := midgardState.Pools[thornodePool.Pool]
		prompt := fmt.Sprintf("\t- [Pool:%s]:", thornodePool.Pool)
		delete(midgardState.Pools, thornodePool.Pool)
		if !ok {
			fmt.Fprintf(&errors, "%s Did not find pool in Midgard (Exists in Thornode)\n", prompt)
			continue
		}

		if midgardPool.RuneDepth != thornodePool.RuneDepth {
			mismatchingPools[thornodePool.Pool] = true
			fmt.Fprintf(
				&errors, "%s RUNE Depth mismatch Thornode: %d, Midgard: %d\n",
				prompt, thornodePool.RuneDepth, midgardPool.RuneDepth)

		}

		if midgardPool.AssetDepth != thornodePool.AssetDepth {
			mismatchingPools[thornodePool.Pool] = true
			fmt.Fprintf(
				&errors, "%s Asset Depth mismatch Thornode: %d, Midgard: %d\n",
				prompt, thornodePool.AssetDepth, midgardPool.AssetDepth)
		}

		if midgardPool.Status != strings.ToLower(thornodePool.Status) {
			fmt.Fprintf(&errors, "%s Status mismatch Thornode: %s, Midgard: %s\n",
				prompt, strings.ToLower(thornodePool.Status), midgardPool.Status)
		}
	}

	for pool := range midgardState.Pools {
		prompt := fmt.Sprintf("\t- [Pool:%s]:", pool)
		fmt.Fprintf(&errors, "%s Did not find pool in Thornode (Exists in Midgard)\n", prompt)
		continue
	}

	if thornodeState.ActiveNodeCount != midgardState.ActiveNodeCount {
		fmt.Fprintf(
			&errors, "\t- [Nodes]: Active Node Count mismatch Thornode: %d, Midgard %d\n",
			thornodeState.ActiveNodeCount, midgardState.ActiveNodeCount)
	}

	if errors.Len() > 0 {
		logrus.Warnf("ERRORS where found\n%s", errors.String())
	} else {
		logrus.Infof("All state checks OK")
	}

	for pool := range mismatchingPools {
		poolsWithMismatchingDepths = append(poolsWithMismatchingDepths, pool)
	}
	return
}

func midgardPoolAtHight(ctx context.Context, pool string, height int64) Pool {
	logrus.Debug("Gettting Midgard data at height: ", height)

	q := `
	SELECT block_timestamp, asset_e8, rune_e8
	FROM block_pool_depths
	WHERE
		block_timestamp < (SELECT timestamp FROM block_log WHERE height=$1)
		AND pool = $2
	ORDER BY block_timestamp DESC
	LIMIT 1
	`
	rows, err := db.Query(ctx, q, height, pool)
	if err != nil {
		logrus.Fatal(err)
	}
	defer rows.Close()

	ret := Pool{Pool: pool}
	if rows.Next() {
		err := rows.Scan(&ret.Timestamp, &ret.AssetDepth, &ret.RuneDepth)
		if err != nil {
			logrus.Fatal(err)
		}
	}

	return ret
}

// Looks up the first difference in the (min, max) range. May return max.
func binarySearch(ctx context.Context, thorNodeUrl string, pool string, minHeight, maxHeight int64) {
	logrus.Infof("[%s] Binary searching in range [%d, %d)", pool, minHeight, maxHeight)

	for 1 < maxHeight-minHeight {
		middleHeight := (minHeight + maxHeight) / 2
		logrus.Debugf(
			"--- [%s] Binary search step [%d, %d] height: %d",
			pool, minHeight, maxHeight, middleHeight)
		var thorNodePool Pool
		queryThorNode(thorNodeUrl, "/pool/"+pool, middleHeight, &thorNodePool)
		logrus.Debug("Thornode: ", thorNodePool)
		midgardPool := midgardPoolAtHight(ctx, pool, middleHeight)
		logrus.Debug("Midgard: ", midgardPool)
		ok := thorNodePool.AssetDepth == midgardPool.AssetDepth && thorNodePool.RuneDepth == midgardPool.RuneDepth
		if ok {
			logrus.Debug("Same at height ", middleHeight)
			minHeight = middleHeight
		} else {
			logrus.Debug("Differ at height ", middleHeight)
			maxHeight = middleHeight
		}
	}

	logrus.Infof("[%s] First differenct at height: %d", pool, maxHeight)
	var thorNodePool Pool
	queryThorNode(thorNodeUrl, "/pool/"+pool, maxHeight, &thorNodePool)
	midgardPool := midgardPoolAtHight(ctx, pool, maxHeight)
	logrus.Info("Thornode: ", thorNodePool)
	logrus.Info("Midgard:  ", midgardPool)

	logrus.Info("Midgard Asset excess:  ", midgardPool.AssetDepth-thorNodePool.AssetDepth)
	logrus.Info("Midgard Rune excess:   ", midgardPool.RuneDepth-thorNodePool.RuneDepth)
}
