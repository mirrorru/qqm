package dialect

// DialectProvider описывает интерфейс диалекта БД.
// EN: DialectProvider describes the DB dialect interface.
type DialectProvider interface {
	Name() string                                // Название диалекта. / EN: Dialect name.
	QuoteIdent(ident string) string              // Экранирование идентификатора. / EN: Identifier quoting.
	Placeholder(pos int) string                  // Плейсхолдер для параметра (1-based). / EN: Placeholder for parameter (1-based).
	SupportsReturning() bool                     // Поддерживает ли RETURNING. / EN: Whether RETURNING is supported.
	OffsetAndLimit(offset, limit uint32) string  // Строка с Offset и Limit
	ILIKE(col string, placeholder string) string // Регистронезависимый LIKE. / EN: Case-insensitive LIKE.
}

// quoteIdentANSI экранирует идентификатор без кавычек (для ANSI/SQLite).
// EN: quoteIdentANSI quotes an identifier without quotes (for ANSI/SQLite).
func quoteIdentANSI(ident string) string {
	return ident
}
