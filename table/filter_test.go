package table

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/executor"
	"github.com/mirrorru/qqm/test/fixtures"
)

type mockRows struct {
	data   [][]any
	cursor int
}

func (m *mockRows) Next() bool {
	if m.cursor < len(m.data) {
		m.cursor++
		return true
	}
	return false
}

func (m *mockRows) Scan(dest ...any) error {
	row := m.data[m.cursor-1]
	for i, d := range dest {
		if i >= len(row) {
			break
		}
		v := reflect.ValueOf(d)
		if v.Kind() == reflect.Pointer && !v.IsNil() {
			elem := v.Elem()
			srcVal := reflect.ValueOf(row[i])
			if srcVal.Type().AssignableTo(elem.Type()) {
				elem.Set(srcVal)
			}
		}
	}
	return nil
}

func (m *mockRows) Close() error { return nil }

type mockResult struct{}

func (mockResult) LastInsertId() (int64, error) { return 0, nil }
func (mockResult) RowsAffected() (int64, error) { return 0, nil }

type mockExecutor struct {
	query string
	args  []any
	rows  *mockRows
}

func (m *mockExecutor) ExecContext(_ context.Context, _ string, _ ...any) (executor.Result, error) {
	return mockResult{}, nil
}

func (m *mockExecutor) QueryContext(_ context.Context, query string, args ...any) (executor.Rows, error) {
	m.query = query
	m.args = args
	return m.rows, nil
}

func (m *mockExecutor) QueryRowContext(_ context.Context, _ string, _ ...any) executor.Row {
	return m.rows
}

func TestFilter_Helpers(t *testing.T) {
	t.Run("Eq", func(t *testing.T) {
		c := Eq(42)
		assert.Equal(t, OpEq, c.Op)
		assert.Equal(t, 42, c.Value)
	})

	t.Run("Gt", func(t *testing.T) {
		c := Gt(10)
		assert.Equal(t, OpGt, c.Op)
		assert.Equal(t, 10, c.Value)
	})

	t.Run("Lt", func(t *testing.T) {
		c := Lt(20)
		assert.Equal(t, OpLt, c.Op)
		assert.Equal(t, 20, c.Value)
	})

	t.Run("Gte", func(t *testing.T) {
		c := Gte(5)
		assert.Equal(t, OpGte, c.Op)
		assert.Equal(t, 5, c.Value)
	})

	t.Run("Lte", func(t *testing.T) {
		c := Lte(100)
		assert.Equal(t, OpLte, c.Op)
		assert.Equal(t, 100, c.Value)
	})

	t.Run("Between", func(t *testing.T) {
		c := Between(10, 20)
		assert.Equal(t, OpBetween, c.Op)
		pair, ok := c.Value.([2]any)
		require.True(t, ok)
		assert.Equal(t, 10, pair[0])
		assert.Equal(t, 20, pair[1])
	})

	t.Run("In", func(t *testing.T) {
		c := In(1, 2, 3)
		assert.Equal(t, OpIn, c.Op)
		vals, ok := c.Value.([]any)
		require.True(t, ok)
		assert.Equal(t, []any{1, 2, 3}, vals)
	})

	t.Run("Field", func(t *testing.T) {
		ff := Field("Age", And, Gt(18), Lt(65))
		assert.Equal(t, "Age", ff.Field)
		assert.Equal(t, And, ff.Op)
		assert.Len(t, ff.Conditions, 2)
	})

	t.Run("AndFilter", func(t *testing.T) {
		f := AndFilter(
			Field("Age", And, Gt(18)),
		)
		assert.Equal(t, And, f.Op)
		assert.Len(t, f.Fields, 1)
	})

	t.Run("OrFilter", func(t *testing.T) {
		f := OrFilter(
			Field("Name", Or, Eq("Alice"), Eq("Bob")),
		)
		assert.Equal(t, Or, f.Op)
		assert.Len(t, f.Fields, 1)
	})
}

