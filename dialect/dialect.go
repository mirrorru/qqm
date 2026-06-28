// Created at 2026-06-28
package dialect

// DialectProvider — интерфейс для диалектов БД.
type DialectProvider interface {
	// Name — название диалекта
	Name() string
	// QuoteIdent — экранирование идентификатора (таблицы, колонки)
	QuoteIdent(ident string) string
	// Placeholder — плейсхолдер для параметра по позиции (1-based)
	Placeholder(pos int) string
	// SupportsReturning — поддерживает ли диалект RETURNING
	SupportsReturning() bool
}

func quoteIdentANSI(ident string) string {
	return ident
}
