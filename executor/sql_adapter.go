// Created at 2026-06-28
package executor

import (
	"context"
	"database/sql"
)

// DBAdapter адаптирует *sql.DB к интерфейсу Executor.
type DBAdapter struct {
	DB *sql.DB
}

// Created at 2026-06-28
// ExecContext выполняет запрос, не возвращающий строк.
func (a *DBAdapter) ExecContext(ctx context.Context, query string, args ...any) (Result, error) {
	return a.DB.ExecContext(ctx, query, args...)
}

// Created at 2026-06-28
// QueryContext выполняет запрос, возвращающий строки.
func (a *DBAdapter) QueryContext(ctx context.Context, query string, args ...any) (Rows, error) {
	return a.DB.QueryContext(ctx, query, args...)
}

// TxAdapter адаптирует *sql.Tx к интерфейсу Executor.
type TxAdapter struct {
	Tx *sql.Tx
}

// Created at 2026-06-28
// ExecContext выполняет запрос, не возвращающий строк.
func (a *TxAdapter) ExecContext(ctx context.Context, query string, args ...any) (Result, error) {
	return a.Tx.ExecContext(ctx, query, args...)
}

// Created at 2026-06-28
// QueryContext выполняет запрос, возвращающий строки.
func (a *TxAdapter) QueryContext(ctx context.Context, query string, args ...any) (Rows, error) {
	return a.Tx.QueryContext(ctx, query, args...)
}
