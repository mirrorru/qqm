package field_info

type Filter struct {
	Offset uint32
	Limit  uint32
	Range  FilterRange
}

type FilterRange struct {
	OpCondition

	Logic  OpLogic
	Ranges []FilterRange
}

type OpCondition struct {
	FieldIdx int         // Индекс поля в структуре таблицы
	Operator CmpOperator // Логический оператор
	OpArg    any         // Аргумент(ы) операции
}

type OpLogic int32

const (
	OpLogicNone OpLogic = iota
	OpLogicOR
	OpLogicAND
	OpLogicNOT
)

func (ol OpLogic) String() string {
	switch ol {
	case OpLogicOR:
		return "OR"
	case OpLogicAND:
		return "AND"
	case OpLogicNOT:
		return "NOT"
	default:
		return ""
	}
}

type CmpOperator int32

const (
	CmpOperatorNone CmpOperator = iota
	CmpOpEq
	CmpOpNe
	CmpOpLt
	CmpOpLe
	CmpOpGt
	CmpOpGe
	CmpOpIn
	CmpOpNotIn
	CmpOpLike
	CmpOpNotLike
	CmpOpILike
	CmpOpNotILike
	CmpOpIsNull
	CmpOpIsNotNull
)

func (fr *FilterRange) BuildWhere(tf TableFields) string {
	return ""
}
