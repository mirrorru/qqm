//go:build functional

package functional

import (
	"context"
	"testing"

	"github.com/mirrorru/qqm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/test/fixtures"
)

func TestFunctional_MultiQuery_INNER_JOIN_PostgreSQL(t *testing.T) {
	t.Parallel()
	_, ex := beginTxPG(t)
	ctx := context.Background()

	userTbl := qqm.NewTable[fixtures.User](dialect.PostgreSQLDialect{})
	orderTbl := qqm.NewTable[fixtures.Order](dialect.PostgreSQLDialect{})

	alice, err := userTbl.Insert(ctx, ex, &fixtures.User{Name: "Alice", Email: "alice@test.com"})
	require.NoError(t, err)
	_, err = userTbl.Insert(ctx, ex, &fixtures.User{Name: "Bob", Email: "bob@test.com"})
	require.NoError(t, err)

	_, err = orderTbl.Insert(ctx, ex, &fixtures.Order{UserID: alice.ID, Amount: 150.0})
	require.NoError(t, err)
	_, err = orderTbl.Insert(ctx, ex, &fixtures.Order{UserID: alice.ID, Amount: 250.0})
	require.NoError(t, err)

	q, err := qqm.NewQuery[fixtures.UserWithOrder](dialect.PostgreSQLDialect{})
	require.NoError(t, err)

	t.Run("List without filters returns all matching rows", func(t *testing.T) {
		results, err := q.List(ctx, ex)
		require.NoError(t, err)
		assert.Len(t, results, 2)
		for _, r := range results {
			assert.Equal(t, alice.ID, r.User.ID)
			assert.Equal(t, "Alice", r.User.Name)
		}
	})

	t.Run("List with filter on User field", func(t *testing.T) {
		results, err := q.List(ctx, ex, qqm.AndFilter(
			qqm.Field("User.Name", qqm.And, qqm.Eq("Alice")),
		))
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("List with filter on Order field", func(t *testing.T) {
		results, err := q.List(ctx, ex, qqm.AndFilter(
			qqm.Field("Order.Amount", qqm.And, qqm.Gt(200.0)),
		))
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, 250.0, results[0].Order.Amount)
	})

	t.Run("List with combined filters across tables", func(t *testing.T) {
		results, err := q.List(ctx, ex, qqm.AndFilter(
			qqm.Field("User.Name", qqm.And, qqm.Eq("Alice")),
			qqm.Field("Order.Amount", qqm.And, qqm.Gt(100.0)),
		))
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})
}

func TestFunctional_MultiQuery_LEFT_JOIN_PostgreSQL(t *testing.T) {
	t.Parallel()
	_, ex := beginTxPG(t)
	ctx := context.Background()

	userTbl := qqm.NewTable[fixtures.User](dialect.PostgreSQLDialect{})
	orderTbl := qqm.NewTable[fixtures.Order](dialect.PostgreSQLDialect{})

	alice, err := userTbl.Insert(ctx, ex, &fixtures.User{Name: "Alice", Email: "alice@test.com"})
	require.NoError(t, err)
	_, err = userTbl.Insert(ctx, ex, &fixtures.User{Name: "Bob", Email: "bob@test.com"})
	require.NoError(t, err)

	_, err = orderTbl.Insert(ctx, ex, &fixtures.Order{UserID: alice.ID, Amount: 150.0})
	require.NoError(t, err)

	q, err := qqm.NewQuery[fixtures.UserWithOrderPtr](dialect.PostgreSQLDialect{})
	require.NoError(t, err)

	results, err := q.List(ctx, ex)
	require.NoError(t, err)
	assert.Len(t, results, 2)

	byName := make(map[string]fixtures.UserWithOrderPtr)
	for _, r := range results {
		byName[r.User.Name] = *r
	}

	aliceRow, ok := byName["Alice"]
	require.True(t, ok)
	require.NotNil(t, aliceRow.Order)
	assert.Equal(t, 150.0, aliceRow.Order.Amount)

	bobRow, ok := byName["Bob"]
	require.True(t, ok)
	assert.Nil(t, bobRow.Order)
}

func TestFunctional_MultiQuery_ThreeTableJoin_PostgreSQL(t *testing.T) {
	t.Parallel()
	_, ex := beginTxPG(t)
	ctx := context.Background()

	userTbl := qqm.NewTable[fixtures.User](dialect.PostgreSQLDialect{})
	orderTbl := qqm.NewTable[fixtures.Order](dialect.PostgreSQLDialect{})
	itemTbl := qqm.NewTable[fixtures.OrderItem](dialect.PostgreSQLDialect{})

	alice, err := userTbl.Insert(ctx, ex, &fixtures.User{Name: "Alice", Email: "alice@test.com"})
	require.NoError(t, err)
	_, err = userTbl.Insert(ctx, ex, &fixtures.User{Name: "Bob", Email: "bob@test.com"})
	require.NoError(t, err)

	insertedOrder, err := orderTbl.Insert(ctx, ex, &fixtures.Order{UserID: alice.ID, Amount: 100.0})
	require.NoError(t, err)

	_, err = itemTbl.Insert(ctx, ex, &fixtures.OrderItem{OrderID: insertedOrder.ID, Quantity: 2, Price: 25.0})
	require.NoError(t, err)
	_, err = itemTbl.Insert(ctx, ex, &fixtures.OrderItem{OrderID: insertedOrder.ID, Quantity: 1, Price: 50.0})
	require.NoError(t, err)

	q, err := qqm.NewQuery[fixtures.UserOrderItem](dialect.PostgreSQLDialect{})
	require.NoError(t, err)

	results, err := q.List(ctx, ex)
	require.NoError(t, err)
	assert.Len(t, results, 2)

	for _, r := range results {
		assert.Equal(t, alice.ID, r.User.ID)
		assert.Equal(t, "Alice", r.User.Name)
		assert.Equal(t, insertedOrder.ID, r.Order.ID)
		require.NotNil(t, r.OrderItem)
	}
}

func TestFunctional_MultiQuery_FilterOnlyPrimary_PostgreSQL(t *testing.T) {
	t.Parallel()
	_, ex := beginTxPG(t)
	ctx := context.Background()

	userTbl := qqm.NewTable[fixtures.User](dialect.PostgreSQLDialect{})
	orderTbl := qqm.NewTable[fixtures.Order](dialect.PostgreSQLDialect{})

	alice, err := userTbl.Insert(ctx, ex, &fixtures.User{Name: "Alice", Email: "alice@test.com"})
	require.NoError(t, err)
	_, err = userTbl.Insert(ctx, ex, &fixtures.User{Name: "Bob", Email: "bob@test.com"})
	require.NoError(t, err)

	_, err = orderTbl.Insert(ctx, ex, &fixtures.Order{UserID: alice.ID, Amount: 100.0})
	require.NoError(t, err)

	q, err := qqm.NewQuery[fixtures.UserWithOrderPtr](dialect.PostgreSQLDialect{})
	require.NoError(t, err)

	t.Run("Gt filter on primary table only", func(t *testing.T) {
		results, err := q.List(ctx, ex, qqm.AndFilter(
			qqm.Field("User.ID", qqm.And, qqm.Gt(alice.ID)),
		))
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, "Bob", results[0].User.Name)
		assert.Nil(t, results[0].Order)
	})

	t.Run("Eq filter on primary table only", func(t *testing.T) {
		results, err := q.List(ctx, ex, qqm.AndFilter(
			qqm.Field("User.Name", qqm.And, qqm.Eq("Alice")),
		))
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, "Alice", results[0].User.Name)
		require.NotNil(t, results[0].Order)
	})
}

