package qqm

import (
	"fmt"
	"strings"

	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/meta"
)

// FilterOp определяет оператор комбинации фильтров (AND/OR).
// EN: FilterOp defines the filter combination operator (AND/OR).
type FilterOp int

const (
	// And объединяет фильтры условием AND.
	// EN: And combines filters with AND condition.
	And FilterOp = iota
	// Or объединяет фильтры условием OR.
	// EN: Or combines filters with OR condition.
	Or
)

// ConditionOp определяет оператор сравнения в условии.
// EN: ConditionOp defines the comparison operator in a condition.
type ConditionOp int

const (
	// OpEq — оператор равенства.
	// EN: OpEq — equality operator.
	OpEq ConditionOp = iota
	// OpGt — оператор "больше".
	// EN: OpGt — greater than operator.
	OpGt
	// OpLt — оператор "меньше".
	// EN: OpLt — less than operator.
	OpLt
	// OpGte — оператор "больше или равно".
	// EN: OpGte — greater than or equal operator.
	OpGte
	// OpLte — оператор "меньше или равно".
	// EN: OpLte — less than or equal operator.
	OpLte
	// OpBetween — оператор BETWEEN (для диапазона).
	// EN: OpBetween — BETWEEN operator (for range).
	OpBetween
	// OpIn — оператор IN (для списка значений).
	// EN: OpIn — IN operator (for list of values).
	OpIn
)

// Condition представляет одно условие фильтра (оператор + значение).
// EN: Condition represents a single filter condition (operator + value).
type Condition struct {
	Op    ConditionOp // Оператор сравнения. / EN: Comparison operator.
	Value any         // Значение для сравнения. / EN: Value for comparison.
}

// FieldFilter — фильтр по одному полю с множеством условий.
// EN: FieldFilter is a filter for a single field with multiple conditions.
type FieldFilter struct {
	Field      string      // Имя поля в структуре ROW. / EN: Field name in ROW struct.
	Conditions []Condition // Условия для поля (через AND/OR). / EN: Conditions for the field (via AND/OR).
	Op         FilterOp    // Оператор между условиями. / EN: Operator between conditions.
}

// Filter объединяет фильтры по полям.
// EN: Filter combines field filters.
type Filter struct {
	Fields []FieldFilter // Фильтры по полям. / EN: Filters by fields.
	Op     FilterOp      // Оператор между FieldFilter (AND/OR). / EN: Operator between FieldFilters (AND/OR).
}

// Eq создаёт условие равенства.
// EN: Eq creates an equality condition.
func Eq(value any) Condition {
	return Condition{Op: OpEq, Value: value}
}

// Gt создаёт условие "больше чем".
// EN: Gt creates a "greater than" condition.
func Gt(value any) Condition {
	return Condition{Op: OpGt, Value: value}
}

// Lt создаёт условие "меньше чем".
// EN: Lt creates a "less than" condition.
func Lt(value any) Condition {
	return Condition{Op: OpLt, Value: value}
}

// Gte создаёт условие "больше или равно".
// EN: Gte creates a "greater than or equal" condition.
func Gte(value any) Condition {
	return Condition{Op: OpGte, Value: value}
}

// Lte создаёт условие "меньше или равно".
// EN: Lte creates a "less than or equal" condition.
func Lte(value any) Condition {
	return Condition{Op: OpLte, Value: value}
}

// Between создаёт условие диапазона (min, max).
// EN: Between creates a range condition (min, max).
func Between(min, max any) Condition {
	return Condition{Op: OpBetween, Value: [2]any{min, max}}
}

// In создаёт условие IN для списка значений.
// EN: In creates an IN condition for a list of values.
func In(values ...any) Condition {
	return Condition{Op: OpIn, Value: values}
}

// Field создаёт фильтр для указанного поля с условиями.
// EN: Field creates a filter for the specified field with conditions.
func Field(fieldName string, op FilterOp, conditions ...Condition) FieldFilter {
	return FieldFilter{
		Field:      fieldName,
		Op:         op,
		Conditions: conditions,
	}
}

// AndFilter объединяет фильтры по полям условием AND.
// EN: AndFilter combines field filters with AND condition.
func AndFilter(fields ...FieldFilter) Filter {
	return Filter{
		Fields: fields,
		Op:     And,
	}
}

// OrFilter объединяет фильтры по полям условием OR.
// EN: OrFilter combines field filters with OR condition.
func OrFilter(fields ...FieldFilter) Filter {
	return Filter{
		Fields: fields,
		Op:     Or,
	}
}

// opToSQL преобразует ConditionOp в строку оператора SQL.
// EN: opToSQL converts ConditionOp to SQL operator string.
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

// filterOpJoin возвращает строку-оператор SQL для соединения условий.
// EN: filterOpJoin returns the SQL operator string for joining conditions.
func filterOpJoin(op FilterOp) string {
	if op == Or {
		return " OR "
	}
	return " AND "
}

// whereBuilder строит WHERE-условие для простых табличных запросов.
// EN: whereBuilder builds WHERE clause for simple table queries.
type whereBuilder struct {
	dialect     dialect.DialectProvider
	fields      []*meta.FieldMeta
	fieldByName map[string]*meta.FieldMeta
}

// newWhereBuilder создаёт whereBuilder для списка полей.
// EN: newWhereBuilder creates a whereBuilder for the field list.
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

// findField находит FieldMeta по имени поля.
// EN: findField finds FieldMeta by field name.
func (wb *whereBuilder) findField(fieldName string) (*meta.FieldMeta, error) {
	fm, ok := wb.fieldByName[fieldName]
	if !ok {
		return nil, fmt.Errorf("qqm: unknown field %q in filter", fieldName)
	}
	return fm, nil
}

// buildConditionSQL формирует SQL-условие для одного Condition.
// argIdx — индекс параметра (используется для генерации плейсхолдеров).
// EN: buildConditionSQL builds SQL condition for a single Condition.
// argIdx — parameter index (used for generating placeholders).
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

// joinStrings соединяет строки с разделителем без лишних аллокаций.
// EN: joinStrings joins strings with a separator without extra allocations.
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

// buildFieldFilterSQL формирует SQL-условие для FieldFilter.
// EN: buildFieldFilterSQL builds SQL condition for a FieldFilter.
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

// buildWhereClause формирует WHERE-условие и аргументы из фильтров.
// EN: buildWhereClause builds the WHERE clause and arguments from filters.
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

// multiWhereBuilder строит WHERE для multi-табличных запросов с квалифицированными именами.
// EN: multiWhereBuilder builds WHERE clauses for multi-table queries with qualified names.
type multiWhereBuilder struct {
	dialect dialect.DialectProvider
	qmeta   *queryMeta
}

// findQualifiedField разбирает квалифицированное имя "Order.Amount" на entry и FieldMeta.
// EN: findQualifiedField parses a qualified name "Order.Amount" into entry and FieldMeta.
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

// listEntryNames формирует строку с именами таблиц через запятую.
// EN: listEntryNames builds a comma-separated string of table names.
func listEntryNames(entries []queryTableEntry) string {
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.FieldName
	}
	return strings.Join(names, ", ")
}

// buildConditionSQL формирует SQL-условие с квалифицированным именем колонки.
// EN: buildConditionSQL builds SQL condition with qualified column name.
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

// buildFieldFilterSQL формирует SQL-условие для FieldFilter с квалифицированными именами.
// EN: buildFieldFilterSQL builds SQL condition for FieldFilter with qualified names.
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

// buildWhereClause формирует WHERE-условие для много-табличных запросов.
// EN: buildWhereClause builds WHERE clause for multi-table queries.
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
