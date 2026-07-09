package field_info

import (
	"reflect"
	"strings"

	"github.com/mirrorru/qqm/defs"
	"github.com/mirrorru/qqm/dialect"
)

type TableDefinition struct {
	TableName  string
	Fields     TableFields
	FieldNames map[string]int
	Indexes    fieldsIndexes
}

func (td *TableDefinition) makeSQLs(dlct dialect.DialectProvider) sqlTexts {
	return sqlTexts{
		InsertCmd:      td.buildInsertSQL(dlct),
		UpdateCmd:      td.buildUpdateSQL(dlct),
		DeleteCmd:      td.buildDeleteSQL(dlct),
		GetOneCmd:      td.buildGetOneSQL(dlct),
		ListCmdStart:   td.buildListSQL(),
		ListSortString: td.buildOrderByClause(),
	}
}

func (td *TableDefinition) extractArgs(src any, indexes []int) []any {
	rv := reflect.ValueOf(src).Elem()
	result := make([]any, len(indexes))
	for pos, idx := range indexes {
		fld := rv.FieldByIndex(td.Fields[idx].Index)
		result[pos] = fld.Interface()
	}

	return result
}

func (td *TableDefinition) extractRefs(src any, indexes []int) (refs []any) {
	rv := reflect.ValueOf(src).Elem()
	result := make([]any, len(indexes))
	for pos, idx := range indexes {
		fld := rv.FieldByIndex(td.Fields[idx].Index)
		result[pos] = fld.Addr().Interface()
	}

	return result
}

// buildUpdateSQL формирует SQL UPDATE с учётом диалекта и метаданных.
// EN: buildUpdateSQL builds UPDATE SQL accounting for dialect and metadata.
func (td *TableDefinition) buildUpdateSQL(dlct dialect.DialectProvider) string {
	if len(td.Indexes.UpdateCols) == 0 || len(td.Indexes.PKCols) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.Grow(len(td.Indexes.UpdateCols) * 50)
	sb.WriteString(defs.SQLUpdate)
	sb.WriteString(td.TableName)
	sb.WriteString(defs.SQLSet)
	for pos, idx := range td.Indexes.UpdateCols {
		if pos > 0 {
			sb.WriteString(defs.SQLCommaSpace)
		}
		sb.WriteString(td.Fields[idx].SQLName)
		sb.WriteString(defs.SQLEquals)
		sb.WriteString(dlct.Placeholder(pos + 1))
	}
	td.writeWhereClauses(dlct, len(td.Indexes.UpdateCols), &sb)
	if dlct.SupportsReturning() {
		td.writeReturning(&sb)
	}

	return sb.String()
}

// buildInsertSQL формирует SQL INSERT с учётом диалекта и метаданных.
// EN: buildInsertSQL builds INSERT SQL accounting for dialect and metadata.
func (td *TableDefinition) buildInsertSQL(dlct dialect.DialectProvider) string {
	if len(td.Indexes.InsertCols) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.Grow(len(td.Fields) * 50)
	sb.WriteString(defs.SQLInsertInto)
	sb.WriteString(dlct.QuoteIdent(td.TableName))
	sb.WriteString(defs.SQLOpenParen)
	for pos, idx := range td.Indexes.InsertCols {
		if pos > 0 {
			sb.WriteString(defs.SQLCommaSpace)
		}
		sb.WriteString(td.Fields[idx].SQLName)
	}
	sb.WriteString(defs.SQLCloseParen)
	sb.WriteString(defs.SQLValues)
	sb.WriteString(defs.SQLOpenParen)
	for pos := range td.Indexes.InsertCols {
		if pos > 0 {
			sb.WriteString(defs.SQLCommaSpace)
		}
		sb.WriteString(dlct.Placeholder(pos + 1))
	}
	sb.WriteString(defs.SQLCloseParen)

	if dlct.SupportsReturning() {
		td.writeReturning(&sb)
	}

	return sb.String()
}

