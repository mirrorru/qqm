package qqm

import (
	"context"
	"errors"
	"reflect"
	"strings"

	"github.com/mirrorru/dot"
	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/txproc"
)

type Table[ROW any] struct {
	tableDef TableDefinition
	sql      sqlTexts
	dialect  dialect.DialectProvider
}

func NewTable[ROW any](dialect dialect.DialectProvider) *Table[ROW] {
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

	result := &Table[ROW]{
		tableDef: tableDef,
		dialect:  dialect,
		sql:      tableDef.makeSQLs(dialect),
	}

	return result
}

func (t *Table[ROW]) Defs() TableDefinition {
	return t.tableDef
}

func (t *Table[ROW]) SQLs() sqlTexts {
	return t.sql
}

func (t *Table[ROW]) Ins(ctx context.Context, tx txproc.TxProcessor, row *ROW) (*ROW, txproc.Result, error) {
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

func (t *Table[ROW]) Upd(ctx context.Context, tx txproc.TxProcessor, row *ROW) (*ROW, txproc.Result, error) {
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

func (t *Table[ROW]) One(ctx context.Context, tx txproc.TxProcessor, keys ...any) (*ROW, error) {
	buf := new(ROW)
	refs := t.tableDef.extractRefs(buf, t.tableDef.Indexes.SelectCols)
	err := tx.QueryRowContext(ctx, t.sql.GetOneCmd, keys...).Scan(refs...)

	return buf, err
}

func (t *Table[ROW]) Del(ctx context.Context, tx txproc.TxProcessor, keys ...any) (txproc.Result, error) {
	sqlResult, err := tx.ExecContext(ctx, t.sql.DeleteCmd, keys...)
	return sqlResult, err
}

func (t *Table[ROW]) Many(ctx context.Context, tx txproc.TxProcessor, filter *Filter) (result []*ROW, err error) {
	var sb strings.Builder
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
