// Created at 2026-06-28
package meta

import "reflect"

// FieldMeta — метаданные одного поля структуры ROW.
type FieldMeta struct {
	// Name — имя поля в Go-структуре
	Name string
	// Column — имя колонки в БД (из тега qqm:"col=...")
	Column string
	// Index — путь к полю через reflection (для вложенных структур)
	Index []int
	// GoType — Go-тип поля
	GoType reflect.Type
	// IsPK — поле является первичным ключом (определяется по тегу pk)
	IsPK bool
	// PkOrder — порядок поля в составном первичном ключе (1-based, по порядку объявления)
	PkOrder int
	// IsReadonly — поле только для чтения (не участвует в UPDATE)
	IsReadonly bool
	// IsAuto — автогенерируемое поле (не участвует в INSERT)
	IsAuto bool
	// RefTable — имя таблицы для внешнего ключа (из тега ref=table.column)
	RefTable string
	// RefColumn — имя колонки для внешнего ключа
	RefColumn string
	// IsOmit — поле пропускается при генерации SQL
	IsOmit bool
}
