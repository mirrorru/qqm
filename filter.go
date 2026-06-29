package qqm

import (
	"fmt"
	"strings"

	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/meta"
)

type FilterOp int

const (
	And FilterOp = iota
	Or
)

type ConditionOp int

const (
	OpEq ConditionOp = iota
	OpGt
	OpLt
	OpGte
	OpLte
	OpBetween
	OpIn
)

type Condition struct {
	Op    ConditionOp
	Value any
}

type FieldFilter struct {
	Field      string
	Conditions []Condition
	Op         FilterOp
}

type Filter struct {
	Fields []FieldFilter
	Op     FilterOp
}

func Eq(value any) Condition {
	return Condition{Op: OpEq, Value: value}
}

func Gt(value any) Condition {
	return Condition{Op: OpGt, Value: value}
}

func Lt(value any) Condition {
	return Condition{Op: OpLt, Value: value}
}

func Gte(value any) Condition {
	return Condition{Op: OpGte, Value: value}
}

func Lte(value any) Condition {
	return Condition{Op: OpLte, Value: value}
}

func Between(min, max any) Condition {
	return Condition{Op: OpBetween, Value: [2]any{min, max}}
}

func In(values ...any) Condition {
	return Condition{Op: OpIn, Value: values}
}

func Field(fieldName string, op FilterOp, conditions ...Condition) FieldFilter {
	return FieldFilter{
		Field:      fieldName,
		Op:         op,
		Conditions: conditions,
	}
}

func AndFilter(fields ...FieldFilter) Filter {
	return Filter{
		Fields: fields,
		Op:     And,
	}
}

func OrFilter(fields ...FieldFilter) Filter {
	return Filter{
		Fields: fields,
		Op:     Or,
	}
}

func opToSQL(op ConditionOp) string {
	switch op {
	case OpEq:
		return "="
	case OpGt:
		return ">"
	case OpLt:
		return "<"
	case OpGte:
		return ">="
	case OpLte:
		return "<="
	case OpBetween:
		return "BETWEEN"
	case OpIn:
		return "IN"
	default:
		return "="
	}
}

func filterOpJoin(op FilterOp) string {
	if op == Or {
		return " OR "
	}
	return " AND "
}

type whereBuilder struct {
	dialect     dialect.DialectProvider
	fields      []*meta.FieldMeta
	fieldByName map[string]*meta.FieldMeta
}

func newWhereBuilder(d dialect.DialectProvider, fields []*meta.FieldMeta) *whereBuilder {
	m := make(map[string]*meta.FieldMeta, len(fields))
	for _, f := range fields {
		m[f.Name] = f
	}
	return &whereBuilder{
		dialect:     d,
		fields:      fields,
		fieldByName: m,
	}
}

func (wb *whereBuilder) findField(fieldName string) (*meta.FieldMeta, error) {
	fm, ok := wb.fieldByName[fieldName]
	if !ok {
		return nil, fmt.Errorf("qqm: unknown field %q in filter", fieldName)
	}
	return fm, nil
}

func (wb *whereBuilder) buildConditionSQL(column string, cond Condition, argIdx *int) (string, []any, error) {
	col := wb.dialect.QuoteIdent(column)

	switch cond.Op {
	case OpBetween:
		pair, ok := cond.Value.([2]any)
		if !ok {
			return "", nil, fmt.Errorf("qqm: Between requires [2]any value, got %T", cond.Value)
		}
		p1 := wb.dialect.Placeholder(*argIdx)
		*argIdx++
		p2 := wb.dialect.Placeholder(*argIdx)
		*argIdx++
		return col + " BETWEEN " + p1 + " AND " + p2, []any{pair[0], pair[1]}, nil

	case OpIn:
		vals, ok := cond.Value.([]any)
		if !ok {
			return "", nil, fmt.Errorf("qqm: In requires []any value, got %T", cond.Value)
		}
		placeholders := make([]string, len(vals))
		for i := range vals {
			placeholders[i] = wb.dialect.Placeholder(*argIdx)
			*argIdx++
		}
		return col + sqlIn + sqlOpenParen + joinStrings(placeholders, sqlCommaSpace) + sqlCloseParen, vals, nil

	default:
		p := wb.dialect.Placeholder(*argIdx)
		*argIdx++
		return col + " " + opToSQL(cond.Op) + " " + p, []any{cond.Value}, nil
	}
}

func joinStrings(elems []string, sep string) string {
	if len(elems) == 0 {
		return ""
	}
	n := len(sep) * (len(elems) - 1)
	for _, e := range elems {
		n += len(e)
	}
	b := make([]byte, 0, n)
	for i, e := range elems {
		if i > 0 {
			b = append(b, sep...)
		}
		b = append(b, e...)
	}
	return string(b)
}

func (wb *whereBuilder) buildFieldFilterSQL(ff FieldFilter, argIdx *int) (string, []any, error) {
	fm, err := wb.findField(ff.Field)
	if err != nil {
		return "", nil, err
	}

	if len(ff.Conditions) == 0 {
		return "", nil, nil
	}

	var clauses []string
	var args []any

	for _, cond := range ff.Conditions {
		clause, condArgs, err := wb.buildConditionSQL(fm.Column, cond, argIdx)
		if err != nil {
			return "", nil, err
		}
		clauses = append(clauses, clause)
		args = append(args, condArgs...)
	}

	joinOp := filterOpJoin(ff.Op)
	combined := joinStrings(clauses, joinOp)
	return sqlOpenParen + combined + sqlCloseParen, args, nil
}

