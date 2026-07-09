package test_structs

// LogicalOperator определяет, как объединяются условия (AND/OR)
type LogicalOperator string

const (
	OpAnd LogicalOperator = "AND"
	OpOr  LogicalOperator = "OR"
)

// ComparisonOperator определяет математическую операцию сравнения
type ComparisonOperator string

const (
	OpEq      ComparisonOperator = "="
	OpNotEq   ComparisonOperator = "!="
	OpGt      ComparisonOperator = ">"
	OpGte     ComparisonOperator = ">="
	OpLt      ComparisonOperator = "<"
	OpLte     ComparisonOperator = "<="
	OpLike    ComparisonOperator = "LIKE"
	OpILike   ComparisonOperator = "ILIKE" // Для PostgreSQL (регистронезависимый поиск)
	OpIn      ComparisonOperator = "IN"
	OpNotIn   ComparisonOperator = "NOT IN"
	OpIsNull  ComparisonOperator = "IS NULL"
	OpNotNull ComparisonOperator = "IS NOT NULL"
)

// Condition представляет собой атомарное условие фильтрации (например: age > 18)
type Condition struct {
	Field    string             `json:"field"`           // Имя колонки/поля
	Operator ComparisonOperator `json:"operator"`        // Оператор сравнения (=, LIKE, IN...)
	Value    interface{}        `json:"value,omitempty"` // Значение (может быть string, int, []interface{} и т.д.)
}

// FilterRule — это узел дерева. Он может содержать либо конкретное атомарное условие,
// либо группу вложенных фильтров, объединенных логикой AND/OR.
type FilterRule struct {
	Logic   LogicalOperator `json:"logic,omitempty"`   // AND или OR (заполняется, если это группа)
	Filters []FilterRule    `json:"filters,omitempty"` // Вложенные правила (для создания дерева)

	// Встраиваем атомарное условие напрямую. Если Logic пустое,
	// значит этот узел является "листом" дерева и представляет собой конкретное условие.
	Condition
}