// buildGetOneSQL формирует SQL SELECT по PK с учётом диалекта и метаданных.
// EN: buildGetOneSQL builds SELECT by PK SQL accounting for dialect and metadata.
func (td *TableDefinition) buildGetOneSQL(dlct dialect.DialectProvider) string {
	if len(td.Indexes.SelectCols) == 0 || len(td.Indexes.PKCols) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.Grow(len(td.Indexes.SelectCols) * 25)
	sb.WriteString(defs.SQLSelect)
	for pos, idx := range td.Indexes.SelectCols {
		if pos > 0 {
			sb.WriteString(defs.SQLCommaSpace)
		}
		sb.WriteString(td.Fields[idx].SQLName)
	}
	sb.WriteString(defs.SQLFrom)
	sb.WriteString(td.TableName)
	td.writeWhereClauses(dlct, 0, &sb)
	return sb.String()
}

// buildDeleteSQL формирует SQL DELETE по PK с учётом диалекта и метаданных.
// EN: buildDeleteSQL builds DELETE by PK SQL accounting for dialect and metadata.

func (td *TableDefinition) buildDeleteSQL(dlct dialect.DialectProvider) string {
	if len(td.Indexes.PKCols) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.Grow(150)
	sb.WriteString(defs.SQLDelete)
	sb.WriteString(td.TableName)
	td.writeWhereClauses(dlct, 0, &sb)
	return sb.String()
}

// buildListSQL формирует SQL SELECT ALL с учётом диалекта, метаданных и сортировки.
// EN: buildListSQL builds SELECT ALL SQL accounting for dialect, metadata and sort.

func (td *TableDefinition) buildListSQL() string {
	if len(td.Indexes.SelectCols) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.Grow(len(td.Indexes.SelectCols) * 25)
	sb.WriteString(defs.SQLSelect)
	for pos, idx := range td.Indexes.SelectCols {
		if pos > 0 {
			sb.WriteString(defs.SQLCommaSpace)
		}
		sb.WriteString(td.Fields[idx].SQLName)
	}
	sb.WriteString(defs.SQLFrom)
	sb.WriteString(td.TableName)
	return sb.String()
}

// buildOrderByClause строит "ORDER BY col1 ASC, col2 DESC" из SortFields.
// tableAlias — алиас таблицы (например, "t1") для квалифицированных имён в Query; пустая строка для простой таблицы.
// EN: buildOrderByClause builds "ORDER BY col1 ASC, col2 DESC" from SortFields.
// tableAlias — table alias (e.g. "t1") for qualified names in Query; empty string for a simple table.

func (td *TableDefinition) buildOrderByClause() string {
	var sb strings.Builder
	sb.Grow(len(td.Indexes.SelectCols) * 15)
	sb.WriteString(defs.SQLOrderBy)
	for pos, idx := range td.Indexes.SortingCols {
		if pos > 0 {
			sb.WriteString(defs.SQLCommaSpace)
		}
		sb.WriteString(td.Fields[idx].SQLName)
		if td.Fields[idx].Flags.SortBackward {
			sb.WriteString(defs.SQLDesc)
		}
	}
	return sb.String()
}

// writeWhereClauses формирует условия WHERE для PK-полей.
// offset добавляется к индексу плейсхолдеров.
// EN: writeWhereClauses builds WHERE conditions for PK fields.
// offset is added to placeholder index.
func (td *TableDefinition) writeWhereClauses(dlct dialect.DialectProvider, offset int, sb *strings.Builder) {
	sb.WriteString(defs.SQLWhere)
	for pos, idx := range td.Indexes.PKCols {
		if pos > 0 {
			sb.WriteString(defs.SQLAnd)
		}
		sb.WriteString(td.Fields[idx].SQLName)
		sb.WriteString(defs.SQLEquals)
		sb.WriteString(dlct.Placeholder(offset + pos + 1))
	}
}

func (td *TableDefinition) writeReturning(sb *strings.Builder) {
	allCols := make([]string, len(td.Indexes.SelectCols))
	for i := range td.Indexes.SelectCols {
		allCols[i] = td.Fields[td.Indexes.SelectCols[i]].SQLName
	}
	sb.WriteString(defs.SQLReturning)
	sb.WriteString(strings.Join(allCols, defs.SQLCommaSpace))
}
