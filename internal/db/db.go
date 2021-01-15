package db

import (
	"bytes"
	"context"
	"crypto/md5"
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

const ddlHashKeyName = "ddl_hash"

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

	MustCheckAndUpdateDDLVersion(dbObj)
}

func MustCheckAndUpdateDDLVersion(dbObj *sql.DB) {
	ddl := Ddl()

	fileDdlHash := md5.Sum(ddl)
	currentDdlHash := mustGetDdlHash(dbObj)

	if !bytes.Equal(fileDdlHash[:], currentDdlHash) {
		log.Printf("ddl hash mismatch\n\tstored value is %x\n\tddl.sql is %x\n", currentDdlHash, fileDdlHash)
		log.Println("Applying new ddl from ddl.go...")
		_, err := dbObj.Exec(string(ddl))
		if err != nil {
			log.Fatal("exit on PostgresSQL ddl setup: ", err)
		}
		_, err = dbObj.Exec("INSERT INTO bin_constants (key, value) VALUES ($1, $2)", ddlHashKeyName, fileDdlHash[:])
		if err != nil {
			log.Fatal("exit on PostgresSQL ddl setup: ", err)
		}
		log.Println("Successfully applied new db schema (Will start syncing from genesis block)")
	}
}

// Returns current file md5 hash stored in table or an empty hash if either bin_metadata table
// does not exist or ddl_hash key is not found. Will panic on other error
// (Don't want to reconstruct the whole database if some other random error ocurs)
func mustGetDdlHash(dbObj *sql.DB) []byte {
	ret := make([]byte, 16)
	tableExists := true
	err := dbObj.QueryRow(`SELECT EXISTS (
		SELECT * FROM pg_tables WHERE tablename = 'bin_constants' AND schemaname = current_schema()
	)`).Scan(&tableExists)
	if err != nil {
		log.Fatal("exit on PostgresSQL ddl setup: ", err)
	}
	if !tableExists {
		return ret
	}

	err = dbObj.QueryRow(`SELECT value FROM bin_constants WHERE key = $1`, ddlHashKeyName).Scan(&ret)
	if err != nil && err != sql.ErrNoRows {
		log.Fatal("exit on PostgresSQL ddl setup: ", err)
	}
	return ret
}
