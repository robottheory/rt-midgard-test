package db

import (
	"context"
	"database/sql"
	"fmt"
)

// Query is the SQL client.
var Query func(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)

// Exec is the SQL client.
var Exec func(query string, args ...interface{}) (sql.Result, error)

// Select field that truncates value to a date using interval priovided in intervalValue
// and gets the Unix timestamp in seconds
func QuerySelectTimestampInSecondsForInterval(targetColumn, intervalValue string) string {
	return fmt.Sprintf(
		"EXTRACT(EPOCH FROM (date_trunc(%s, to_timestamp(%s/1000000000/300*300))))::BIGINT", intervalValue, targetColumn)
}
