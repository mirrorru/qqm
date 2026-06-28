// Created at 2026-06-28
package meta

import (
	"reflect"
)

const (
	// errRequiresStruct — ошибка: тип не является структурой
	errRequiresStruct = "qqm: BuildRowMeta requires struct type, got "

	// errDuplicateColumn — ошибка: дублирующееся имя колонки
	errDuplicateColumn = "qqm: duplicate column name "

	// errInTable — часть сообщения об ошибке
	errInTable = " in table "

	// errScanDestAddressable — ошибка: ScanDest требует адресуемого значения
	errScanDestAddressable = "qqm: ScanDest requires an addressable value (pass a pointer to struct)"
)

// RowMeta — метаданные структуры ROW для работы с БД.
type RowMeta struct {
	// TableName — имя таблицы в БД
	TableName string
	// Fields — метаданные всех полей
	Fields []*FieldMeta
	// PKFields — поля первичного ключа (в порядке объявления)
	PKFields []*FieldMeta
	// Columns — все имена колонок (для SELECT)
	Columns []string
}

// Updated at 2026-06-28
// BuildRowMeta строит метаданные для типа t с именем таблицы tableName.
func BuildRowMeta(t reflect.Type, tableName string) *RowMeta {
	// разыменовываем указатель
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		panic(errRequiresStruct + t.String())
	}

	rm := &RowMeta{
		TableName: tableName,
		Fields:    make([]*FieldMeta, 0),
		PKFields:  make([]*FieldMeta, 0),
		Columns:   make([]string, 0),
	}

	pkCounter := 1
	rm.walkFields(t, nil, "", &pkCounter)

	rm.validateUniqueColumns()

	return rm
}

// Created at 2026-06-28
// validateUniqueColumns проверяет уникальность имён колонок.
func (rm *RowMeta) validateUniqueColumns() {
	seen := make(map[string]bool, len(rm.Columns))
	for _, col := range rm.Columns {
		if seen[col] {
			panic(errDuplicateColumn + col + errInTable + rm.TableName)
		}
		seen[col] = true
	}
}

// Updated at 2026-06-28
// walkFields рекурсивно обходит поля структуры, включая embedded.
// prefix — префикс для колонок из anonymous struct (из тега prefix=).
// pkCounter — счётчик для назначения порядка PK-полей по порядку объявления.
func (rm *RowMeta) walkFields(t reflect.Type, parentIndex []int, prefix string, pkCounter *int) {
	for i := range t.NumField() {
		sf := t.Field(i)

		// формируем индекс поля
		idx := make([]int, len(parentIndex)+1)
		copy(idx, parentIndex)
		idx[len(parentIndex)] = i

		// пропускаем неэкспортируемые поля (включая anonymous)
		if !sf.IsExported() {
			continue
		}

		// anonymous поле: struct — набор полей, non-struct — обычное поле
		if sf.Anonymous {
			ft := sf.Type
			for ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}

			if ft.Kind() == reflect.Struct {
				// читаем тег anonymous struct для получения префикса
				tagRaw := sf.Tag.Get(tagName)
				opts := ParseTag(tagRaw)
				childPrefix := prefix + opts.Prefix
				rm.walkFields(ft, idx, childPrefix, pkCounter)
				continue
			}

			// парсим тег qqm для anonymous non-struct
			tagRaw := sf.Tag.Get(tagName)
			opts := ParseTag(tagRaw)

			col := prefix + ToSnakeCase(sf.Name)
			if opts.Col != "" {
				col = prefix + opts.Col
			}

			pkOrder := 0
			if opts.IsPK {
				pkOrder = *pkCounter
				*pkCounter++
			}

			fm := &FieldMeta{
				Name:       sf.Name,
				Column:     col,
				Index:      idx,
				GoType:     sf.Type,
				IsPK:       opts.IsPK,
				PkOrder:    pkOrder,
				IsReadonly: opts.Readonly,
				IsAuto:     opts.Auto,
				RefTable:   opts.RefTable,
				RefColumn:  opts.RefCol,
				IsOmit:     opts.Omit,
			}
			rm.Fields = append(rm.Fields, fm)
			if fm.IsPK {
				rm.PKFields = append(rm.PKFields, fm)
			}
			if !fm.IsOmit {
				rm.Columns = append(rm.Columns, fm.Column)
			}
			continue
		}

		// парсим тег qqm
		tagRaw := sf.Tag.Get(tagName)
		opts := ParseTag(tagRaw)

		// неанонимное поле-структура с префиксом — разворачиваем её поля
		if opts.Prefix != "" {
			ft := sf.Type
			for ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct {
				childPrefix := prefix + opts.Prefix
				rm.walkFields(ft, idx, childPrefix, pkCounter)
				continue
			}
		}

		// если нет col=, используем ToSnakeCase от имени поля
		col := opts.Col
		if col == "" {
			col = ToSnakeCase(sf.Name)
		}
		col = prefix + col

		pkOrder := 0
		if opts.IsPK {
			pkOrder = *pkCounter
			*pkCounter++
		}

		fm := &FieldMeta{
			Name:       sf.Name,
			Column:     col,
			Index:      idx,
			GoType:     sf.Type,
			IsPK:       opts.IsPK,
			PkOrder:    pkOrder,
			IsReadonly: opts.Readonly,
			IsAuto:     opts.Auto,
			RefTable:   opts.RefTable,
			RefColumn:  opts.RefCol,
			IsOmit:     opts.Omit,
		}

		rm.Fields = append(rm.Fields, fm)
		if fm.IsPK {
			rm.PKFields = append(rm.PKFields, fm)
		}
		if !fm.IsOmit {
			rm.Columns = append(rm.Columns, fm.Column)
		}
	}
}

