package qqm

import (
	"fmt"
	"strings"

	"github.com/mirrorru/qqm/defs"
	"github.com/mirrorru/qqm/dialect"
)

type CommandOp int32

const (
	CmdEq CommandOp = iota
	CmdNotEq
	CmdGt
	CmdGte
	CmdLt
	CmdLte
	CmdIsNull
	CmdIsNotNull
	CmdLike
	CmdILike
	CmdIn
)

type LogicOp int32

const (
	LogicAnd LogicOp = iota
	LogicOr
	LogicNot
)

// Filter — структура фильтра для Many().
// Offset и Limit задают пагинацию, Range — дерево условий.
// EN: Filter — filter structure for Many().
// Offset and Limit set pagination, Range — condition tree.
type Filter struct {
	Offset uint32
	Limit  uint32
	Range  FilterNode
}

// FilterNode — интерфейс узла дерева фильтра.
// Реализации: ConditionNode (условие), GroupNode (And/Or/Not-группа).
// EN: FilterNode — filter tree node interface.
// Implementations: ConditionNode (condition), GroupNode (And/Or/Not group).
type FilterNode interface {
	Build(tf TableFields, d dialect.DialectProvider, argIdx *int) (clause string, args []any, err error)
}

// ConditionNode — лист дерева фильтра: одно условие над полем.
// EN: ConditionNode — filter tree leaf: a single condition on a field.
type ConditionNode struct {
	FieldIdx int
	Op       CommandOp
	Value    any
}

// GroupNode — группа условий с логическим оператором.
// EN: GroupNode — group of conditions with a logical operator.
type GroupNode struct {
	Logic    LogicOp
	Children []FilterNode
}

func (f *Filter) BuildOffsetAndLimit(d dialect.DialectProvider) string {
	if f == nil || (f.Offset == 0 && f.Limit == 0) {
		return ""
	}

	return d.OffsetAndLimit(f.Offset, f.Limit)
}

func (f *Filter) BuildWhere(tf TableFields, d dialect.DialectProvider) (query string, args []any, err error) {
	if f == nil || f.Range == nil {
		return "", nil, nil
	}
	argIdx := 1
	clause, args, err := f.Range.Build(tf, d, &argIdx)
	if err != nil || clause == "" {
		return "", nil, err
	}
	return defs.SQLWhere + clause, args, nil
}

func (cn ConditionNode) Build(tf TableFields, d dialect.DialectProvider, argIdx *int) (string, []any, error) {
	if cn.FieldIdx < 0 || cn.FieldIdx >= len(tf) {
		return "", nil, fmt.Errorf("qqm: FieldIdx %d out of range [0, %d)", cn.FieldIdx, len(tf))
	}

	col := tf[cn.FieldIdx].SQLName

	switch cn.Op {
	case CmdEq, CmdNotEq, CmdGt, CmdGte, CmdLt, CmdLte, CmdLike:
		op := cmdOpToSQL(cn.Op)
		p := d.Placeholder(*argIdx)
		*argIdx++
		return col + op + p, []any{cn.Value}, nil

	case CmdIn:
		vals, ok := cn.Value.([]any)
		if !ok {
			return "", nil, fmt.Errorf("qqm: CmdIn requires []any value, got %T", cn.Value)
		}
		placeholders := make([]string, len(vals))
		for i := range vals {
			placeholders[i] = d.Placeholder(*argIdx)
			*argIdx++
		}
		return col + defs.SQLIn + defs.SQLOpenParen +
			strings.Join(placeholders, defs.SQLCommaSpace) +
			defs.SQLCloseParen, vals, nil

	case CmdILike:
		p := d.Placeholder(*argIdx)
		*argIdx++
		return d.ILIKE(col, p), []any{cn.Value}, nil

	case CmdIsNull:
		return col + defs.SQLIsNull, nil, nil

	case CmdIsNotNull:
		return col + defs.SQLIsNotNull, nil, nil
	}

	return "", nil, fmt.Errorf("qqm: unknown CommandOp %d", cn.Op)
}

func (gn GroupNode) Build(tf TableFields, d dialect.DialectProvider, argIdx *int) (string, []any, error) {
	switch gn.Logic {
	case LogicAnd, LogicOr:
		sep := defs.SQLAnd
		if gn.Logic == LogicOr {
			sep = defs.SQLOr
		}
		clauses, args, err := buildChildren(gn.Children, tf, d, argIdx, sep)
		if err != nil {
			return "", nil, err
		}
		if clauses == "" {
			return "", nil, nil
		}
		return defs.SQLOpenParen + clauses + defs.SQLCloseParen, args, nil

	case LogicNot:
		if len(gn.Children) != 1 {
			return "", nil, fmt.Errorf("qqm: LogicNot requires exactly 1 child, got %d", len(gn.Children))
		}
		childClause, childArgs, err := gn.Children[0].Build(tf, d, argIdx)
		if err != nil {
			return "", nil, err
		}
		if childClause == "" {
			return "", nil, nil
		}
		return defs.SQLNot + defs.SQLOpenParen + childClause + defs.SQLCloseParen, childArgs, nil
	}

	return "", nil, fmt.Errorf("qqm: unknown LogicOp %d", gn.Logic)
}

func buildChildren(children []FilterNode, tf TableFields, d dialect.DialectProvider, argIdx *int, separator string) (string, []any, error) {
	if len(children) == 0 {
		return "", nil, nil
	}

	clauses := make([]string, 0, len(children))
	allArgs := make([]any, 0, len(children)*2)

	for _, child := range children {
		clause, args, err := child.Build(tf, d, argIdx)
		if err != nil {
			return "", nil, err
		}
		if clause == "" {
			continue
		}
		clauses = append(clauses, clause)
		allArgs = append(allArgs, args...)
	}

	if len(clauses) == 0 {
		return "", nil, nil
	}

	return strings.Join(clauses, separator), allArgs, nil
}

func cmdOpToSQL(op CommandOp) string {
	switch op {
	case CmdEq:
		return defs.SQLEquals
	case CmdNotEq:
		return " <> "
	case CmdGt:
		return " > "
	case CmdGte:
		return " >= "
	case CmdLt:
		return " < "
	case CmdLte:
		return " <= "
	case CmdLike:
		return defs.SQLLike
	default:
		return defs.SQLEquals
	}
}

// Cond создаёт узел условия: fieldIdx — индекс поля в TableFields, op — оператор, value — значение.
// EN: Cond creates a condition node: fieldIdx — field index in TableFields, op — operator, value — value.
func Cond(fieldIdx int, op CommandOp, value any) *ConditionNode {
	return &ConditionNode{FieldIdx: fieldIdx, Op: op, Value: value}
}

// And создаёт группу с логическим AND.
// EN: And creates a logical AND group.
func And(children ...FilterNode) *GroupNode {
	return &GroupNode{Logic: LogicAnd, Children: children}
}

// Or создаёт группу с логическим OR.
// EN: Or creates a logical OR group.
func Or(children ...FilterNode) *GroupNode {
	return &GroupNode{Logic: LogicOr, Children: children}
}

// Not создаёт группу с логическим NOT (ровно один ребёнок).
// EN: Not creates a logical NOT group (exactly one child).
func Not(child FilterNode) *GroupNode {
	return &GroupNode{Logic: LogicNot, Children: []FilterNode{child}}
}
