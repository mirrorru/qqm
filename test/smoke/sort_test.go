// Created at 2026-06-29
//go:build smoke

package smoke

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/executor"
	"github.com/mirrorru/qqm/table"
	_ "modernc.org/sqlite"
)

// sortRow — структура с sort-тегами для smoke-тестов
type sortRow struct {
	ID    int64  `qqm:"pk;auto"`
	Name  string `qqm:"sort=1"`
	Value int    `qqm:"sort=2,desc"`
}

func (s *sortRow) SQLName() string { return "sort_test" }

func TestSmoke_List_SortAsc(t *testing.T) {
	t.Parallel()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	ex := executor.NewDBAdapter(db)
	ctx := context.Background()

	_, err = db.Exec(`
		CREATE TABLE sort_test (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			value INTEGER NOT NULL
		)
	`)
	require.NoError(t, err)

	tbl := table.NewTable[sortRow](dialect.SQLiteDialect{})

	rows := []*sortRow{
		{Name: "Charlie", Value: 300},
		{Name: "Alice", Value: 100},
		{Name: "Bob", Value: 200},
	}

	for _, r := range rows {
		_, err := tbl.Insert(ctx, ex, r)
		require.NoError(t, err)
	}

	list, err := tbl.List(ctx, ex)
	require.NoError(t, err)
	require.Len(t, list, 3)

	assert.Equal(t, "Alice", list[0].Name)
	assert.Equal(t, "Bob", list[1].Name)
	assert.Equal(t, "Charlie", list[2].Name)
}

func TestSmoke_List_SortDesc(t *testing.T) {
	t.Parallel()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	ex := executor.NewDBAdapter(db)
	ctx := context.Background()

	_, err = db.Exec(`
		CREATE TABLE sort_test (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			value INTEGER NOT NULL
		)
	`)
	require.NoError(t, err)

	tbl := table.NewTable[sortRow](dialect.SQLiteDialect{})

	rows := []*sortRow{
		{Name: "Alice", Value: 100},
		{Name: "Alice", Value: 300},
		{Name: "Alice", Value: 200},
	}

	for _, r := range rows {
		_, err := tbl.Insert(ctx, ex, r)
		require.NoError(t, err)
	}

	list, err := tbl.List(ctx, ex)
	require.NoError(t, err)
	require.Len(t, list, 3)

	// sort=1: name ASC → все Alice одинаковы
	// sort=2,desc: value DESC → 300, 200, 100
	assert.Equal(t, int(300), list[0].Value)
	assert.Equal(t, int(200), list[1].Value)
	assert.Equal(t, int(100), list[2].Value)
}

func TestSmoke_List_SortNoTags(t *testing.T) {
	t.Parallel()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	ex := executor.NewDBAdapter(db)
	ctx := context.Background()

	_, err = db.Exec(`
		CREATE TABLE user_row (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			email TEXT NOT NULL
		)
	`)
	require.NoError(t, err)

	type userRow struct {
		ID    int64 `qqm:"pk;auto"`
		Name  string
		Email string
	}

	tbl := table.NewTable[userRow](dialect.SQLiteDialect{})

	for _, r := range []*userRow{
		{Name: "Z", Email: "z@test"},
		{Name: "A", Email: "a@test"},
	} {
		_, err := tbl.Insert(ctx, ex, r)
		require.NoError(t, err)
	}

	list, err := tbl.List(ctx, ex)
	require.NoError(t, err)
	require.Len(t, list, 2)

	// без sort-тегов порядок не гарантирован — просто проверяем что оба есть
	names := map[string]bool{}
	for _, r := range list {
		names[r.Name] = true
	}
	assert.True(t, names["Z"])
	assert.True(t, names["A"])
}
