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
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	"gitlab.com/thorchain/midgard/config"
	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/internal/timeseries/stat"
)

type Pool struct {
	Pool       string `json:"asset"`
	AssetDepth int64  `json:"balance_asset,string"`
	RuneDepth  int64  `json:"balance_rune,string"`
	Units      int64  `json:"pool_units,string"`
	Status     string `json:"status"`
	Timestamp  db.Nano
}

func (pool Pool) String() string {
	return fmt.Sprintf("%s [Asset: %d, Rune: %d, Units: %d]",
		pool.Pool, pool.AssetDepth, pool.RuneDepth, pool.Units)
}

type State struct {
	Pools           map[string]Pool
	ActiveNodeCount int64
}

type Node struct {
	Status  string `json:"status"`
	Address string `json:"node_address"`
	Bond    string `json:"bond"`
}

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: "2006-01-02 15:04:05", FullTimestamp: true})
	logrus.SetLevel(logrus.InfoLevel)
	// logrus.SetLevel(logrus.DebugLevel)

	var c config.Config = config.ReadConfig()

	ctx := context.Background()

	db.Setup(&c.TimeScale)

	db.LoadFirstBlockFromDB(ctx)

	lastHeight, lastTimestamp := getLastBlockFromDB(ctx)
	logrus.Infof("Latest height: %d, timestamp: %d", lastHeight, lastTimestamp)

	midgardState := getMidgardState(ctx, lastHeight, lastTimestamp)
	logrus.Debug("Pools checked: ", midgardState)

	thornodeState := getThornodeState(ctx, c.ThorChain.ThorNodeURL, lastHeight)

	problems := compareStates(midgardState, thornodeState)

	for _, pool := range problems.mismatchingPools {
		binarySearchPool(ctx, c.ThorChain.ThorNodeURL, pool, 1, lastHeight)
	}

	if problems.activeNodeCountError {
		binarySearchNodes(ctx, c.ThorChain.ThorNodeURL, 1, lastHeight)
	}
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
	pools := []string{}
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
		if pool.Status != "suspended" {
			state.Pools[pool.Pool] = pool
			pools = append(pools, pool.Pool)
		}
	}

	until := timestamp + 1
	unitsMap, err := stat.PoolsLiquidityUnitsBefore(ctx, pools, &until)
	for pool, units := range unitsMap {
		s := state.Pools[pool]
		s.Units = units
		state.Pools[pool] = s
	}

	state.ActiveNodeCount, err = timeseries.ActiveNodeCount(ctx, timestamp)
	if err != nil {
		logrus.Fatal(err)
	}
	return
}

