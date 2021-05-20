package main

// Deletes all blocks including and after certain height.

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/sirupsen/logrus"
	"gitlab.com/thorchain/midgard/config"
	"gitlab.com/thorchain/midgard/internal/api"
	"gitlab.com/thorchain/midgard/internal/db"
)

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: "2006-01-02 15:04:05", FullTimestamp: true})
	logrus.SetLevel(logrus.InfoLevel)

	// TODO(huginn): enforce this
	logrus.Warn("If Midgard is running, stop it and rerun this tool!")

	if len(os.Args) != 3 {
		logrus.Fatalf("Provide 2 arguments, %d provided\nUsage: $ trimdb config heightOrTimestamp",
			len(os.Args)-1)
	}

	var c config.Config = config.ReadConfigFrom(os.Args[1])
	ctx := context.Background()

	db.Setup(&c.TimeScale)

	idStr := os.Args[2]
	heightOrTimestamp, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		logrus.Fatal("Couldn't parse height or timestamp: ", idStr)
	}
	height, timestamp, err := api.TimestampAndHeight(ctx, heightOrTimestamp)
	if err != nil {
		logrus.Fatal("Couldn't find height for ", heightOrTimestamp)
	}

	logrus.Info("Deleting aggregates")
	err = db.DropAggregates()
	if err != nil {
		logrus.Fatal(err)
	}

	logrus.Infof("Deleting rows including and after height %d , timestamp %d", height, timestamp)
	tables := GetTableColumns(ctx)
	for table, columns := range tables {
		if columns["block_timestamp"] {
			logrus.Infof("%s  deleting by block_timestamp", table)
			DeleteAfter(table, "block_timestamp", timestamp.ToI())
		} else if columns["height"] {
			logrus.Infof("%s deleting by height", table)
			DeleteAfter(table, "height", height)
		} else if table == "constants" {
			logrus.Infof("Skipping table %s", table)
		} else {
			logrus.Warnf("talbe %s has no good column", table)
		}
	}
}

func DeleteAfter(table string, columnName string, value int64) {
	q := fmt.Sprintf("DELETE FROM %s WHERE $1 <= %s", table, columnName)
	_, err := db.Exec(q, value)
	if err != nil {
		logrus.Fatal("delete failed: ", err)
	}
}

type TableMap map[string]map[string]bool

func GetTableColumns(ctx context.Context) TableMap {
	q := `
	SELECT
		table_name,
		column_name
	FROM information_schema.columns
	WHERE table_schema='midgard'
	`
	rows, err := db.Query(ctx, q)
	if err != nil {
		logrus.Fatal(err)
	}
	defer rows.Close()

	ret := TableMap{}
	for rows.Next() {
		var table, column string
		err := rows.Scan(&table, &column)
		if err != nil {
			logrus.Fatal(err)
		}
		if _, ok := ret[table]; !ok {
			ret[table] = map[string]bool{}
		}
		ret[table][column] = true
	}
	return ret
}
