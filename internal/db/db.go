package db

import (
	"context"
	"crypto/md5"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/pascaldekloe/metrics"
	"github.com/rs/zerolog/log"
	"gitlab.com/thorchain/midgard/config"
)

// The Query part of the SQL client.
var Query func(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)

// Global RowInserter object used by block recorder
var Inserter RowInserter

var TheImmediateInserter *ImmediateInserter
var TheBatchInserter *BatchInserter

// The SQL client object used for ad-hoc DB manipulation like aggregate refreshing (and by tests).
var TheDB *sql.DB

const (
	ddlHashKey           = "ddl_hash"
	aggregatesDdlHashKey = "aggregates_ddl_hash"
)

// By default we use the BatchInserter to process blocks and insert data into the DB.
// If the BatchInserter fails to flush a batch of rows, that means that we are trying to insert
// some data that doesn't match our database schema. In such a case we make a "mark"
// in the 'constants' table and exit. On restart we detect this and switch to using TxInserter,
// which can handle such a block gracefully.
// This situation should be investigated and fixed. When it's fixed the version below can be
// incremented, and updated Midgard will switch back to using BatchInserter.
// (Note: versions compared as strings, lexicographically. That's why the zeroes.)
//
// TODO(huginn): figure out how to test this well
const (
	inserterFailKey     = "batch_inserter_failed"
	inserterFailVersion = "0001"
)

var inserterFailVar = metrics.MustCounter("batch_inserter_marked_failed",
	"1 if using TxInserter because BatchInserter was marked as failed")

type md5Hash [md5.Size]byte

func Setup() {
	timeScale := config.Global.TimeScale

	dbObj, err := sql.Open("pgx",
		fmt.Sprintf("user=%s dbname=%s sslmode=%s password=%s host=%s port=%d",
			timeScale.UserName, timeScale.Database, timeScale.Sslmode,
			timeScale.Password, timeScale.Host, timeScale.Port))
	if err != nil {
		log.Fatal().Err(err).Msg("Exit on PostgreSQL client instantiation")
	}

	dbObj.SetMaxOpenConns(timeScale.MaxOpenConns)

	dbConn, err := dbObj.Conn(context.Background())
	if err != nil {
		log.Fatal().Err(err).Msg("Opening a connection to PostgreSQL failed")
	}

	Query = dbObj.QueryContext

	TheDB = dbObj

	UpdateDDLsIfNeeded(dbObj, timeScale)

	TheImmediateInserter = &ImmediateInserter{db: dbConn}
	TheBatchInserter = &BatchInserter{db: dbConn}
	Inserter = TheBatchInserter
	if CheckBatchInserterMarked() {
		log.Error().Msg("BatchInserter maked as failed, sync will be slow!")
		inserterFailVar.Add(1)
		Inserter = TheImmediateInserter
	}
}

func UpdateDDLsIfNeeded(dbObj *sql.DB, cfg config.TimeScale) {
	UpdateDDLIfNeeded(dbObj, "data", Ddl(), ddlHashKey,
		cfg.NoAutoUpdateDDL || cfg.NoAutoUpdateAggregatesDDL)
	// If 'data' DDL is updated the 'aggregates' DDL is automatically updated too, as
	// the `constants` table is recreated with the 'data' DDL.
	UpdateDDLIfNeeded(dbObj, "aggregates", AggregatesDdl(), aggregatesDdlHashKey,
		cfg.NoAutoUpdateAggregatesDDL)
}

func UpdateDDLIfNeeded(dbObj *sql.DB, tag string, ddl string, hashKey string, noauto bool) {
	fileDdlHash := md5.Sum([]byte(ddl))
	currentDdlHash := liveDDLHash(dbObj, hashKey)

	if fileDdlHash != currentDdlHash {
		log.Info().Msgf("DDL hash mismatch for %s\n\tstored value is %x\n\thash of the code is %x",
			tag, currentDdlHash, fileDdlHash)
		if noauto {
			log.Fatal().Msg("DDL update prohibited in config")
		}
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

func MarkBatchInserterFail() {
	_, err := TheDB.Exec(`INSERT INTO constants (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`,
		inserterFailKey, inserterFailVersion[:])
	if err != nil {
		log.Error().Err(err).Msg("Marking batch inserter failed, probably bad DB connection")
	}
}

func CheckBatchInserterMarked() bool {
	value := []byte{}
	err := TheDB.QueryRow("SELECT value FROM constants WHERE key = $1", inserterFailKey).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			// This is the expected state
			return false
		}
		log.Fatal().Err(err).Msgf("Querying 'constants' table for '%s' failed", inserterFailKey)
	}
	return inserterFailVersion <= string(value)
}
