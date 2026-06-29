package qqm

import (
	"context"
	"reflect"

	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/meta"
)

type Query[QROW any] struct {
	dialect      dialect.DialectProvider
	qmeta        *queryMeta
	qrowType     reflect.Type
	scanTemplate *scanContext
}

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

type scanContext struct {
	dest    []any
	entries []entryScanCtx
}

type entryScanCtx struct {
	isPointer    bool
	fieldStart   int
	pkFieldIdx   []int
	tempAny      []any
	rowMeta      *meta.RowMeta
	fieldIndexes [][]int
	applyFields  []*meta.FieldMeta
}

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

func countNonOmitFields(rm *meta.RowMeta) int {
	count := 0
	for _, fm := range rm.Fields {
		if !fm.IsOmit {
			count++
		}
	}
	return count
}
