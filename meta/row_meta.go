package meta

import (
	"reflect"
)

const (
	// errRequiresStruct возвращается, когда тип не является структурой.
	// EN: Error returned when type is not a struct.
	errRequiresStruct = "qqm: BuildRowMeta requires struct type, got "

	// errDuplicateColumn возвращается при дублировании имени колонки.
	// EN: Error returned when a column name is duplicated.
	errDuplicateColumn = "qqm: duplicate column name "

	// errInTable добавляется в сообщения об ошибках для указания таблицы.
	// EN: Added to error messages to indicate the table.
	errInTable = " in table "

	// errScanDestAddressable возвращается, когда ScanDest требует адресуемое значение.
	// EN: Error returned when ScanDest requires an addressable value.
	errScanDestAddressable = "qqm: ScanDest requires an addressable value (pass a pointer to struct)"
)

// RowMeta содержит метаданные структуры ROW для работы с БД.
// EN: RowMeta holds ROW struct metadata for database operations.
type RowMeta struct {
	// TableName — имя таблицы в БД.
	TableName string
	// Fields — метаданные всех полей.
	Fields []*FieldMeta
	// PKFields — поля первичного ключа в порядке объявления.
	PKFields []*FieldMeta
	// Columns — все имена колонок для SELECT.
	Columns []string
	// insertColumns — кэшированные колонки для INSERT.
	insertColumns []string
	// updateColumns — кэшированные колонки для UPDATE.
	updateColumns []string
	// SortFields — поля для ORDER BY, отсортированные по SortPosition.
	SortFields []*FieldMeta
}

// BuildRowMeta строит метаданные для типа t с именем таблицы tableName.
// EN: BuildRowMeta builds metadata for type t with table name tableName.
func BuildRowMeta(t reflect.Type, tableName string) *RowMeta {
	// Разыменовываем указатель.
	// EN: Dereference pointer.
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

	rm.sortSortFields()

	rm.validateUniqueColumns()

	rm.insertColumns = buildInsertColumns(rm)
	rm.updateColumns = buildUpdateColumns(rm)

	return rm
}

// validateUniqueColumns проверяет уникальность имён колонок и вызывает panic при дублировании.
// EN: validateUniqueColumns checks column name uniqueness and panics on duplicates.
func (rm *RowMeta) validateUniqueColumns() {
	seen := make(map[string]bool, len(rm.Columns))
	for _, col := range rm.Columns {
		if seen[col] {
			panic(errDuplicateColumn + col + errInTable + rm.TableName)
		}
		seen[col] = true
	}
}

// walkFields рекурсивно обходит поля структуры, включая embedded.
// prefix — префикс для колонок из anonymous struct (из тега prefix=).
// pkCounter — счётчик для назначения порядка PK-полей по порядку объявления.
// EN: walkFields recursively traverses struct fields, including embedded.
// prefix — prefix for columns from anonymous struct (from prefix= tag).
// pkCounter — counter for assigning PK field order by declaration order.
func (rm *RowMeta) walkFields(t reflect.Type, parentIndex []int, prefix string, pkCounter *int) {
	for i := range t.NumField() {
		sf := t.Field(i)

		// Формируем индекс поля.
		// EN: Build field index.
		idx := make([]int, len(parentIndex)+1)
		copy(idx, parentIndex)
		idx[len(parentIndex)] = i

		// Пропускаем неэкспортируемые поля (включая anonymous).
		// EN: Skip unexported fields (including anonymous).
		if !sf.IsExported() {
			continue
		}

		// Anonymous поле: struct — набор полей, non-struct — обычное поле.
		// EN: Anonymous field: struct — field set, non-struct — regular field.
		if sf.Anonymous {
			ft := sf.Type
			for ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}

			if ft.Kind() == reflect.Struct {
				// Читаем тег anonymous struct для получения префикса.
				// EN: Read anonymous struct tag to get prefix.
				tagRaw := sf.Tag.Get(TagName)
				opts := ParseTag(tagRaw)
				childPrefix := prefix + opts.Prefix
				rm.walkFields(ft, idx, childPrefix, pkCounter)
				continue
			}

			// Парсим тег qqm для anonymous non-struct.
			// EN: Parse qqm tag for anonymous non-struct.
			tagRaw := sf.Tag.Get(TagName)
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
				Name:          sf.Name,
				Column:        col,
				Index:         idx,
				GoType:        sf.Type,
				IsPK:          opts.IsPK,
				PkOrder:       pkOrder,
				IsAuto:        opts.Auto,
				IsUpdate:      opts.Update,
				RefTable:      opts.RefTable,
				RefColumn:     opts.RefCol,
				IsOmit:        opts.Omit,
				SortPosition:  opts.Sort,
				SortDirection: opts.SortDir,
				CreateClause:  opts.Create,
			}
			rm.Fields = append(rm.Fields, fm)
			if fm.IsPK {
				rm.PKFields = append(rm.PKFields, fm)
			}
			if !fm.IsOmit {
				rm.Columns = append(rm.Columns, fm.Column)
			}
			if fm.SortPosition > 0 {
				rm.SortFields = append(rm.SortFields, fm)
			}
			continue
		}

		// Парсим тег qqm.
		// EN: Parse qqm tag.
		tagRaw := sf.Tag.Get(TagName)
		opts := ParseTag(tagRaw)

		// Неанонимное поле-структура с префиксом — разворачиваем её поля.
		// EN: Non-anonymous struct field with prefix — expand its fields.
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

		// Если нет col=, используем ToSnakeCase от имени поля.
		// EN: If no col=, use ToSnakeCase from field name.
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
			Name:          sf.Name,
			Column:        col,
			Index:         idx,
			GoType:        sf.Type,
			IsPK:          opts.IsPK,
			PkOrder:       pkOrder,
			IsAuto:        opts.Auto,
			IsUpdate:      opts.Update,
			RefTable:      opts.RefTable,
			RefColumn:     opts.RefCol,
			IsOmit:        opts.Omit,
			SortPosition:  opts.Sort,
			SortDirection: opts.SortDir,
			CreateClause:  opts.Create,
		}

		rm.Fields = append(rm.Fields, fm)
		if fm.IsPK {
			rm.PKFields = append(rm.PKFields, fm)
		}
		if !fm.IsOmit {
			rm.Columns = append(rm.Columns, fm.Column)
		}
		if fm.SortPosition > 0 {
			rm.SortFields = append(rm.SortFields, fm)
		}
	}
}

