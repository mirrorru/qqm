// Created at 2026-06-28
package dialect

// SQLiteDialect — диалект SQLite.
type SQLiteDialect struct{}

// проверка интерфейса на этапе компиляции
var _ DialectProvider = SQLiteDialect{}

// Created at 2026-06-28
func (SQLiteDialect) Name() string { return "sqlite" }

// Created at 2026-06-28
func (SQLiteDialect) QuoteIdent(ident string) string { return quoteIdentANSI(ident) }

// Created at 2026-06-28
func (SQLiteDialect) Placeholder(_ int) string { return "?" }

// Created at 2026-06-28
func (SQLiteDialect) SupportsReturning() bool { return true }
