package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v4"
	pgxstd "github.com/jackc/pgx/v4/stdlib"
	"github.com/rs/zerolog/log"
)

// Abstraction for block recorder inserting rows into tables.
//
// Note: Does not support concurrent use.
type RowInserter interface {
	StartBlock() error
	EndBlock() error
	Flush() error
	Insert(table string, columns []string, values ...interface{}) error
	FlushesOnEndBlock() bool
}

////////////////////////////////////////////////////////////////////////////////////////////////////

// Creates a separate transaction for every block and inserts rows as is, in separate SQL query.
// This is the fallback Inserter and also used for testing.
//
// When in a transaction, a SAVEPOINT is created before any operation, and if the operation fails
// the transaction is rolled back to the state before it.
// This is necessary at the moment, as we can't guarantee that we won't run an invalid operation
// while processing a block.
// TODO(huginn): remove this functionality when all inter-operation issues are fixed.
type TxInserter struct {
	db  *sql.Conn
	txn *sql.Tx
}

func (txi *TxInserter) StartBlock() (err error) {
	if txi.txn != nil {
		log.Panic().Msg("Txn still open")
	}
	txn, err := txi.db.BeginTx(context.Background(), nil)
	if err != nil {
		log.Error().Err(err).Msg("BEGIN failed")
		return
	}
	txi.txn = txn
	_, err = txi.txn.Exec("SAVEPOINT sp")
	if err != nil {
		log.Error().Err(err).Msg("SAVEPOINT failed")
	}
	return
}

func (txi *TxInserter) EndBlock() (err error) {
	if txi.txn == nil {
		log.Panic().Msg("No txn open")
	}
	err = txi.txn.Commit()
	if err != nil {
		log.Error().Err(err).Msg("COMMIT failed")
	}
	txi.txn = nil
	return
}

func (txi *TxInserter) Flush() error {
	if txi.txn != nil {
		log.Panic().Msg("Flush while txn open")
	}
	return nil
}

func (txi *TxInserter) Insert(table string, columns []string, values ...interface{}) (err error) {
	if txi.txn == nil {
		log.Panic().Msg("Insert outside open txn")
	}
	var q strings.Builder
	fmt.Fprintf(&q, "INSERT INTO %s (%s) VALUES (", table, strings.Join(columns, ", "))
	for i := range columns {
		if i > 0 {
			fmt.Fprint(&q, ", ")
		}
		fmt.Fprintf(&q, "$%d", i+1)
	}
	fmt.Fprint(&q, ")")

	_, err = txi.txn.Exec(q.String(), values...)
	if err != nil {
		_, err2 := txi.txn.Exec("ROLLBACK TO SAVEPOINT sp")
		if err2 != nil {
			log.Error().Err(err2).Msg("ROLLBACK TO SAVEPOINT failed")
		}
		return
	}
	_, err = txi.txn.Exec("RELEASE SAVEPOINT sp; SAVEPOINT sp")
	if err != nil {
		log.Error().Err(err).Msg("Resetting SAVEPOINT failed")
	}
	return
}

func (txi *TxInserter) FlushesOnEndBlock() bool {
	return true
}

////////////////////////////////////////////////////////////////////////////////////////////////////

type batchRows struct {
	table   string
	columns []string
	rows    [][]interface{}
}

type BatchInserter struct {
	db      *sql.Conn
	batches map[string]batchRows
}

func (bi *BatchInserter) StartBlock() error {
	if bi.batches == nil {
		bi.batches = make(map[string]batchRows)
	}
	return nil
}

func (bi *BatchInserter) EndBlock() error {
	return nil
}

func (bi *BatchInserter) Insert(table string, columns []string, values ...interface{}) error {
	key := table + "(" + strings.Join(columns, ",") + ")"
	brows, ok := bi.batches[key]
	if !ok {
		brows = batchRows{table: table, columns: columns}
	}
	brows.rows = append(brows.rows, values)
	bi.batches[key] = brows
	return nil
}

func (bi *BatchInserter) flushRaw(rawConn interface{}) (err error) {
	batches := bi.batches
	bi.batches = nil

	innerConn, ok := rawConn.(*pgxstd.Conn)
	if !ok {
		log.Fatal().Msg("Not a pgx connection")
	}
	conn := innerConn.Conn()

	txn, err := conn.Begin(context.Background())
	if err != nil {
		log.Fatal().Err(err).Msg("BEGIN failed")
		return
	}

	for _, batch := range batches {
		_, err = txn.CopyFrom(context.Background(),
			pgx.Identifier{batch.table}, batch.columns, pgx.CopyFromRows(batch.rows))
		if err != nil {
			err2 := txn.Rollback(context.Background())
			if err2 != nil {
				log.Error().Err(err).Msg("ROLLBACK failed")
			}
			return
		}
	}

	err = txn.Commit(context.Background())

	return
}

func (bi *BatchInserter) Flush() error {
	if bi.batches == nil {
		return nil
	}

	return bi.db.Raw(bi.flushRaw)
}

func (bi *BatchInserter) FlushesOnEndBlock() bool {
	return false
}
