//go:build smoke

package smoke

import (
	"context"
	"database/sql"
	"testing"

	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/test/fixtures"
	"github.com/mirrorru/qqm/txproc"
	v2 "github.com/mirrorru/qqm/v2/field_info"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

func TestV2Smoke_Table_CRUD(t *testing.T) {
	t.Parallel()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			email TEXT NOT NULL
		)
	`)
	require.NoError(t, err)

	ex := txproc.NewDBAdapterVal(db)
	ctx := context.Background()
	tbl := v2.NewTable[fixtures.V2User](dialect.SQLiteDialect{})

	user := &fixtures.V2User{
		Name:  "Alice",
		Email: "alice@test.com",
	}

	inserted, _, err := tbl.Ins(ctx, ex, user)
	require.NoError(t, err)
	assert.NotZero(t, inserted.ID)
	assert.Equal(t, "Alice", inserted.Name)
	assert.Equal(t, "alice@test.com", inserted.Email)

	fetched, err := tbl.One(ctx, ex, inserted.ID)
	require.NoError(t, err)
	assert.Equal(t, inserted.ID, fetched.ID)
	assert.Equal(t, "Alice", fetched.Name)

	fetched.Name = "Alice Updated"
	fetched.Email = "alice-upd@test.com"
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

func TestV2Smoke_Table_Insert_NoPK(t *testing.T) {
	t.Parallel()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	_, err = db.Exec(`
		CREATE TABLE users (
			name TEXT NOT NULL,
			email TEXT NOT NULL
		)
	`)
	require.NoError(t, err)

	ex := txproc.NewDBAdapterVal(db)
	ctx := context.Background()
	tbl := v2.NewTable[fixtures.V2UserNoPK](dialect.SQLiteDialect{})

	user := &fixtures.V2UserNoPK{
		Name:  "NoPKUser",
		Email: "nopk@test.com",
	}

	inserted, _, err := tbl.Ins(ctx, ex, user)
	require.NoError(t, err)
	assert.Equal(t, "NoPKUser", inserted.Name)
}

func TestV2Smoke_Table_Many(t *testing.T) {
	t.Parallel()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			email TEXT NOT NULL,
			age INTEGER NOT NULL DEFAULT 0
		)
	`)
	require.NoError(t, err)

	ex := txproc.NewDBAdapterVal(db)
	ctx := context.Background()
	tbl := v2.NewTable[fixtures.V2UserWithSort](dialect.SQLiteDialect{})

	_, _, err = tbl.Ins(ctx, ex, &fixtures.V2UserWithSort{Name: "Charlie", Email: "charlie@test.com", Age: 30})
	require.NoError(t, err)
	_, _, err = tbl.Ins(ctx, ex, &fixtures.V2UserWithSort{Name: "Alice", Email: "alice@test.com", Age: 25})
	require.NoError(t, err)
	_, _, err = tbl.Ins(ctx, ex, &fixtures.V2UserWithSort{Name: "Bob", Email: "bob@test.com", Age: 35})
	require.NoError(t, err)

	t.Run("Many returns all rows", func(t *testing.T) {
		results, err := tbl.Many(ctx, ex, nil)
		require.NoError(t, err)
		assert.Len(t, results, 3)
	})

	t.Run("Many respects sort order", func(t *testing.T) {
		results, err := tbl.Many(ctx, ex, nil)
		require.NoError(t, err)
		require.Len(t, results, 3)
		assert.Equal(t, "Alice", results[0].Name)
		assert.Equal(t, "Bob", results[1].Name)
		assert.Equal(t, "Charlie", results[2].Name)
	})

	t.Run("Many with Name filter", func(t *testing.T) {
		filter := &v2.Filter{
			Range: v2.And(v2.Cond(1, v2.CmdEq, "Alice")),
		}
		results, err := tbl.Many(ctx, ex, filter)
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, "Alice", results[0].Name)
	})

	t.Run("Many with Limit", func(t *testing.T) {
		filter := &v2.Filter{
			Limit: 2,
			Range: nil,
		}
		results, err := tbl.Many(ctx, ex, filter)
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("Many with Age > filter", func(t *testing.T) {
		filter := &v2.Filter{
			Range: v2.And(v2.Cond(3, v2.CmdGt, 30)),
		}
		results, err := tbl.Many(ctx, ex, filter)
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, "Bob", results[0].Name)
	})
}

func TestV2Smoke_Query_Many_INNER_JOIN(t *testing.T) {
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

	ex := txproc.NewDBAdapterVal(db)
	ctx := context.Background()

	userTbl := v2.NewTable[fixtures.V2User](dialect.SQLiteDialect{})
	orderTbl := v2.NewTable[fixtures.V2Order](dialect.SQLiteDialect{})

	_, _, err = userTbl.Ins(ctx, ex, &fixtures.V2User{ID: 1, Name: "Alice", Email: "alice@test.com"})
	require.NoError(t, err)
	_, _, err = userTbl.Ins(ctx, ex, &fixtures.V2User{ID: 2, Name: "Bob", Email: "bob@test.com"})
	require.NoError(t, err)

	_, _, err = orderTbl.Ins(ctx, ex, &fixtures.V2Order{UserID: 1, Amount: 150.0})
	require.NoError(t, err)
	_, _, err = orderTbl.Ins(ctx, ex, &fixtures.V2Order{UserID: 1, Amount: 250.0})
	require.NoError(t, err)

	query := v2.NewQuery[fixtures.V2UserWithOrder](dialect.SQLiteDialect{})

	results, err := query.Many(ctx, ex, nil)
	require.NoError(t, err)
	assert.Len(t, results, 2)
	for _, r := range results {
		assert.Equal(t, int64(1), r.User.ID)
	}
}

