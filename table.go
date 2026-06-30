package qqm

import (
	"context"
	"reflect"

	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/meta"
)

type SQLNamer interface {
	SQLName() string
}

type CRUD[ROW any] interface {
	Internals() *tableInternals

	Insert(ctx context.Context, ex Executor, src *ROW) (*ROW, error)
	Update(ctx context.Context, ex Executor, src *ROW) error
	GetByPK(ctx context.Context, ex Executor, keys ...any) (*ROW, error)
	Delete(ctx context.Context, ex Executor, keys ...any) error
	List(ctx context.Context, ex Executor, filters ...Filter) ([]*ROW, error)
}

type tableInternals struct {
	dialect      dialect.DialectProvider
	meta         *meta.RowMeta
	queries      *queryBuilder
	whereBuilder *whereBuilder
	scanHelper   *scanDestHelper
}

type scanDestHelper struct {
	fieldIndexes [][]int
	dest         []any
}

func newScanDestHelper(rm *meta.RowMeta) *scanDestHelper {
	var indexes [][]int
	for _, fm := range rm.Fields {
		if fm.IsOmit {
			continue
		}
		idx := make([]int, len(fm.Index))
		copy(idx, fm.Index)
		indexes = append(indexes, idx)
	}
	return &scanDestHelper{
		fieldIndexes: indexes,
		dest:         make([]any, len(indexes)),
	}
}

func (h *scanDestHelper) resetForRow(rowVal reflect.Value) []any {
	for i, idx := range h.fieldIndexes {
		fv := rowVal.FieldByIndex(idx)
		h.dest[i] = fv.Addr().Interface()
	}
	return h.dest
}

type Table[ROW any] struct {
	internal tableInternals
	rowType  reflect.Type
}

func NewTable[ROW any](d dialect.DialectProvider) *Table[ROW] {
	var zero ROW
	rt := reflect.TypeOf(zero)

	if rt.Kind() == reflect.Pointer {
		panic("qqm: ROW must not be a pointer type, use struct value")
	}

	tableName := resolveTableName(rt, zero)
	rm := meta.GetOrBuildRowMeta(rt, tableName)

	elemType := rt
	for elemType.Kind() == reflect.Pointer {
		elemType = elemType.Elem()
	}

	return &Table[ROW]{
		internal: tableInternals{
			dialect:      d,
			meta:         rm,
			queries:      newQueryBuilder(),
			whereBuilder: newWhereBuilder(d, rm.Fields),
			scanHelper:   newScanDestHelper(rm),
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

func (i *tableInternals) CreateTableSQL() string {
	return i.queries.CreateTableSQL(i.dialect, i.meta)
}

func (t *Table[ROW]) Insert(ctx context.Context, ex Executor, src *ROW) (*ROW, error) {
	args := t.internal.meta.InsertValues(src)

	if t.internal.dialect.SupportsReturning() {
		row := ex.QueryRowContext(ctx, t.internal.InsertSQL(), args...)
		buf := new(ROW)
		dest := t.internal.scanHelper.resetForRow(t.rowValue(buf))
		if err := row.Scan(dest...); err != nil {
			return nil, err
		}
		result := new(ROW)
		*result = *buf
		return result, nil
	}

	_, err := ex.ExecContext(ctx, t.internal.InsertSQL(), args...)
	return nil, err
}

func (t *Table[ROW]) Update(ctx context.Context, ex Executor, src *ROW) error {
	updateVals := t.internal.meta.UpdateValues(src)
	pkVals := t.internal.meta.PKFieldValues(src)
	args := append(updateVals, pkVals...)

	_, err := ex.ExecContext(ctx, t.internal.UpdateSQL(), args...)
	return err
}

func (t *Table[ROW]) GetByPK(ctx context.Context, ex Executor, keys ...any) (*ROW, error) {
	row := ex.QueryRowContext(ctx, t.internal.SelectSQL(), keys...)

	buf := new(ROW)
	dest := t.internal.scanHelper.resetForRow(t.rowValue(buf))
	if err := row.Scan(dest...); err != nil {
		return nil, err
	}
	result := new(ROW)
	*result = *buf
	return result, nil
}

func (t *Table[ROW]) Delete(ctx context.Context, ex Executor, keys ...any) error {
	_, err := ex.ExecContext(ctx, t.internal.DeleteSQL(), keys...)
	return err
}

func (t *Table[ROW]) List(ctx context.Context, ex Executor, filters ...Filter) ([]*ROW, error) {
	if len(filters) == 0 {
		rows, err := ex.QueryContext(ctx, t.internal.ListSQL())
		if err != nil {
			return nil, err
		}
		defer func() { _ = rows.Close() }()

		var result []*ROW
		buf := new(ROW)
		dest := t.internal.scanHelper.resetForRow(t.rowValue(buf))
		for rows.Next() {
			if err := rows.Scan(dest...); err != nil {
				return nil, err
			}
			row := new(ROW)
			*row = *buf
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

	var result []*ROW
	for rows.Next() {
		buf := new(ROW)
		dest := t.internal.scanHelper.resetForRow(t.rowValue(buf))
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		row := new(ROW)
		*row = *buf
		result = append(result, row)
	}
	return result, nil
}

func (t *Table[ROW]) rowValue(row *ROW) reflect.Value {
	return reflect.ValueOf(row).Elem()
}

func (t *Table[ROW]) buildFilterWhereClause(filters []Filter) (string, []any, error) {
	return t.internal.whereBuilder.buildWhereClause(filters)
}