func TestFunctional_MultiQuery_OrFilter_PostgreSQL(t *testing.T) {
	t.Parallel()
	_, ex := beginTxPG(t)
	ctx := context.Background()

	userTbl := qqm.NewTable[fixtures.User](dialect.PostgreSQLDialect{})
	orderTbl := qqm.NewTable[fixtures.Order](dialect.PostgreSQLDialect{})

	alice, err := userTbl.Insert(ctx, ex, &fixtures.User{Name: "Alice", Email: "alice@test.com"})
	require.NoError(t, err)
	bob, err := userTbl.Insert(ctx, ex, &fixtures.User{Name: "Bob", Email: "bob@test.com"})
	require.NoError(t, err)

	_, err = orderTbl.Insert(ctx, ex, &fixtures.Order{UserID: alice.ID, Amount: 100.0})
	require.NoError(t, err)
	_, err = orderTbl.Insert(ctx, ex, &fixtures.Order{UserID: bob.ID, Amount: 200.0})
	require.NoError(t, err)

	q, err := qqm.NewQuery[fixtures.UserWithOrder](dialect.PostgreSQLDialect{})
	require.NoError(t, err)

	t.Run("OR filter on primary table field", func(t *testing.T) {
		results, err := q.List(ctx, ex, qqm.OrFilter(
			qqm.Field("User.Name", qqm.And, qqm.Eq("Alice")),
			qqm.Field("User.Name", qqm.And, qqm.Eq("Bob")),
		))
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("OR filter on joined table field", func(t *testing.T) {
		results, err := q.List(ctx, ex, qqm.OrFilter(
			qqm.Field("Order.Amount", qqm.And, qqm.Eq(100.0)),
			qqm.Field("Order.Amount", qqm.And, qqm.Eq(200.0)),
		))
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})
}

func TestFunctional_MultiQuery_One_INNER_PostgreSQL(t *testing.T) {
	t.Parallel()
	_, ex := beginTxPG(t)
	ctx := context.Background()

	userTbl := qqm.NewTable[fixtures.User](dialect.PostgreSQLDialect{})
	orderTbl := qqm.NewTable[fixtures.Order](dialect.PostgreSQLDialect{})

	alice, err := userTbl.Insert(ctx, ex, &fixtures.User{Name: "Alice", Email: "alice@test.com"})
	require.NoError(t, err)
	_, err = userTbl.Insert(ctx, ex, &fixtures.User{Name: "Bob", Email: "bob@test.com"})
	require.NoError(t, err)

	order, err := orderTbl.Insert(ctx, ex, &fixtures.Order{UserID: alice.ID, Amount: 150.0})
	require.NoError(t, err)

	q, err := qqm.NewQuery[fixtures.UserWithOrder](dialect.PostgreSQLDialect{})
	require.NoError(t, err)

	t.Run("One returns single row by primary table PK", func(t *testing.T) {
		row, err := q.One(ctx, ex, alice.ID)
		require.NoError(t, err)
		require.NotNil(t, row)
		assert.Equal(t, alice.ID, row.User.ID)
		assert.Equal(t, "Alice", row.User.Name)
		assert.Equal(t, order.ID, row.Order.ID)
		assert.Equal(t, 150.0, row.Order.Amount)
	})

	t.Run("One returns error when no rows match", func(t *testing.T) {
		_, err := q.One(ctx, ex, int64(999))
		require.Error(t, err)
	})
}

func TestFunctional_MultiQuery_One_LEFT_PostgreSQL(t *testing.T) {
	t.Parallel()
	_, ex := beginTxPG(t)
	ctx := context.Background()

	userTbl := qqm.NewTable[fixtures.User](dialect.PostgreSQLDialect{})
	orderTbl := qqm.NewTable[fixtures.Order](dialect.PostgreSQLDialect{})

	alice, err := userTbl.Insert(ctx, ex, &fixtures.User{Name: "Alice", Email: "alice@test.com"})
	require.NoError(t, err)
	bob, err := userTbl.Insert(ctx, ex, &fixtures.User{Name: "Bob", Email: "bob@test.com"})
	require.NoError(t, err)

	_, err = orderTbl.Insert(ctx, ex, &fixtures.Order{UserID: alice.ID, Amount: 150.0})
	require.NoError(t, err)

	q, err := qqm.NewQuery[fixtures.UserWithOrderPtr](dialect.PostgreSQLDialect{})
	require.NoError(t, err)

	t.Run("One returns user with order", func(t *testing.T) {
		row, err := q.One(ctx, ex, alice.ID)
		require.NoError(t, err)
		require.NotNil(t, row)
		assert.Equal(t, "Alice", row.User.Name)
		require.NotNil(t, row.Order)
		assert.Equal(t, 150.0, row.Order.Amount)
	})

	t.Run("One returns user without order as nil", func(t *testing.T) {
		row, err := q.One(ctx, ex, bob.ID)
		require.NoError(t, err)
		require.NotNil(t, row)
		assert.Equal(t, "Bob", row.User.Name)
		assert.Nil(t, row.Order)
	})
}

func TestFunctional_MultiQuery_One_ThreeTables_PostgreSQL(t *testing.T) {
	t.Parallel()
	_, ex := beginTxPG(t)
	ctx := context.Background()

	userTbl := qqm.NewTable[fixtures.User](dialect.PostgreSQLDialect{})
	orderTbl := qqm.NewTable[fixtures.Order](dialect.PostgreSQLDialect{})
	itemTbl := qqm.NewTable[fixtures.OrderItem](dialect.PostgreSQLDialect{})

	alice, err := userTbl.Insert(ctx, ex, &fixtures.User{Name: "Alice", Email: "alice@test.com"})
	require.NoError(t, err)

	insertedOrder, err := orderTbl.Insert(ctx, ex, &fixtures.Order{UserID: alice.ID, Amount: 100.0})
	require.NoError(t, err)

	_, err = itemTbl.Insert(ctx, ex, &fixtures.OrderItem{OrderID: insertedOrder.ID, Quantity: 3, Price: 30.0})
	require.NoError(t, err)

	q, err := qqm.NewQuery[fixtures.UserOrderItem](dialect.PostgreSQLDialect{})
	require.NoError(t, err)

	row, err := q.One(ctx, ex, alice.ID)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, alice.ID, row.User.ID)
	assert.Equal(t, "Alice", row.User.Name)
	assert.Equal(t, insertedOrder.ID, row.Order.ID)
	assert.Equal(t, 100.0, row.Order.Amount)
	require.NotNil(t, row.OrderItem)
	assert.Equal(t, insertedOrder.ID, row.OrderItem.OrderID)
	assert.Equal(t, 30.0, row.OrderItem.Price)
}
