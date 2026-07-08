package field_info

import (
	"reflect"
	"strings"

	"github.com/mirrorru/dot"
	"github.com/mirrorru/qqm/defs"
	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/meta"
)

type SQLNamer interface {
	SQLName() string
}

type Table[ROW any] struct {
	tableDef      TableDefinition
	SelectOneSQL  string
	SelectManySQL string
	InsertSQL     string
	UpdateSQL     string
	DeleteSQL     string
}
type TableDefinition struct {
	TableName string
	Fields    TableFields
	Dialect   dialect.DialectProvider
}

func NewTable[ROW any](dialect dialect.DialectProvider) *Table[ROW] {
	var (
		ptr     *ROW
		sqlName string
	)
	rowType := reflect.TypeOf(ptr).Elem()

	if namer1, ok1 := any(ptr).(SQLNamer); ok1 {
		sqlName = namer1.SQLName()
	} else if namer2, ok2 := any(*new(ROW)).(SQLNamer); ok2 {
		sqlName = namer2.SQLName()
	} else {
		sqlName = meta.ToSnakeCase(rowType.Name())
	}

	fields := dot.MustMake(CollectTableFields(rowType))

	for idx := range fields {
		fields[idx].SQLName = dialect.QuoteIdent(fields[idx].SQLName)
	}
	result := &Table[ROW]{
		tableDef: TableDefinition{
			TableName: sqlName,
			Fields:    fields,
			Dialect:   dialect,
		},
	}
	result.InsertSQL = buildInsertSQL(&result.tableDef)

	return result
}

// buildInsertSQL формирует SQL INSERT с учётом диалекта и метаданных.
// EN: buildInsertSQL builds INSERT SQL accounting for dialect and metadata.
func buildInsertSQL(td *TableDefinition) string {
	insCols := td.Fields.InsertingColsIdx()
	if len(insCols) == 0 {
		return ""
	}

	d := td.Dialect
	colNames := make([]string, len(insCols))
	placeholders := make([]string, len(insCols))
	for i := range insCols {
		colNames[i] = td.Fields[i].SQLName
		placeholders[i] = d.Placeholder(i + 1)
	}

	var sb strings.Builder
	sb.Grow(len(td.Fields) * 50)
	sb.WriteString(defs.SQLInsertInto)
	sb.WriteString(td.Dialect.QuoteIdent(td.TableName))
	sb.WriteString(defs.SQLOpenParen)
	sb.WriteString(strings.Join(colNames, defs.SQLCommaSpace))
	sb.WriteString(defs.SQLCloseParen)
	sb.WriteString(defs.SQLValues)
	sb.WriteString(defs.SQLOpenParen)
	sb.WriteString(strings.Join(placeholders, defs.SQLCommaSpace))
	sb.WriteString(defs.SQLCloseParen)

	if d.SupportsReturning() {
		retIdx := td.Fields.SelectingColsIdx()
		allCols := make([]string, len(retIdx))
		for i := range retIdx {
			allCols[i] = td.Fields[retIdx[i]].SQLName
		}
		sb.WriteString(defs.SQLReturning)
		sb.WriteString(strings.Join(allCols, defs.SQLCommaSpace))
	}

	return sb.String()
}

