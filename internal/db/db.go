package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/jackc/pgx/v4/stdlib"
)

// Query is the SQL client.
var Query func(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)

// Exec is the SQL client.
var Exec func(query string, args ...interface{}) (sql.Result, error)

type Config struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	UserName string `json:"user_name"`
	Password string `json:"password"`
	Database string `json:"database"`
	Sslmode  string `json:"sslmode"`
}

func Setup(config *Config) {
	dbObj, err := sql.Open("pgx",
		fmt.Sprintf("user=%s dbname=%s sslmode=%s password=%s host=%s port=%d",
			config.UserName, config.Database, config.Sslmode,
			config.Password, config.Host, config.Port))
	if err != nil {
		log.Fatal("exit on PostgreSQL client instantiation: ", err)
	}

	Exec = dbObj.Exec
	Query = dbObj.QueryContext
}

// Select field that truncates value to a date using interval priovided in intervalValue
// and gets the Unix timestamp in seconds
func SelectTruncatedTimestamp(targetColumn, intervalValue string) string {
	return fmt.Sprintf(
		"EXTRACT(EPOCH FROM (date_trunc(%s, to_timestamp(%s/1000000000/300*300))))::BIGINT",
		intervalValue, targetColumn)
}
