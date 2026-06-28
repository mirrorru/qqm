package table

import (
	"context"
	"reflect"

	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/executor"
	"github.com/mirrorru/qqm/meta"
)

type SQLNamer interface {
	SQLName() string
}

type CRUD[ROW any] interface {
	Internals() *tableInternals

	Insert(ctx context.Context, ex executor.Executor, src ROW) (ROW, error)
	Update(ctx context.Context, ex executor.Executor, src ROW) error
	GetByKey(ctx context.Context, ex executor.Executor, keys ...any) (ROW, error)
	Delete(ctx context.Context, ex executor.Executor, keys ...any) error
	List(ctx context.Context, ex executor.Executor, filters ...Filter) ([]ROW, error)
}

type tableInternals struct {
	dialect dialect.DialectProvider
	meta    *meta.RowMeta
	queries *queryBuilder
}

type Table[ROW any] struct {
	internal tableInternals
	rowType  reflect.Type
}

var _ CRUD[any] = (*Table[any])(nil)

func NewTable[ROW any](d dialect.DialectProvider) *Table[ROW] {
	var zero ROW
	rt := reflect.TypeOf(zero)

	tableName := resolveTableName(rt, zero)
	rm := meta.GetOrBuildRowMeta(rt, tableName)

	elemType := rt
	for elemType.Kind() == reflect.Pointer {
		elemType = elemType.Elem()
	}

	return &Table[ROW]{
		internal: tableInternals{
			dialect: d,
			meta:    rm,
			queries: newQueryBuilder(),
		},
		rowType: elemType,
	}
}

func resolveTableName[ROW any](rt reflect.Type, zero ROW) string {
	base := rt
	for base.Kind() == reflect.Pointer {
		base = base.Elem()
	}

	if namer, ok := any(zero).(SQLNamer); ok {
		return namer.SQLName()
	}

	if rt.Kind() != reflect.Pointer {
		ptrVal := reflect.New(rt)
		if namer, ok := ptrVal.Interface().(SQLNamer); ok {
			return namer.SQLName()
		}
	}

	return meta.ToSnakeCase(base.Name())
}

func (t *Table[ROW]) Internals() *tableInternals {
	return &t.internal
}

func (i *tableInternals) Meta() *meta.RowMeta {
	return i.meta
}

func (i *tableInternals) Dialect() dialect.DialectProvider {
	return i.dialect
}

func (i *tableInternals) InsertSQL() string {
	return i.queries.InsertSQL(i.dialect, i.meta)
}

func (i *tableInternals) UpdateSQL() string {
	return i.queries.UpdateSQL(i.dialect, i.meta)
}

func (i *tableInternals) SelectSQL() string {
	return i.queries.SelectSQL(i.dialect, i.meta)
}

func (i *tableInternals) DeleteSQL() string {
	return i.queries.DeleteSQL(i.dialect, i.meta)
}

func (i *tableInternals) ListSQL() string {
	return i.queries.ListSQL(i.dialect, i.meta)
}

func (t *Table[ROW]) Insert(ctx context.Context, ex executor.Executor, src ROW) (ROW, error) {
	args := t.internal.meta.InsertValues(src)

	if t.internal.dialect.SupportsReturning() {
		row := ex.QueryRowContext(ctx, t.internal.InsertSQL(), args...)
		result := t.newRow()
		dest := t.internal.meta.ScanDest(result)
		if err := row.Scan(dest...); err != nil {
			var zero ROW
			return zero, err
		}
		return result, nil
	}

	_, err := ex.ExecContext(ctx, t.internal.InsertSQL(), args...)
	var zero ROW
	return zero, err
}

func (t *Table[ROW]) Update(ctx context.Context, ex executor.Executor, src ROW) error {
	updateVals := t.internal.meta.UpdateValues(src)
	pkVals := t.internal.meta.PKFieldValues(src)
	args := append(updateVals, pkVals...)

	_, err := ex.ExecContext(ctx, t.internal.UpdateSQL(), args...)
	return err
}

func (t *Table[ROW]) GetByKey(ctx context.Context, ex executor.Executor, keys ...any) (ROW, error) {
	row := ex.QueryRowContext(ctx, t.internal.SelectSQL(), keys...)

	result := t.newRow()
	dest := t.internal.meta.ScanDest(result)
	if err := row.Scan(dest...); err != nil {
		var zero ROW
		return zero, err
	}
	return result, nil
}

func (t *Table[ROW]) Delete(ctx context.Context, ex executor.Executor, keys ...any) error {
	_, err := ex.ExecContext(ctx, t.internal.DeleteSQL(), keys...)
	return err
}

func (t *Table[ROW]) List(ctx context.Context, ex executor.Executor, filters ...Filter) ([]ROW, error) {
	if len(filters) == 0 {
		rows, err := ex.QueryContext(ctx, t.internal.ListSQL())
		if err != nil {
			return nil, err
		}
		defer func() { _ = rows.Close() }()

		var result []ROW
		for rows.Next() {
			row := t.newRow()
			dest := t.internal.meta.ScanDest(row)
			if err := rows.Scan(dest...); err != nil {
				return nil, err
			}
			result = append(result, row)
		}
		return result, nil
	}

	sql, args, err := t.buildFilterWhereClause(filters)
	if err != nil {
		return nil, err
	}

	query := t.internal.ListSQL() + sql
	rows, err := ex.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var result []ROW
	for rows.Next() {
		row := t.newRow()
		dest := t.internal.meta.ScanDest(row)
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, nil
}

func (t *Table[ROW]) newRow() ROW {
	v := reflect.New(t.rowType)
	return v.Interface().(ROW)
}

func (t *Table[ROW]) buildFilterWhereClause(filters []Filter) (string, []any, error) {
	wb := &whereBuilder{
		dialect: t.internal.dialect,
		fields:  t.internal.meta.Fields,
	}
	return wb.buildWhereClause(filters)
}
