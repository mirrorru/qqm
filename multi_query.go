package qqm

import (
	"context"
	"reflect"
	"strings"

	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/meta"
)

// Query represents a multi-table query.
// QROW is a struct with nested ROW types.
// EN: Query представляет много-табличный запрос.
// QROW — структура с вложенными ROW-типами.
type Query[QROW any] struct {
	dialect      dialect.DialectProvider
	qmeta        *queryMeta
	qrowType     reflect.Type
	scanTemplate *scanContext
}

// NewQuery создаёт Query[QROW] для указанного диалекта.
// QROW должен быть структурой с минимум одним struct-полем.
// EN: NewQuery creates a Query[QROW] for the specified dialect.
// QROW must be a struct with at least one struct field.
func NewQuery[QROW any](d dialect.DialectProvider) (*Query[QROW], error) {
	query, err := NewQueryVal[QROW](d)
	if err != nil {
		return nil, err
	}
	return new(query), nil
}

func NewQueryVal[QROW any](d dialect.DialectProvider) (Query[QROW], error) {
	var zero QROW
	rt := reflect.TypeOf(zero)
	for rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}

	qmeta, err := buildQueryMeta[QROW]()
	if err != nil {
		return Query[QROW]{}, err
	}

	qmeta.listSQL = buildQueryListSQL(d, qmeta)

	return Query[QROW]{
		dialect:      d,
		qmeta:        qmeta,
		qrowType:     rt,
		scanTemplate: buildScanTemplate(qmeta),
	}, nil
}

func (q *Query[QROW]) Internals() *queryMeta {
	return q.qmeta
}

// List выполняет много-табличный запрос и возвращает строки.
// EN: List executes a multi-table query and returns rows.
func (q *Query[QROW]) List(ctx context.Context, ex Executor, filters ...Filter) ([]*QROW, error) {
	query := q.qmeta.listSQL

	var args []any
	if len(filters) > 0 {
		wb := &multiWhereBuilder{
			dialect: q.dialect,
			qmeta:   q.qmeta,
		}
		whereSQL, whereArgs, err := wb.buildWhereClause(filters)
		if err != nil {
			return nil, err
		}
		query += whereSQL
		args = whereArgs
	}

	rows, err := ex.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var result []*QROW
	buf := new(QROW)
	qrowVal := reflect.ValueOf(buf).Elem()
	q.scanTemplate.resetForRow(qrowVal)
	dest := q.scanTemplate.dest
	for rows.Next() {
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}

		q.scanTemplate.apply(q.qmeta, qrowVal)

		row := new(QROW)
		*row = *buf
		result = append(result, row)
	}

	return result, nil
}

// One возвращает одну строку по первичному ключу основной таблицы.
// Основная таблица — первая запись в QROW (алиас t1).
// EN: One returns a single row by the primary key of the main table.
// The main table is the first entry in QROW (alias t1).
func (q *Query[QROW]) One(ctx context.Context, ex Executor, keys ...any) (*QROW, error) {
	primary := q.qmeta.entries[0]
	pkFields := primary.RowMeta.PKFields

	whereClauses := make([]string, len(pkFields))
	for i, pk := range pkFields {
		whereClauses[i] = q.dialect.QuoteIdent(primary.Alias) + "." + q.dialect.QuoteIdent(pk.Column) + sqlEquals + q.dialect.Placeholder(i+1)
	}

	query := q.qmeta.listSQL + sqlWhere + strings.Join(whereClauses, sqlAnd)

	row := ex.QueryRowContext(ctx, query, keys...)

	buf := new(QROW)
	qrowVal := reflect.ValueOf(buf).Elem()
	q.scanTemplate.resetForRow(qrowVal)
	dest := q.scanTemplate.dest

	if err := row.Scan(dest...); err != nil {
		return nil, err
	}

	q.scanTemplate.apply(q.qmeta, qrowVal)

	return buf, nil
}

// scanContext хранит подготовленные dest для много-табличного сканирования.
// EN: scanContext holds prepared dest for multi-table scanning.
type scanContext struct {
	dest    []any
	entries []entryScanCtx
}

// entryScanCtx содержит контекст для сканирования одной таблицы в Query.
// EN: entryScanCtx holds scan context for one table in Query.
type entryScanCtx struct {
	fieldStart   int
	fieldIndexes [][]int
	tempAny      []any
}

// buildScanTemplate создаёт шаблон для сканирования строк результата.
// Обрабатывает как обычные поля, так и указатели (*ROW).
// EN: buildScanTemplate creates a template for scanning result rows.
func buildScanTemplate(qm *queryMeta) *scanContext {
	sc := &scanContext{}

	for _, entry := range qm.entries {
		ec := entryScanCtx{
			fieldStart: len(sc.dest),
		}

		nonOmitCount := countNonOmitFields(entry.RowMeta)
		ec.tempAny = make([]any, nonOmitCount)
		for i := 0; i < nonOmitCount; i++ {
			sc.dest = append(sc.dest, &ec.tempAny[i])
		}

		for _, fm := range entry.RowMeta.Fields {
			if fm.IsOmit {
				continue
			}
			combinedIdx := make([]int, 0, len(fm.Index)+1)
			combinedIdx = append(combinedIdx, entry.FieldIndex)
			combinedIdx = append(combinedIdx, fm.Index...)
			ec.fieldIndexes = append(ec.fieldIndexes, combinedIdx)
		}

		sc.entries = append(sc.entries, ec)
	}

	return sc
}

// resetForRow обновляет dest-ы для нового значения QROW.
// Для pointer-полей переиспользует уже аллоцированные структуры из result slice.
// EN: resetForRow updates dests for a new QROW value.
func (sc *scanContext) resetForRow(qrow reflect.Value) {
	for ei := range sc.entries {
		ec := &sc.entries[ei]
		for i := range ec.tempAny {
			ec.tempAny[i] = nil
		}
	}
}

// apply handles NULL detection for outer joins and copies data to struct fields.
// For outer join entries: if all scanned values are NULL (nil), struct is set to zero value.
// Otherwise: copies data from tempAny to struct fields via fieldIndexes.
// EN: apply handles NULL detection and data copying for each entry.
func (sc *scanContext) apply(qm *queryMeta, qrow reflect.Value) {
	for ei, entry := range qm.entries {
		ec := sc.entries[ei]

		isOuterJoin := entry.JoinType == "LEFT" || entry.JoinType == "RIGHT" || entry.JoinType == "FULL"

		allNull := true
		for _, v := range ec.tempAny {
			if v != nil {
				allNull = false
				break
			}
		}

		if allNull && isOuterJoin {
			ptrField := qrow.Field(entry.FieldIndex)
			ptrField.Set(reflect.Zero(ptrField.Type()))
			continue
		}

		for fi, idx := range ec.fieldIndexes {
			fv := qrow.FieldByIndex(idx)
			srcVal := reflect.ValueOf(ec.tempAny[fi])
			if !srcVal.IsValid() {
				continue
			}
			if srcVal.Type().AssignableTo(fv.Type()) {
				fv.Set(srcVal)
			} else if srcVal.Type().ConvertibleTo(fv.Type()) {
				fv.Set(srcVal.Convert(fv.Type()))
			}
		}
	}
}

// countNonOmitFields counts non-omit fields in RowMeta.
// EN: countNonOmitFields counts non-omit fields in RowMeta.
func countNonOmitFields(rm *meta.RowMeta) int {
	count := 0
	for _, fm := range rm.Fields {
		if !fm.IsOmit {
			count++
		}
	}
	return count
}
