package qqm

import "context"

// Executor описывает интерфейс выполнения SQL-запросов.
// Абстрагирует database/sql.DB, database/sql.Tx, pgx.Conn, pgx.Tx.
// EN: Executor describes the SQL execution interface.
// Abstracts database/sql.DB, database/sql.Tx, pgx.Conn, pgx.Tx.
type Executor interface {
	ExecContext(ctx context.Context, query string, args ...any) (Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) Row
}

// Row представляет одну строку результата запроса.
// EN: Row represents a single query result row.
type Row interface {
	Scan(dest ...any) error
}

// Result представляет результат выполнения ExecContext.
// EN: Result represents the result of ExecContext execution.
type Result interface {
	LastInsertId() (int64, error)
	RowsAffected() (int64, error)
}

// Rows представляет курсор результатов запроса.
// EN: Rows represents a query result cursor.
type Rows interface {
	Next() bool
	Scan(dest ...any) error
	Close() error
}
