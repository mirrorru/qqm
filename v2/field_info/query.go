package field_info

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
	"github.com/mirrorru/qqm/txproc"
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
	tempDest     []any // промежуточный буфер для не-primary таблиц
	fieldCount   int   // количество selectable полей
}

type Query[QROW any] struct {
	dialect    dialect.DialectProvider
	tables     []queryTableEntry
	primaryIdx int
	flatFields TableFields
	idxMapping map[string]int
	sql        sqlTexts
	dest       []any
	qrowType   reflect.Type
}

func NewQuery[QROW any](d dialect.DialectProvider) *Query[QROW] {
	var ptr *QROW
	t := reflect.TypeOf(ptr).Elem()

	if t.Kind() != reflect.Struct {
		panic("QROW must be a struct")
	}

	entries, primaryIdx := buildEntries(t)
	q := &Query[QROW]{
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

func (q *Query[QROW]) SQLs() sqlTexts {
	return q.sql
}

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
		return "", fmt.Errorf("field_info: no FK relationship found for table %q", cur.tableDef.TableName)
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
	q.dest = nil

	for ei := range q.tables {
		entry := &q.tables[ei]
		entry.offsets.destStart = len(q.dest)
		entry.refFieldIdxs = nil

		if !entry.isPrimary {
			entry.tempDest = make([]any, entry.fieldCount)
		}

		ti := 0
		for _, field := range entry.tableDef.Fields {
			if !field.Flags.canSelect() {
				continue
			}
			if entry.isPrimary {
				q.dest = append(q.dest, nil)
			} else {
				if field.Flags.Ref != "" {
					entry.refFieldIdxs = append(entry.refFieldIdxs, ti)
				}
				q.dest = append(q.dest, &entry.tempDest[ti])
				ti++
			}
		}
	}
}

func (q *Query[QROW]) resetDest(buf *QROW) {
	rv := reflect.ValueOf(buf).Elem()
	for ei := range q.tables {
		entry := &q.tables[ei]
		di := entry.offsets.destStart
		if !entry.isPrimary {
			for i := range entry.tempDest {
				entry.tempDest[i] = nil
			}
			return
		}
		for _, field := range entry.tableDef.Fields {
			if !field.Flags.canSelect() {
				continue
			}
			fullPath := append([]int{entry.fieldIdx}, field.Index...)
			fv := rv.FieldByIndex(fullPath)
			q.dest[di] = fv.Addr().Interface()
			di++
		}
	}
}

func (q *Query[QROW]) applyNulls(buf *QROW) {
	rv := reflect.ValueOf(buf).Elem()
	for ei := range q.tables {
		entry := &q.tables[ei]
		if entry.isPrimary {
			continue
		}
		if len(entry.refFieldIdxs) == 0 {
			continue
		}

		allNull := true
		for _, idx := range entry.refFieldIdxs {
			if idx < len(entry.tempDest) && entry.tempDest[idx] != nil {
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
				if ti < len(entry.tempDest) && entry.tempDest[ti] != nil {
					fv := rowVal.FieldByIndex(field.Index)
					srcVal := reflect.ValueOf(entry.tempDest[ti])
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

func (q *Query[QROW]) One(ctx context.Context, tx txproc.TxProcessor, keys ...any) (*QROW, error) {
	buf := new(QROW)
	q.resetDest(buf)
	err := tx.QueryRowContext(ctx, q.sql.GetOneCmd, keys...).Scan(q.dest...)
	if err != nil {
		return nil, err
	}
	q.applyNulls(buf)
	return buf, nil
}

func (q *Query[QROW]) Many(ctx context.Context, tx txproc.TxProcessor, filter *Filter) (result []*QROW, err error) {
	var sb strings.Builder
	sb.WriteString(q.sql.ListCmdStart)
	where, args := filter.BuildWhere(q.flatFields, q.dialect)
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
	for rows.Next() {
		q.resetDest(buf)
		if err = rows.Scan(q.dest...); err != nil {
			return nil, err
		}
		q.applyNulls(buf)
		rowBuf := new(QROW)
		*rowBuf = *buf
		result = append(result, rowBuf)
	}

	return result, rows.Err()
}