func (wb *whereBuilder) buildWhereClause(filters []Filter) (string, []any, error) {
	if len(filters) == 0 {
		return "", nil, nil
	}

	var allClauses []string
	var allArgs []any

	for _, f := range filters {
		argStart := len(allArgs) + 1
		for _, ff := range f.Fields {
			clause, fieldArgs, err := wb.buildFieldFilterSQL(ff, &argStart)
			if err != nil {
				return "", nil, err
			}
			if clause == "" {
				continue
			}
			allClauses = append(allClauses, clause)
			allArgs = append(allArgs, fieldArgs...)
		}
	}

	if len(allClauses) == 0 {
		return "", nil, nil
	}

	joinOp := filterOpJoin(filters[0].Op)
	combined := joinStrings(allClauses, joinOp)
	return sqlWhere + combined, allArgs, nil
}

// multiWhereBuilder — построитель WHERE для multi-табличных запросов с квалифицированными именами.
type multiWhereBuilder struct {
	dialect dialect.DialectProvider
	qmeta   *queryMeta
}

// findQualifiedField разбирает квалифицированное имя "Order.Amount" на entry и FieldMeta.
func (wb *multiWhereBuilder) findQualifiedField(fieldName string) (string, *meta.FieldMeta, error) {
	dot := strings.IndexByte(fieldName, '.')
	if dot < 0 {
		return "", nil, fmt.Errorf("qqm: qualified field name required in multi-table query, got %q (use \"TableName.FieldName\")", fieldName)
	}

	entryName := fieldName[:dot]
	fieldPart := fieldName[dot+1:]

	entry := wb.qmeta.findEntryByFieldName(entryName)
	if entry == nil {
		return "", nil, fmt.Errorf("qqm: unknown table %q in filter (valid: %s)", entryName, listEntryNames(wb.qmeta.entries))
	}

	for _, fm := range entry.RowMeta.Fields {
		if fm.Name == fieldPart {
			return entry.Alias, fm, nil
		}
	}

	return "", nil, fmt.Errorf("qqm: unknown field %q in table %q", fieldPart, entryName)
}

func listEntryNames(entries []queryTableEntry) string {
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.FieldName
	}
	return strings.Join(names, ", ")
}

func (wb *multiWhereBuilder) buildConditionSQL(alias, column string, cond Condition, argIdx *int) (string, []any, error) {
	col := wb.dialect.QuoteIdent(alias) + "." + wb.dialect.QuoteIdent(column)

	switch cond.Op {
	case OpBetween:
		pair, ok := cond.Value.([2]any)
		if !ok {
			return "", nil, fmt.Errorf("qqm: Between requires [2]any value, got %T", cond.Value)
		}
		p1 := wb.dialect.Placeholder(*argIdx)
		*argIdx++
		p2 := wb.dialect.Placeholder(*argIdx)
		*argIdx++
		return col + " BETWEEN " + p1 + " AND " + p2, []any{pair[0], pair[1]}, nil

	case OpIn:
		vals, ok := cond.Value.([]any)
		if !ok {
			return "", nil, fmt.Errorf("qqm: In requires []any value, got %T", cond.Value)
		}
		placeholders := make([]string, len(vals))
		for i := range vals {
			placeholders[i] = wb.dialect.Placeholder(*argIdx)
			*argIdx++
		}
		return col + sqlIn + sqlOpenParen + joinStrings(placeholders, sqlCommaSpace) + sqlCloseParen, vals, nil

	default:
		p := wb.dialect.Placeholder(*argIdx)
		*argIdx++
		return col + " " + opToSQL(cond.Op) + " " + p, []any{cond.Value}, nil
	}
}

func (wb *multiWhereBuilder) buildFieldFilterSQL(ff FieldFilter, argIdx *int) (string, []any, error) {
	alias, fm, err := wb.findQualifiedField(ff.Field)
	if err != nil {
		return "", nil, err
	}

	if len(ff.Conditions) == 0 {
		return "", nil, nil
	}

	var clauses []string
	var args []any

	for _, cond := range ff.Conditions {
		clause, condArgs, err := wb.buildConditionSQL(alias, fm.Column, cond, argIdx)
		if err != nil {
			return "", nil, err
		}
		clauses = append(clauses, clause)
		args = append(args, condArgs...)
	}

	joinOp := filterOpJoin(ff.Op)
	combined := joinStrings(clauses, joinOp)
	return sqlOpenParen + combined + sqlCloseParen, args, nil
}

func (wb *multiWhereBuilder) buildWhereClause(filters []Filter) (string, []any, error) {
	if len(filters) == 0 {
		return "", nil, nil
	}

	var allClauses []string
	var allArgs []any

	for _, f := range filters {
		argStart := len(allArgs) + 1
		for _, ff := range f.Fields {
			clause, fieldArgs, err := wb.buildFieldFilterSQL(ff, &argStart)
			if err != nil {
				return "", nil, err
			}
			if clause == "" {
				continue
			}
			allClauses = append(allClauses, clause)
			allArgs = append(allArgs, fieldArgs...)
		}
	}

	if len(allClauses) == 0 {
		return "", nil, nil
	}

	joinOp := filterOpJoin(filters[0].Op)
	combined := joinStrings(allClauses, joinOp)
	return sqlWhere + combined, allArgs, nil
}
