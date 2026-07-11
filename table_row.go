package qqm

import (
	"context"
	"errors"
	"reflect"
	"strings"

	"github.com/mirrorru/dot"
	"github.com/mirrorru/qqm/dialect"
)

// Table — типизированная таблица, параметризованная типом строки ROW.
// Методы: Ins (вставка), Upd (обновление), One (SELECT по PK), Del (удаление), Many (SELECT с фильтром).
// EN: Table — a typed table parameterized by row type ROW.
// Methods: Ins (insert), Upd (update), One (SELECT by PK), Del (delete), Many (SELECT with filter).
type Table[ROW any] struct {
	tableDef TableDefinition
	sql      sqlTexts
	dialect  dialect.DialectProvider
}

// NewTable создаёт типизированную таблицу для типа ROW.
// ROW должен быть value-типом (структура), не указателем.
// Собирает метаданные полей и генерирует SQL-запросы (кешируются).
// EN: NewTable creates a typed table for the ROW type.
// ROW must be a value type (struct), not a pointer.
// Collects field metadata and generates SQL queries (cached).
func NewTable[ROW any](dialect dialect.DialectProvider) *Table[ROW] {
	return new(NewTableVal[ROW](dialect))
}

func NewTableVal[ROW any](dialect dialect.DialectProvider) Table[ROW] {
	var (
		row ROW
	)
	rowType := reflect.TypeOf(row)

	fields := dot.MustMake(CollectTableFields(rowType))
	tableDef := TableDefinition{
		TableName:  getTableName(rowType),
		Fields:     fields,
		Indexes:    fields.allIndexes(),
		FieldNames: buildFieldNames(fields),
	}

	result := Table[ROW]{
		tableDef: tableDef,
		dialect:  dialect,
		sql:      tableDef.makeSQLs(dialect),
	}

	return result
}

// Defs возвращает метаданные таблицы (поля, колонки, индексы).
// EN: Defs returns table metadata (fields, columns, indexes).
func (t *Table[ROW]) Defs() TableDefinition {
	return t.tableDef
}

// SQLs возвращает сгенерированные SQL-запросы (InsertCmd, UpdateCmd, DeleteCmd, GetOneCmd, ListCmdStart, ListSortString).
// EN: SQLs returns generated SQL queries (InsertCmd, UpdateCmd, DeleteCmd, GetOneCmd, ListCmdStart, ListSortString).
func (t *Table[ROW]) SQLs() sqlTexts {
	return t.sql
}

// Ins вставляет строку. Если диалект поддерживает RETURNING — возвращает вставленную строку.
// EN: Ins inserts a row. If the dialect supports RETURNING — returns the inserted row.
func (t *Table[ROW]) Ins(ctx context.Context, tx TxProcessor, row *ROW) (*ROW, Result, error) {
	args := t.tableDef.extractArgs(row, t.tableDef.Indexes.InsertCols)
	if !t.dialect.SupportsReturning() {
		sqlResult, err := tx.ExecContext(ctx, t.sql.InsertCmd, args...)
		return row, sqlResult, err
	}
	buf := new(ROW)
	refs := t.tableDef.extractRefs(buf, t.tableDef.Indexes.SelectCols)
	err := tx.QueryRowContext(ctx, t.sql.InsertCmd, args...).Scan(refs...)

	return buf, nil, err
}

// Upd обновляет строку по PK. Если диалект поддерживает RETURNING — возвращает обновлённую строку.
// EN: Upd updates a row by PK. If the dialect supports RETURNING — returns the updated row.
func (t *Table[ROW]) Upd(ctx context.Context, tx TxProcessor, row *ROW) (*ROW, Result, error) {
	args := t.tableDef.extractArgs(row, t.tableDef.Indexes.UpdateCols)
	args = append(args, t.tableDef.extractArgs(row, t.tableDef.Indexes.PKCols)...)
	if !t.dialect.SupportsReturning() {
		sqlResult, err := tx.ExecContext(ctx, t.sql.UpdateCmd, args...)
		return row, sqlResult, err
	}
	buf := new(ROW)
	refs := t.tableDef.extractRefs(buf, t.tableDef.Indexes.SelectCols)
	err := tx.QueryRowContext(ctx, t.sql.UpdateCmd, args...).Scan(refs...)

	return buf, nil, err
}

// One возвращает одну строку по первичному ключу.
// EN: One returns a single row by primary key.
func (t *Table[ROW]) One(ctx context.Context, tx TxProcessor, keys ...any) (*ROW, error) {
	buf := new(ROW)
	refs := t.tableDef.extractRefs(buf, t.tableDef.Indexes.SelectCols)
	err := tx.QueryRowContext(ctx, t.sql.GetOneCmd, keys...).Scan(refs...)

	return buf, err
}

// Del удаляет строку по первичному ключу.
// EN: Del deletes a row by primary key.
func (t *Table[ROW]) Del(ctx context.Context, tx TxProcessor, keys ...any) (Result, error) {
	sqlResult, err := tx.ExecContext(ctx, t.sql.DeleteCmd, keys...)
	return sqlResult, err
}

// Many возвращает срез строк с фильтрацией и сортировкой.
// filter может быть nil — тогда возвращаются все строки с ORDER BY из sort-тегов.
// EN: Many returns a slice of rows with filtering and sorting.
// filter may be nil — then all rows are returned with ORDER BY from sort tags.
func (t *Table[ROW]) Many(ctx context.Context, tx TxProcessor, filter *Filter) (result []*ROW, err error) {
	var sb strings.Builder
	fieldCount := len(t.tableDef.Indexes.SelectCols)
	sb.Grow(len(t.sql.ListCmdStart) + fieldCount*20 + 128)
	sb.WriteString(t.sql.ListCmdStart)
	where, args, buildErr := filter.BuildWhere(t.tableDef.Fields, t.dialect)
	if buildErr != nil {
		return nil, buildErr
	}
	sb.WriteString(where)
	sb.WriteString(t.sql.ListSortString)
	sb.WriteString(filter.BuildOffsetAndLimit(t.dialect))

	q := sb.String()
	rows, err := tx.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = errors.Join(err, rows.Close())
	}()
	resultCap := uint32(0)
	if filter != nil {
		resultCap = filter.Limit
	}
	if resultCap == 0 {
		resultCap = 64
	}
	result = make([]*ROW, 0, resultCap)
	buf := new(ROW)
	refs := t.tableDef.extractRefs(buf, t.tableDef.Indexes.SelectCols)

	for rows.Next() {
		if err = rows.Scan(refs...); err != nil {
			return nil, err
		}
		rowBuf := new(ROW)
		*rowBuf = *buf
		result = append(result, rowBuf)
	}

	return result, rows.Err()
}
