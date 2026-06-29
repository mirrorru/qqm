// Created at 2026-06-29
//go:build smoke

package smoke

import (
	"context"
	"database/sql"
	"testing"

	"github.com/mirrorru/qqm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/executor"
	"github.com/mirrorru/qqm/test/fixtures"
	_ "modernc.org/sqlite"
)

func TestSmoke_MultiQuery_INNER_JOIN(t *testing.T) {
	t.Parallel()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT NOT NULL
		);
		CREATE TABLE orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL REFERENCES users(id),
			amount REAL NOT NULL
		)
	`)
	require.NoError(t, err)

	ex := executor.NewDBAdapter(db)
	ctx := context.Background()

	userTbl := qqm.NewTable[fixtures.User](dialect.SQLiteDialect{})
	orderTbl := qqm.NewTable[fixtures.Order](dialect.SQLiteDialect{})

	_, err = userTbl.Insert(ctx, ex, &fixtures.User{ID: 1, Name: "Alice", Email: "alice@test.com"})
	require.NoError(t, err)
	_, err = userTbl.Insert(ctx, ex, &fixtures.User{ID: 2, Name: "Bob", Email: "bob@test.com"})
	require.NoError(t, err)

	_, err = orderTbl.Insert(ctx, ex, &fixtures.Order{UserID: 1, Amount: 150.0})
	require.NoError(t, err)
	_, err = orderTbl.Insert(ctx, ex, &fixtures.Order{UserID: 1, Amount: 250.0})
	require.NoError(t, err)

	q, err := qqm.NewQuery[fixtures.UserWithOrder](dialect.SQLiteDialect{})
	require.NoError(t, err)

	t.Run("List without filters returns all matching rows", func(t *testing.T) {
		results, err := q.List(ctx, ex)
		require.NoError(t, err)
		assert.Len(t, results, 2)
		for _, r := range results {
			assert.Equal(t, int64(1), r.User.ID)
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
}

func TestSmoke_MultiQuery_LEFT_JOIN(t *testing.T) {
	t.Parallel()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT NOT NULL
		);
		CREATE TABLE orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL REFERENCES users(id),
			amount REAL NOT NULL
		)
	`)
	require.NoError(t, err)

	ex := executor.NewDBAdapter(db)
	ctx := context.Background()

	userTbl := qqm.NewTable[fixtures.User](dialect.SQLiteDialect{})
	orderTbl := qqm.NewTable[fixtures.Order](dialect.SQLiteDialect{})

	_, err = userTbl.Insert(ctx, ex, &fixtures.User{ID: 1, Name: "Alice", Email: "alice@test.com"})
	require.NoError(t, err)
	_, err = userTbl.Insert(ctx, ex, &fixtures.User{ID: 2, Name: "Bob", Email: "bob@test.com"})
	require.NoError(t, err)

	_, err = orderTbl.Insert(ctx, ex, &fixtures.Order{UserID: 1, Amount: 150.0})
	require.NoError(t, err)

	q, err := qqm.NewQuery[fixtures.UserWithOrderPtr](dialect.SQLiteDialect{})
	require.NoError(t, err)

	results, err := q.List(ctx, ex)
	require.NoError(t, err)
	assert.Len(t, results, 2)

	// Map results by user name for order-independent assertions
	byName := make(map[string]fixtures.UserWithOrderPtr)
	for _, r := range results {
		byName[r.User.Name] = *r
	}

	alice, ok := byName["Alice"]
	require.True(t, ok)
	require.NotNil(t, alice.Order)
	assert.Equal(t, 150.0, alice.Order.Amount)

	bob, ok := byName["Bob"]
	require.True(t, ok)
	assert.Nil(t, bob.Order)
}

func TestSmoke_MultiQuery_ThreeTableJoin(t *testing.T) {
	t.Parallel()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT NOT NULL
		);
		CREATE TABLE orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL REFERENCES users(id),
			amount REAL NOT NULL
		);
		CREATE TABLE order_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			order_id INTEGER NOT NULL REFERENCES orders(id),
			quantity INTEGER NOT NULL,
			price REAL NOT NULL
		)
	`)
	require.NoError(t, err)

	ex := executor.NewDBAdapter(db)
	ctx := context.Background()

	userTbl := qqm.NewTable[fixtures.User](dialect.SQLiteDialect{})
	orderTbl := qqm.NewTable[fixtures.Order](dialect.SQLiteDialect{})
	itemTbl := qqm.NewTable[fixtures.OrderItem](dialect.SQLiteDialect{})

	_, err = userTbl.Insert(ctx, ex, &fixtures.User{ID: 1, Name: "Alice", Email: "alice@test.com"})
	require.NoError(t, err)
	_, err = userTbl.Insert(ctx, ex, &fixtures.User{ID: 2, Name: "Bob", Email: "bob@test.com"})
	require.NoError(t, err)

	insertedOrder, err := orderTbl.Insert(ctx, ex, &fixtures.Order{UserID: 1, Amount: 100.0})
	require.NoError(t, err)

	_, err = itemTbl.Insert(ctx, ex, &fixtures.OrderItem{OrderID: insertedOrder.ID, Quantity: 2, Price: 25.0})
	require.NoError(t, err)
	_, err = itemTbl.Insert(ctx, ex, &fixtures.OrderItem{OrderID: insertedOrder.ID, Quantity: 1, Price: 50.0})
	require.NoError(t, err)

	q, err := qqm.NewQuery[fixtures.UserOrderItem](dialect.SQLiteDialect{})
	require.NoError(t, err)

	results, err := q.List(ctx, ex)
	require.NoError(t, err)
	assert.Len(t, results, 2)

	for _, r := range results {
		assert.Equal(t, int64(1), r.User.ID)
		assert.Equal(t, "Alice", r.User.Name)
		assert.Equal(t, int64(1), r.Order.UserID)
		require.NotNil(t, r.OrderItem)
	}
}
