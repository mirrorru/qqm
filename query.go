//nolint:nestif
package qqm

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/mirrorru/dot"
	"github.com/mirrorru/qqm/defs"
	"github.com/mirrorru/qqm/dialect"
)

type queryTableEntry struct {
	fieldIdx  int
	tableDef  *TableDefinition
	flags     TableFlags
	alias     string
	isPrimary bool
	offsets   struct {
		destStart   int
		fieldsStart int
	}
	refFieldIdxs []int // позиции ref-полей в tempDest (для NULL-проверки)
	fieldCount   int   // количество selectable полей
}

type scanState struct {
	dest      []any
	tempDests [][]any
}

// Query — типизированный multi-table SELECT с JOIN.
// QROW — структура, поля которой — ROW-типы таблиц.
// JOIN-условия выводятся автоматически из тегов ref= на полях ROW-структур.
// EN: Query — typed multi-table SELECT with JOIN.
// QROW — struct whose fields are ROW table types.
// JOIN conditions are auto-inferred from ref= tags on ROW struct fields.
type Query[QROW any] struct {
	dialect    dialect.DialectProvider
	tables     []queryTableEntry
	primaryIdx int
	flatFields TableFields
	idxMapping map[string]int
	sql        sqlTexts
	qrowType   reflect.Type
}

// NewQuery создаёт типизированный multi-table запрос для типа QROW.
// Собирает метаданные всех таблиц, строит JOIN-условия и генерирует SQL.
// EN: NewQuery creates a typed multi-table query for the QROW type.
// Collects metadata for all tables, builds JOIN conditions and generates SQL.
func NewQuery[QROW any](d dialect.DialectProvider) *Query[QROW] {
	return new(NewQueryVal[QROW](d))
}

func NewQueryVal[QROW any](d dialect.DialectProvider) Query[QROW] {
	var ptr *QROW
	t := reflect.TypeOf(ptr).Elem()

	if t.Kind() != reflect.Struct {
		panic("QROW must be a struct")
	}

	entries, primaryIdx := buildEntries(t)
	q := Query[QROW]{
		dialect:    d,
		tables:     entries,
		primaryIdx: primaryIdx,
		qrowType:   t,
	}
	q.buildJoinOnClauses()
	q.buildFlatFields()
	q.buildDest()
	q.sql = q.buildSQL()

	return q
}

// SQLs возвращает сгенерированные SQL-запросы (GetOneCmd, ListCmdStart, ListSortString).
// EN: SQLs returns generated SQL queries (GetOneCmd, ListCmdStart, ListSortString).
func (q *Query[QROW]) SQLs() sqlTexts {
	return q.sql
}

// FlatFields возвращает плоский список всех selectable полей всех таблиц.
// Используется для определения индексов полей в Cond().
// EN: FlatFields returns the flat list of all selectable fields from all tables.
// Used to determine field indices in Cond().
func (q *Query[QROW]) FlatFields() TableFields {
	return q.flatFields
}

func buildEntries(t reflect.Type) ([]queryTableEntry, int) {
	entries := make([]queryTableEntry, 0, t.NumField())
	primaryIdx := -1

	for idx := range t.NumField() {
		sf := t.Field(idx)
		if !sf.IsExported() || sf.Anonymous {
			continue
		}
		tFlags, ok := parseTableTag(sf.Tag.Get(tagName))
		if !ok {
			continue
		}
		if tFlags.IsFrom {
			if primaryIdx != -1 {
				panic("multiple primary tag fields found")
			}
			primaryIdx = len(entries)
		}

		sqlName := getTableName(sf.Type)
		fields := dot.MustMake(CollectTableFields(sf.Type))
		tableDef := &TableDefinition{
			TableName:  sqlName,
			Fields:     fields,
			Indexes:    fields.allIndexes(),
			FieldNames: buildFieldNames(fields),
		}

		alias := tFlags.Alias
		if alias == "" {
			alias = sqlName
		}

		fieldCount := 0
		for _, f := range fields {
			if f.Flags.canSelect() {
				fieldCount++
			}
		}

		entries = append(entries, queryTableEntry{
			fieldIdx:   idx,
			tableDef:   tableDef,
			flags:      tFlags,
			alias:      alias,
			isPrimary:  tFlags.IsFrom,
			fieldCount: fieldCount,
		})
	}

	if primaryIdx == -1 {
		primaryIdx = 0
	}
	if len(entries) > 0 {
		entries[primaryIdx].isPrimary = true
	}

	return entries, primaryIdx
}

