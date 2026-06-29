// Created at 2026-06-29
//go:build functional

package functional

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/table"
	"github.com/mirrorru/qqm/test/fixtures"
)

func TestFunctional_MultiQuery_INNER_JOIN_PostgreSQL(t *testing.T) {
	t.Parallel()
	_, ex := beginTxPG(t)
	ctx := context.Background()

	userTbl := table.NewTable[fixtures.User](dialect.PostgreSQLDialect{})
	orderTbl := table.NewTable[fixtures.Order](dialect.PostgreSQLDialect{})

	alice, err := userTbl.Insert(ctx, ex, &fixtures.User{Name: "Alice", Email: "alice@test.com"})
	require.NoError(t, err)
	_, err = userTbl.Insert(ctx, ex, &fixtures.User{Name: "Bob", Email: "bob@test.com"})
	require.NoError(t, err)

	_, err = orderTbl.Insert(ctx, ex, &fixtures.Order{UserID: alice.ID, Amount: 150.0})
	require.NoError(t, err)
	_, err = orderTbl.Insert(ctx, ex, &fixtures.Order{UserID: alice.ID, Amount: 250.0})
	require.NoError(t, err)

	q, err := table.NewQuery[fixtures.UserWithOrder](dialect.PostgreSQLDialect{})
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
		results, err := q.List(ctx, ex, table.AndFilter(
			table.Field("User.Name", table.And, table.Eq("Alice")),
		))
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("List with filter on Order field", func(t *testing.T) {
		results, err := q.List(ctx, ex, table.AndFilter(
			table.Field("Order.Amount", table.And, table.Gt(200.0)),
		))
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, 250.0, results[0].Order.Amount)
	})

	t.Run("List with combined filters across tables", func(t *testing.T) {
		results, err := q.List(ctx, ex, table.AndFilter(
			table.Field("User.Name", table.And, table.Eq("Alice")),
			table.Field("Order.Amount", table.And, table.Gt(100.0)),
		))
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})
}

func TestFunctional_MultiQuery_LEFT_JOIN_PostgreSQL(t *testing.T) {
	t.Parallel()
	_, ex := beginTxPG(t)
	ctx := context.Background()

	userTbl := table.NewTable[fixtures.User](dialect.PostgreSQLDialect{})
	orderTbl := table.NewTable[fixtures.Order](dialect.PostgreSQLDialect{})

	alice, err := userTbl.Insert(ctx, ex, &fixtures.User{Name: "Alice", Email: "alice@test.com"})
	require.NoError(t, err)
	_, err = userTbl.Insert(ctx, ex, &fixtures.User{Name: "Bob", Email: "bob@test.com"})
	require.NoError(t, err)

	_, err = orderTbl.Insert(ctx, ex, &fixtures.Order{UserID: alice.ID, Amount: 150.0})
	require.NoError(t, err)

	q, err := table.NewQuery[fixtures.UserWithOrderPtr](dialect.PostgreSQLDialect{})
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

	userTbl := table.NewTable[fixtures.User](dialect.PostgreSQLDialect{})
	orderTbl := table.NewTable[fixtures.Order](dialect.PostgreSQLDialect{})
	itemTbl := table.NewTable[fixtures.OrderItem](dialect.PostgreSQLDialect{})

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

	q, err := table.NewQuery[fixtures.UserOrderItem](dialect.PostgreSQLDialect{})
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

	userTbl := table.NewTable[fixtures.User](dialect.PostgreSQLDialect{})
	orderTbl := table.NewTable[fixtures.Order](dialect.PostgreSQLDialect{})

	alice, err := userTbl.Insert(ctx, ex, &fixtures.User{Name: "Alice", Email: "alice@test.com"})
	require.NoError(t, err)
	_, err = userTbl.Insert(ctx, ex, &fixtures.User{Name: "Bob", Email: "bob@test.com"})
	require.NoError(t, err)

	_, err = orderTbl.Insert(ctx, ex, &fixtures.Order{UserID: alice.ID, Amount: 100.0})
	require.NoError(t, err)

	q, err := table.NewQuery[fixtures.UserWithOrderPtr](dialect.PostgreSQLDialect{})
	require.NoError(t, err)

	t.Run("Gt filter on primary table only", func(t *testing.T) {
		results, err := q.List(ctx, ex, table.AndFilter(
			table.Field("User.ID", table.And, table.Gt(alice.ID)),
		))
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, "Bob", results[0].User.Name)
		assert.Nil(t, results[0].Order)
	})

	t.Run("Eq filter on primary table only", func(t *testing.T) {
		results, err := q.List(ctx, ex, table.AndFilter(
			table.Field("User.Name", table.And, table.Eq("Alice")),
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

	userTbl := table.NewTable[fixtures.User](dialect.PostgreSQLDialect{})
	orderTbl := table.NewTable[fixtures.Order](dialect.PostgreSQLDialect{})

	alice, err := userTbl.Insert(ctx, ex, &fixtures.User{Name: "Alice", Email: "alice@test.com"})
	require.NoError(t, err)
	bob, err := userTbl.Insert(ctx, ex, &fixtures.User{Name: "Bob", Email: "bob@test.com"})
	require.NoError(t, err)

	_, err = orderTbl.Insert(ctx, ex, &fixtures.Order{UserID: alice.ID, Amount: 100.0})
	require.NoError(t, err)
	_, err = orderTbl.Insert(ctx, ex, &fixtures.Order{UserID: bob.ID, Amount: 200.0})
	require.NoError(t, err)

	q, err := table.NewQuery[fixtures.UserWithOrder](dialect.PostgreSQLDialect{})
	require.NoError(t, err)

	t.Run("OR filter on primary table field", func(t *testing.T) {
		results, err := q.List(ctx, ex, table.OrFilter(
			table.Field("User.Name", table.And, table.Eq("Alice")),
			table.Field("User.Name", table.And, table.Eq("Bob")),
		))
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("OR filter on joined table field", func(t *testing.T) {
		results, err := q.List(ctx, ex, table.OrFilter(
			table.Field("Order.Amount", table.And, table.Eq(100.0)),
			table.Field("Order.Amount", table.And, table.Eq(200.0)),
		))
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})
}
