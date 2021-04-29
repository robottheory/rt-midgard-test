package main

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"gitlab.com/thorchain/midgard/config"
	"gitlab.com/thorchain/midgard/internal/db"
)

const Mode ModeEnum = ModeHTTPFetch
const N = 100
const Threads = 7

const URL = "http://localhost:8080/v2/history/depths/ETH.ETH?interval=day"

type ModeEnum int

const (
	ModeHTTPFetch ModeEnum = iota
	ModeDBFetch
)

type SingleSummary struct {
	id    int
	ok    bool
	milli int
}

func Measure() func() (milli int) {
	start := time.Now()
	return func() int {
		return int(time.Since(start).Milliseconds())
	}
}

func measureHTTP(result *SingleSummary) {
	result.ok = false
	timer := Measure()
	resp, err := http.Get(URL)
	if err != nil {
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
	result.milli = timer()
	result.ok = true
	logrus.Debugf("#%d (http) OK - %d ms", result.id, result.milli)
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
	result.ok = false

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
	result.milli = timer()
	result.ok = true
	logrus.Debugf("#%d (db) OK - %d ms", result.id, result.milli)
}

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: "2006-01-02 15:04:05", FullTimestamp: true})
	logrus.SetLevel(logrus.DebugLevel)

	var measureFunc func(*SingleSummary)
	switch Mode {
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

	done := make(chan struct{}, Threads)
	for i := 0; i < Threads; i++ {
		done <- struct{}{}
	}
	for i := range summaries {
		ilocal := i
		<-done
		go func() {
			measureFunc(&summaries[ilocal])
			done <- struct{}{}
		}()
	}
	for i := 0; i < Threads; i++ {
		<-done
	}

	var sum float64
	failNum := 0
	for _, v := range summaries {
		if v.ok {
			sum += float64(v.milli)
		} else {
			failNum++
		}
	}
	success := 100 * float64(failNum) / float64(N)
	avg := sum / float64(N)
	qps := Threads * 1000 / avg
	logrus.Infof(
		"Failures: %.2f%%, Average response time: %.2f milli, Average qps: %.2f",
		success, avg, qps)
}
