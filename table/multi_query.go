package table

import (
	"context"
	"reflect"

	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/executor"
	"github.com/mirrorru/qqm/meta"
)

// Query — типизированный запрос для SELECT с JOIN по нескольким таблицам.
type Query[QROW any] struct {
	dialect  dialect.DialectProvider
	qmeta    *queryMeta
	qrowType reflect.Type
}

// NewQuery создаёт новый Query для QROW.
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
		dialect:  d,
		qmeta:    qmeta,
		qrowType: rt,
	}, nil
}

// List выполняет SELECT с JOIN и опциональными фильтрами.
func (q *Query[QROW]) List(ctx context.Context, ex executor.Executor, filters ...Filter) ([]QROW, error) {
	query := q.qmeta.listSQL

	var args []any
	if len(filters) > 0 {
		wb := &multiWhereBuilder{
			dialect: q.dialect,
			qmeta:   q.qmeta,
		}
		whereSQL, whereArgs, err := wb.buildWhereClause(filters)
		if err != nil {
			var zero []QROW
			return zero, err
		}
		query += whereSQL
		args = whereArgs
	}

	rows, err := ex.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var result []QROW
	for rows.Next() {
		qrow := reflect.New(q.qrowType).Elem()

		sc := newScanContext(q.qmeta, qrow)
		if err := rows.Scan(sc.dest...); err != nil {
			return nil, err
		}

		sc.apply(q.qmeta, qrow)

		result = append(result, qrow.Interface().(QROW))
	}

	return result, nil
}

// scanContext хранит flat scan destination и временные значения для pointer-полей.
type scanContext struct {
	dest    []any
	entries []entryScanCtx
}

// entryScanCtx — контекст сканирования для одной таблицы.
type entryScanCtx struct {
	isPointer  bool
	fieldStart int
	pkFieldIdx []int // индексы PK-полей в entry.RowMeta.Fields
	tempAny    []any // для pointer: временные any-переменные (указатели на них в dest)
	rowMeta    *meta.RowMeta
}

// newScanContext создаёт scanContext для QROW.
func newScanContext(qm *queryMeta, qrow reflect.Value) *scanContext {
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
				fv := qrow.FieldByIndex(combinedIdx)
				sc.dest = append(sc.dest, fv.Addr().Interface())
				if fm.IsPK {
					ec.pkFieldIdx = append(ec.pkFieldIdx, len(sc.dest)-1-ec.fieldStart)
				}
			}
		} else {
			nonOmitCount := countNonOmitFields(entry.RowMeta)
			tempSlice := make([]any, nonOmitCount)
			ec.tempAny = tempSlice
			tempIdx := 0
			for _, fm := range entry.RowMeta.Fields {
				if fm.IsOmit {
					continue
				}
				sc.dest = append(sc.dest, &tempSlice[tempIdx])
				if fm.IsPK {
					ec.pkFieldIdx = append(ec.pkFieldIdx, tempIdx)
				}
				tempIdx++
			}
		}

		sc.entries = append(sc.entries, ec)
	}

	return sc
}

// apply обрабатывает результаты сканирования и устанавливает значения в QROW.
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
			tempIdx := 0
			for _, fm := range entry.RowMeta.Fields {
				if fm.IsOmit {
					continue
				}
				dst := allocated.FieldByIndex(fm.Index)
				srcVal := reflect.ValueOf(ec.tempAny[tempIdx])
				if srcVal.IsValid() && srcVal.Type().AssignableTo(dst.Type()) {
					dst.Set(srcVal)
				}
				tempIdx++
			}
			ptrField := qrow.Field(entry.FieldIndex)
			ptrField.Set(allocated.Addr())
		}
	}
}

// countNonOmitFields подсчитывает количество не-omit полей в RowMeta.
func countNonOmitFields(rm *meta.RowMeta) int {
	count := 0
	for _, fm := range rm.Fields {
		if !fm.IsOmit {
			count++
		}
	}
	return count
}
