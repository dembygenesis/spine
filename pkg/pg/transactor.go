package pg

import (
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// Executor runs SQL. Both *sqlx.DB and *sqlx.Tx satisfy it, which is what lets
// one store method run inside a transaction or outside one, unchanged.
type Executor interface {
	sqlx.ExecerContext
	sqlx.QueryerContext
	sqlx.ExtContext
}

// Transactor owns the connection and hands out transactions. Only the business
// layer holds one: stores receive an Executor and never decide where a
// transaction starts or ends.
type Transactor struct {
	db *sqlx.DB
}

// NewTransactor wraps an open connection.
func NewTransactor(db *sqlx.DB) (*Transactor, error) {
	if db == nil {
		return nil, errors.New("pg: db is required")
	}
	return &Transactor{db: db}, nil
}

// DB is the connection as an Executor, for work that needs no transaction.
func (t *Transactor) DB() Executor { return t.db }

// Begin starts a transaction.
func (t *Transactor) Begin() (*sqlx.Tx, error) {
	tx, err := t.db.Beginx()
	if err != nil {
		return nil, fmt.Errorf("begin: %w", err)
	}
	return tx, nil
}

// Rollback undoes tx unless it already committed, so it is safe to defer
// straight after Begin.
func (t *Transactor) Rollback(tx *sqlx.Tx) {
	_ = tx.Rollback() // after a commit this is sql.ErrTxDone, the expected case
}
