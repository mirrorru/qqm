//go:build functional

package functional

import (
	"context"
	"testing"

	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/test/fixtures"
	"github.com/mirrorru/qqm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestV2Functional_Table_CRUD(t *testing.T) {
	t.Parallel()
	_, ex := beginTxPG(t)
	ctx := context.Background()

	_, err := ex.ExecContext(ctx, `DELETE FROM orders`)
	require.NoError(t, err)
	_, err = ex.ExecContext(ctx, `DELETE FROM users`)
	require.NoError(t, err)

	tbl := qqm.NewTable[fixtures.User](dialect.PostgreSQLDialect{})

	user := &fixtures.User{
		Name:  "Alice",
		Email: "alice@test.com",
	}

	inserted, _, err := tbl.Ins(ctx, ex, user)
	require.NoError(t, err)
	assert.NotZero(t, inserted.ID)
	assert.Equal(t, "Alice", inserted.Name)

	fetched, err := tbl.One(ctx, ex, inserted.ID)
	require.NoError(t, err)
	assert.Equal(t, inserted.ID, fetched.ID)
	assert.Equal(t, "Alice", fetched.Name)

	fetched.Name = "Alice Updated"
	returned, _, err := tbl.Upd(ctx, ex, fetched)
	require.NoError(t, err)
	assert.Equal(t, "Alice Updated", returned.Name)

	updated, err := tbl.One(ctx, ex, inserted.ID)
	require.NoError(t, err)
	assert.Equal(t, "Alice Updated", updated.Name)

	delResult, err := tbl.Del(ctx, ex, inserted.ID)
	require.NoError(t, err)
	delAffected, err := delResult.RowsAffected()
	require.NoError(t, err)
	assert.Equal(t, int64(1), delAffected)

	_, err = tbl.One(ctx, ex, inserted.ID)
	require.Error(t, err)
}

func TestV2Functional_Table_Many(t *testing.T) {
	t.Parallel()
	_, ex := beginTxPG(t)
	ctx := context.Background()

	_, err := ex.ExecContext(ctx, `DELETE FROM orders`)
	require.NoError(t, err)
	_, err = ex.ExecContext(ctx, `DELETE FROM users`)
	require.NoError(t, err)

	tbl := qqm.NewTable[fixtures.User](dialect.PostgreSQLDialect{})

	_, _, err = tbl.Ins(ctx, ex, &fixtures.User{Name: "Charlie", Email: "c@test.com"})
	require.NoError(t, err)
	_, _, err = tbl.Ins(ctx, ex, &fixtures.User{Name: "Alice", Email: "a@test.com"})
	require.NoError(t, err)
	_, _, err = tbl.Ins(ctx, ex, &fixtures.User{Name: "Bob", Email: "b@test.com"})
	require.NoError(t, err)

	results, err := tbl.Many(ctx, ex, nil)
	require.NoError(t, err)
	assert.Len(t, results, 3)

	t.Run("filter by name", func(t *testing.T) {
		filter := &qqm.Filter{
			Range: qqm.And(qqm.Cond(1, qqm.CmdEq, "Bob")),
		}
		results, err := tbl.Many(ctx, ex, filter)
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, "Bob", results[0].Name)
	})

	t.Run("Offset+Limit", func(t *testing.T) {
		filter := &qqm.Filter{
			Offset: 1,
			Limit:  1,
		}
		results, err := tbl.Many(ctx, ex, filter)
		require.NoError(t, err)
		require.Len(t, results, 1)
	})
}

