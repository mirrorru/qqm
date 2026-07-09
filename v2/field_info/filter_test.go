package field_info_test

import (
	"testing"

	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/v2/field_info"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type filterTestRow struct {
	ID   int64  `tbl:"pk;auto"`
	Name string `tbl:"col=user_name"`
	Age  int
}

func TestBuildWhere_NilRoot(t *testing.T) {
	t.Parallel()

	table := field_info.NewTable[filterTestRow](dialect.SQLiteDialect{})
	f := field_info.Filter{Range: nil}
	query, args := f.BuildWhere(table.Defs().Fields, dialect.SQLiteDialect{})
	assert.Empty(t, query)
	assert.Nil(t, args)
}

func TestBuildWhere_SingleCondition_SQLite(t *testing.T) {
	t.Parallel()

	table := field_info.NewTable[filterTestRow](dialect.SQLiteDialect{})
	tf := table.Defs().Fields

	root := field_info.Cond(1, field_info.CmdEq, "Alice")
	f := field_info.Filter{Range: root}

	query, args := f.BuildWhere(tf, dialect.SQLiteDialect{})
	assert.Contains(t, query, "WHERE")
	assert.Contains(t, query, "user_name")
	assert.Contains(t, query, "= ?")
	assert.Equal(t, []any{"Alice"}, args)
}

func TestBuildWhere_SingleCondition_PG(t *testing.T) {
	t.Parallel()

	table := field_info.NewTable[filterTestRow](dialect.PostgreSQLDialect{})
	tf := table.Defs().Fields

	root := field_info.Cond(1, field_info.CmdEq, "Alice")
	f := field_info.Filter{Range: root}

	query, args := f.BuildWhere(tf, dialect.PostgreSQLDialect{})
	assert.Contains(t, query, "WHERE")
	assert.Contains(t, query, "user_name")
	assert.Contains(t, query, "= $1")
	assert.Equal(t, []any{"Alice"}, args)
}

func TestBuildWhere_MultipleAND(t *testing.T) {
	t.Parallel()

	table := field_info.NewTable[filterTestRow](dialect.SQLiteDialect{})
	tf := table.Defs().Fields

	root := field_info.And(
		field_info.Cond(1, field_info.CmdEq, "Alice"),
		field_info.Cond(2, field_info.CmdGte, 25),
	)
	f := field_info.Filter{Range: root}

	query, args := f.BuildWhere(tf, dialect.SQLiteDialect{})
	assert.Contains(t, query, "WHERE")
	assert.Contains(t, query, "user_name")
	assert.Contains(t, query, "age")
	assert.Contains(t, query, " AND ")
	assert.Equal(t, []any{"Alice", 25}, args)
}

func TestBuildWhere_OR(t *testing.T) {
	t.Parallel()

	table := field_info.NewTable[filterTestRow](dialect.SQLiteDialect{})
	tf := table.Defs().Fields

	root := field_info.Or(
		field_info.Cond(1, field_info.CmdEq, "Alice"),
		field_info.Cond(1, field_info.CmdEq, "Bob"),
	)
	f := field_info.Filter{Range: root}

	query, args := f.BuildWhere(tf, dialect.SQLiteDialect{})
	assert.Contains(t, query, "WHERE")
	assert.Contains(t, query, " OR ")
	assert.Equal(t, []any{"Alice", "Bob"}, args)
}

func TestBuildWhere_NOT(t *testing.T) {
	t.Parallel()

	table := field_info.NewTable[filterTestRow](dialect.SQLiteDialect{})
	tf := table.Defs().Fields

	root := field_info.Not(field_info.Cond(2, field_info.CmdEq, 18))
	f := field_info.Filter{Range: root}

	query, args := f.BuildWhere(tf, dialect.SQLiteDialect{})
	assert.Contains(t, query, "WHERE")
	assert.Contains(t, query, "NOT")
	assert.Contains(t, query, "age")
	assert.Equal(t, []any{18}, args)
}

func TestBuildWhere_Nested(t *testing.T) {
	t.Parallel()

	table := field_info.NewTable[filterTestRow](dialect.SQLiteDialect{})
	tf := table.Defs().Fields

	root := field_info.And(
		field_info.Cond(1, field_info.CmdEq, "Alice"),
		field_info.Or(
			field_info.Cond(2, field_info.CmdGt, 20),
			field_info.Cond(2, field_info.CmdLt, 10),
		),
	)
	f := field_info.Filter{Range: root}

	query, args := f.BuildWhere(tf, dialect.SQLiteDialect{})
	assert.Contains(t, query, "WHERE")
	assert.Contains(t, query, "user_name")
	assert.Contains(t, query, "age")
	assert.Contains(t, query, " AND ")
	assert.Contains(t, query, " OR ")
	assert.Equal(t, []any{"Alice", 20, 10}, args)
}

func TestBuildWhere_Operators(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		op         field_info.CommandOp
		value      any
		wantSQL    string
		wantArgs   []any
	}{
		{"Eq", field_info.CmdEq, 42, "= ?", []any{42}},
		{"NotEq", field_info.CmdNotEq, 42, "<> ?", []any{42}},
		{"Gt", field_info.CmdGt, 42, "> ?", []any{42}},
		{"Gte", field_info.CmdGte, 42, ">= ?", []any{42}},
		{"Lt", field_info.CmdLt, 42, "< ?", []any{42}},
		{"Lte", field_info.CmdLte, 42, "<= ?", []any{42}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			table := field_info.NewTable[filterTestRow](dialect.SQLiteDialect{})
			tf := table.Defs().Fields

			root := field_info.Cond(2, tt.op, tt.value)
			f := field_info.Filter{Range: root}

			query, args := f.BuildWhere(tf, dialect.SQLiteDialect{})
			assert.Contains(t, query, tt.wantSQL)
			assert.Equal(t, tt.wantArgs, args)
		})
	}
}

