package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

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
