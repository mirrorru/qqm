package field_info

import (
	"context"
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
	TableName  string
	Fields     TableFields
	Dialect    dialect.DialectProvider
	FieldNames map[string]int
	Indexes    fieldsIndexes
	SQL        sqlTexts
}

type sqlTexts struct {
	InsertCmd      string
	UpdateCmd      string
	DeleteCmd      string
	GetOneCmd      string
	ListCmdStart   string
	ListSortString string
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
	names := make(map[string]int, len(fields))
	for idx := range fields {
		fields[idx].SQLName = dialect.QuoteIdent(fields[idx].SQLName)
		names[fields[idx].SQLName] = idx
	}
	tableDef := TableDefinition{
		TableName:  sqlName,
		Fields:     fields,
		Dialect:    dialect,
		Indexes:    fields.allIndexes(),
		FieldNames: names,
	}
	tableDef.makeSQLs()
	result := &Table[ROW]{
		tableDef: tableDef,
	}

	return result
}

type TX interface{}
type Filter interface{}

func (t *Table[ROW]) Defs() TableDefinition {
	return t.tableDef
}

func (t *Table[ROW]) Ins(ctx context.Context, tx TX, row *ROW) (*ROW, error) {
	return nil, nil
}

func (t *Table[ROW]) Upd(ctx context.Context, tx TX, row *ROW) (*ROW, error) {
	return nil, nil
}

func (t *Table[ROW]) One(ctx context.Context, tx TX, keys ...any) (*ROW, error) {
	return nil, nil
}

func (t *Table[ROW]) Del(ctx context.Context, tx TX, keys ...any) (*ROW, error) {
	return nil, nil
}

func (t *Table[ROW]) Many(ctx context.Context, tx TX, filter Filter) (*ROW, error) {
	return nil, nil
}

func (td *TableDefinition) makeSQLs() {
	td.SQL = sqlTexts{
		InsertCmd:      buildInsertSQL(td),
		UpdateCmd:      buildUpdateSQL(td),
		DeleteCmd:      buildDeleteSQL(td),
		GetOneCmd:      buildGetOneSQL(td),
		ListCmdStart:   buildListSQL(td),
		ListSortString: buildOrderByClause(td),
	}
}

// buildInsertSQL формирует SQL INSERT с учётом диалекта и метаданных.
// EN: buildInsertSQL builds INSERT SQL accounting for dialect and metadata.
func buildInsertSQL(td *TableDefinition) string {
	if len(td.Indexes.InsertIdx) == 0 {
		return ""
	}

	colNames := make([]string, len(td.Indexes.InsertIdx))
	placeholders := make([]string, len(td.Indexes.InsertIdx))
	for i := range td.Indexes.InsertIdx {
		colNames[i] = td.Fields[i].SQLName
		placeholders[i] = td.Dialect.Placeholder(i + 1)
	}

	var sb strings.Builder
	sb.Grow(len(td.Fields) * 50)
	sb.WriteString(defs.SQLInsertInto)
	sb.WriteString(td.Dialect.QuoteIdent(td.TableName))
	sb.WriteString(defs.SQLOpenParen)
	for pos, idx := range td.Indexes.InsertIdx {
		if pos > 0 {
			sb.WriteString(defs.SQLCommaSpace)
		}
		sb.WriteString(td.Fields[idx].SQLName)
	}
	sb.WriteString(defs.SQLCloseParen)
	sb.WriteString(defs.SQLValues)
	sb.WriteString(defs.SQLOpenParen)
	for pos := range td.Indexes.InsertIdx {
		if pos > 0 {
			sb.WriteString(defs.SQLCommaSpace)
		}
		sb.WriteString(td.Dialect.Placeholder(pos + 1))
	}
	sb.WriteString(defs.SQLCloseParen)

	if td.Dialect.SupportsReturning() {
		writeReturning(td, &sb)
	}

	return sb.String()
}

