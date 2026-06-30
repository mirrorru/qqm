package qqm

import (
	"reflect"
	"strings"
	"sync"

	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/meta"
)

const (
	sqlInsertInto = "INSERT INTO "
	sqlValues     = " VALUES "
	sqlReturning  = " RETURNING "
	sqlUpdate     = "UPDATE "
	sqlSet        = " SET "
	sqlSelect     = "SELECT "
	sqlFrom       = " FROM "
	sqlWhere      = " WHERE "
	sqlDelete     = "DELETE FROM "
	sqlAnd        = " AND "
	sqlEquals     = " = "
	sqlCommaSpace = ", "
	sqlSpace      = " "
	sqlOpenParen  = "("
	sqlCloseParen = ")"
	sqlIn         = " IN "
	sqlOrderBy    = " ORDER BY "
	sqlAsc        = " ASC"
	sqlDesc       = " DESC"

	sqlCreateTable   = "CREATE TABLE "
	sqlNotNull       = " NOT NULL"
	sqlPrimaryKey    = " PRIMARY KEY"
	sqlReferences    = " REFERENCES "
	sqlAutoincrement = " AUTOINCREMENT"
)

type queryBuilder struct {
	insertOnce      sync.Once
	updateOnce      sync.Once
	selectOnce      sync.Once
	deleteOnce      sync.Once
	listOnce        sync.Once
	createTableOnce sync.Once

	insertSQL      string
	updateSQL      string
	selectSQL      string
	deleteSQL      string
	listSQL        string
	createTableSQL string
}

func newQueryBuilder() *queryBuilder {
	return &queryBuilder{}
}

func (qb *queryBuilder) InsertSQL(d dialect.DialectProvider, m *meta.RowMeta) string {
	qb.insertOnce.Do(func() {
		qb.insertSQL = buildInsertSQL(d, m)
	})
	return qb.insertSQL
}

func (qb *queryBuilder) UpdateSQL(d dialect.DialectProvider, m *meta.RowMeta) string {
	qb.updateOnce.Do(func() {
		qb.updateSQL = buildUpdateSQL(d, m)
	})
	return qb.updateSQL
}

func (qb *queryBuilder) SelectSQL(d dialect.DialectProvider, m *meta.RowMeta) string {
	qb.selectOnce.Do(func() {
		qb.selectSQL = buildSelectSQL(d, m)
	})
	return qb.selectSQL
}

func (qb *queryBuilder) DeleteSQL(d dialect.DialectProvider, m *meta.RowMeta) string {
	qb.deleteOnce.Do(func() {
		qb.deleteSQL = buildDeleteSQL(d, m)
	})
	return qb.deleteSQL
}

func (qb *queryBuilder) ListSQL(d dialect.DialectProvider, m *meta.RowMeta) string {
	qb.listOnce.Do(func() {
		qb.listSQL = buildListSQL(d, m)
	})
	return qb.listSQL
}

func (qb *queryBuilder) CreateTableSQL(d dialect.DialectProvider, m *meta.RowMeta) string {
	qb.createTableOnce.Do(func() {
		qb.createTableSQL = buildCreateTableSQL(d, m)
	})
	return qb.createTableSQL
}

func buildInsertSQL(d dialect.DialectProvider, m *meta.RowMeta) string {
	cols := m.InsertColumns()
	if len(cols) == 0 {
		return ""
	}

	quotedCols := make([]string, len(cols))
	placeholders := make([]string, len(cols))
	for i, col := range cols {
		quotedCols[i] = d.QuoteIdent(col)
		placeholders[i] = d.Placeholder(i + 1)
	}

	sql := sqlInsertInto + d.QuoteIdent(m.TableName) + sqlSpace +
		sqlOpenParen + strings.Join(quotedCols, sqlCommaSpace) + sqlCloseParen +
		sqlValues + sqlOpenParen + strings.Join(placeholders, sqlCommaSpace) + sqlCloseParen

	if d.SupportsReturning() {
		allCols := make([]string, len(m.Columns))
		for i, col := range m.Columns {
			allCols[i] = d.QuoteIdent(col)
		}
		sql += sqlReturning + strings.Join(allCols, sqlCommaSpace)
	}

	return sql
}

func buildUpdateSQL(d dialect.DialectProvider, m *meta.RowMeta) string {
	cols := m.UpdateColumns()
	if len(cols) == 0 || len(m.PKFields) == 0 {
		return ""
	}

	setClauses := make([]string, len(cols))
	for i, col := range cols {
		setClauses[i] = d.QuoteIdent(col) + sqlEquals + d.Placeholder(i+1)
	}

	whereClauses := buildWhereClauses(d, m, len(cols))

	return sqlUpdate + d.QuoteIdent(m.TableName) +
		sqlSet + strings.Join(setClauses, sqlCommaSpace) +
		sqlWhere + strings.Join(whereClauses, sqlAnd)
}

func buildSelectSQL(d dialect.DialectProvider, m *meta.RowMeta) string {
	if len(m.PKFields) == 0 {
		return ""
	}

	allCols := make([]string, len(m.Columns))
	for i, col := range m.Columns {
		allCols[i] = d.QuoteIdent(col)
	}

	whereClauses := buildWhereClauses(d, m, 0)

	return sqlSelect + strings.Join(allCols, sqlCommaSpace) +
		sqlFrom + d.QuoteIdent(m.TableName) +
		sqlWhere + strings.Join(whereClauses, sqlAnd)
}

