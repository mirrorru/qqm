package qqm_test

import (
	"testing"

	"github.com/mirrorru/qqm"
	"github.com/mirrorru/qqm/dialect"
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

	table := qqm.NewTable[filterTestRow](dialect.SQLiteDialect{})
	f := qqm.Filter{Range: nil}
	query, args, err := f.BuildWhere(table.Defs().Fields, dialect.SQLiteDialect{})
	require.NoError(t, err)
	assert.Empty(t, query)
	assert.Nil(t, args)
}

func TestBuildWhere_SingleCondition_SQLite(t *testing.T) {
	t.Parallel()

	table := qqm.NewTable[filterTestRow](dialect.SQLiteDialect{})
	tf := table.Defs().Fields

	root := qqm.Cond(1, qqm.CmdEq, "Alice")
	f := qqm.Filter{Range: root}

	query, args, err := f.BuildWhere(tf, dialect.SQLiteDialect{})
	require.NoError(t, err)
	assert.Contains(t, query, "WHERE")
	assert.Contains(t, query, "user_name")
	assert.Contains(t, query, "= ?")
	assert.Equal(t, []any{"Alice"}, args)
}

func TestBuildWhere_SingleCondition_PG(t *testing.T) {
	t.Parallel()

	table := qqm.NewTable[filterTestRow](dialect.PostgreSQLDialect{})
	tf := table.Defs().Fields

	root := qqm.Cond(1, qqm.CmdEq, "Alice")
	f := qqm.Filter{Range: root}

	query, args, err := f.BuildWhere(tf, dialect.PostgreSQLDialect{})
	require.NoError(t, err)
	assert.Contains(t, query, "WHERE")
	assert.Contains(t, query, "user_name")
	assert.Contains(t, query, "= $1")
	assert.Equal(t, []any{"Alice"}, args)
}

func TestBuildWhere_MultipleAND(t *testing.T) {
	t.Parallel()

	table := qqm.NewTable[filterTestRow](dialect.SQLiteDialect{})
	tf := table.Defs().Fields

	root := qqm.And(
		qqm.Cond(1, qqm.CmdEq, "Alice"),
		qqm.Cond(2, qqm.CmdGte, 25),
	)
	f := qqm.Filter{Range: root}

	query, args, err := f.BuildWhere(tf, dialect.SQLiteDialect{})
	require.NoError(t, err)
	assert.Contains(t, query, "WHERE")
	assert.Contains(t, query, "user_name")
	assert.Contains(t, query, "age")
	assert.Contains(t, query, " AND ")
	assert.Equal(t, []any{"Alice", 25}, args)
}

func TestBuildWhere_OR(t *testing.T) {
	t.Parallel()

	table := qqm.NewTable[filterTestRow](dialect.SQLiteDialect{})
	tf := table.Defs().Fields

	root := qqm.Or(
		qqm.Cond(1, qqm.CmdEq, "Alice"),
		qqm.Cond(1, qqm.CmdEq, "Bob"),
	)
	f := qqm.Filter{Range: root}

	query, args, err := f.BuildWhere(tf, dialect.SQLiteDialect{})
	require.NoError(t, err)
	assert.Contains(t, query, "WHERE")
	assert.Contains(t, query, " OR ")
	assert.Equal(t, []any{"Alice", "Bob"}, args)
}

func TestBuildWhere_NOT(t *testing.T) {
	t.Parallel()

	table := qqm.NewTable[filterTestRow](dialect.SQLiteDialect{})
	tf := table.Defs().Fields

	root := qqm.Not(qqm.Cond(2, qqm.CmdEq, 18))
	f := qqm.Filter{Range: root}

	query, args, err := f.BuildWhere(tf, dialect.SQLiteDialect{})
	require.NoError(t, err)
	assert.Contains(t, query, "WHERE")
	assert.Contains(t, query, "NOT")
	assert.Contains(t, query, "age")
	assert.Equal(t, []any{18}, args)
}

func TestBuildWhere_Nested(t *testing.T) {
	t.Parallel()

	table := qqm.NewTable[filterTestRow](dialect.SQLiteDialect{})
	tf := table.Defs().Fields

	root := qqm.And(
		qqm.Cond(1, qqm.CmdEq, "Alice"),
		qqm.Or(
			qqm.Cond(2, qqm.CmdGt, 20),
			qqm.Cond(2, qqm.CmdLt, 10),
		),
	)
	f := qqm.Filter{Range: root}

	query, args, err := f.BuildWhere(tf, dialect.SQLiteDialect{})
	require.NoError(t, err)
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
		name     string
		op       qqm.CommandOp
		value    any
		wantSQL  string
		wantArgs []any
	}{
		{"Eq", qqm.CmdEq, 42, "= ?", []any{42}},
		{"NotEq", qqm.CmdNotEq, 42, "<> ?", []any{42}},
		{"Gt", qqm.CmdGt, 42, "> ?", []any{42}},
		{"Gte", qqm.CmdGte, 42, ">= ?", []any{42}},
		{"Lt", qqm.CmdLt, 42, "< ?", []any{42}},
		{"Lte", qqm.CmdLte, 42, "<= ?", []any{42}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			table := qqm.NewTable[filterTestRow](dialect.SQLiteDialect{})
			tf := table.Defs().Fields

			root := qqm.Cond(2, tt.op, tt.value)
			f := qqm.Filter{Range: root}

			query, args, err := f.BuildWhere(tf, dialect.SQLiteDialect{})
			require.NoError(t, err)
			assert.Contains(t, query, tt.wantSQL)
			assert.Equal(t, tt.wantArgs, args)
		})
	}
}