func buildFieldNames(fields TableFields) map[string]int {
	names := make(map[string]int, len(fields))
	for idx := range fields {
		names[fields[idx].SQLName] = idx
	}
	return names
}

func (q *Query[QROW]) buildJoinOnClauses() {
	for i := range q.tables {
		if q.tables[i].isPrimary {
			continue
		}
		_, err := buildJoinOnClause(&q.tables[i], q.tables[:i])
		if err != nil {
			panic(err)
		}
	}
}

func buildJoinOnClause(cur *queryTableEntry, prev []queryTableEntry) (string, error) {
	var conditions []string
	curAlias := cur.alias

	buildCond := func(refAlias, refCol, col string) {
		conditions = append(conditions,
			fmt.Sprintf("%s.%s = %s.%s", curAlias, col, refAlias, refCol))
	}

	for _, field := range cur.tableDef.Fields {
		if field.Flags.Ref == "" {
			continue
		}
		refTable, refCol := parseRef(field.Flags.Ref)
		refAlias := lookupAlias(cur.flags.RefMap, refTable)

		for pi := range prev {
			if prev[pi].alias == refAlias || prev[pi].tableDef.TableName == refTable {
				buildCond(prev[pi].alias, refCol, field.SQLName)
			}
		}
	}

	for pi := range prev {
		pe := &prev[pi]
		for _, field := range pe.tableDef.Fields {
			if field.Flags.Ref == "" {
				continue
			}
			refTable, refCol := parseRef(field.Flags.Ref)
			translated := lookupAlias(pe.flags.RefMap, refTable)
			if translated == cur.alias || refTable == cur.tableDef.TableName {
				buildCond(pe.alias, field.SQLName, refCol)
			}
		}
	}

	if len(conditions) == 0 {
		return "", fmt.Errorf("qqm: no FK relationship found for table %q", cur.tableDef.TableName)
	}

	return strings.Join(conditions, defs.SQLAnd), nil
}