func TestV2Functional_Query_Many_INNER_JOIN(t *testing.T) {
	t.Parallel()
	_, ex := beginTxPG(t)
	ctx := context.Background()

	_, err := ex.ExecContext(ctx, `DELETE FROM orders`)
	require.NoError(t, err)
	_, err = ex.ExecContext(ctx, `DELETE FROM users`)
	require.NoError(t, err)

	userTbl := qqm.NewTable[fixtures.User](dialect.PostgreSQLDialect{})
	orderTbl := qqm.NewTable[fixtures.Order](dialect.PostgreSQLDialect{})

	alice, _, err := userTbl.Ins(ctx, ex, &fixtures.User{Name: "Alice", Email: "alice@test.com"})
	require.NoError(t, err)
	_, _, err = userTbl.Ins(ctx, ex, &fixtures.User{Name: "Bob", Email: "bob@test.com"})
	require.NoError(t, err)

	_, _, err = orderTbl.Ins(ctx, ex, &fixtures.Order{UserID: alice.ID, Amount: 150.0})
	require.NoError(t, err)
	_, _, err = orderTbl.Ins(ctx, ex, &fixtures.Order{UserID: alice.ID, Amount: 250.0})
	require.NoError(t, err)

	query := qqm.NewQuery[fixtures.UserWithOrder](dialect.PostgreSQLDialect{})

	results, err := query.Many(ctx, ex, nil)
	require.NoError(t, err)
	assert.Len(t, results, 2)
	for _, r := range results {
		assert.Equal(t, "Alice", r.User.Name)
		assert.NotZero(t, r.Order.Amount)
	}
}

func TestV2Functional_Query_Many_LEFT_JOIN(t *testing.T) {
	t.Parallel()
	_, ex := beginTxPG(t)
	ctx := context.Background()

	_, err := ex.ExecContext(ctx, `DELETE FROM orders`)
	require.NoError(t, err)
	_, err = ex.ExecContext(ctx, `DELETE FROM users`)
	require.NoError(t, err)

	userTbl := qqm.NewTable[fixtures.User](dialect.PostgreSQLDialect{})
	orderTbl := qqm.NewTable[fixtures.Order](dialect.PostgreSQLDialect{})

	alice, _, err := userTbl.Ins(ctx, ex, &fixtures.User{Name: "Alice", Email: "alice@test.com"})
	require.NoError(t, err)
	_, _, err = userTbl.Ins(ctx, ex, &fixtures.User{Name: "Bob", Email: "bob@test.com"})
	require.NoError(t, err)

	_, _, err = orderTbl.Ins(ctx, ex, &fixtures.Order{UserID: alice.ID, Amount: 150.0})
	require.NoError(t, err)

	query := qqm.NewQuery[fixtures.UserWithOrderLeft](dialect.PostgreSQLDialect{})

	results, err := query.Many(ctx, ex, nil)
	require.NoError(t, err)
	assert.Len(t, results, 2)

	byName := make(map[string]fixtures.UserWithOrderLeft)
	for _, r := range results {
		byName[r.User.Name] = *r
	}

	aliceRow, ok := byName["Alice"]
	require.True(t, ok)
	assert.NotZero(t, aliceRow.Order.ID)
	assert.Equal(t, 150.0, aliceRow.Order.Amount)

	bobRow, ok := byName["Bob"]
	require.True(t, ok)
	assert.Zero(t, bobRow.Order.ID)
}

func TestV2Functional_Query_One_INNER_JOIN(t *testing.T) {
	t.Parallel()
	_, ex := beginTxPG(t)
	ctx := context.Background()

	_, err := ex.ExecContext(ctx, `DELETE FROM orders`)
	require.NoError(t, err)
	_, err = ex.ExecContext(ctx, `DELETE FROM users`)
	require.NoError(t, err)

	userTbl := qqm.NewTable[fixtures.User](dialect.PostgreSQLDialect{})
	orderTbl := qqm.NewTable[fixtures.Order](dialect.PostgreSQLDialect{})

	alice, _, err := userTbl.Ins(ctx, ex, &fixtures.User{Name: "Alice", Email: "alice@test.com"})
	require.NoError(t, err)
	_, _, err = orderTbl.Ins(ctx, ex, &fixtures.Order{UserID: alice.ID, Amount: 150.0})
	require.NoError(t, err)

	query := qqm.NewQuery[fixtures.UserWithOrder](dialect.PostgreSQLDialect{})

	row, err := query.One(ctx, ex, alice.ID)
	require.NoError(t, err)
	assert.Equal(t, "Alice", row.User.Name)
	assert.Equal(t, 150.0, row.Order.Amount)
}

