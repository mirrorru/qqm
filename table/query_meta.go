package table

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/meta"
)

// queryTableEntry — одна таблица в запросе.
type queryTableEntry struct {
	FieldName  string        // имя поля в QROW (например, "Order")
	FieldIndex int           // индекс поля в QROW-структуре
	RowMeta    *meta.RowMeta // метаданные ROW-типа
	RowType    reflect.Type  // базовый struct-тип (без указателя)
	IsPointer  bool          // true если поле — указатель (*ROW)
	JoinType   string        // INNER, LEFT, RIGHT, FULL
	TableName  string        // итоговое имя таблицы (с учётом override)
	OnClause   string        // условие JOIN (auto или explicit)
	Alias      string        // алиас таблицы (t1, t2, ...)
}

// qualifiedColumn — колонка с алиасом таблицы для SELECT.
type qualifiedColumn struct {
	TableAlias string
	Column     string
}

// queryMeta — кэшируемые метаданные Query.
type queryMeta struct {
	entries     []queryTableEntry // первая = primary
	listSQL     string            // полный SELECT ... FROM ... JOIN ...
	columns     []qualifiedColumn // все колонки в порядке SELECT
	entryByName map[string]*queryTableEntry
}

// resolveQueryFieldTableName определяет имя таблицы для поля QROW.
// Приоритет: тег `table=` > SQLNamer > snake_case(TypeName).
func resolveQueryFieldTableName(sf reflect.StructField) string {
	tagRaw := sf.Tag.Get("qqm")
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

		// только struct или *struct
		ft := sf.Type
		isPtr := false
		if ft.Kind() == reflect.Pointer {
			isPtr = true
			ft = ft.Elem()
		}
		if ft.Kind() != reflect.Struct {
			continue
		}

		tagRaw := sf.Tag.Get("qqm")
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

	// первая запись — primary, если не отмечено явно
	if !primaryFound && len(entries) > 0 {
		for i := range entries {
			if !entries[i].IsPointer {
				entries[i].JoinType = "INNER"
				break
			}
		}
	}

	// строим JOIN-условия для non-primary таблиц
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

	// назначаем алиасы таблиц
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

// buildJoinOnClause автоматически строит ON условие для entry.
// currentAlias — алиас текущей таблицы (t1, t2, ...).
// prevEntries — уже добавленные таблицы (их алиасы t1, t2, ..., tN).
func buildJoinOnClause(entry *queryTableEntry, prevEntries []queryTableEntry, currentAlias int) (string, error) {
	var conditions []string

	// Прямое направление: поля entry ссылаются на prevEntries
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

	// Обратное направление: поля prevEntries ссылаются на entry
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
func (qm *queryMeta) findEntryByFieldName(name string) *queryTableEntry {
	return qm.entryByName[name]
}
