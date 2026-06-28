package qqm

import (
	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/table"
)

var (
	SQLiteDialect     = dialect.SQLiteDialect{}
	PostgreSQLDialect = dialect.PostgreSQLDialect{}
)

// FilterOp
type FilterOp = table.FilterOp

const (
	And FilterOp = iota
	Or
)

// ConditionOp
type ConditionOp = table.ConditionOp

// Condition
type Condition = table.Condition

// FieldFilter
type FieldFilter = table.FieldFilter

// Filter
type Filter = table.Filter

func NewTable[ROW any](d dialect.DialectProvider) *table.Table[ROW] {
	return table.NewTable[ROW](d)
}

func Eq(value any) Condition {
	return table.Eq(value)
}

func Gt(value any) Condition {
	return table.Gt(value)
}

func Lt(value any) Condition {
	return table.Lt(value)
}

func Gte(value any) Condition {
	return table.Gte(value)
}

func Lte(value any) Condition {
	return table.Lte(value)
}

func Between(min, max any) Condition {
	return table.Between(min, max)
}

func In(values ...any) Condition {
	return table.In(values...)
}

func Field(fieldName string, op FilterOp, conditions ...Condition) FieldFilter {
	return table.Field(fieldName, op, conditions...)
}

func AndFilter(fields ...FieldFilter) Filter {
	return table.AndFilter(fields...)
}

func OrFilter(fields ...FieldFilter) Filter {
	return table.OrFilter(fields...)
}
