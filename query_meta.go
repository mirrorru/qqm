package qqm

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/meta"
)

// queryTableEntry содержит описание одной таблицы в multi-table запросе.
// EN: queryTableEntry holds the description of one table in a multi-table query.
type queryTableEntry struct {
	FieldName  string        // Имя поля в QROW (например, "Order"). / EN: Field name in QROW (e.g. "Order").
	FieldIndex int           // Индекс поля в QROW-структуре. / EN: Field index in QROW struct.
	RowMeta    *meta.RowMeta // Метаданные ROW-типа. / EN: ROW type metadata.
	RowType    reflect.Type  // Базовый struct-тип (без указателя). / EN: Base struct type (without pointer).
	IsPointer  bool          // true, если поле — указатель (*ROW). / EN: true if field is a pointer (*ROW).
	JoinType   string        // Тип JOIN: INNER, LEFT, RIGHT, FULL.
	TableName  string        // Итоговое имя таблицы (с учётом override). / EN: Final table name (with override).
	OnClause   string        // Условие JOIN (авто или явное). / EN: JOIN condition (auto or explicit).
	Alias      string        // Алиас таблицы (t1, t2, ...). / EN: Table alias (t1, t2, ...).
}

// qualifiedColumn содержит колонку с алиасом таблицы для SELECT.
// EN: qualifiedColumn holds a column with a table alias for SELECT.
type qualifiedColumn struct {
	TableAlias string
	Column     string
}

// queryMeta содержит кэшируемые метаданные Query.
// EN: queryMeta holds cacheable Query metadata.
type queryMeta struct {
	entries     []queryTableEntry // Первая запись — primary. / EN: First entry is primary.
	listSQL     string            // Полный SELECT ... FROM ... JOIN ... .
	columns     []qualifiedColumn // Все колонки в порядке SELECT.
	entryByName map[string]*queryTableEntry
}

// resolveQueryFieldTableName определяет имя таблицы для поля QROW.
// Приоритет: тег `table=` > SQLNamer > snake_case(TypeName).
// EN: resolveQueryFieldTableName determines the table name for a QROW field.
// Priority: `table=` tag > SQLNamer > snake_case(TypeName).
func resolveQueryFieldTableName(sf reflect.StructField) string {
	tagRaw := sf.Tag.Get(meta.TagName)
	if tagRaw != "" {
		opts := meta.ParseTag(tagRaw)
		if opts.TableName != "" {
			return opts.TableName
		}
	}

	ft := sf.Type
	for ft.Kind() == reflect.Pointer {
		ft = ft.Elem()
	}

	zero := reflect.New(ft).Interface()
	if namer, ok := zero.(SQLNamer); ok {
		return namer.SQLName()
	}

	return meta.ToSnakeCase(ft.Name())
}

// buildQueryMeta строит метаданные для QROW.
// EN: buildQueryMeta builds metadata for QROW.
func buildQueryMeta[QROW any]() (*queryMeta, error) {
	var zero QROW
	t := reflect.TypeOf(zero)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("qqm: QROW must be a struct, got %s", t.Kind())
	}

	var entries []queryTableEntry
	primaryFound := false

	for i := range t.NumField() {
		sf := t.Field(i)
		if !sf.IsExported() {
			continue
		}

		// Только struct или *struct.
		// EN: Only struct or *struct.
		ft := sf.Type
		isPtr := false
		if ft.Kind() == reflect.Pointer {
			isPtr = true
			ft = ft.Elem()
		}
		if ft.Kind() != reflect.Struct {
			continue
		}

		tagRaw := sf.Tag.Get(meta.TagName)
		opts := meta.ParseTag(tagRaw)

		tableName := resolveQueryFieldTableName(sf)
		rowMeta := meta.GetOrBuildRowMeta(ft, tableName)

		joinType := opts.JoinType
		if joinType == "" {
			if isPtr {
				joinType = "LEFT"
			} else {
				joinType = "INNER"
			}
		}

		isPrimary := opts.IsPrimary
		if !isPrimary && !primaryFound && !isPtr {
			isPrimary = true
		}

		entry := queryTableEntry{
			FieldName:  sf.Name,
			FieldIndex: i,
			RowMeta:    rowMeta,
			RowType:    ft,
			IsPointer:  isPtr,
			JoinType:   joinType,
			TableName:  tableName,
			OnClause:   opts.On,
		}
		entries = append(entries, entry)

		if isPrimary {
			primaryFound = true
		}
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("qqm: QROW must have at least one struct field")
	}

	// Первая запись — primary, если не отмечено явно.
	// EN: First entry is primary if not explicitly marked.
	if !primaryFound && len(entries) > 0 {
		for i := range entries {
			if !entries[i].IsPointer {
				entries[i].JoinType = "INNER"
				break
			}
		}
	}

	// Строим JOIN-условия для non-primary таблиц.
	// EN: Build JOIN conditions for non-primary tables.
	for i := 1; i < len(entries); i++ {
		if entries[i].OnClause != "" {
			continue
		}

		onClause, err := buildJoinOnClause(&entries[i], entries[:i], i+1)
		if err != nil {
			return nil, err
		}
		entries[i].OnClause = onClause
	}

	qm := &queryMeta{
		entries: entries,
	}

	// Назначаем алиасы таблиц.
	// EN: Assign table aliases.
	for i := range qm.entries {
		qm.entries[i].Alias = fmt.Sprintf("t%d", i+1)
	}

	qm.columns = buildQualifiedColumns(qm)

	m := make(map[string]*queryTableEntry, len(entries))
	for i := range entries {
		m[entries[i].FieldName] = &entries[i]
	}
	qm.entryByName = m

	return qm, nil
}

