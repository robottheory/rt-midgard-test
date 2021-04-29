package db

import (
	"context"
	"crypto/md5"
	"database/sql"
	"fmt"
	"log"
	"strings"

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

	// -1 sets it to infinite
	MaxOpenConns int `json:"max_open_conns"`
}

const ddlHashKeyName = "ddl_hash"

type md5Hash [md5.Size]byte

func Setup(config *Config) {
	dbObj, err := sql.Open("pgx",
		fmt.Sprintf("user=%s dbname=%s sslmode=%s password=%s host=%s port=%d",
			config.UserName, config.Database, config.Sslmode,
			config.Password, config.Host, config.Port))
	if err != nil {
		log.Fatal("exit on PostgreSQL client instantiation: ", err)
	}

	dbObj.SetMaxOpenConns(config.MaxOpenConns)

	Exec = dbObj.Exec
	Query = dbObj.QueryContext

	UpdateDDLIfNeeded(dbObj)
}

func UpdateDDLIfNeeded(dbObj *sql.DB) {
	ddl := Ddl()

	fileDdlHash := md5.Sum([]byte(ddl))
	currentDdlHash := liveDDLHash(dbObj)

	if fileDdlHash != currentDdlHash {
		log.Printf("ddl hash mismatch\n\tstored value is %x\n\tddl.sql is %x\n", currentDdlHash, fileDdlHash)
		log.Println("Applying new ddl from ddl.go...")
		_, err := dbObj.Exec(ddl)
		if err != nil {
			log.Fatal("exit on PostgresSQL ddl setup: ", err)
		}
		_, err = dbObj.Exec("INSERT INTO constants (key, value) VALUES ($1, $2)", ddlHashKeyName, fileDdlHash[:])
		if err != nil {
			log.Fatal("exit on PostgresSQL ddl setup: ", err)
		}
		log.Println("Successfully applied new db schema (Will start syncing from genesis block)")
	}
}

// Returns current file md5 hash stored in table or an empty hash if either constants table
// does not exist or ddl_hash key is not found. Will panic on other error
// (Don't want to reconstruct the whole database if some other random error ocurs)
func liveDDLHash(dbObj *sql.DB) (ret md5Hash) {
	tableExists := true
	err := dbObj.QueryRow(`SELECT EXISTS (
		SELECT * FROM pg_tables WHERE tablename = 'constants' AND schemaname = current_schema()
	)`).Scan(&tableExists)
	if err != nil {
		log.Fatal("exit on PostgresSQL ddl setup: ", err)
	}
	if !tableExists {
		return
	}

	value := []byte{}
	err = dbObj.QueryRow(`SELECT value FROM constants WHERE key = $1`, ddlHashKeyName).Scan(&value)
	if err != nil && err != sql.ErrNoRows {
		log.Fatal("exit on PostgresSQL ddl setup: ", err)
	}
	if len(ret) != len(value) {
		log.Printf(
			"Warning: %s in constants table had with wrong format, will recreate database anyway",
			ddlHashKeyName)
		return
	}
	copy(ret[:], value)
	return
}

// Helper function to join posibbly empty filters for a WHERE clause.
// Empty strings are discarded.
func Where(filters ...string) string {
	actualFilters := []string{}
	for _, filter := range filters {
		if filter != "" {
			actualFilters = append(actualFilters, filter)
		}
	}
	if 0 == len(actualFilters) {
		return ""
	}
	return "WHERE (" + strings.Join(actualFilters, ") AND (") + ")"
}