func TestV2Smoke_Query_Many_LEFT_JOIN(t *testing.T) {
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

	ex := txproc.NewDBAdapterVal(db)
	ctx := context.Background()

	userTbl := v2.NewTable[fixtures.V2User](dialect.SQLiteDialect{})
	orderTbl := v2.NewTable[fixtures.V2Order](dialect.SQLiteDialect{})

	_, _, err = userTbl.Ins(ctx, ex, &fixtures.V2User{ID: 1, Name: "Alice", Email: "alice@test.com"})
	require.NoError(t, err)
	_, _, err = userTbl.Ins(ctx, ex, &fixtures.V2User{ID: 2, Name: "Bob", Email: "bob@test.com"})
	require.NoError(t, err)

	_, _, err = orderTbl.Ins(ctx, ex, &fixtures.V2Order{UserID: 1, Amount: 150.0})
	require.NoError(t, err)

	query := v2.NewQuery[fixtures.V2UserWithOrderLeft](dialect.SQLiteDialect{})
	results, err := query.Many(ctx, ex, nil)
	require.NoError(t, err)
	assert.Len(t, results, 2)

	byName := make(map[string]fixtures.V2UserWithOrderLeft)
	for _, r := range results {
		byName[r.User.Name] = *r
	}

	alice, ok := byName["Alice"]
	require.True(t, ok)
	assert.NotZero(t, alice.Order.ID)
	assert.Equal(t, 150.0, alice.Order.Amount)

	bob, ok := byName["Bob"]
	require.True(t, ok)
	assert.Zero(t, bob.Order.ID)
	assert.Zero(t, bob.Order.UserID)
}

func TestV2Smoke_Query_One_INNER_JOIN(t *testing.T) {
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

	ex := txproc.NewDBAdapterVal(db)
	ctx := context.Background()

	userTbl := v2.NewTable[fixtures.V2User](dialect.SQLiteDialect{})
	orderTbl := v2.NewTable[fixtures.V2Order](dialect.SQLiteDialect{})

	alice, _, err := userTbl.Ins(ctx, ex, &fixtures.V2User{ID: 1, Name: "Alice", Email: "alice@test.com"})
	require.NoError(t, err)
	_, _, err = userTbl.Ins(ctx, ex, &fixtures.V2User{ID: 2, Name: "Bob", Email: "bob@test.com"})
	require.NoError(t, err)

	_, _, err = orderTbl.Ins(ctx, ex, &fixtures.V2Order{UserID: alice.ID, Amount: 150.0})
	require.NoError(t, err)

	query := v2.NewQuery[fixtures.V2UserWithOrder](dialect.SQLiteDialect{})

	t.Run("One returns single row by PK", func(t *testing.T) {
		row, err := query.One(ctx, ex, int64(1))
		require.NoError(t, err)
		assert.Equal(t, int64(1), row.User.ID)
		assert.Equal(t, "Alice", row.User.Name)
		assert.Equal(t, 150.0, row.Order.Amount)
	})

	t.Run("One returns error when no row matches", func(t *testing.T) {
		_, err := query.One(ctx, ex, int64(999))
		require.Error(t, err)
	})
}

func TestV2Smoke_Query_One_LEFT_JOIN(t *testing.T) {
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

	ex := txproc.NewDBAdapterVal(db)
	ctx := context.Background()

	userTbl := v2.NewTable[fixtures.V2User](dialect.SQLiteDialect{})
	orderTbl := v2.NewTable[fixtures.V2Order](dialect.SQLiteDialect{})

	alice, _, err := userTbl.Ins(ctx, ex, &fixtures.V2User{ID: 1, Name: "Alice", Email: "alice@test.com"})
	require.NoError(t, err)
	_, _, err = userTbl.Ins(ctx, ex, &fixtures.V2User{ID: 2, Name: "Bob", Email: "bob@test.com"})
	require.NoError(t, err)

	_, _, err = orderTbl.Ins(ctx, ex, &fixtures.V2Order{UserID: alice.ID, Amount: 150.0})
	require.NoError(t, err)

	query := v2.NewQuery[fixtures.V2UserWithOrderLeft](dialect.SQLiteDialect{})

	t.Run("One with LEFT JOIN returns user with order", func(t *testing.T) {
		row, err := query.One(ctx, ex, int64(1))
		require.NoError(t, err)
		assert.Equal(t, "Alice", row.User.Name)
		assert.NotZero(t, row.Order.ID)
		assert.Equal(t, 150.0, row.Order.Amount)
	})

	t.Run("One with LEFT JOIN returns zero-value Order when no match", func(t *testing.T) {
		row, err := query.One(ctx, ex, int64(2))
		require.NoError(t, err)
		assert.Equal(t, "Bob", row.User.Name)
		assert.Zero(t, row.Order.ID)
		assert.Zero(t, row.Order.UserID)
	})
}