func TestBuildWhere_IsNull_IsNotNull(t *testing.T) {
	t.Parallel()

	t.Run("IsNull", func(t *testing.T) {
		t.Parallel()

		table := field_info.NewTable[filterTestRow](dialect.SQLiteDialect{})
		tf := table.Defs().Fields

		root := field_info.Cond(1, field_info.CmdIsNull, nil)
		f := field_info.Filter{Range: root}

		query, args := f.BuildWhere(tf, dialect.SQLiteDialect{})
		assert.Contains(t, query, "IS NULL")
		assert.Nil(t, args)
	})

	t.Run("IsNotNull", func(t *testing.T) {
		t.Parallel()

		table := field_info.NewTable[filterTestRow](dialect.SQLiteDialect{})
		tf := table.Defs().Fields

		root := field_info.Cond(1, field_info.CmdIsNotNull, nil)
		f := field_info.Filter{Range: root}

		query, args := f.BuildWhere(tf, dialect.SQLiteDialect{})
		assert.Contains(t, query, "IS NOT NULL")
		assert.Nil(t, args)
	})
}

func TestBuildWhere_Like(t *testing.T) {
	t.Parallel()

	table := field_info.NewTable[filterTestRow](dialect.SQLiteDialect{})
	tf := table.Defs().Fields

	root := field_info.Cond(1, field_info.CmdLike, "%Alice%")
	f := field_info.Filter{Range: root}

	query, args := f.BuildWhere(tf, dialect.SQLiteDialect{})
	assert.Contains(t, query, "LIKE ?")
	assert.Equal(t, []any{"%Alice%"}, args)
}

func TestBuildWhere_ILike_PG(t *testing.T) {
	t.Parallel()

	table := field_info.NewTable[filterTestRow](dialect.PostgreSQLDialect{})
	tf := table.Defs().Fields

	root := field_info.Cond(1, field_info.CmdILike, "%Alice%")
	f := field_info.Filter{Range: root}

	query, args := f.BuildWhere(tf, dialect.PostgreSQLDialect{})
	assert.Contains(t, query, "ILIKE")
	assert.Equal(t, []any{"%Alice%"}, args)
}

func TestBuildWhere_ILike_SQLite(t *testing.T) {
	t.Parallel()

	table := field_info.NewTable[filterTestRow](dialect.SQLiteDialect{})
	tf := table.Defs().Fields

	root := field_info.Cond(1, field_info.CmdILike, "%Alice%")
	f := field_info.Filter{Range: root}

	query, args := f.BuildWhere(tf, dialect.SQLiteDialect{})
	assert.Contains(t, query, "LOWER(")
	assert.Contains(t, query, ") LIKE LOWER(")
	assert.Equal(t, []any{"%Alice%"}, args)
}

func TestBuildWhere_In(t *testing.T) {
	t.Parallel()

	table := field_info.NewTable[filterTestRow](dialect.SQLiteDialect{})
	tf := table.Defs().Fields

	root := field_info.Cond(2, field_info.CmdIn, []any{20, 30, 40})
	f := field_info.Filter{Range: root}

	query, args := f.BuildWhere(tf, dialect.SQLiteDialect{})
	assert.Contains(t, query, "IN")
	assert.Contains(t, query, "?, ?, ?")
	assert.Equal(t, []any{20, 30, 40}, args)
}

func TestBuildWhere_PlaceholderContinuity(t *testing.T) {
	t.Parallel()

	table := field_info.NewTable[filterTestRow](dialect.PostgreSQLDialect{})
	tf := table.Defs().Fields

	root := field_info.And(
		field_info.Cond(1, field_info.CmdEq, "Alice"),
		field_info.Cond(2, field_info.CmdGte, 25),
		field_info.Cond(2, field_info.CmdLte, 50),
	)
	f := field_info.Filter{Range: root}

	query, args := f.BuildWhere(tf, dialect.PostgreSQLDialect{})
	assert.Contains(t, query, "$1")
	assert.Contains(t, query, "$2")
	assert.Contains(t, query, "$3")
	assert.NotContains(t, query, "$4")
	assert.Len(t, args, 3)
	assert.Equal(t, []any{"Alice", 25, 50}, args)
}

func TestBuildWhere_OutOfRange(t *testing.T) {
	t.Parallel()

	table := field_info.NewTable[filterTestRow](dialect.SQLiteDialect{})
	tf := table.Defs().Fields

	root := field_info.Cond(99, field_info.CmdEq, "test")
	f := field_info.Filter{Range: root}

	query, args := f.BuildWhere(tf, dialect.SQLiteDialect{})
	assert.Empty(t, query)
	assert.Nil(t, args)
}

func TestCond_HasRequiredMethods(t *testing.T) {
	t.Parallel()

	node := field_info.Cond(0, field_info.CmdEq, "test")
	assert.NotNil(t, node)

	andNode := field_info.And(node)
	assert.NotNil(t, andNode)

	orNode := field_info.Or(node, node)
	assert.NotNil(t, orNode)

	notNode := field_info.Not(node)
	require.NotNil(t, notNode)
}