func TestFilter_BuildWhereClause_SQLite(t *testing.T) {
	tbl := NewTable[*fixtures.UserWithAge](dialect.SQLiteDialect{})

	t.Run("single field single condition", func(t *testing.T) {
		sql, args, err := tbl.buildFilterWhereClause([]Filter{
			AndFilter(Field("Age", And, Gt(18))),
		})
		require.NoError(t, err)
		assert.Equal(t, ` WHERE (age > ?)`, sql)
		assert.Equal(t, []any{18}, args)
	})

	t.Run("single field multiple conditions AND", func(t *testing.T) {
		sql, args, err := tbl.buildFilterWhereClause([]Filter{
			AndFilter(Field("Age", And, Gt(18), Lt(65))),
		})
		require.NoError(t, err)
		assert.Equal(t, ` WHERE (age > ? AND age < ?)`, sql)
		assert.Equal(t, []any{18, 65}, args)
	})

	t.Run("single field multiple conditions OR", func(t *testing.T) {
		sql, args, err := tbl.buildFilterWhereClause([]Filter{
			AndFilter(Field("Name", Or, Eq("Alice"), Eq("Bob"))),
		})
		require.NoError(t, err)
		assert.Equal(t, ` WHERE (name = ? OR name = ?)`, sql)
		assert.Equal(t, []any{"Alice", "Bob"}, args)
	})

	t.Run("multiple fields AND", func(t *testing.T) {
		sql, args, err := tbl.buildFilterWhereClause([]Filter{
			AndFilter(
				Field("Age", And, Gt(18)),
				Field("Name", And, Eq("Alice")),
			),
		})
		require.NoError(t, err)
		assert.Equal(t, ` WHERE (age > ?) AND (name = ?)`, sql)
		assert.Equal(t, []any{18, "Alice"}, args)
	})

	t.Run("Between condition", func(t *testing.T) {
		sql, args, err := tbl.buildFilterWhereClause([]Filter{
			AndFilter(Field("Age", And, Between(10, 20))),
		})
		require.NoError(t, err)
		assert.Equal(t, ` WHERE (age BETWEEN ? AND ?)`, sql)
		assert.Equal(t, []any{10, 20}, args)
	})

	t.Run("In condition", func(t *testing.T) {
		sql, args, err := tbl.buildFilterWhereClause([]Filter{
			AndFilter(Field("Age", And, In(1, 2, 3))),
		})
		require.NoError(t, err)
		assert.Equal(t, ` WHERE (age IN (?, ?, ?))`, sql)
		assert.Equal(t, []any{1, 2, 3}, args)
	})

	t.Run("no filters", func(t *testing.T) {
		sql, args, err := tbl.buildFilterWhereClause(nil)
		require.NoError(t, err)
		assert.Equal(t, "", sql)
		assert.Nil(t, args)
	})

	t.Run("empty filters", func(t *testing.T) {
		sql, args, err := tbl.buildFilterWhereClause([]Filter{})
		require.NoError(t, err)
		assert.Equal(t, "", sql)
		assert.Nil(t, args)
	})

	t.Run("empty fields in filter", func(t *testing.T) {
		sql, args, err := tbl.buildFilterWhereClause([]Filter{
			AndFilter(),
		})
		require.NoError(t, err)
		assert.Equal(t, "", sql)
		assert.Nil(t, args)
	})

	t.Run("unknown field", func(t *testing.T) {
		_, _, err := tbl.buildFilterWhereClause([]Filter{
			AndFilter(Field("Unknown", And, Eq("x"))),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), `qqm: unknown field "Unknown" in filter`)
	})

	t.Run("combined conditions on one field", func(t *testing.T) {
		sql, args, err := tbl.buildFilterWhereClause([]Filter{
			AndFilter(Field("Age", And, Gte(18), Lte(100))),
		})
		require.NoError(t, err)
		assert.Equal(t, ` WHERE (age >= ? AND age <= ?)`, sql)
		assert.Equal(t, []any{18, 100}, args)
	})
}