//// buildUpdateSQL формирует SQL UPDATE с учётом диалекта и метаданных.
//// EN: buildUpdateSQL builds UPDATE SQL accounting for dialect and metadata.
//func buildUpdateSQL(d dialect.DialectProvider, m *meta.RowMeta) string {
//	cols := m.UpdateColumns()
//	if len(cols) == 0 || len(m.PKFields) == 0 {
//		return ""
//	}
//
//	setClauses := make([]string, len(cols))
//	for i, col := range cols {
//		setClauses[i] = d.QuoteIdent(col) + defs.SQLEquals + d.Placeholder(i+1)
//	}
//
//	whereClauses := buildWhereClauses(d, m, len(cols))
//
//	sql := defs.SQLUpdate + d.QuoteIdent(m.TableName) +
//		sqlSet + strings.Join(setClauses, defs.SQLCommaSpace) +
//		sqlWhere + strings.Join(whereClauses, defs.SQLAnd)
//
//	if d.SupportsReturning() {
//		allCols := make([]string, len(m.Columns))
//		for i, col := range m.Columns {
//			allCols[i] = d.QuoteIdent(col)
//		}
//		sql += defs.SQLReturning + strings.Join(allCols, defs.SQLCommaSpace)
//	}
//
//	return defs.SQL
//}
//
//// buildSelectSQL формирует SQL SELECT по PK с учётом диалекта и метаданных.
//// EN: buildSelectSQL builds SELECT by PK SQL accounting for dialect and metadata.
//func buildSelectSQL(d dialect.DialectProvider, m *meta.RowMeta) string {
//	if len(m.PKFields) == 0 {
//		return ""
//	}
//
//	allCols := make([]string, len(m.Columns))
//	for i, col := range m.Columns {
//		allCols[i] = d.QuoteIdent(col)
//	}
//
//	whereClauses := buildWhereClauses(d, m, 0)
//
//	return defs.SQLSelect + strings.Join(allCols, defs.SQLCommaSpace) +
//		sqlFrom + d.QuoteIdent(m.TableName) +
//		sqlWhere + strings.Join(whereClauses, defs.SQLAnd)
//}
//
//// buildDeleteSQL формирует SQL DELETE по PK с учётом диалекта и метаданных.
//// EN: buildDeleteSQL builds DELETE by PK SQL accounting for dialect and metadata.
//func buildDeleteSQL(d dialect.DialectProvider, m *meta.RowMeta) string {
//	if len(m.PKFields) == 0 {
//		return ""
//	}
//
//	whereClauses := buildWhereClauses(d, m, 0)
//
//	return defs.SQLDelete + d.QuoteIdent(m.TableName) +
//		sqlWhere + strings.Join(whereClauses, defs.SQLAnd)
//}
//
//// buildListSQL формирует SQL SELECT ALL с учётом диалекта, метаданных и сортировки.
//// EN: buildListSQL builds SELECT ALL SQL accounting for dialect, metadata and sort.
//func buildListSQL(d dialect.DialectProvider, m *meta.RowMeta) string {
//	allCols := make([]string, len(m.Columns))
//	for i, col := range m.Columns {
//		allCols[i] = d.QuoteIdent(col)
//	}
//
//	sql := defs.SQLSelect + strings.Join(allCols, defs.SQLCommaSpace) +
//		sqlFrom + d.QuoteIdent(m.TableName)
//
//	if len(m.SortFields) > 0 {
//		sql += buildOrderByClause(d, m, "")
//	}
//
//	return defs.SQL
//}
//
//// buildOrderByClause строит "ORDER BY col1 ASC, col2 DESC" из SortFields.
//// tableAlias — алиас таблицы (например, "t1") для квалифицированных имён в Query; пустая строка для простой таблицы.
//// EN: buildOrderByClause builds "ORDER BY col1 ASC, col2 DESC" from SortFields.
//// tableAlias — table alias (e.g. "t1") for qualified names in Query; empty string for a simple table.
//func buildOrderByClause(d dialect.DialectProvider, m *meta.RowMeta, tableAlias string) string {
//	parts := make([]string, len(m.SortFields))
//	for i, sf := range m.SortFields {
//		dir := defs.SQLAsc
//		if sf.SortDirection == "DESC" {
//			dir = defs.SQLDesc
//		}
//		col := d.QuoteIdent(sf.Column)
//		if tableAlias != "" {
//			col = d.QuoteIdent(tableAlias) + "." + col
//		}
//		parts[i] = col + dir
//	}
//	return defs.SQLOrderBy + strings.Join(parts, defs.SQLCommaSpace)
//}
//
//// buildWhereClauses формирует условия WHERE для PK-полей.
//// offset добавляется к индексу плейсхолдеров.
//// EN: buildWhereClauses builds WHERE conditions for PK fields.
//// offset is added to placeholder index.
//func buildWhereClauses(d dialect.DialectProvider, m *meta.RowMeta, offset int) []string {
//	whereClauses := make([]string, len(m.PKFields))
//	for i, pk := range m.PKFields {
//		whereClauses[i] = d.QuoteIdent(pk.Column) + defs.SQLEquals + d.Placeholder(offset+i+1)
//	}
//	return whereClauses
//}
