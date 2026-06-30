package qqm

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// PGXAdapter адаптирует *pgx.Conn к интерфейсу Executor.
// EN: PGXAdapter adapts *pgx.Conn to the Executor interface.
type PGXAdapter struct {
	conn *pgx.Conn
}

// NewPGXAdapterVal создаёт адаптер для *pgx.Conn к интерфейсу Executor.
// EN: NewPGXAdapterVal creates an adapter from *pgx.Conn to the Executor interface.
func NewPGXAdapterVal(conn *pgx.Conn) PGXAdapter {
	return PGXAdapter{conn: conn}
}

// ExecContext выполняет запрос, не возвращающий строк.
// EN: ExecContext executes a query that does not return rows.
func (a PGXAdapter) ExecContext(ctx context.Context, query string, args ...any) (Result, error) {
	tag, err := a.conn.Exec(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &PgxResult{tag: tag}, nil
}

// QueryContext выполняет запрос, возвращающий строки.
// EN: QueryContext executes a query that returns rows.
func (a PGXAdapter) QueryContext(ctx context.Context, query string, args ...any) (Rows, error) {
	rows, err := a.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &PgxRows{rows: rows}, nil
}

// QueryRowContext выполняет запрос, возвращающий одну строку.
// EN: QueryRowContext executes a query that returns a single row.
func (a PGXAdapter) QueryRowContext(ctx context.Context, query string, args ...any) Row {
	return a.conn.QueryRow(ctx, query, args...)
}

// PGXTxAdapter адаптирует pgx.Tx к интерфейсу Executor.
// EN: PGXTxAdapter adapts pgx.Tx to the Executor interface.
type PGXTxAdapter struct {
	tx pgx.Tx
}

// NewPGXTxAdapterVal создаёт адаптер для pgx.Tx к интерфейсу Executor.
// EN: NewPGXTxAdapterVal creates an adapter from pgx.Tx to the Executor interface.
func NewPGXTxAdapterVal(tx pgx.Tx) PGXTxAdapter {
	return PGXTxAdapter{tx: tx}
}

// ExecContext выполняет запрос, не возвращающий строк.
// EN: ExecContext executes a query that does not return rows.
func (a PGXTxAdapter) ExecContext(ctx context.Context, query string, args ...any) (Result, error) {
	tag, err := a.tx.Exec(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &PgxResult{tag: tag}, nil
}

// QueryContext выполняет запрос, возвращающий строки.
// EN: QueryContext executes a query that returns rows.
func (a PGXTxAdapter) QueryContext(ctx context.Context, query string, args ...any) (Rows, error) {
	rows, err := a.tx.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &PgxRows{rows: rows}, nil
}

// QueryRowContext выполняет запрос, возвращающий одну строку.
// EN: QueryRowContext executes a query that returns a single row.
func (a PGXTxAdapter) QueryRowContext(ctx context.Context, query string, args ...any) Row {
	return a.tx.QueryRow(ctx, query, args...)
}

// PgxResult адаптирует pgconn.CommandTag к интерфейсу Result.
// EN: PgxResult adapts pgconn.CommandTag to the Result interface.
type PgxResult struct {
	tag pgconn.CommandTag
}

// LastInsertId возвращает ID последней вставленной строки (не поддерживается в PostgreSQL).
// EN: LastInsertId returns the ID of the last inserted row (not supported in PostgreSQL).
func (r *PgxResult) LastInsertId() (int64, error) {
	return 0, nil
}

// RowsAffected возвращает количество затронутых строк.
// EN: RowsAffected returns the number of affected rows.
func (r *PgxResult) RowsAffected() (int64, error) {
	return r.tag.RowsAffected(), nil
}

// PgxRows адаптирует pgx.Rows к интерфейсу Rows.
// EN: PgxRows adapts pgx.Rows to the Rows interface.
type PgxRows struct {
	rows pgx.Rows
}

// Next переходит к следующей строке результата.
// EN: Next advances to the next row in the result set.
func (r *PgxRows) Next() bool {
	return r.rows.Next()
}

// Scan сканирует значения текущей строки в dest.
// EN: Scan scans the current row values into dest.
func (r *PgxRows) Scan(dest ...any) error {
	return r.rows.Scan(dest...)
}

// Close закрывает курсор результатов.
// EN: Close closes the result cursor.
func (r *PgxRows) Close() error {
	r.rows.Close()
	return nil
}