func TestFilter_BuildWhereClause_PostgreSQL(t *testing.T) {
	tbl := NewTable[*fixtures.UserWithAge](dialect.PostgreSQLDialect{})

	t.Run("single field single condition", func(t *testing.T) {
		sql, args, err := tbl.buildFilterWhereClause([]Filter{
			AndFilter(Field("Age", And, Gt(18))),
		})
		require.NoError(t, err)
		assert.Equal(t, ` WHERE (age > $1)`, sql)
		assert.Equal(t, []any{18}, args)
	})

	t.Run("multiple conditions with placeholders", func(t *testing.T) {
		sql, args, err := tbl.buildFilterWhereClause([]Filter{
			AndFilter(Field("Age", And, Gt(18), Lt(65))),
		})
		require.NoError(t, err)
		assert.Equal(t, ` WHERE (age > $1 AND age < $2)`, sql)
		assert.Equal(t, []any{18, 65}, args)
	})

	t.Run("Between with PostgreSQL placeholders", func(t *testing.T) {
		sql, args, err := tbl.buildFilterWhereClause([]Filter{
			AndFilter(Field("Age", And, Between(10, 20))),
		})
		require.NoError(t, err)
		assert.Equal(t, ` WHERE (age BETWEEN $1 AND $2)`, sql)
		assert.Equal(t, []any{10, 20}, args)
	})

	t.Run("In with PostgreSQL placeholders", func(t *testing.T) {
		sql, args, err := tbl.buildFilterWhereClause([]Filter{
			AndFilter(Field("Age", And, In(1, 2, 3))),
		})
		require.NoError(t, err)
		assert.Equal(t, ` WHERE (age IN ($1, $2, $3))`, sql)
		assert.Equal(t, []any{1, 2, 3}, args)
	})

	t.Run("multiple fields with placeholder continuity", func(t *testing.T) {
		sql, args, err := tbl.buildFilterWhereClause([]Filter{
			AndFilter(
				Field("Age", And, Gt(18), Lt(65)),
				Field("Name", And, Eq("Alice")),
			),
		})
		require.NoError(t, err)
		assert.Equal(t, ` WHERE (age > $1 AND age < $2) AND (name = $3)`, sql)
		assert.Equal(t, []any{18, 65, "Alice"}, args)
	})
}