func queryThorNode(thorNodeUrl string, urlPath string, height int64, dest interface{}) {
	url := thorNodeUrl + urlPath
	if 0 < height {
		url += "?height=" + strconv.FormatInt(height, 10)
	}
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

// Note: we collect total bonded, but we don't use this data.
// Delete totalBonded if it's not used in the future.
func getThornodeNodesInfo(ctx context.Context, thorNodeUrl string, height int64) (
	nodeCount int64, totalBonded int64) {
	var nodes []Node
	queryThorNode(thorNodeUrl, "/nodes", height, &nodes)
	for _, node := range nodes {
		if strings.ToLower(node.Status) == "active" {
			nodeCount++
		}
		bond, err := strconv.ParseInt(node.Bond, 10, 64)
		if err != nil {
			log.Fatal(err)
		}
		totalBonded += bond
	}
	return
}

// true if active
func allThornodeNodes(ctx context.Context, thorNodeUrl string, height int64) map[string]bool {
	var nodes []Node
	queryThorNode(thorNodeUrl, "/nodes", height, &nodes)
	ret := map[string]bool{}
	for _, node := range nodes {
		ret[node.Address] = (strings.ToLower(node.Status) == "active")
	}
	return ret
}

func getThornodeState(ctx context.Context, thorNodeUrl string, height int64) (state State) {
	logrus.Debug("Getting ThorNode data...")

	var pools []Pool

	queryThorNode(thorNodeUrl, "/pools", height, &pools)
	state.Pools = map[string]Pool{}
	for _, pool := range pools {
		state.Pools[pool.Pool] = pool
	}

	state.ActiveNodeCount, _ = getThornodeNodesInfo(ctx, thorNodeUrl, height)
	return
}

type Problems struct {
	mismatchingPools     []string
	activeNodeCountError bool
}

func compareStates(midgardState, thornodeState State) (problems Problems) {
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

		if midgardPool.Units != thornodePool.Units {
			mismatchingPools[thornodePool.Pool] = true
			fmt.Fprintf(
				&errors, "%s Pool Units mismatch Thornode: %d, Midgard: %d\n",
				prompt, thornodePool.Units, midgardPool.Units)
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
		problems.activeNodeCountError = true
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
		problems.mismatchingPools = append(problems.mismatchingPools, pool)
	}
	sort.Strings(problems.mismatchingPools)
	return
}

func midgardPoolAtHight(ctx context.Context, pool string, height int64) Pool {
	logrus.Debug("Gettting Midgard data at height: ", height)

	q := `
	SELECT block_log.timestamp, asset_e8, rune_e8
	FROM block_pool_depths
	INNER JOIN block_log
	ON block_pool_depths.block_timestamp <= block_log.timestamp
	WHERE height=$1 AND pool = $2
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

	until := ret.Timestamp + 1
	unitsMap, err := stat.PoolsLiquidityUnitsBefore(ctx, []string{pool}, &until)
	if err != nil {
		logrus.Fatal(err)
	}
	ret.Units = unitsMap[pool]

	return ret
}

func findTablesWithColumns(ctx context.Context, columnName string) map[string]bool {
	q := `
	SELECT
		table_name
	FROM information_schema.columns
	WHERE table_schema='public' and column_name=$1
	`
	rows, err := db.Query(ctx, q, columnName)
	if err != nil {
		logrus.Fatal(err)
	}
	defer rows.Close()

	ret := map[string]bool{}
	for rows.Next() {
		var table string
		err := rows.Scan(&table)
		if err != nil {
			logrus.Fatal(err)
		}
		ret[table] = true
	}

	return ret
}

type EventTable struct {
	TableName      string
	PoolColumnName string // "pool" or "asset" or ""
}

func findEventTables(ctx context.Context) []EventTable {
	blockTimestampTables := findTablesWithColumns(ctx, "block_timestamp")
	blockTimestampTables["block_pool_depths"] = false

	poolTables := findTablesWithColumns(ctx, "pool")
	assetTables := findTablesWithColumns(ctx, "asset")
	ret := []EventTable{}
	for table := range blockTimestampTables {
		if poolTables[table] {
			ret = append(ret, EventTable{TableName: table, PoolColumnName: "pool"})
		} else if assetTables[table] {
			ret = append(ret, EventTable{TableName: table, PoolColumnName: "asset"})
		} else {
			ret = append(ret, EventTable{TableName: table, PoolColumnName: ""})
		}
	}
	return ret
}

var eventTablesCache []EventTable
var eventTablesOnce sync.Once

func getEventTables(ctx context.Context) []EventTable {
	eventTablesOnce.Do(func() {
		eventTablesCache = findEventTables(ctx)
	})
	return eventTablesCache
}

func logEventsFromTable(ctx context.Context, eventTable EventTable, pool string, timestamp db.Nano) {
	qargs := []interface{}{timestamp}
	poolFilter := ""
	if eventTable.PoolColumnName != "" {
		poolFilter = eventTable.PoolColumnName + " = $2"
		qargs = append(qargs, pool)
	}

	q := `
	SELECT *
	FROM ` + eventTable.TableName + `
	` + db.Where("block_timestamp = $1", poolFilter)
	rows, err := db.Query(ctx, q, qargs...)
	if err != nil {
		logrus.Fatal(err)
	}
	defer rows.Close()

	colNames, err := rows.Columns()
	if err != nil {
		panic(err)
	}

	eventNum := 0
	for rows.Next() {
		eventNum++
		colsPtr := make([]interface{}, len(colNames))
		for i := range colsPtr {
			var tmp interface{}
			colsPtr[i] = &tmp
		}
		err := rows.Scan(colsPtr...)
		if err != nil {
			logrus.Fatal(err)
		}
		buf := bytes.Buffer{}

		fmt.Fprintf(&buf, "%s [", eventTable.TableName)
		for i := range colNames {
			if i != 0 {
				fmt.Fprintf(&buf, ", ")
			}
			fmt.Fprintf(&buf, "%s: %v", colNames[i], *(colsPtr[i].(*interface{})))
		}
		fmt.Fprintf(&buf, "]")
		logrus.Infof(buf.String())
	}
}

func logAllEventsAtHeight(ctx context.Context, pool string, timestamp db.Nano) {
	eventTables := getEventTables(ctx)
	for _, eventTable := range eventTables {
		logEventsFromTable(ctx, eventTable, pool, timestamp)
	}
}

// Looks up the first difference in the (min, max) range. May choose max.
func binarySearchPool(ctx context.Context, thorNodeUrl string, pool string, minHeight, maxHeight int64) {
	logrus.Infof("=====  [%s] Binary searching in range [%d, %d)", pool, minHeight, maxHeight)

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
		ok := (thorNodePool.AssetDepth == midgardPool.AssetDepth &&
			thorNodePool.RuneDepth == midgardPool.RuneDepth &&
			thorNodePool.Units == midgardPool.Units)
		if ok {
			logrus.Debug("Same at height ", middleHeight)
			minHeight = middleHeight
		} else {
			logrus.Debug("Differ at height ", middleHeight)
			maxHeight = middleHeight
		}
	}

	midgardPoolBefore := midgardPoolAtHight(ctx, pool, maxHeight-1)

	var thorNodePool Pool
	queryThorNode(thorNodeUrl, "/pool/"+pool, maxHeight, &thorNodePool)
	midgardPool := midgardPoolAtHight(ctx, pool, maxHeight)

	logrus.Infof("[%s] First differenct at height: %d timestamp: %d",
		pool, maxHeight, midgardPool.Timestamp)
	logrus.Info("Previous state:  ", midgardPoolBefore)
	logrus.Info("Thornode:        ", thorNodePool)
	logrus.Info("Midgard:         ", midgardPool)

	logrus.Info("Midgard Asset excess:  ", midgardPool.AssetDepth-thorNodePool.AssetDepth)
	logrus.Info("Midgard Rune excess:   ", midgardPool.RuneDepth-thorNodePool.RuneDepth)
	logrus.Info("Midgard Unit excess:   ", midgardPool.Units-thorNodePool.Units)

	logAllEventsAtHeight(ctx, pool, midgardPool.Timestamp)
}

func timestampAtHeight(ctx context.Context, height int64) db.Nano {
	q := `
	SELECT timestamp
	FROM block_log
	WHERE height=$1
	`
	rows, err := db.Query(ctx, q, height)
	if err != nil {
		logrus.Fatal(err)
	}
	defer rows.Close()

	if !rows.Next() {
		logrus.Fatal("No rows selected:", q)
	}
	var ts db.Nano
	err = rows.Scan(&ts)
	if err != nil {
		logrus.Fatal(err)
	}
	return ts
}

func midgardActiveNodeCount(ctx context.Context, height int64) int64 {
	timestamp := timestampAtHeight(ctx, height)
	midgardCount, err := timeseries.ActiveNodeCount(ctx, timestamp)
	if err != nil {
		logrus.Fatal(err)
	}
	return midgardCount
}

func allMidgardNodes(ctx context.Context, height int64) map[string]bool {
	timestamp := timestampAtHeight(ctx, height)
	q := `
	SELECT
		node_addr,
		last(current, block_timestamp) AS status
	FROM update_node_account_status_events
	WHERE block_timestamp <= $1
	GROUP BY node_addr
	`
	rows, err := db.Query(ctx, q, timestamp)
	if err != nil {
		logrus.Fatal(err)
	}
	defer rows.Close()

	ret := map[string]bool{}
	for rows.Next() {
		var addr, status string
		err = rows.Scan(&addr, &status)
		if err != nil {
			logrus.Fatal(err)
		}
		logrus.Debug("Status: ", strings.ToLower(status))
		ret[addr] = (strings.ToLower(status) == "active")
	}
	return ret
}

func excessNodes(str string, a, b map[string]bool) {
	buf := bytes.Buffer{}
	hasdiff := false
	for node, status := range a {
		if !status {
			continue
		}
		status2b, ok := b[node]
		if !ok {
			fmt.Fprintf(&buf, "present: %s - ", node)
			hasdiff = true
		} else if status2b == false {
			fmt.Fprintf(&buf, "active: %s - ", node)
			hasdiff = true
		}
	}
	if hasdiff {
		logrus.Info(str, " excess: ", buf.String())
	} else {
		logrus.Info(str, " OK")
	}
}

// Looks up the first difference in the (min, max) range. May choose max.
func binarySearchNodes(ctx context.Context, thorNodeUrl string, minHeight, maxHeight int64) {
	logrus.Infof("=====  Binary searching active nodes in range [%d, %d)", minHeight, maxHeight)

	for 1 < maxHeight-minHeight {
		middleHeight := (minHeight + maxHeight) / 2
		logrus.Debugf(
			"--- Binary search step [%d, %d] height: %d",
			minHeight, maxHeight, middleHeight)
		thorNodeCount, _ := getThornodeNodesInfo(ctx, thorNodeUrl, middleHeight)
		logrus.Debug("Thornode: ", thorNodeCount)

		midgardCount := midgardActiveNodeCount(ctx, middleHeight)
		logrus.Debug("Midgard: ", midgardCount)
		ok := midgardCount == thorNodeCount
		if ok {
			logrus.Debug("Same at height ", middleHeight)
			minHeight = middleHeight
		} else {
			logrus.Debug("Differ at height ", middleHeight)
			maxHeight = middleHeight
		}
	}

	countBefore := midgardActiveNodeCount(ctx, maxHeight-1)

	thorNodeCount, _ := getThornodeNodesInfo(ctx, thorNodeUrl, maxHeight)
	midgardCount := midgardActiveNodeCount(ctx, maxHeight)

	logrus.Infof("First node differenct at height: %d timestamp: %d",
		maxHeight, timestampAtHeight(ctx, maxHeight))
	logrus.Info("Previous state:  ", countBefore)
	logrus.Info("Thornode:        ", thorNodeCount)
	logrus.Info("Midgard:         ", midgardCount)

	prevThornodeNodes := allThornodeNodes(ctx, thorNodeUrl, maxHeight-1)
	prevMidgardNodes := allMidgardNodes(ctx, maxHeight-1)
	excessNodes("previous thornode vs midgard", prevThornodeNodes, prevMidgardNodes)
	excessNodes("previous midgard vs thornode", prevMidgardNodes, prevThornodeNodes)

	curentThornodeNodes := allThornodeNodes(ctx, thorNodeUrl, maxHeight)
	curentMidgardNodes := allMidgardNodes(ctx, maxHeight)
	excessNodes("Current thornode vs midgard", curentThornodeNodes, curentMidgardNodes)
	excessNodes("Current midgard vs thornode", curentMidgardNodes, curentThornodeNodes)

}