// buildJoinOnClause автоматически строит ON-условие для entry.
// currentAlias — алиас текущей таблицы (t1, t2, ...).
// prevEntries — уже добавленные таблицы (их алиасы t1, t2, ..., tN).
// EN: buildJoinOnClause automatically builds the ON condition for entry.
// currentAlias — alias of the current table (t1, t2, ...).
// prevEntries — already added tables (their aliases t1, t2, ..., tN).
func buildJoinOnClause(entry *queryTableEntry, prevEntries []queryTableEntry, currentAlias int) (string, error) {
	var conditions []string

	// Прямое направление: поля entry ссылаются на prevEntries.
	// EN: Forward direction: entry fields reference prevEntries.
	for _, fm := range entry.RowMeta.Fields {
		if fm.RefTable == "" || fm.IsOmit {
			continue
		}
		for pi, pe := range prevEntries {
			if pe.TableName == fm.RefTable {
				refCol := fm.RefColumn
				if refCol == "" {
					refCol = "id"
				}
				leftAlias := fmt.Sprintf("t%d", currentAlias)
				rightAlias := fmt.Sprintf("t%d", pi+1)
				conditions = append(conditions,
					fmt.Sprintf("%s.%s = %s.%s",
						leftAlias, fm.Column,
						rightAlias, refCol))
			}
		}
	}

	// Обратное направление: поля prevEntries ссылаются на entry.
	// EN: Reverse direction: prevEntries fields reference entry.
	for pi, pe := range prevEntries {
		for _, fm := range pe.RowMeta.Fields {
			if fm.RefTable == "" || fm.IsOmit {
				continue
			}
			if fm.RefTable == entry.TableName {
				refCol := fm.RefColumn
				if refCol == "" {
					refCol = "id"
				}
				leftAlias := fmt.Sprintf("t%d", currentAlias)
				rightAlias := fmt.Sprintf("t%d", pi+1)
				conditions = append(conditions,
					fmt.Sprintf("%s.%s = %s.%s",
						leftAlias, refCol,
						rightAlias, fm.Column))
			}
		}
	}

	if len(conditions) == 0 {
		return "", fmt.Errorf("qqm: no FK relationship found for table %q (field %q)", entry.TableName, entry.FieldName)
	}

	return strings.Join(conditions, " AND "), nil
}

// buildQueryListSQL строит полный SQL для SELECT с JOIN.
// EN: buildQueryListSQL builds the full SQL for SELECT with JOIN.
func buildQueryListSQL(d dialect.DialectProvider, qm *queryMeta) string {
	if len(qm.entries) == 0 {
		return ""
	}

	var selectCols []string
	for i, entry := range qm.entries {
		alias := fmt.Sprintf("t%d", i+1)
		for _, col := range entry.RowMeta.Columns {
			selectCols = append(selectCols, d.QuoteIdent(alias)+"."+d.QuoteIdent(col))
		}
	}

	primaryAlias := "t1"
	sql := sqlSelect + strings.Join(selectCols, sqlCommaSpace) +
		sqlFrom + d.QuoteIdent(qm.entries[0].TableName) + " AS " + primaryAlias

	for i := 1; i < len(qm.entries); i++ {
		entry := qm.entries[i]
		alias := fmt.Sprintf("t%d", i+1)
		joinType := entry.JoinType
		if joinType == "" {
			joinType = "INNER"
		}
		sql += fmt.Sprintf(" %s JOIN %s AS %s ON %s",
			joinType, d.QuoteIdent(entry.TableName), alias, entry.OnClause)
	}

	if len(qm.entries[0].RowMeta.SortFields) > 0 {
		sql += buildOrderByClause(d, qm.entries[0].RowMeta, "t1")
	}

	return sql
}

// buildQualifiedColumns строит маппинг колонок SELECT к полям QROW.
// EN: buildQualifiedColumns builds the mapping of SELECT columns to QROW fields.
func buildQualifiedColumns(qm *queryMeta) []qualifiedColumn {
	var cols []qualifiedColumn
	for i, entry := range qm.entries {
		alias := fmt.Sprintf("t%d", i+1)
		for _, fm := range entry.RowMeta.Fields {
			if fm.IsOmit {
				continue
			}
			cols = append(cols, qualifiedColumn{
				TableAlias: alias,
				Column:     fm.Column,
			})
		}
	}
	return cols
}

// findEntryByFieldName находит entry по имени поля в QROW.
// EN: findEntryByFieldName finds an entry by field name in QROW.
func (qm *queryMeta) findEntryByFieldName(name string) *queryTableEntry {
	return qm.entryByName[name]
}