func TestV2Functional_Query_One_LEFT_JOIN(t *testing.T) {
	t.Parallel()
	_, ex := beginTxPG(t)
	ctx := context.Background()

	_, err := ex.ExecContext(ctx, `DELETE FROM orders`)
	require.NoError(t, err)
	_, err = ex.ExecContext(ctx, `DELETE FROM users`)
	require.NoError(t, err)

	userTbl := qqm.NewTable[fixtures.User](dialect.PostgreSQLDialect{})
	orderTbl := qqm.NewTable[fixtures.Order](dialect.PostgreSQLDialect{})

	alice, _, err := userTbl.Ins(ctx, ex, &fixtures.User{Name: "Alice", Email: "alice@test.com"})
	require.NoError(t, err)
	bob, _, err := userTbl.Ins(ctx, ex, &fixtures.User{Name: "Bob", Email: "bob@test.com"})
	require.NoError(t, err)

	_, _, err = orderTbl.Ins(ctx, ex, &fixtures.Order{UserID: alice.ID, Amount: 150.0})
	require.NoError(t, err)

	query := qqm.NewQuery[fixtures.UserWithOrderLeft](dialect.PostgreSQLDialect{})

	t.Run("One with order", func(t *testing.T) {
		row, err := query.One(ctx, ex, alice.ID)
		require.NoError(t, err)
		assert.Equal(t, "Alice", row.User.Name)
		assert.NotZero(t, row.Order.ID)
		assert.Equal(t, 150.0, row.Order.Amount)
	})

	t.Run("One without order — zero-value Order", func(t *testing.T) {
		row, err := query.One(ctx, ex, bob.ID)
		require.NoError(t, err)
		assert.Equal(t, "Bob", row.User.Name)
		assert.Zero(t, row.Order.ID)
	})
}

func TestV2Functional_Query_Many_WithFilter(t *testing.T) {
	t.Parallel()
	_, ex := beginTxPG(t)
	ctx := context.Background()

	_, err := ex.ExecContext(ctx, `DELETE FROM orders`)
	require.NoError(t, err)
	_, err = ex.ExecContext(ctx, `DELETE FROM users`)
	require.NoError(t, err)

	userTbl := qqm.NewTable[fixtures.User](dialect.PostgreSQLDialect{})
	orderTbl := qqm.NewTable[fixtures.Order](dialect.PostgreSQLDialect{})

	alice, _, err := userTbl.Ins(ctx, ex, &fixtures.User{Name: "Alice", Email: "alice@test.com"})
	require.NoError(t, err)
	_, _, err = userTbl.Ins(ctx, ex, &fixtures.User{Name: "Bob", Email: "bob@test.com"})
	require.NoError(t, err)

	_, _, err = orderTbl.Ins(ctx, ex, &fixtures.Order{UserID: alice.ID, Amount: 150.0})
	require.NoError(t, err)
	_, _, err = orderTbl.Ins(ctx, ex, &fixtures.Order{UserID: alice.ID, Amount: 250.0})
	require.NoError(t, err)

	query := qqm.NewQuery[fixtures.UserWithOrder](dialect.PostgreSQLDialect{})

	t.Run("Filter by user name", func(t *testing.T) {
		filter := &qqm.Filter{
			Range: qqm.And(qqm.Cond(1, qqm.CmdEq, "Alice")),
		}
		results, err := query.Many(ctx, ex, filter)
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("Filter by order amount", func(t *testing.T) {
		filter := &qqm.Filter{
			Range: qqm.And(qqm.Cond(5, qqm.CmdGt, 200.0)),
		}
		results, err := query.Many(ctx, ex, filter)
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, 250.0, results[0].Order.Amount)
	})

	t.Run("Filter with Offset+Limit", func(t *testing.T) {
		filter := &qqm.Filter{
			Offset: 1,
			Limit:  1,
		}
		results, err := query.Many(ctx, ex, filter)
		require.NoError(t, err)
		assert.Len(t, results, 1)
	})
}
