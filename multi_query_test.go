package qqm

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/meta"
	"github.com/mirrorru/qqm/test/fixtures"
)

func TestNewQuery_InvalidType(t *testing.T) {
	t.Run("non-struct QROW", func(t *testing.T) {
		_, err := NewQuery[int](dialect.SQLiteDialect{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "QROW must be a struct")
	})

	t.Run("non-struct field in QROW", func(t *testing.T) {
		type NotAStruct struct {
			Name string
		}
		_, err := NewQuery[NotAStruct](dialect.SQLiteDialect{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be a struct")
	})
}

func TestNewQuery_Metadata(t *testing.T) {
	t.Run("UserWithOrder has two entries", func(t *testing.T) {
		q, err := NewQuery[fixtures.UserWithOrder](dialect.SQLiteDialect{})
		require.NoError(t, err)
		require.Len(t, q.qmeta.entries, 2)
	})

	t.Run("primary is first field", func(t *testing.T) {
		q, err := NewQuery[fixtures.UserWithOrder](dialect.SQLiteDialect{})
		require.NoError(t, err)

		assert.Equal(t, "User", q.qmeta.entries[0].FieldName)
		assert.Equal(t, "INNER", q.qmeta.entries[0].JoinType)

		assert.Equal(t, "Order", q.qmeta.entries[1].FieldName)
		assert.Equal(t, "INNER", q.qmeta.entries[1].JoinType)
	})

	t.Run("UserWithOrderPtr has LEFT JOIN", func(t *testing.T) {
		q, err := NewQuery[fixtures.UserWithOrderLeft](dialect.SQLiteDialect{})
		require.NoError(t, err)

		assert.Equal(t, "User", q.qmeta.entries[0].FieldName)
		assert.Equal(t, "INNER", q.qmeta.entries[0].JoinType)

		assert.Equal(t, "Order", q.qmeta.entries[1].FieldName)
		assert.Equal(t, "LEFT", q.qmeta.entries[1].JoinType)
	})

	t.Run("UserOrderItem has three entries", func(t *testing.T) {
		q, err := NewQuery[fixtures.UserOrderItem](dialect.SQLiteDialect{})
		require.NoError(t, err)

		assert.Len(t, q.qmeta.entries, 3)
		assert.Equal(t, "User", q.qmeta.entries[0].FieldName)
		assert.Equal(t, "Order", q.qmeta.entries[1].FieldName)
		assert.Equal(t, "OrderItem", q.qmeta.entries[2].FieldName)
	})
}

func TestNewQuery_JOINSQL(t *testing.T) {
	t.Run("UserWithOrder SQL has JOIN with FK condition", func(t *testing.T) {
		q, err := NewQuery[fixtures.UserWithOrder](dialect.SQLiteDialect{})
		require.NoError(t, err)

		sql := q.qmeta.listSQL
		assert.Contains(t, sql, "SELECT")
		assert.Contains(t, sql, "FROM")
		assert.Contains(t, sql, "JOIN")
		assert.Contains(t, sql, "t2.user_id = t1.id")
	})

	t.Run("UserWithOrderLeft SQL has LEFT JOIN", func(t *testing.T) {
		q, err := NewQuery[fixtures.UserWithOrderLeft](dialect.SQLiteDialect{})
		require.NoError(t, err)

		sql := q.qmeta.listSQL
		assert.Contains(t, sql, "LEFT JOIN")
	})

	t.Run("UserOrderItem SQL has two JOINs", func(t *testing.T) {
		q, err := NewQuery[fixtures.UserOrderItem](dialect.SQLiteDialect{})
		require.NoError(t, err)

		sql := q.qmeta.listSQL
		// t2 = orders (Order.UserID ref=users.id → t2.user_id = t1.id)
		assert.Contains(t, sql, "t2.user_id = t1.id")
		// t3 = order_items (OrderItem.OrderID ref=orders.id → t3.order_id = t2.id)
		assert.Contains(t, sql, "t3.order_id = t2.id")
	})

	t.Run("PostgreSQL dialect quoting", func(t *testing.T) {
		q, err := NewQuery[fixtures.UserWithOrder](dialect.PostgreSQLDialect{})
		require.NoError(t, err)

		sql := q.qmeta.listSQL
		assert.Contains(t, sql, "t1.id")
		assert.Contains(t, sql, "users")
		assert.Contains(t, sql, "orders")
	})
}

func TestNewQuery_ColumnOrder(t *testing.T) {
	t.Run("UserWithOrder columns include all User and Order fields", func(t *testing.T) {
		q, err := NewQuery[fixtures.UserWithOrder](dialect.SQLiteDialect{})
		require.NoError(t, err)

		// User: id, name, email (3) + Order: id, user_id, amount (3) = 6
		assert.Len(t, q.qmeta.columns, 6)

		// User fields
		assert.Equal(t, "t1", q.qmeta.columns[0].TableAlias)
		assert.Equal(t, "id", q.qmeta.columns[0].Column)

		assert.Equal(t, "t1", q.qmeta.columns[1].TableAlias)
		assert.Equal(t, "name", q.qmeta.columns[1].Column)

		assert.Equal(t, "t1", q.qmeta.columns[2].TableAlias)
		assert.Equal(t, "email", q.qmeta.columns[2].Column)

		// Order fields
		assert.Equal(t, "t2", q.qmeta.columns[3].TableAlias)
		assert.Equal(t, "id", q.qmeta.columns[3].Column)

		assert.Equal(t, "t2", q.qmeta.columns[4].TableAlias)
		assert.Equal(t, "user_id", q.qmeta.columns[4].Column)

		assert.Equal(t, "t2", q.qmeta.columns[5].TableAlias)
		assert.Equal(t, "amount", q.qmeta.columns[5].Column)
	})
}

func TestNewQuery_MissingFKError(t *testing.T) {
	t.Run("no FK relationship returns error", func(t *testing.T) {
		type User struct {
			ID   int64 `qqm:"pk"`
			Name string
		}
		type NoRef struct {
			Value string
		}
		type BadQuery struct {
			User  User
			NoRef NoRef
		}

		_, err := NewQuery[BadQuery](dialect.SQLiteDialect{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no FK relationship found")
	})
}

func TestNewQuery_TableNameOverride(t *testing.T) {
	t.Run("table= tag overrides table name", func(t *testing.T) {
		type User struct {
			ID   int64 `qqm:"pk"`
			Name string
		}
		type Ref struct {
			ID     int64 `qqm:"pk"`
			UserID int64 `qqm:"ref=app_users.id"`
			Value  string
		}
		type QueryWithOverride struct {
			User User `qqm:"table=app_users"`
			Ref  Ref
		}

		q, err := NewQuery[QueryWithOverride](dialect.SQLiteDialect{})
		require.NoError(t, err)

		assert.Equal(t, "app_users", q.qmeta.entries[0].TableName)
		assert.Equal(t, "ref", q.qmeta.entries[1].TableName) // default snake_case
	})
}

func TestNewQuery_PrimaryTag(t *testing.T) {
	t.Run("primary tag explicitly sets primary table", func(t *testing.T) {
		type Ref struct {
			ID     int64 `qqm:"pk"`
			UserID int64 `qqm:"ref=users.id"`
			Value  string
		}
		type QueryWithPrimary struct {
			Ref  Ref `qqm:"primary"`
			User fixtures.User
		}

		q, err := NewQuery[QueryWithPrimary](dialect.SQLiteDialect{})
		require.NoError(t, err)

		// Ref is primary (t1), fixtures.User is t2 (table "users" via SQLName)
		// Ref.UserID has ref=users.id → ON t1.user_id = t2.id
		assert.Equal(t, "ref", q.qmeta.entries[0].TableName)
		assert.Equal(t, "users", q.qmeta.entries[1].TableName)
		sql := q.qmeta.listSQL
		assert.Contains(t, sql, "t2.id = t1.user_id")
		assert.Contains(t, sql, "INNER JOIN")
	})
}

func TestMultiWhereBuilder(t *testing.T) {
	t.Run("qualified field name resolves correctly", func(t *testing.T) {
		q, err := NewQuery[fixtures.UserWithOrder](dialect.SQLiteDialect{})
		require.NoError(t, err)

		wb := &multiWhereBuilder{
			dialect: dialect.SQLiteDialect{},
			qmeta:   q.qmeta,
		}

		alias, fm, err := wb.findQualifiedField("Order.Amount")
		require.NoError(t, err)
		assert.Equal(t, "t2", alias)
		assert.Equal(t, "Amount", fm.Name)
		assert.Equal(t, "amount", fm.Column)

		alias, fm, err = wb.findQualifiedField("User.Name")
		require.NoError(t, err)
		assert.Equal(t, "t1", alias)
		assert.Equal(t, "Name", fm.Name)
	})

	t.Run("unknown table returns error", func(t *testing.T) {
		q, err := NewQuery[fixtures.UserWithOrder](dialect.SQLiteDialect{})
		require.NoError(t, err)

		wb := &multiWhereBuilder{
			dialect: dialect.SQLiteDialect{},
			qmeta:   q.qmeta,
		}

		_, _, err = wb.findQualifiedField("Unknown.Field")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown table")
	})

	t.Run("unknown field returns error", func(t *testing.T) {
		q, err := NewQuery[fixtures.UserWithOrder](dialect.SQLiteDialect{})
		require.NoError(t, err)

		wb := &multiWhereBuilder{
			dialect: dialect.SQLiteDialect{},
			qmeta:   q.qmeta,
		}

		_, _, err = wb.findQualifiedField("User.Unknown")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown field")
	})

	t.Run("unqualified field name returns error", func(t *testing.T) {
		q, err := NewQuery[fixtures.UserWithOrder](dialect.SQLiteDialect{})
		require.NoError(t, err)

		wb := &multiWhereBuilder{
			dialect: dialect.SQLiteDialect{},
			qmeta:   q.qmeta,
		}

		_, _, err = wb.findQualifiedField("Amount")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "qualified field name required")
	})
}

func TestMultiWhere_BuildWhereSQL(t *testing.T) {
	t.Run("simple filter with qualified name", func(t *testing.T) {
		q, err := NewQuery[fixtures.UserWithOrder](dialect.SQLiteDialect{})
		require.NoError(t, err)

		wb := &multiWhereBuilder{
			dialect: dialect.SQLiteDialect{},
			qmeta:   q.qmeta,
		}

		sql, args, err := wb.buildWhereClause([]Filter{
			AndFilter(Field("Order.Amount", And, Gt(100.0))),
		})
		require.NoError(t, err)
		assert.Equal(t, ` WHERE (t2.amount > ?)`, sql)
		assert.Equal(t, []any{100.0}, args)
	})

	t.Run("filter with PostgreSQL placeholders", func(t *testing.T) {
		q, err := NewQuery[fixtures.UserWithOrder](dialect.PostgreSQLDialect{})
		require.NoError(t, err)

		wb := &multiWhereBuilder{
			dialect: dialect.PostgreSQLDialect{},
			qmeta:   q.qmeta,
		}

		sql, args, err := wb.buildWhereClause([]Filter{
			AndFilter(Field("User.Name", And, Eq("Alice"))),
		})
		require.NoError(t, err)
		assert.Equal(t, ` WHERE (t1.name = $1)`, sql)
		assert.Equal(t, []any{"Alice"}, args)
	})

	t.Run("multiple filters across tables", func(t *testing.T) {
		q, err := NewQuery[fixtures.UserWithOrder](dialect.SQLiteDialect{})
		require.NoError(t, err)

		wb := &multiWhereBuilder{
			dialect: dialect.SQLiteDialect{},
			qmeta:   q.qmeta,
		}

		sql, args, err := wb.buildWhereClause([]Filter{
			AndFilter(
				Field("User.Name", And, Eq("Alice")),
				Field("Order.Amount", And, Gt(50.0)),
			),
		})
		require.NoError(t, err)
		assert.Equal(t, ` WHERE (t1.name = ?) AND (t2.amount > ?)`, sql)
		assert.Equal(t, []any{"Alice", 50.0}, args)
	})
}

func TestClearMetaCache(t *testing.T) {
	meta.ClearCache()
}

type mockOneRow struct {
	data []any
}

func (m *mockOneRow) Scan(dest ...any) error {
	for i, d := range dest {
		if i >= len(m.data) {
			break
		}
		v := reflect.ValueOf(d)
		if v.Kind() != reflect.Pointer || v.IsNil() {
			continue
		}
		elem := v.Elem()
		if m.data[i] == nil {
			if elem.CanSet() {
				elem.Set(reflect.Zero(elem.Type()))
			}
			continue
		}
		srcVal := reflect.ValueOf(m.data[i])
		if srcVal.Type().AssignableTo(elem.Type()) {
			elem.Set(srcVal)
		} else if srcVal.Type().ConvertibleTo(elem.Type()) {
			elem.Set(srcVal.Convert(elem.Type()))
		}
	}
	return nil
}

type mockOneEx struct {
	query string
	args  []any
	row   *mockOneRow
}

func (m *mockOneEx) ExecContext(_ context.Context, _ string, _ ...any) (Result, error) {
	return mockResult{}, nil
}

func (m *mockOneEx) QueryContext(_ context.Context, query string, args ...any) (Rows, error) {
	m.query = query
	m.args = args
	return nil, nil //nolint:nilnil
}

func (m *mockOneEx) QueryRowContext(_ context.Context, query string, args ...any) Row {
	m.query = query
	m.args = args
	return m.row
}

func TestQuery_One_INNER_JOIN(t *testing.T) {
	q, err := NewQuery[fixtures.UserWithOrder](dialect.SQLiteDialect{})
	require.NoError(t, err)

	mock := &mockOneEx{
		row: &mockOneRow{
			data: []any{int64(1), "Alice", "alice@test.com", int64(1), int64(1), 150.0},
		},
	}

	row, err := q.One(context.Background(), mock, int64(1))
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, int64(1), row.User.ID)
	assert.Equal(t, "Alice", row.User.Name)
	assert.Equal(t, "alice@test.com", row.User.Email)
	assert.Equal(t, int64(1), row.Order.ID)
	assert.Equal(t, int64(1), row.Order.UserID)
	assert.InEpsilon(t, 150.0, row.Order.Amount, 0)

	assert.Contains(t, mock.query, "WHERE")
	assert.Contains(t, mock.query, "t1.id = ?")
	assert.Equal(t, []any{int64(1)}, mock.args)
}

func TestQuery_One_LEFT_JOIN(t *testing.T) {
	q, err := NewQuery[fixtures.UserWithOrderLeft](dialect.SQLiteDialect{})
	require.NoError(t, err)

	mock := &mockOneEx{
		row: &mockOneRow{
			data: []any{int64(1), "Alice", "alice@test.com", int64(1), int64(1), 150.0},
		},
	}

	row, err := q.One(context.Background(), mock, int64(1))
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, int64(1), row.User.ID)
	assert.Equal(t, "Alice", row.User.Name)
	assert.Equal(t, int64(1), row.Order.ID)
	assert.InEpsilon(t, 150.0, row.Order.Amount, 0)

	assert.Contains(t, mock.query, "WHERE")
	assert.Contains(t, mock.query, "t1.id = ?")
}

func TestQuery_One_LEFT_JOIN_NoOrder(t *testing.T) {
	q, err := NewQuery[fixtures.UserWithOrderLeft](dialect.SQLiteDialect{})
	require.NoError(t, err)

	mock := &mockOneEx{
		row: &mockOneRow{
			data: []any{int64(2), "Bob", "bob@test.com", nil, nil, nil},
		},
	}

	row, err := q.One(context.Background(), mock, int64(2))
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, int64(2), row.User.ID)
	assert.Equal(t, "Bob", row.User.Name)
	assert.Equal(t, int64(0), row.Order.ID)
	assert.Equal(t, float64(0), row.Order.Amount)
}

func TestQuery_One_ThreeTableJoin(t *testing.T) {
	q, err := NewQuery[fixtures.UserOrderItem](dialect.SQLiteDialect{})
	require.NoError(t, err)

	mock := &mockOneEx{
		row: &mockOneRow{
			data: []any{
				int64(1), "Alice", "alice@test.com",
				int64(1), int64(1), 100.0,
				int64(1), int64(1), 2, 25.0,
			},
		},
	}

	row, err := q.One(context.Background(), mock, int64(1))
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, int64(1), row.User.ID)
	assert.Equal(t, "Alice", row.User.Name)
	assert.Equal(t, int64(1), row.Order.ID)
	assert.Equal(t, 2, row.OrderItem.Quantity)
	assert.InEpsilon(t, 25.0, row.OrderItem.Price, 0)
}