func TestBuildWhere_IsNull_IsNotNull(t *testing.T) {
	t.Parallel()

	t.Run("IsNull", func(t *testing.T) {
		t.Parallel()

		table := qqm.NewTable[filterTestRow](dialect.SQLiteDialect{})
		tf := table.Defs().Fields

		root := qqm.Cond(1, qqm.CmdIsNull, nil)
		f := qqm.Filter{Range: root}

		query, args, err := f.BuildWhere(tf, dialect.SQLiteDialect{})
		require.NoError(t, err)
		assert.Contains(t, query, "IS NULL")
		assert.Nil(t, args)
	})

	t.Run("IsNotNull", func(t *testing.T) {
		t.Parallel()

		table := qqm.NewTable[filterTestRow](dialect.SQLiteDialect{})
		tf := table.Defs().Fields

		root := qqm.Cond(1, qqm.CmdIsNotNull, nil)
		f := qqm.Filter{Range: root}

		query, args, err := f.BuildWhere(tf, dialect.SQLiteDialect{})
		require.NoError(t, err)
		assert.Contains(t, query, "IS NOT NULL")
		assert.Nil(t, args)
	})
}

func TestBuildWhere_Like(t *testing.T) {
	t.Parallel()

	table := qqm.NewTable[filterTestRow](dialect.SQLiteDialect{})
	tf := table.Defs().Fields

	root := qqm.Cond(1, qqm.CmdLike, "%Alice%")
	f := qqm.Filter{Range: root}

	query, args, err := f.BuildWhere(tf, dialect.SQLiteDialect{})
	require.NoError(t, err)
	assert.Contains(t, query, "LIKE ?")
	assert.Equal(t, []any{"%Alice%"}, args)
}

func TestBuildWhere_ILike_PG(t *testing.T) {
	t.Parallel()

	table := qqm.NewTable[filterTestRow](dialect.PostgreSQLDialect{})
	tf := table.Defs().Fields

	root := qqm.Cond(1, qqm.CmdILike, "%Alice%")
	f := qqm.Filter{Range: root}

	query, args, err := f.BuildWhere(tf, dialect.PostgreSQLDialect{})
	require.NoError(t, err)
	assert.Contains(t, query, "ILIKE")
	assert.Equal(t, []any{"%Alice%"}, args)
}

func TestBuildWhere_ILike_SQLite(t *testing.T) {
	t.Parallel()

	table := qqm.NewTable[filterTestRow](dialect.SQLiteDialect{})
	tf := table.Defs().Fields

	root := qqm.Cond(1, qqm.CmdILike, "%Alice%")
	f := qqm.Filter{Range: root}

	query, args, err := f.BuildWhere(tf, dialect.SQLiteDialect{})
	require.NoError(t, err)
	assert.Contains(t, query, "LOWER(")
	assert.Contains(t, query, ") LIKE LOWER(")
	assert.Equal(t, []any{"%Alice%"}, args)
}

func TestBuildWhere_In(t *testing.T) {
	t.Parallel()

	table := qqm.NewTable[filterTestRow](dialect.SQLiteDialect{})
	tf := table.Defs().Fields

	root := qqm.Cond(2, qqm.CmdIn, []any{20, 30, 40})
	f := qqm.Filter{Range: root}

	query, args, err := f.BuildWhere(tf, dialect.SQLiteDialect{})
	require.NoError(t, err)
	assert.Contains(t, query, "IN")
	assert.Contains(t, query, "?, ?, ?")
	assert.Equal(t, []any{20, 30, 40}, args)
}

func TestBuildWhere_PlaceholderContinuity(t *testing.T) {
	t.Parallel()

	table := qqm.NewTable[filterTestRow](dialect.PostgreSQLDialect{})
	tf := table.Defs().Fields

	root := qqm.And(
		qqm.Cond(1, qqm.CmdEq, "Alice"),
		qqm.Cond(2, qqm.CmdGte, 25),
		qqm.Cond(2, qqm.CmdLte, 50),
	)
	f := qqm.Filter{Range: root}

	query, args, err := f.BuildWhere(tf, dialect.PostgreSQLDialect{})
	require.NoError(t, err)
	assert.Contains(t, query, "$1")
	assert.Contains(t, query, "$2")
	assert.Contains(t, query, "$3")
	assert.NotContains(t, query, "$4")
	assert.Len(t, args, 3)
	assert.Equal(t, []any{"Alice", 25, 50}, args)
}

func TestBuildWhere_OutOfRange(t *testing.T) {
	t.Parallel()

	table := qqm.NewTable[filterTestRow](dialect.SQLiteDialect{})
	tf := table.Defs().Fields

	root := qqm.Cond(99, qqm.CmdEq, "test")
	f := qqm.Filter{Range: root}

	query, args, err := f.BuildWhere(tf, dialect.SQLiteDialect{})
	require.Error(t, err)
	assert.Empty(t, query)
	assert.Nil(t, args)
}

func TestCond_HasRequiredMethods(t *testing.T) {
	t.Parallel()

	node := qqm.Cond(0, qqm.CmdEq, "test")
	assert.NotNil(t, node)

	andNode := qqm.And(node)
	assert.NotNil(t, andNode)

	orNode := qqm.Or(node, node)
	assert.NotNil(t, orNode)

	notNode := qqm.Not(node)
	require.NotNil(t, notNode)
}