func TestV2Smoke_Query_Many_Sort(t *testing.T) {
	t.Parallel()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT NOT NULL,
			age INTEGER NOT NULL DEFAULT 0
		);
		CREATE TABLE orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL REFERENCES users(id),
			amount REAL NOT NULL
		)
	`)
	require.NoError(t, err)

	ex := txproc.NewDBAdapterVal(db)
	ctx := context.Background()

	userTbl := v2.NewTable[fixtures.V2UserWithSort](dialect.SQLiteDialect{})
	orderTbl := v2.NewTable[fixtures.V2Order](dialect.SQLiteDialect{})

	_, _, err = userTbl.Ins(ctx, ex, &fixtures.V2UserWithSort{ID: 1, Name: "Charlie", Email: "c@test.com", Age: 30})
	require.NoError(t, err)
	_, _, err = userTbl.Ins(ctx, ex, &fixtures.V2UserWithSort{ID: 2, Name: "Alice", Email: "a@test.com", Age: 25})
	require.NoError(t, err)
	_, _, err = userTbl.Ins(ctx, ex, &fixtures.V2UserWithSort{ID: 3, Name: "Bob", Email: "b@test.com", Age: 35})
	require.NoError(t, err)

	_, _, err = orderTbl.Ins(ctx, ex, &fixtures.V2Order{UserID: 1, Amount: 100.0})
	require.NoError(t, err)
	_, _, err = orderTbl.Ins(ctx, ex, &fixtures.V2Order{UserID: 2, Amount: 200.0})
	require.NoError(t, err)
	_, _, err = orderTbl.Ins(ctx, ex, &fixtures.V2Order{UserID: 3, Amount: 300.0})
	require.NoError(t, err)

	query := v2.NewQuery[fixtures.V2UserWithSortAndOrder](dialect.SQLiteDialect{})

	results, err := query.Many(ctx, ex, nil)
	require.NoError(t, err)
	require.Len(t, results, 3)
	assert.Equal(t, "Alice", results[0].User.Name)
	assert.Equal(t, "Bob", results[1].User.Name)
	assert.Equal(t, "Charlie", results[2].User.Name)
}

func TestV2Smoke_Query_Many_WithFilter(t *testing.T) {
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

	ex := txproc.NewDBAdapterVal(db)
	ctx := context.Background()

	userTbl := v2.NewTable[fixtures.V2User](dialect.SQLiteDialect{})
	orderTbl := v2.NewTable[fixtures.V2Order](dialect.SQLiteDialect{})

	_, _, err = userTbl.Ins(ctx, ex, &fixtures.V2User{ID: 1, Name: "Alice", Email: "alice@test.com"})
	require.NoError(t, err)
	_, _, err = userTbl.Ins(ctx, ex, &fixtures.V2User{ID: 2, Name: "Bob", Email: "bob@test.com"})
	require.NoError(t, err)

	_, _, err = orderTbl.Ins(ctx, ex, &fixtures.V2Order{UserID: 1, Amount: 150.0})
	require.NoError(t, err)
	_, _, err = orderTbl.Ins(ctx, ex, &fixtures.V2Order{UserID: 1, Amount: 250.0})
	require.NoError(t, err)
	_, _, err = orderTbl.Ins(ctx, ex, &fixtures.V2Order{UserID: 2, Amount: 100.0})
	require.NoError(t, err)

	query := v2.NewQuery[fixtures.V2UserWithOrder](dialect.SQLiteDialect{})

	t.Run("Filter by User.Name (flatField idx=1)", func(t *testing.T) {
		filter := &v2.Filter{
			Range: v2.And(v2.Cond(1, v2.CmdEq, "Alice")),
		}
		results, err := query.Many(ctx, ex, filter)
		require.NoError(t, err)
		assert.Len(t, results, 2)
		for _, r := range results {
			assert.Equal(t, "Alice", r.User.Name)
		}
	})

	t.Run("Filter by Order.Amount > (flatField idx=5)", func(t *testing.T) {
		filter := &v2.Filter{
			Range: v2.And(v2.Cond(5, v2.CmdGt, 200.0)),
		}
		results, err := query.Many(ctx, ex, filter)
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, 250.0, results[0].Order.Amount)
	})

	t.Run("Filter with Offset and Limit", func(t *testing.T) {
		filter := &v2.Filter{
			Offset: 1,
			Limit:  1,
			Range:  v2.And(v2.Cond(1, v2.CmdEq, "Alice")),
		}
		results, err := query.Many(ctx, ex, filter)
		require.NoError(t, err)
		assert.Len(t, results, 1)
	})
}
