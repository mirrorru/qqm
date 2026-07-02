package meta

import "reflect"

// FieldMeta содержит метаданные одного поля структуры ROW.
// EN: FieldMeta holds metadata for one ROW struct field.
type FieldMeta struct {
	Name          string       // Имя поля в Go-структуре. / EN: Field name in Go struct.
	Column        string       // Имя колонки в БД (из тега qqm:"col=..."). / EN: Column name in DB (from qqm:"col=..." tag).
	Index         []int        // Путь к полю через reflection для вложенных структур. / EN: Path to field via reflection for nested structs.
	GoType        reflect.Type // Go-тип поля. / EN: Go type of the field.
	IsPK          bool         // Поле является первичным ключом (определяется по тегу pk). / EN: Field is a primary key (determined by pk tag).
	PkOrder       int          // Порядок поля в составном первичном ключе (1-based, по порядку объявления). / EN: Field order in composite primary key (1-based, by declaration order).
	IsAuto        bool         // Автогенерируемое поле (не участвует в INSERT). / EN: Auto-generated field (excluded from INSERT).
	IsUpdate      bool         // Разрешено обновление auto-поля через UPDATE. / EN: Auto field allowed in UPDATE.
	RefTable      string       // Имя таблицы для внешнего ключа (из тега ref=table.column). / EN: Table name for foreign key (from ref=table.column tag).
	RefColumn     string       // Имя колонки для внешнего ключа. / EN: Column name for foreign key.
	IsOmit        bool         // Поле пропускается при генерации SQL. / EN: Field is skipped during SQL generation.
	SortPosition  int          // Позиция поля в сортировке (0 если не задана). / EN: Field position in ordering (0 if not set).
	SortDirection string       // Направление сортировки: "ASC" или "DESC". / EN: Sort direction: "ASC" or "DESC".
	CreateClause  string       // Строка для колонки в CREATE TABLE (из тега create=...). / EN: Column definition string in CREATE TABLE (from create=... tag).
	IsInsert      bool         // Участвует в INSERT, исключается из UPDATE (из тега insert).
	// EN: Participates in INSERT, excluded from UPDATE (from insert tag).
}
