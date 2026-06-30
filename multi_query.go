package qqm

import (
	"context"
	"reflect"

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
	var zero QROW
	rt := reflect.TypeOf(zero)
	for rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}

	qmeta, err := buildQueryMeta[QROW]()
	if err != nil {
		return nil, err
	}

	qmeta.listSQL = buildQueryListSQL(d, qmeta)

	return &Query[QROW]{
		dialect:      d,
		qmeta:        qmeta,
		qrowType:     rt,
		scanTemplate: buildScanTemplate(qmeta),
	}, nil
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

// scanContext хранит подготовленные dest для много-табличного сканирования.
// EN: scanContext holds prepared dest for multi-table scanning.
type scanContext struct {
	dest    []any
	entries []entryScanCtx
}

// entryScanCtx содержит контекст для сканирования одной таблицы в Query.
// EN: entryScanCtx holds scan context for one table in Query.
type entryScanCtx struct {
	isPointer    bool
	fieldStart   int
	pkFieldIdx   []int
	tempAny      []any
	rowMeta      *meta.RowMeta
	fieldIndexes [][]int
	applyFields  []*meta.FieldMeta
}

// buildScanTemplate создаёт шаблон для сканирования строк результата.
// Обрабатывает как обычные поля, так и указатели (*ROW).
// EN: buildScanTemplate creates a template for scanning result rows.
// Handles both regular fields and pointers (*ROW).
func buildScanTemplate(qm *queryMeta) *scanContext {
	sc := &scanContext{}

	for _, entry := range qm.entries {
		ec := entryScanCtx{
			isPointer:  entry.IsPointer,
			fieldStart: len(sc.dest),
			rowMeta:    entry.RowMeta,
		}

		if !entry.IsPointer {
			for _, fm := range entry.RowMeta.Fields {
				if fm.IsOmit {
					continue
				}
				combinedIdx := make([]int, 0, len(fm.Index)+1)
				combinedIdx = append(combinedIdx, entry.FieldIndex)
				combinedIdx = append(combinedIdx, fm.Index...)
				ec.fieldIndexes = append(ec.fieldIndexes, combinedIdx)

				sc.dest = append(sc.dest, nil)
				if fm.IsPK {
					ec.pkFieldIdx = append(ec.pkFieldIdx, len(sc.dest)-1-ec.fieldStart)
				}
			}
		} else {
			nonOmitCount := countNonOmitFields(entry.RowMeta)
			ec.tempAny = make([]any, nonOmitCount)
			for i := 0; i < nonOmitCount; i++ {
				sc.dest = append(sc.dest, &ec.tempAny[i])
			}

			for _, fm := range entry.RowMeta.Fields {
				if fm.IsOmit {
					continue
				}
				ec.applyFields = append(ec.applyFields, fm)
				if fm.IsPK {
					ec.pkFieldIdx = append(ec.pkFieldIdx, len(ec.applyFields)-1)
				}
			}
		}

		sc.entries = append(sc.entries, ec)
	}

	return sc
}

// resetForRow обновляет dest-ы для нового значения QROW.
// EN: resetForRow updates dests for a new QROW value.
func (sc *scanContext) resetForRow(qrow reflect.Value) {
	for ei := range sc.entries {
		ec := &sc.entries[ei]
		if ec.isPointer {
			for i := range ec.tempAny {
				ec.tempAny[i] = nil
			}
		} else {
			for fi := range ec.fieldIndexes {
				fv := qrow.FieldByIndex(ec.fieldIndexes[fi])
				sc.dest[ec.fieldStart+fi] = fv.Addr().Interface()
			}
		}
	}
}

// apply копирует данные из tempAny в QROW для указательных полей.
// Обрабатывает случай NULL: если все PK-значения nil, поле остаётся nil.
// EN: apply copies data from tempAny into QROW for pointer fields.
// Handles NULL case: if all PK values are nil, the field remains nil.
func (sc *scanContext) apply(qm *queryMeta, qrow reflect.Value) {
	for ei, entry := range qm.entries {
		ec := sc.entries[ei]
		if !ec.isPointer {
			continue
		}

		allPKNull := true
		for _, pkIdx := range ec.pkFieldIdx {
			if ec.tempAny[pkIdx] != nil {
				allPKNull = false
				break
			}
		}

		if allPKNull {
			ptrField := qrow.Field(entry.FieldIndex)
			ptrField.Set(reflect.Zero(ptrField.Type()))
		} else {
			allocated := reflect.New(entry.RowType).Elem()
			for fi, fm := range ec.applyFields {
				dst := allocated.FieldByIndex(fm.Index)
				srcVal := reflect.ValueOf(ec.tempAny[fi])
				if srcVal.IsValid() && srcVal.Type().AssignableTo(dst.Type()) {
					dst.Set(srcVal)
				}
			}
			ptrField := qrow.Field(entry.FieldIndex)
			ptrField.Set(allocated.Addr())
		}
	}
}

// countNonOmitFields считает количество не-omit полей в RowMeta.
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
