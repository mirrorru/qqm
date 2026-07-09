package dialect

import (
	"fmt"
	"strconv"
)

// PostgreSQLDialect реализует диалект PostgreSQL.
// EN: PostgreSQLDialect implements the PostgreSQL dialect.
type PostgreSQLDialect struct{}

// Проверка интерфейса на этапе компиляции.
// EN: Interface check at compile time.
var _ DialectProvider = PostgreSQLDialect{}

// Name возвращает название диалекта.
// EN: Name returns the dialect name.
func (PostgreSQLDialect) Name() string { return "postgres" }

// QuoteIdent экранирует идентификатор.
// EN: QuoteIdent quotes an identifier.
func (PostgreSQLDialect) QuoteIdent(ident string) string { return quoteIdentANSI(ident) }

// Placeholder возвращает плейсхолдер вида $N для параметра.
// EN: Placeholder returns a $N-style placeholder for a parameter.
func (PostgreSQLDialect) Placeholder(pos int) string { return "$" + strconv.Itoa(pos) }

// SupportsReturning сообщает, поддерживает ли диалект RETURNING.
// EN: SupportsReturning reports whether the dialect supports RETURNING.
func (PostgreSQLDialect) SupportsReturning() bool { return true }

func (PostgreSQLDialect) OffsetAndLimit(offset, limit uint32) string {
	if limit == 0 && offset == 0 {
		return ""
	}
	return fmt.Sprintf(" OFFSET %d LIMIT %d", offset, limit)
}