// buildUpdateSQL формирует SQL UPDATE с учётом диалекта и метаданных.
// EN: buildUpdateSQL builds UPDATE SQL accounting for dialect and metadata.
func buildUpdateSQL(td *TableDefinition) string {
	if len(td.Indexes.UpdateIdx) == 0 || len(td.Indexes.PKIdx) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.Grow(len(td.Indexes.UpdateIdx) * 50)
	sb.WriteString(defs.SQLUpdate)
	sb.WriteString(td.TableName)
	sb.WriteString(defs.SQLSet)
	for pos, idx := range td.Indexes.UpdateIdx {
		if pos > 0 {
			sb.WriteString(defs.SQLCommaSpace)
		}
		sb.WriteString(td.Fields[idx].SQLName)
		sb.WriteString(defs.SQLEquals)
		sb.WriteString(td.Dialect.Placeholder(pos + 1))
	}
	writeWhereClauses(td, len(td.Indexes.UpdateIdx), &sb)
	if td.Dialect.SupportsReturning() {
		writeReturning(td, &sb)
	}

	return sb.String()
}

// buildGetOneSQL формирует SQL SELECT по PK с учётом диалекта и метаданных.
// EN: buildGetOneSQL builds SELECT by PK SQL accounting for dialect and metadata.
func buildGetOneSQL(td *TableDefinition) string {
	if len(td.Indexes.SelectIdx) == 0 || len(td.Indexes.PKIdx) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.Grow(len(td.Indexes.SelectIdx) * 25)
	sb.WriteString(defs.SQLSelect)
	for pos, idx := range td.Indexes.SelectIdx {
		if pos > 0 {
			sb.WriteString(defs.SQLCommaSpace)
		}
		sb.WriteString(td.Fields[idx].SQLName)
	}
	sb.WriteString(defs.SQLFrom)
	sb.WriteString(td.TableName)
	writeWhereClauses(td, len(td.Indexes.SelectIdx), &sb)
	return sb.String()
}

// buildDeleteSQL формирует SQL DELETE по PK с учётом диалекта и метаданных.
// EN: buildDeleteSQL builds DELETE by PK SQL accounting for dialect and metadata.

func buildDeleteSQL(td *TableDefinition) string {
	if len(td.Indexes.PKIdx) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.Grow(150)
	sb.WriteString(defs.SQLDelete)
	sb.WriteString(td.TableName)
	writeWhereClauses(td, 0, &sb)
	return sb.String()
}

// buildListSQL формирует SQL SELECT ALL с учётом диалекта, метаданных и сортировки.
// EN: buildListSQL builds SELECT ALL SQL accounting for dialect, metadata and sort.

func buildListSQL(td *TableDefinition) string {
	if len(td.Indexes.SelectIdx) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.Grow(len(td.Indexes.SelectIdx) * 25)
	sb.WriteString(defs.SQLSelect)
	for pos, idx := range td.Indexes.SelectIdx {
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

func buildOrderByClause(td *TableDefinition) string {
	var sb strings.Builder
	sb.Grow(len(td.Indexes.SelectIdx) * 15)
	sb.WriteString(defs.SQLOrderBy)
	for pos, idx := range td.Indexes.SortingIdx {
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
func writeWhereClauses(td *TableDefinition, offset int, sb *strings.Builder) {
	sb.WriteString(defs.SQLWhere)
	for pos, idx := range td.Indexes.PKIdx {
		if pos > 0 {
			sb.WriteString(defs.SQLAnd)
		}
		sb.WriteString(td.Fields[idx].SQLName)
		sb.WriteString(defs.SQLEquals)
		sb.WriteString(td.Dialect.Placeholder(offset + pos + 1))
	}
}

func writeReturning(td *TableDefinition, sb *strings.Builder) {
	allCols := make([]string, len(td.Indexes.SelectIdx))
	for i := range td.Indexes.SelectIdx {
		allCols[i] = td.Fields[td.Indexes.SelectIdx[i]].SQLName
	}
	sb.WriteString(defs.SQLReturning)
	sb.WriteString(strings.Join(allCols, defs.SQLCommaSpace))
}
