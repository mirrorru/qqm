// Created at 2026-06-28
package qqm

import (
	"context"
	"database/sql"
)

// DBAdapter адаптирует *sql.DB к интерфейсу Executor.
type DBAdapter struct {
	db *sql.DB
}

// NewDBAdapterVal создает адаптер для *sql.DB к интерфейсу Executor
func NewDBAdapterVal(db *sql.DB) DBAdapter {
	return DBAdapter{db: db}
}

// ExecContext выполняет запрос, не возвращающий строк.
func (a DBAdapter) ExecContext(ctx context.Context, query string, args ...any) (Result, error) {
	return a.db.ExecContext(ctx, query, args...)
}

// QueryContext выполняет запрос, возвращающий строки.
func (a DBAdapter) QueryContext(ctx context.Context, query string, args ...any) (Rows, error) {
	return a.db.QueryContext(ctx, query, args...)
}

// QueryRowContext выполняет запрос, возвращающий одну строку.
func (a DBAdapter) QueryRowContext(ctx context.Context, query string, args ...any) Row {
	return a.db.QueryRowContext(ctx, query, args...)
}

// TxAdapter адаптирует *sql.Tx к интерфейсу Executor.
type TxAdapter struct {
	tx *sql.Tx
}

// NewTxAdapterVal создает адаптер для *sql.Tx к интерфейсу Executor
func NewTxAdapterVal(tx *sql.Tx) TxAdapter {
	return TxAdapter{tx: tx}
}

// ExecContext выполняет запрос, не возвращающий строк.
func (a TxAdapter) ExecContext(ctx context.Context, query string, args ...any) (Result, error) {
	return a.tx.ExecContext(ctx, query, args...)
}

// QueryContext выполняет запрос, возвращающий строки.
func (a TxAdapter) QueryContext(ctx context.Context, query string, args ...any) (Rows, error) {
	return a.tx.QueryContext(ctx, query, args...)
}

// QueryRowContext выполняет запрос, возвращающий одну строку.
func (a TxAdapter) QueryRowContext(ctx context.Context, query string, args ...any) Row {
	return a.tx.QueryRowContext(ctx, query, args...)
}
