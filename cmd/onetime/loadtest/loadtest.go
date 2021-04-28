package main

import (
	"context"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"gitlab.com/thorchain/midgard/config"
	"gitlab.com/thorchain/midgard/internal/db"
)

const MODE Mode = ModeDBFetch
const N = 100
const PARALLEL = true
const QPS float64 = 6

const URL = "http://localhost:8080/v2/history/depths/ETH.ETH?interval=day"

type Mode int

const (
	ModeHTTPFetch Mode = iota
	ModeDBFetch
)

type SingleSummary struct {
	id int
	ok bool
}

func Measure() func() (milli int) {
	start := time.Now()
	return func() int {
		return int(time.Since(start).Milliseconds())
	}
}

func measureHTTP(result *SingleSummary) {
	timer := Measure()
	resp, err := http.Get(URL)
	if err != nil {
		result.ok = false
		logrus.Debugf("#%d (http) Failed to get: %v", result.id, err)
		return
	}
	defer resp.Body.Close()
	buf := new(strings.Builder)
	n, err := io.Copy(buf, resp.Body)
	if err != nil || n < 10 {
		logrus.Debugf("#%d (http) Failed to read: %v", result.id, err)
		return
	}
	logrus.Debugf("#%d (http) OK - %d ms", result.id, timer())
}

const ColumnNumber = 4
const Query = `
SELECT
        pool,
        last(asset_e8, block_timestamp) as asset_e8,
        last(rune_e8, block_timestamp) as rune_e8,
        EXTRACT(EPOCH FROM (date_trunc('day', to_timestamp(block_timestamp/1000000000/300*300))))::BIGINT AS truncated
FROM block_pool_depths
WHERE pool = ANY($1) AND $2 <= block_timestamp AND block_timestamp < $3
GROUP BY truncated, pool
ORDER BY truncated ASC;`

var QArgs = []interface{}{
	[]string{"ETH.ETH", "BNB.BUSD-BAF BNB.USDT-DC8", "ETH.USDT-0X62E273709DA575835C7F6AEF4A31140CA5B1D190"},
	db.Nano(1618012800000000000),
	db.Nano(1619654400000000000)}

func measureDB(result *SingleSummary) {
	ctx := context.Background()

	timer := Measure()
	lastHeightRows, err := db.Query(ctx, Query, QArgs...)
	if err != nil {
		logrus.Debugf("#%d (db) Failed query: %v", result.id, err)
		return
	}
	defer lastHeightRows.Close()

	data := make([]string, ColumnNumber)
	results := []interface{}{}
	for i := range data {
		results = append(results, &data[i])
	}
	if lastHeightRows.Next() {
		err := lastHeightRows.Scan(results...)
		if err != nil {
			logrus.Debugf("#%d (db) Failed read: %v", result.id, err)
			return
		}
	}

	logrus.Debugf("#%d (db) OK - %d ms", result.id, timer())
}

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: "2006-01-02 15:04:05", FullTimestamp: true})
	logrus.SetLevel(logrus.DebugLevel)

	var measureFunc func(*SingleSummary)
	switch MODE {
	case ModeHTTPFetch:
		measureFunc = measureHTTP
	case ModeDBFetch:
		measureFunc = measureDB
		var c config.Config = config.ReadConfig()
		db.Setup(&c.TimeScale)
	}

	var summaries []SingleSummary
	for i := 0; i < N; i++ {
		summaries = append(summaries, SingleSummary{id: i})
	}
	if PARALLEL {
		done := make(chan struct{}, N)
		waitTime := time.Duration(math.Round(float64(time.Second) / QPS))
		for i := range summaries {
			go func() {
				measureFunc(&summaries[i])
				done <- struct{}{}
			}()
			time.Sleep(waitTime)
		}
		for range summaries {
			<-done
		}
	} else {
		for i := range summaries {
			measureFunc(&summaries[i])
		}
	}
}
