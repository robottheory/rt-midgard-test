package db

import (
	"context"
	"crypto/md5"
	"database/sql"
	"fmt"
	"strings"
	"sync"

	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/rs/zerolog/log"
)

// Query is the SQL client.
var Query func(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)

// Exec is the SQL client.
var Exec func(query string, args ...interface{}) (sql.Result, error)

var Begin func() error
var Commit func() error

// Wrapper for `sql.DB` that can operate in transactional or non-transactional mode.
//
// When in a transaction a SAVEPOINT is created before any operation, and if the operation failed
// the transaction is rolled back to the state before it.
// This is necessary at the moment, as we can't guarantee that we won't run an invalid operation
// while processing a block.
// TODO(huginn): remove this functionality when all inter-operation issues are fixed.
//
// Both `sql.DB` and `sql.Tx` are thread-safe, but handling whether we are in a transaction or not
// (ie, whether txn is nil or not) isn't. That's what `mu` protects.
type TxDB struct {
	sync.Mutex
	db  *sql.DB
	txn *sql.Tx
}

func (txdb *TxDB) Begin() (err error) {
	txdb.Lock()
	defer txdb.Unlock()
	if txdb.txn != nil {
		log.Fatal().Msg("Txn still open")
	}
	txn, err := txdb.db.Begin()
	if err != nil {
		log.Error().Err(err).Msg("BEGIN failed")
		return
	}
	txdb.txn = txn
	_, err = txdb.txn.Exec("SAVEPOINT sp")
	if err != nil {
		log.Error().Err(err).Msg("SAVEPOINT failed")
	}
	return
}

func (txdb *TxDB) Commit() (err error) {
	txdb.Lock()
	defer txdb.Unlock()
	if txdb.txn == nil {
		log.Fatal().Msg("No txn open")
	}
	err = txdb.txn.Commit()
	if err != nil {
		log.Error().Err(err).Msg("COMMIT failed")
	}
	txdb.txn = nil
	return
}

func (txdb *TxDB) Exec(query string, args ...interface{}) (res sql.Result, err error) {
	txdb.Lock()
	defer txdb.Unlock()
	if txdb.txn == nil {
		return txdb.db.Exec(query, args...)
	}
	res, err = txdb.txn.Exec(query, args...)
	if err != nil {
		_, err2 := txdb.txn.Exec("ROLLBACK TO SAVEPOINT sp")
		if err2 != nil {
			log.Error().Err(err2).Msg("ROLLBACK TO SAVEPOINT failed")
		}
		return
	}
	_, err = txdb.txn.Exec("RELEASE SAVEPOINT sp; SAVEPOINT sp")
	if err != nil {
		log.Error().Err(err).Msg("Resetting SAVEPOINT failed")
	}
	return
}

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

	txdb := TxDB{
		db:  dbObj,
		txn: nil,
	}

	Exec = txdb.Exec
	Query = dbObj.QueryContext
	Begin = txdb.Begin
	Commit = txdb.Commit

	UpdateDDLsIfNeeded(dbObj)
}

func UpdateDDLsIfNeeded(dbObj *sql.DB) {
	UpdateDDLIfNeeded(dbObj, "data", Ddl(), ddlHashKey)
	UpdateDDLIfNeeded(dbObj, "aggregates", AggregatesDdl(), aggregatesDdlHashKey)
}

func UpdateDDLIfNeeded(dbObj *sql.DB, tag string, ddl string, hashKey string) {
	fileDdlHash := md5.Sum([]byte(ddl))
	currentDdlHash := liveDDLHash(dbObj, hashKey)

	if fileDdlHash != currentDdlHash {
		log.Info().Msgf("DDL hash mismatch for %s\n\tstored value is %x\n\tddl.sql is %x\n",
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