func parseRef(ref string) (table, column string) {
	parts := strings.SplitN(ref, ".", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return parts[0], ""
}

func lookupAlias(refMap map[string]string, name string) string {
	if refMap != nil {
		if alias, ok := refMap[name]; ok {
			return alias
		}
	}
	return name
}

func (q *Query[QROW]) buildFlatFields() {
	q.flatFields = nil
	q.idxMapping = make(map[string]int)

	for ei := range q.tables {
		entry := &q.tables[ei]
		entry.offsets.fieldsStart = len(q.flatFields)

		for _, field := range entry.tableDef.Fields {
			if !field.Flags.canSelect() {
				continue
			}
			qf := TableField{
				Index:   append([]int{entry.fieldIdx}, field.Index...),
				Path:    field.Path,
				SQLName: entry.alias + "." + field.SQLName,
				Flags:   field.Flags,
			}
			q.idxMapping[qf.SQLName] = len(q.flatFields)
			q.flatFields = append(q.flatFields, qf)
		}
	}
}

func (q *Query[QROW]) buildDest() {
	var di int
	for ei := range q.tables {
		entry := &q.tables[ei]
		entry.offsets.destStart = di
		entry.refFieldIdxs = nil

		ti := 0
		for _, field := range entry.tableDef.Fields {
			if !field.Flags.canSelect() {
				continue
			}
			if !entry.isPrimary && field.Flags.Ref != "" {
				entry.refFieldIdxs = append(entry.refFieldIdxs, ti)
			}
			if !entry.isPrimary {
				ti++
			}
			di++
		}
	}
}

func (q *Query[QROW]) newScanState(row *QROW) *scanState {
	ss := &scanState{
		dest:      make([]any, 0, len(q.flatFields)),
		tempDests: make([][]any, len(q.tables)),
	}

	rv := reflect.ValueOf(row).Elem()

	for ei := range q.tables {
		entry := &q.tables[ei]

		if !entry.isPrimary {
			ss.tempDests[ei] = make([]any, entry.fieldCount)
		}

		ti := 0
		for _, field := range entry.tableDef.Fields {
			if !field.Flags.canSelect() {
				continue
			}
			if entry.isPrimary {
				fullPath := append([]int{entry.fieldIdx}, field.Index...)
				fv := rv.FieldByIndex(fullPath)
				ss.dest = append(ss.dest, fv.Addr().Interface())
			} else {
				ss.dest = append(ss.dest, &ss.tempDests[ei][ti])
				ti++
			}
		}
	}

	return ss
}

func (ss *scanState) clearTempDests() {
	for _, td := range ss.tempDests {
		for i := range td {
			td[i] = nil
		}
	}
}

func (q *Query[QROW]) applyNulls(buf *QROW, ss *scanState) { //nolint:gocognit
	rv := reflect.ValueOf(buf).Elem()
	for ei := range q.tables {
		entry := &q.tables[ei]
		if entry.isPrimary {
			continue
		}
		if len(entry.refFieldIdxs) == 0 {
			continue
		}

		tempDest := ss.tempDests[ei]

		allNull := true
		for _, idx := range entry.refFieldIdxs {
			if idx < len(tempDest) && tempDest[idx] != nil {
				allNull = false
				break
			}
		}

		if allNull {
			fv := rv.Field(entry.fieldIdx)
			fv.Set(reflect.Zero(fv.Type()))
		} else {
			rowVal := rv.Field(entry.fieldIdx)
			ti := 0
			for _, field := range entry.tableDef.Fields {
				if !field.Flags.canSelect() {
					continue
				}
				if ti < len(tempDest) && tempDest[ti] != nil {
					fv := rowVal.FieldByIndex(field.Index)
					srcVal := reflect.ValueOf(tempDest[ti])
					if srcVal.Type().AssignableTo(fv.Type()) {
						fv.Set(srcVal)
					} else if srcVal.Type().ConvertibleTo(fv.Type()) {
						fv.Set(srcVal.Convert(fv.Type()))
					}
				}
				ti++
			}
		}
	}
}

func (q *Query[QROW]) buildSQL() sqlTexts {
	return sqlTexts{
		GetOneCmd:      q.buildGetOneSQL(),
		ListCmdStart:   q.buildListSQL(),
		ListSortString: q.buildOrderByClause(),
	}
}

func (q *Query[QROW]) collectSelectColumns() []string {
	var cols []string
	for _, entry := range q.tables {
		for _, idx := range entry.tableDef.Indexes.SelectCols {
			field := entry.tableDef.Fields[idx]
			cols = append(cols, entry.alias+"."+field.SQLName)
		}
	}
	return cols
}

func (q *Query[QROW]) buildListSQL() string {
	if len(q.tables) == 0 {
		return ""
	}
	cols := q.collectSelectColumns()
	if len(cols) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(defs.SQLSelect)
	sb.WriteString(strings.Join(cols, defs.SQLCommaSpace))
	sb.WriteString(defs.SQLFrom)
	q.writeFromJoin(&sb)
	return sb.String()
}

func (q *Query[QROW]) writeFromJoin(sb *strings.Builder) {
	if len(q.tables) == 0 {
		return
	}
	primary := &q.tables[q.primaryIdx]
	sb.WriteString(primary.tableDef.TableName)
	if primary.alias != primary.tableDef.TableName {
		sb.WriteString(defs.SQLAs)
		sb.WriteString(primary.alias)
	}

	for i := range q.tables {
		if q.tables[i].isPrimary {
			continue
		}
		entry := &q.tables[i]
		joinType := entry.flags.JoinMode
		if joinType == JoinModeNone {
			joinType = JoinModeInner
		}

		var joinSQL string
		switch joinType {
		case JoinModeLeft:
			joinSQL = defs.SQLLeftJoin
		case JoinModeRight:
			joinSQL = defs.SQLRightJoin
		case JoinModeInner:
			joinSQL = defs.SQLInnerJoin
		default:
			joinSQL = defs.SQLJoin
		}

		sb.WriteString(joinSQL)
		sb.WriteString(entry.tableDef.TableName)
		if entry.alias != entry.tableDef.TableName {
			sb.WriteString(defs.SQLAs)
			sb.WriteString(entry.alias)
		}

		onClause, _ := buildJoinOnClause(entry, q.tables[:i])
		sb.WriteString(defs.SQLOn)
		sb.WriteString(onClause)
	}
}

func (q *Query[QROW]) buildGetOneSQL() string {
	if len(q.tables) == 0 {
		return ""
	}
	primary := &q.tables[q.primaryIdx]
	if len(primary.tableDef.Indexes.PKCols) == 0 {
		return ""
	}

	baseSQL := q.buildListSQL()
	if baseSQL == "" {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(baseSQL)

	phIdx := 1
	sb.WriteString(defs.SQLWhere)
	for pos, idx := range primary.tableDef.Indexes.PKCols {
		if pos > 0 {
			sb.WriteString(defs.SQLAnd)
		}
		field := primary.tableDef.Fields[idx]
		sb.WriteString(primary.alias + "." + field.SQLName)
		sb.WriteString(defs.SQLEquals)
		sb.WriteString(q.dialect.Placeholder(phIdx))
		phIdx++
	}

	for i := range q.tables {
		if q.tables[i].isPrimary {
			continue
		}
		entry := &q.tables[i]
		if !entry.flags.UsePk {
			continue
		}
		if len(entry.tableDef.Indexes.PKCols) == 0 {
			continue
		}
		for _, idx := range entry.tableDef.Indexes.PKCols {
			sb.WriteString(defs.SQLAnd)
			field := entry.tableDef.Fields[idx]
			sb.WriteString(entry.alias + "." + field.SQLName)
			sb.WriteString(defs.SQLEquals)
			sb.WriteString(q.dialect.Placeholder(phIdx))
			phIdx++
		}
	}
	sb.WriteString(q.dialect.OffsetAndLimit(0, 1))

	return sb.String()
}

type sortField struct {
	sqlName string
	desc    bool
	key     int
}

func (q *Query[QROW]) buildOrderByClause() string {
	var sorts []sortField
	for _, entry := range q.tables {
		tableOrder := entry.flags.SortOrder
		for _, field := range entry.tableDef.Fields {
			if field.Flags.SortPos == 0 {
				continue
			}
			sorts = append(sorts, sortField{
				sqlName: entry.alias + "." + field.SQLName,
				desc:    field.Flags.SortBackward,
				key:     tableOrder*1000 + field.Flags.SortPos,
			})
		}
	}

	if len(sorts) == 0 {
		return ""
	}

	slices.SortStableFunc(sorts, func(a, b sortField) int {
		return a.key - b.key
	})

	var sb strings.Builder
	sb.WriteString(defs.SQLOrderBy)
	for pos, sf := range sorts {
		if pos > 0 {
			sb.WriteString(defs.SQLCommaSpace)
		}
		sb.WriteString(sf.sqlName)
		if sf.desc {
			sb.WriteString(defs.SQLDesc)
		}
	}

	return sb.String()
}

// One возвращает одну строку Query по PK первичной таблицы (и таблиц с тегом pk).
// Для LEFT JOIN без совпадений поля присоединённых таблиц обнуляются.
// EN: One returns a single Query row by PK of the primary table (and tables with pk tag).
// For LEFT JOIN with no match, joined table fields are zeroed.
func (q *Query[QROW]) One(ctx context.Context, tx TxProcessor, keys ...any) (*QROW, error) {
	buf := new(QROW)
	ss := q.newScanState(buf)
	err := tx.QueryRowContext(ctx, q.sql.GetOneCmd, keys...).Scan(ss.dest...)
	if err != nil {
		return nil, err
	}
	q.applyNulls(buf, ss)
	return buf, nil
}

// Many возвращает срез строк Query с JOIN, фильтрацией и сортировкой.
// filter может быть nil — тогда возвращаются все строки с ORDER BY из sort-тегов.
// Для LEFT JOIN без совпадений поля присоединённых таблиц обнуляются.
// EN: Many returns a slice of Query rows with JOIN, filtering and sorting.
// filter may be nil — then all rows are returned with ORDER BY from sort tags.
// For LEFT JOIN with no match, joined table fields are zeroed.
func (q *Query[QROW]) Many(ctx context.Context, tx TxProcessor, filter *Filter) (result []*QROW, err error) {
	var sb strings.Builder
	sb.WriteString(q.sql.ListCmdStart)
	where, args, buildErr := filter.BuildWhere(q.flatFields, q.dialect)
	if buildErr != nil {
		return nil, buildErr
	}
	sb.WriteString(where)
	sb.WriteString(q.sql.ListSortString)
	sb.WriteString(filter.BuildOffsetAndLimit(q.dialect))

	rows, err := tx.QueryContext(ctx, sb.String(), args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = errors.Join(err, rows.Close())
	}()

	buf := new(QROW)
	ss := q.newScanState(buf)
	for rows.Next() {
		ss.clearTempDests()
		if err = rows.Scan(ss.dest...); err != nil {
			return nil, err
		}
		q.applyNulls(buf, ss)
		rowBuf := new(QROW)
		*rowBuf = *buf
		result = append(result, rowBuf)
	}

	return result, rows.Err()
}