func TestFilter_ValidationErrors(t *testing.T) {
	tbl := NewTable[*fixtures.UserWithAge](dialect.SQLiteDialect{})

	t.Run("unknown field name", func(t *testing.T) {
		_, _, err := tbl.buildFilterWhereClause([]Filter{
			AndFilter(Field("NonExistent", And, Eq("val"))),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "NonExistent")
	})

	t.Run("Between with wrong type", func(t *testing.T) {
		_, _, err := tbl.buildFilterWhereClause([]Filter{
			AndFilter(Field("Age", And, Condition{Op: OpBetween, Value: "not-a-pair"})),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Between requires [2]any value")
	})

	t.Run("In with wrong type", func(t *testing.T) {
		_, _, err := tbl.buildFilterWhereClause([]Filter{
			AndFilter(Field("Age", And, Condition{Op: OpIn, Value: "not-a-slice"})),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "In requires []any value")
	})
}

func TestFilter_CompositeKey(t *testing.T) {
	tbl := NewTable[*fixtures.OrgUser](dialect.SQLiteDialect{})

	t.Run("filter on composite key table", func(t *testing.T) {
		sql, args, err := tbl.buildFilterWhereClause([]Filter{
			AndFilter(Field("Name", And, Eq("test"))),
		})
		require.NoError(t, err)
		assert.Contains(t, sql, `name = ?`)
		assert.Equal(t, []any{"test"}, args)
	})

	t.Run("filter on org_id", func(t *testing.T) {
		sql, args, err := tbl.buildFilterWhereClause([]Filter{
			AndFilter(Field("OrgID", And, Eq(int64(1)))),
		})
		require.NoError(t, err)
		assert.Contains(t, sql, `org_id = ?`)
		assert.Equal(t, []any{int64(1)}, args)
	})
}

func TestFilter_OrFilterOp(t *testing.T) {
	tbl := NewTable[*fixtures.UserWithAge](dialect.SQLiteDialect{})

	t.Run("OR between fields", func(t *testing.T) {
		sql, args, err := tbl.buildFilterWhereClause([]Filter{
			OrFilter(
				Field("Name", And, Eq("Alice")),
				Field("Name", And, Eq("Bob")),
			),
		})
		require.NoError(t, err)
		assert.Equal(t, ` WHERE (name = ?) OR (name = ?)`, sql)
		assert.Equal(t, []any{"Alice", "Bob"}, args)
	})

	t.Run("OR between fields with PostgreSQL", func(t *testing.T) {
		tbl2 := NewTable[*fixtures.UserWithAge](dialect.PostgreSQLDialect{})
		sql, args, err := tbl2.buildFilterWhereClause([]Filter{
			OrFilter(
				Field("Name", And, Eq("Alice")),
				Field("Age", And, Gt(18)),
			),
		})
		require.NoError(t, err)
		assert.Equal(t, ` WHERE (name = $1) OR (age > $2)`, sql)
		assert.Equal(t, []any{"Alice", 18}, args)
	})
}

func TestFilter_ListWithMockExecutor(t *testing.T) {
	tbl := NewTable[*fixtures.UserWithAge](dialect.SQLiteDialect{})
	ctx := context.Background()

	t.Run("List without filters passes original SQL", func(t *testing.T) {
		mockEx := &mockExecutor{
			rows: &mockRows{data: nil},
		}
		result, err := tbl.List(ctx, mockEx)
		require.NoError(t, err)
		assert.Empty(t, result)
		assert.Contains(t, mockEx.query, "SELECT")
		assert.NotContains(t, mockEx.query, "WHERE")
	})

	t.Run("List with filter appends WHERE clause", func(t *testing.T) {
		mockEx := &mockExecutor{
			rows: &mockRows{data: nil},
		}
		_, err := tbl.List(ctx, mockEx, AndFilter(Field("Age", And, Gt(18))))
		require.NoError(t, err)
		assert.Contains(t, mockEx.query, "WHERE")
		assert.Contains(t, mockEx.query, "age > ?")
		assert.Equal(t, []any{18}, mockEx.args)
	})

	t.Run("List with multiple filters passes correct args", func(t *testing.T) {
		mockEx := &mockExecutor{
			rows: &mockRows{data: nil},
		}
		_, err := tbl.List(ctx, mockEx,
			AndFilter(
				Field("Age", And, Gt(18), Lt(65)),
				Field("Name", And, Eq("Alice")),
			),
		)
		require.NoError(t, err)
		assert.Equal(t, []any{18, 65, "Alice"}, mockEx.args)
	})

	t.Run("List with Between filter", func(t *testing.T) {
		mockEx := &mockExecutor{
			rows: &mockRows{data: nil},
		}
		_, err := tbl.List(ctx, mockEx, AndFilter(Field("Age", And, Between(10, 20))))
		require.NoError(t, err)
		assert.Contains(t, mockEx.query, "BETWEEN")
		assert.Equal(t, []any{10, 20}, mockEx.args)
	})

	t.Run("List with In filter", func(t *testing.T) {
		mockEx := &mockExecutor{
			rows: &mockRows{data: nil},
		}
		_, err := tbl.List(ctx, mockEx, AndFilter(Field("Age", And, In(1, 2, 3))))
		require.NoError(t, err)
		assert.Contains(t, mockEx.query, "IN")
		assert.Equal(t, []any{1, 2, 3}, mockEx.args)
	})

	t.Run("List with unknown field returns error", func(t *testing.T) {
		mockEx := &mockExecutor{
			rows: &mockRows{data: nil},
		}
		_, err := tbl.List(ctx, mockEx, AndFilter(Field("NonExistent", And, Eq("x"))))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "NonExistent")
	})

	t.Run("List with filter returns scanned rows", func(t *testing.T) {
		var id any = int64(1)
		var name any = "Alice"
		var email any = "alice@test.com"
		var age any = 25

		mockEx := &mockExecutor{
			rows: &mockRows{
				data: [][]any{{id, name, email, age}},
			},
		}
		result, err := tbl.List(ctx, mockEx, AndFilter(Field("Age", And, Eq(25))))
		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, int64(1), result[0].ID)
		assert.Equal(t, "Alice", result[0].Name)
		assert.Equal(t, 25, result[0].Age)
	})
}
