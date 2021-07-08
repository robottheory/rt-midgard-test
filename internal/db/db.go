package db

import (
	"context"
	"crypto/md5"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/rs/zerolog/log"
)

// The Query part of the SQL client.
var Query func(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)

// Global RowInserter object used by block recorder
var Inserter RowInserter

// The SQL client object used for ad-hoc DB manipulation like aggregate refreshing (and by tests).
var TheDB *sql.DB

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

const (
	ddlHashKey           = "ddl_hash"
	aggregatesDdlHashKey = "aggregates_ddl_hash"
)

type md5Hash [md5.Size]byte

func Setup(config *Config) {
	dbObj, err := sql.Open("pgx",
		fmt.Sprintf("user=%s dbname=%s sslmode=%s password=%s host=%s port=%d",
			config.UserName, config.Database, config.Sslmode,
			config.Password, config.Host, config.Port))
	if err != nil {
		log.Fatal().Err(err).Msg("Exit on PostgreSQL client instantiation")
	}

	dbObj.SetMaxOpenConns(config.MaxOpenConns)

	dbConn, err := dbObj.Conn(context.Background())
	if err != nil {
		log.Fatal().Err(err).Msg("Opening a connection to PostgreSQL failed")
	}

	Inserter = &TxInserter{
		db:  dbConn,
		txn: nil,
	}

	Query = dbObj.QueryContext

	TheDB = dbObj

	UpdateDDLsIfNeeded(dbObj)
}

func UpdateDDLsIfNeeded(dbObj *sql.DB) {
	UpdateDDLIfNeeded(dbObj, "data", Ddl(), ddlHashKey)
	// If 'data' DDL is updated the 'aggregates' DDL is automatically updated too, as
	// the `constants` table is recreated with the 'data' DDL.
	UpdateDDLIfNeeded(dbObj, "aggregates", AggregatesDdl(), aggregatesDdlHashKey)
}

func UpdateDDLIfNeeded(dbObj *sql.DB, tag string, ddl string, hashKey string) {
	fileDdlHash := md5.Sum([]byte(ddl))
	currentDdlHash := liveDDLHash(dbObj, hashKey)

	if fileDdlHash != currentDdlHash {
		log.Info().Msgf("DDL hash mismatch for %s\n\tstored value is %x\n\thash of the code is %x",
			tag, currentDdlHash, fileDdlHash)
		log.Info().Msgf("Applying new %s ddl...", tag)
		_, err := dbObj.Exec(ddl)
		if err != nil {
			log.Fatal().Err(err).Msgf("Applying new %s ddl failed, exiting", tag)
		}
		_, err = dbObj.Exec(`INSERT INTO constants (key, value) VALUES ($1, $2)
							 ON CONFLICT (key) DO UPDATE SET value = $2`,
			hashKey, fileDdlHash[:])
		if err != nil {
			log.Fatal().Err(err).Msg("Updating 'constants' table failed, exiting")
		}
		log.Info().Msgf("Successfully applied new %s schema", tag)
	}
}

// Returns current file md5 hash stored in table or an empty hash if either constants table
// does not exist or the requested hash key is not found. Will panic on other errors
// (Don't want to reconstruct the whole database if some other random error ocurs)
func liveDDLHash(dbObj *sql.DB, hashKey string) (ret md5Hash) {
	tableExists := true
	err := dbObj.QueryRow(`SELECT EXISTS (
		SELECT * FROM pg_tables WHERE tablename = 'constants' AND schemaname = 'midgard'
	)`).Scan(&tableExists)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to look up 'constants' table")
	}
	if !tableExists {
		return
	}

	value := []byte{}
	err = dbObj.QueryRow(`SELECT value FROM midgard.constants WHERE key = $1`, hashKey).Scan(&value)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Fatal().Err(err).Msg("Querying 'constants' table failed")
		}
		return
	}
	if len(ret) != len(value) {
		log.Warn().Msgf(
			"Warning: %s in constants table has wrong format, recreating database anyway",
			hashKey)
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
	if len(actualFilters) == 0 {
		return ""
	}
	return "WHERE (" + strings.Join(actualFilters, ") AND (") + ")"
}