// sortSortFields сортирует SortFields по SortPosition.
// EN: sortSortFields sorts SortFields by SortPosition.
func (rm *RowMeta) sortSortFields() {
	if len(rm.SortFields) < 2 {
		return
	}
	// Insertion sort — SortFields обычно небольшой.
	// EN: Insertion sort — SortFields is usually small.
	for i := 1; i < len(rm.SortFields); i++ {
		j := i
		for j > 0 && rm.SortFields[j-1].SortPosition > rm.SortFields[j].SortPosition {
			rm.SortFields[j-1], rm.SortFields[j] = rm.SortFields[j], rm.SortFields[j-1]
			j--
		}
	}
}

// ScanDest формирует слайс указателей на поля row для sql.Rows.Scan().
// row должен быть указателем на struct.
// EN: ScanDest builds a slice of pointers to row fields for sql.Rows.Scan().
// row must be a pointer to struct.
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

// InsertColumns возвращает колонки для INSERT (без auto, без omit).
// PK-поля включаются, если не помечены как auto.
// EN: InsertColumns returns columns for INSERT (without auto, without omit).
// PK fields are included if not marked as auto.
func (rm *RowMeta) InsertColumns() []string {
	return rm.insertColumns
}

// UpdateColumns возвращает колонки для UPDATE (без pk, без omit, без auto без update).
// EN: UpdateColumns returns columns for UPDATE (without pk, without omit, without auto without update).
func (rm *RowMeta) UpdateColumns() []string {
	return rm.updateColumns
}

// buildInsertColumns строит слайс колонок для INSERT (без auto, без omit).
// EN: buildInsertColumns builds column slice for INSERT (without auto, without omit).
func buildInsertColumns(rm *RowMeta) []string {
	cols := make([]string, 0, len(rm.Fields))
	for _, fm := range rm.Fields {
		if fm.IsAuto || fm.IsOmit {
			continue
		}
		cols = append(cols, fm.Column)
	}
	return cols
}

// buildUpdateColumns строит слайс колонок для UPDATE (без pk, без omit, без auto без update).
// EN: buildUpdateColumns builds column slice for UPDATE (without pk, without omit, without auto without update).
func buildUpdateColumns(rm *RowMeta) []string {
	cols := make([]string, 0, len(rm.Fields))
	for _, fm := range rm.Fields {
		if fm.IsPK || fm.IsOmit || (fm.IsAuto && !fm.IsUpdate) {
			continue
		}
		cols = append(cols, fm.Column)
	}
	return cols
}

// InsertValues извлекает значения полей для INSERT из row.
// EN: InsertValues extracts field values for INSERT from row.
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

// UpdateValues извлекает значения полей для UPDATE из row.
// EN: UpdateValues extracts field values for UPDATE from row.
func (rm *RowMeta) UpdateValues(row any) []any {
	v := reflect.ValueOf(row)
	for v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	vals := make([]any, 0, len(rm.Fields))
	for _, fm := range rm.Fields {
		if fm.IsPK || fm.IsOmit || (fm.IsAuto && !fm.IsUpdate) {
			continue
		}
		fv := v.FieldByIndex(fm.Index)
		vals = append(vals, fv.Interface())
	}
	return vals
}

// PKFieldValues извлекает значения PK-полей из row.
// EN: PKFieldValues extracts PK field values from row.
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
