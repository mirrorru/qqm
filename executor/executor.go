// Created at 2026-06-28
package executor

import "context"

// Executor — интерфейс для выполнения SQL-запросов.
// Абстрагирует database/sql.DB и database/sql.Tx.
type Executor interface {
	ExecContext(ctx context.Context, query string, args ...any) (Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (Rows, error)
}

// Result — результат выполнения ExecContext.
type Result interface {
	LastInsertId() (int64, error)
	RowsAffected() (int64, error)
}

// Rows — курсор результатов запроса.
type Rows interface {
	Next() bool
	Scan(dest ...any) error
	Close() error
}
