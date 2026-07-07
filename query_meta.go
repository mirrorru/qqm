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
	RowType    reflect.Type  // Базовый struct-тип. / EN: Base struct type.
	JoinType   string        // Тип JOIN: INNER, LEFT, RIGHT, FULL.
	TableName  string        // Итоговое имя таблицы (с учётом override). / EN: Final table name (with override).
	OnClause   string        // Условие JOIN (авто или явное). / EN: JOIN condition (auto or explicit).
	Alias      string        // Алиас таблицы (из тега или t1, t2, ...). / EN: Table alias (from tag or t1, t2, ...).
}

// qualifiedColumn содержит колонку с алиасом таблицы для SELECT.
// EN: qualifiedColumn holds a column with a table alias for SELECT.
type qualifiedColumn struct {
	TableAlias string // Алиас таблицы (t1, t2, ...). / EN: Table alias (t1, t2, ...).
	Column     string // Имя колонки. / EN: Column name.
}

// queryMeta содержит кэшируемые метаданные Query.
// EN: queryMeta holds cacheable Query metadata.
type queryMeta struct {
	entries     []queryTableEntry           // Первая запись — primary. / EN: First entry is primary.
	listSQL     string                      // Полный SELECT ... FROM ... JOIN ... .
	columns     []qualifiedColumn           // Все колонки в порядке SELECT. / EN: All columns in SELECT order.
	entryByName map[string]*queryTableEntry // Индекс по имени поля. / EN: Index by field name.
}

func (qm *queryMeta) ListSQL() string {
	return qm.listSQL
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

		ft := sf.Type
		for ft.Kind() == reflect.Pointer {
			return nil, fmt.Errorf("qqm: QROW field %q must be a struct, not a pointer", sf.Name)
		}
		if ft.Kind() != reflect.Struct {
			return nil, fmt.Errorf("qqm: QROW field %q must be a struct", sf.Name)
		}

		tagRaw := sf.Tag.Get(meta.TagName)
		opts := meta.ParseTag(tagRaw)

		tableName := resolveQueryFieldTableName(sf)
		rowMeta := meta.GetOrBuildRowMeta(ft, tableName)

		joinType := opts.JoinType
		if joinType == "" {
			joinType = "INNER"
		}

		isPrimary := opts.IsPrimary
		if !isPrimary && !primaryFound {
			isPrimary = true
		}

		entry := queryTableEntry{
			FieldName:  sf.Name,
			FieldIndex: i,
			RowMeta:    rowMeta,
			RowType:    ft,
			JoinType:   joinType,
			TableName:  tableName,
			Alias:      opts.Alias,
			OnClause:   opts.OnCondition,
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
		entries[0].JoinType = "INNER"
	}

	// Назначаем алиасы таблиц.
	// EN: Assign table aliases.
	for i := range entries {
		if entries[i].Alias == "" {
			entries[i].Alias = fmt.Sprintf("t%d", i+1)
		}
	}

	// Строим JOIN-условия для non-primary таблиц.
	// EN: Build JOIN conditions for non-primary tables.
	for i := 1; i < len(entries); i++ {
		if entries[i].OnClause != "" {
			continue
		}

		onClause, err := buildJoinOnClause(&entries[i], entries[:i])
		if err != nil {
			return nil, err
		}
		entries[i].OnClause = onClause
	}

	qm := &queryMeta{
		entries: entries,
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
// prevEntries — уже добавленные таблицы.
// EN: buildJoinOnClause automatically builds the ON condition for entry.
// prevEntries — already added tables.
func buildJoinOnClause(entry *queryTableEntry, prevEntries []queryTableEntry) (string, error) {
	if entry.OnClause != "" {
		return entry.OnClause, nil
	}

	var conditions []string

	// Прямое направление: поля entry ссылаются на prevEntries.
	// EN: Forward direction: entry fields reference prevEntries.
	for _, fm := range entry.RowMeta.Fields {
		if fm.RefTable == "" || fm.IsOmit {
			continue
		}
		for _, pe := range prevEntries {
			if pe.TableName == fm.RefTable {
				refCol := fm.RefColumn
				if refCol == "" {
					refCol = "id"
				}
				leftAlias := entry.Alias
				rightAlias := pe.Alias
				conditions = append(conditions,
					fmt.Sprintf("%s.%s = %s.%s",
						leftAlias, fm.Column,
						rightAlias, refCol))
			}
		}
	}

	// Обратное направление: поля prevEntries ссылаются на entry.
	// EN: Reverse direction: prevEntries fields reference entry.
	for _, pe := range prevEntries {
		for _, fm := range pe.RowMeta.Fields {
			if fm.RefTable == "" || fm.IsOmit {
				continue
			}
			if fm.RefTable == entry.TableName {
				refCol := fm.RefColumn
				if refCol == "" {
					refCol = "id"
				}
				leftAlias := entry.Alias
				rightAlias := pe.Alias
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
	for _, entry := range qm.entries {
		for _, col := range entry.RowMeta.Columns {
			selectCols = append(selectCols, d.QuoteIdent(entry.Alias)+"."+d.QuoteIdent(col))
		}
	}

	primaryAlias := qm.entries[0].Alias
	sql := sqlSelect + strings.Join(selectCols, sqlCommaSpace) +
		sqlFrom + d.QuoteIdent(qm.entries[0].TableName) + " AS " + primaryAlias

	for i := 1; i < len(qm.entries); i++ {
		entry := qm.entries[i]
		joinType := entry.JoinType
		if joinType == "" {
			joinType = "INNER"
		}
		sql += fmt.Sprintf(" %s JOIN %s AS %s ON %s",
			joinType, d.QuoteIdent(entry.TableName), entry.Alias, entry.OnClause)
	}

	if len(qm.entries[0].RowMeta.SortFields) > 0 {
		sql += buildOrderByClause(d, qm.entries[0].RowMeta, qm.entries[0].Alias)
	}

	return sql
}

// buildQualifiedColumns строит маппинг колонок SELECT к полям QROW.
// EN: buildQualifiedColumns builds the mapping of SELECT columns to QROW fields.
func buildQualifiedColumns(qm *queryMeta) []qualifiedColumn {
	var cols []qualifiedColumn
	for _, entry := range qm.entries {
		for _, fm := range entry.RowMeta.Fields {
			if fm.IsOmit {
				continue
			}
			cols = append(cols, qualifiedColumn{
				TableAlias: entry.Alias,
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