// Created at 2026-06-28
// ScanDest формирует слайс указателей на поля row для sql.Rows.Scan().
// row должен быть указателем на struct.
func (rm *RowMeta) ScanDest(row any) []any {
	v := reflect.ValueOf(row)
	for v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	if !v.CanAddr() {
		panic(errScanDestAddressable)
	}

	dest := make([]any, 0, len(rm.Columns))
	for _, fm := range rm.Fields {
		if fm.IsOmit {
			continue
		}

		fv := v.FieldByIndex(fm.Index)
		dest = append(dest, fv.Addr().Interface())
	}
	return dest
}

// Updated at 2026-06-28
// InsertColumns возвращает колонки для INSERT (без auto, без omit).
// PK-поля включаются, если не помечены как auto.
func (rm *RowMeta) InsertColumns() []string {
	cols := make([]string, 0, len(rm.Fields))
	for _, fm := range rm.Fields {
		if fm.IsAuto || fm.IsOmit {
			continue
		}
		cols = append(cols, fm.Column)
	}
	return cols
}

// Updated at 2026-06-28
// UpdateColumns возвращает колонки для UPDATE (без readonly, без pk, без omit, без auto).
func (rm *RowMeta) UpdateColumns() []string {
	cols := make([]string, 0, len(rm.Fields))
	for _, fm := range rm.Fields {
		if fm.IsReadonly || fm.IsPK || fm.IsOmit || fm.IsAuto {
			continue
		}
		cols = append(cols, fm.Column)
	}
	return cols
}

// Created at 2026-06-28
// InsertValues извлекает значения полей для INSERT из row.
func (rm *RowMeta) InsertValues(row any) []any {
	v := reflect.ValueOf(row)
	for v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	vals := make([]any, 0, len(rm.Fields))
	for _, fm := range rm.Fields {
		if fm.IsAuto || fm.IsOmit {
			continue
		}
		fv := v.FieldByIndex(fm.Index)
		vals = append(vals, fv.Interface())
	}
	return vals
}

// Created at 2026-06-28
// UpdateValues извлекает значения полей для UPDATE из row.
func (rm *RowMeta) UpdateValues(row any) []any {
	v := reflect.ValueOf(row)
	for v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	vals := make([]any, 0, len(rm.Fields))
	for _, fm := range rm.Fields {
		if fm.IsReadonly || fm.IsPK || fm.IsOmit || fm.IsAuto {
			continue
		}
		fv := v.FieldByIndex(fm.Index)
		vals = append(vals, fv.Interface())
	}
	return vals
}

// Created at 2026-06-28
// PKFieldValues извлекает значения PK-полей из row.
func (rm *RowMeta) PKFieldValues(row any) []any {
	v := reflect.ValueOf(row)
	for v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	vals := make([]any, 0, len(rm.PKFields))
	for _, fm := range rm.PKFields {
		fv := v.FieldByIndex(fm.Index)
		vals = append(vals, fv.Interface())
	}
	return vals
}