func buildDeleteSQL(d dialect.DialectProvider, m *meta.RowMeta) string {
	if len(m.PKFields) == 0 {
		return ""
	}

	whereClauses := buildWhereClauses(d, m, 0)

	return sqlDelete + d.QuoteIdent(m.TableName) +
		sqlWhere + strings.Join(whereClauses, sqlAnd)
}

func buildListSQL(d dialect.DialectProvider, m *meta.RowMeta) string {
	allCols := make([]string, len(m.Columns))
	for i, col := range m.Columns {
		allCols[i] = d.QuoteIdent(col)
	}

	sql := sqlSelect + strings.Join(allCols, sqlCommaSpace) +
		sqlFrom + d.QuoteIdent(m.TableName)

	if len(m.SortFields) > 0 {
		sql += buildOrderByClause(d, m, "")
	}

	return sql
}

// buildOrderByClause строит "ORDER BY col1 ASC, col2 DESC" из SortFields.
// tableAlias — алиас таблицы (например, "t1") для квалифицированных имён в Query; пустая строка для простой таблицы.
// EN: buildOrderByClause builds "ORDER BY col1 ASC, col2 DESC" from SortFields.
// tableAlias — table alias (e.g. "t1") for qualified names in Query; empty string for a simple table.
func buildOrderByClause(d dialect.DialectProvider, m *meta.RowMeta, tableAlias string) string {
	parts := make([]string, len(m.SortFields))
	for i, sf := range m.SortFields {
		dir := sqlAsc
		if sf.SortDirection == "DESC" {
			dir = sqlDesc
		}
		col := d.QuoteIdent(sf.Column)
		if tableAlias != "" {
			col = d.QuoteIdent(tableAlias) + "." + col
		}
		parts[i] = col + dir
	}
	return sqlOrderBy + strings.Join(parts, sqlCommaSpace)
}

func buildWhereClauses(d dialect.DialectProvider, m *meta.RowMeta, offset int) []string {
	whereClauses := make([]string, len(m.PKFields))
	for i, pk := range m.PKFields {
		whereClauses[i] = d.QuoteIdent(pk.Column) + sqlEquals + d.Placeholder(offset+i+1)
	}
	return whereClauses
}

// buildCreateTableSQL строит CREATE TABLE с учётом pk, default= и ref=.
// EN: buildCreateTableSQL builds CREATE TABLE accounting for pk, default= and ref=.
func buildCreateTableSQL(d dialect.DialectProvider, m *meta.RowMeta) string {
	var b strings.Builder
	b.WriteString(sqlCreateTable)
	b.WriteString(d.QuoteIdent(m.TableName))
	b.WriteString(" (\n")

	var pkCols []string
	first := true

	for _, fm := range m.Fields {
		if fm.IsOmit {
			continue
		}

		if !first {
			b.WriteString(",\n")
		}
		first = false

		b.WriteString("\t")
		b.WriteString(d.QuoteIdent(fm.Column))
		b.WriteString(sqlSpace)

		sqlType := goTypeToSQL(fm.GoType)
		if fm.IsPK && fm.IsAuto && d.Name() == "sqlite" {
			sqlType = "INTEGER"
		}
		b.WriteString(sqlType)

		if fm.IsPK && fm.IsAuto {
			b.WriteString(sqlPrimaryKey)
			if d.Name() == "sqlite" {
				b.WriteString(sqlAutoincrement)
			}
		} else if fm.IsPK {
			pkCols = append(pkCols, d.QuoteIdent(fm.Column))
		} else {
			b.WriteString(sqlNotNull)
		}

		if fm.CreateClause != "" {
			b.WriteString(sqlSpace)
			b.WriteString(fm.CreateClause)
		}

		if fm.RefTable != "" {
			b.WriteString(sqlReferences)
			b.WriteString(d.QuoteIdent(fm.RefTable))
			refCol := fm.RefColumn
			if refCol == "" {
				refCol = "id"
			}
			b.WriteString(sqlOpenParen)
			b.WriteString(d.QuoteIdent(refCol))
			b.WriteString(sqlCloseParen)
		}
	}

	if len(pkCols) > 0 {
		b.WriteString(",\n\t")
		b.WriteString(sqlPrimaryKey)
		b.WriteString(" (")
		b.WriteString(strings.Join(pkCols, sqlCommaSpace))
		b.WriteString(")")
	}

	b.WriteString("\n)")
	return b.String()
}

// goTypeToSQL преобразует Go-тип в SQL-тип для CREATE TABLE.
// EN: goTypeToSQL converts Go type to SQL type for CREATE TABLE.
func goTypeToSQL(t reflect.Type) string {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Int, reflect.Int32:
		return "INTEGER"
	case reflect.Int64:
		return "BIGINT"
	case reflect.Float32:
		return "REAL"
	case reflect.Float64:
		return "DOUBLE PRECISION"
	case reflect.String:
		return "TEXT"
	case reflect.Bool:
		return "BOOLEAN"
	}

	switch t.String() {
	case "time.Time":
		return "TIMESTAMPTZ"
	}

	return "TEXT"
}
