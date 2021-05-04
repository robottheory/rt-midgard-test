// This tool checks if pool units reported reported for each member sum by ThorNode sums up to the
// total units of the pool.
// At the time of writing it does.
package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"github.com/sirupsen/logrus"
	"gitlab.com/thorchain/midgard/config"
	"gitlab.com/thorchain/midgard/internal/db"
)

type ThorNodeSummary struct {
	TotalUnits int64 `json:"pool_units,string"`
}

type LiquidityMembers []struct {
	Units int64 `json:"units,string"`
}

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: "2006-01-02 15:04:05", FullTimestamp: true})
	logrus.SetLevel(logrus.DebugLevel)

	if len(os.Args) != 4 {
		logrus.Fatalf("Provide 3 arguments, %d provided\nUsage: $ unitsum config pool heightOrTimestamp",
			len(os.Args)-1)
	}

	var c config.Config = config.ReadConfigFrom(os.Args[1])
	ctx := context.Background()

	db.Setup(&c.TimeScale)

	db.LoadFirstBlockFromDB(ctx)

	pool := os.Args[2]

	idStr := os.Args[3]
	height, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		logrus.Fatal("Couldn't parse height or timestamp: ", idStr)
	}

	logrus.Info("Checking pool units sum. Pool: ", pool, " Height: ", height)

	var summary ThorNodeSummary
	queryThorNode(c.ThorChain.ThorNodeURL, "/pool/"+pool, height, &summary)
	logrus.Info("Global units: ", summary.TotalUnits)

	var breakdown LiquidityMembers
	queryThorNode(c.ThorChain.ThorNodeURL, "/pool/"+pool+"/liquidity_providers", height, &breakdown)

	var sum2 int64
	for _, member := range breakdown {
		sum2 += member.Units
	}
	logrus.Info("sum2: ", sum2)
	if sum2 == summary.TotalUnits {
		logrus.Info("SAME")
	} else {
		logrus.Info("DIFFERS")
	}
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
