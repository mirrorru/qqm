// Created at 2026-06-28
package dialect

import "strconv"

// PostgreSQLDialect — диалект PostgreSQL.
type PostgreSQLDialect struct{}

// проверка интерфейса на этапе компиляции
var _ DialectProvider = PostgreSQLDialect{}

// Created at 2026-06-28
func (PostgreSQLDialect) Name() string { return "postgres" }

// Created at 2026-06-28
func (PostgreSQLDialect) QuoteIdent(ident string) string { return quoteIdentANSI(ident) }

// Created at 2026-06-28
func (PostgreSQLDialect) Placeholder(pos int) string { return "$" + strconv.Itoa(pos) }

// Created at 2026-06-28
func (PostgreSQLDialect) SupportsReturning() bool { return true }