func TestQuery_One_CompositePK(t *testing.T) {
	type RefA struct {
		UID int64 `qqm:"pk"`
	}
	type RefB struct {
		OrgID  int64 `qqm:"pk;col=org"`
		UserID int64 `qqm:"pk;col=user"`
		RefAID int64 `qqm:"ref=ref_a.uid"`
		Value  string
	}
	type CompositeQuery struct {
		A RefA
		B RefB
	}

	q, err := NewQuery[CompositeQuery](dialect.SQLiteDialect{})
	require.NoError(t, err)

	mock := &mockOneEx{
		row: &mockOneRow{
			data: []any{int64(1), int64(10), int64(100), int64(1), "value"},
		},
	}

	row, err := q.One(context.Background(), mock, int64(1))
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, int64(1), row.A.UID)
	assert.Equal(t, int64(10), row.B.OrgID)
	assert.Equal(t, int64(100), row.B.UserID)

	assert.Contains(t, mock.query, "WHERE")
	assert.Contains(t, mock.query, "t1.uid = ?")
	assert.Equal(t, []any{int64(1)}, mock.args)
}

func TestQuery_One_PostgreSQLPlaceholders(t *testing.T) {
	q, err := NewQuery[fixtures.UserWithOrder](dialect.PostgreSQLDialect{})
	require.NoError(t, err)

	mock := &mockOneEx{
		row: &mockOneRow{
			data: []any{int64(1), "Alice", "alice@test.com", int64(1), int64(1), 150.0},
		},
	}

	row, err := q.One(context.Background(), mock, int64(1))
	require.NoError(t, err)
	require.NotNil(t, row)

	assert.Contains(t, mock.query, "WHERE")
	assert.Contains(t, mock.query, "t1.id = $1")
	assert.Equal(t, []any{int64(1)}, mock.args)
}
