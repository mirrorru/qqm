// Created at 2026-06-28
package executor

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// PGXAdapter адаптирует *pgx.Conn к интерфейсу Executor.
type PGXAdapter struct {
	conn *pgx.Conn
}

// NewPGXTxAdapter создает адаптер для *pgx.Conn к интерфейсу Executor
func NewPGXAdapter(conn *pgx.Conn) *PGXAdapter {
	return &PGXAdapter{conn: conn}
}

func (a *PGXAdapter) ExecContext(ctx context.Context, query string, args ...any) (Result, error) {
	tag, err := a.conn.Exec(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &PgxResult{tag: tag}, nil
}

func (a *PGXAdapter) QueryContext(ctx context.Context, query string, args ...any) (Rows, error) {
	rows, err := a.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &PgxRows{rows: rows}, nil
}

func (a *PGXAdapter) QueryRowContext(ctx context.Context, query string, args ...any) Row {
	return a.conn.QueryRow(ctx, query, args...)
}

// PGXTxAdapter адаптирует pgx.Tx к интерфейсу Executor.
type PGXTxAdapter struct {
	tx pgx.Tx
}

// NewPGXTxAdapter создает адаптер для pgx.Tx к интерфейсу Executor
func NewPGXTxAdapter(tx pgx.Tx) *PGXTxAdapter {
	return &PGXTxAdapter{tx: tx}
}

func (a *PGXTxAdapter) ExecContext(ctx context.Context, query string, args ...any) (Result, error) {
	tag, err := a.tx.Exec(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &PgxResult{tag: tag}, nil
}

func (a *PGXTxAdapter) QueryContext(ctx context.Context, query string, args ...any) (Rows, error) {
	rows, err := a.tx.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &PgxRows{rows: rows}, nil
}

func (a *PGXTxAdapter) QueryRowContext(ctx context.Context, query string, args ...any) Row {
	return a.tx.QueryRow(ctx, query, args...)
}

// PgxResult адаптирует pgconn.CommandTag к интерфейсу Result.
type PgxResult struct {
	tag pgconn.CommandTag
}

func (r *PgxResult) LastInsertId() (int64, error) {
	return 0, nil
}

func (r *PgxResult) RowsAffected() (int64, error) {
	return r.tag.RowsAffected(), nil
}

// PgxRows адаптирует pgx.Rows к интерфейсу Rows.
type PgxRows struct {
	rows pgx.Rows
}

func (r *PgxRows) Next() bool {
	return r.rows.Next()
}

func (r *PgxRows) Scan(dest ...any) error {
	return r.rows.Scan(dest...)
}

func (r *PgxRows) Close() error {
	r.rows.Close()
	return nil
}
