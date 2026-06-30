package qqm

import (
	"context"
	"database/sql"
)

// DBAdapter адаптирует *sql.DB к интерфейсу Executor.
// EN: DBAdapter adapts *sql.DB to the Executor interface.
type DBAdapter struct {
	db *sql.DB
}

// NewDBAdapterVal создаёт адаптер для *sql.DB к интерфейсу Executor.
// EN: NewDBAdapterVal creates an adapter from *sql.DB to the Executor interface.
func NewDBAdapterVal(db *sql.DB) DBAdapter {
	return DBAdapter{db: db}
}

// ExecContext выполняет запрос, не возвращающий строк.
// EN: ExecContext executes a query that does not return rows.
func (a DBAdapter) ExecContext(ctx context.Context, query string, args ...any) (Result, error) {
	return a.db.ExecContext(ctx, query, args...)
}

// QueryContext выполняет запрос, возвращающий строки.
// EN: QueryContext executes a query that returns rows.
func (a DBAdapter) QueryContext(ctx context.Context, query string, args ...any) (Rows, error) {
	return a.db.QueryContext(ctx, query, args...)
}

// QueryRowContext выполняет запрос, возвращающий одну строку.
// EN: QueryRowContext executes a query that returns a single row.
func (a DBAdapter) QueryRowContext(ctx context.Context, query string, args ...any) Row {
	return a.db.QueryRowContext(ctx, query, args...)
}

// TxAdapter адаптирует *sql.Tx к интерфейсу Executor.
// EN: TxAdapter adapts *sql.Tx to the Executor interface.
type TxAdapter struct {
	tx *sql.Tx
}

// NewTxAdapterVal создаёт адаптер для *sql.Tx к интерфейсу Executor.
// EN: NewTxAdapterVal creates an adapter from *sql.Tx to the Executor interface.
func NewTxAdapterVal(tx *sql.Tx) TxAdapter {
	return TxAdapter{tx: tx}
}

// ExecContext выполняет запрос, не возвращающий строк.
// EN: ExecContext executes a query that does not return rows.
func (a TxAdapter) ExecContext(ctx context.Context, query string, args ...any) (Result, error) {
	return a.tx.ExecContext(ctx, query, args...)
}

// QueryContext выполняет запрос, возвращающий строки.
// EN: QueryContext executes a query that returns rows.
func (a TxAdapter) QueryContext(ctx context.Context, query string, args ...any) (Rows, error) {
	return a.tx.QueryContext(ctx, query, args...)
}

// QueryRowContext выполняет запрос, возвращающий одну строку.
// EN: QueryRowContext executes a query that returns a single row.
func (a TxAdapter) QueryRowContext(ctx context.Context, query string, args ...any) Row {
	return a.tx.QueryRowContext(ctx, query, args...)
}
