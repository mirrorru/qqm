package field_info

import (
	"fmt"
	"strings"

	"github.com/mirrorru/qqm/defs"
	"github.com/mirrorru/qqm/dialect"
)

type CommandOp int32

const (
	CmdEq     CommandOp = iota
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

type Filter struct {
	Offset uint32
	Limit  uint32
	Range  FilterNode
}

type FilterNode interface {
	Build(tf TableFields, d dialect.DialectProvider, argIdx *int) (clause string, args []any, err error)
}

type ConditionNode struct {
	FieldIdx int
	Op       CommandOp
	Value    any
}

type GroupNode struct {
	Logic    LogicOp
	Children []FilterNode
}

func (f *Filter) BuildWhere(tf TableFields, d dialect.DialectProvider) (query string, args []any) {
	if f.Range == nil {
		return "", nil
	}
	argIdx := 1
	clause, args, err := f.Range.Build(tf, d, &argIdx)
	if err != nil || clause == "" {
		return "", nil
	}
	return defs.SQLWhere + clause, args
}

func (cn *ConditionNode) Build(tf TableFields, d dialect.DialectProvider, argIdx *int) (string, []any, error) {
	if cn.FieldIdx < 0 || cn.FieldIdx >= len(tf) {
		return "", nil, fmt.Errorf("field_info: FieldIdx %d out of range [0, %d)", cn.FieldIdx, len(tf))
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
			return "", nil, fmt.Errorf("field_info: CmdIn requires []any value, got %T", cn.Value)
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
		return col + " IS NULL", nil, nil

	case CmdIsNotNull:
		return col + " IS NOT NULL", nil, nil
	}

	return "", nil, fmt.Errorf("field_info: unknown CommandOp %d", cn.Op)
}

func (gn *GroupNode) Build(tf TableFields, d dialect.DialectProvider, argIdx *int) (string, []any, error) {
	switch gn.Logic {
	case LogicAnd, LogicOr:
		sep := " AND "
		if gn.Logic == LogicOr {
			sep = " OR "
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
			return "", nil, fmt.Errorf("field_info: LogicNot requires exactly 1 child, got %d", len(gn.Children))
		}
		childClause, childArgs, err := gn.Children[0].Build(tf, d, argIdx)
		if err != nil {
			return "", nil, err
		}
		if childClause == "" {
			return "", nil, nil
		}
		return "NOT " + defs.SQLOpenParen + childClause + defs.SQLCloseParen, childArgs, nil
	}

	return "", nil, fmt.Errorf("field_info: unknown LogicOp %d", gn.Logic)
}

func buildChildren(children []FilterNode, tf TableFields, d dialect.DialectProvider, argIdx *int, separator string) (string, []any, error) {
	if len(children) == 0 {
		return "", nil, nil
	}

	var clauses []string
	var allArgs []any

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
		return " LIKE "
	default:
		return defs.SQLEquals
	}
}

func Cond(fieldIdx int, op CommandOp, value any) *ConditionNode {
	return &ConditionNode{FieldIdx: fieldIdx, Op: op, Value: value}
}

func And(children ...FilterNode) *GroupNode {
	return &GroupNode{Logic: LogicAnd, Children: children}
}

func Or(children ...FilterNode) *GroupNode {
	return &GroupNode{Logic: LogicOr, Children: children}
}

func Not(child FilterNode) *GroupNode {
	return &GroupNode{Logic: LogicNot, Children: []FilterNode{child}}
}
