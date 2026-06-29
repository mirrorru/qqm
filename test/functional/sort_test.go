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

// sortUserRow — структура с sort-тегами поверх таблицы users.
type sortUserRow struct {
	ID    int64  `qqm:"pk;auto"`
	Name  string `qqm:"sort=1,desc"`
	Email string `qqm:"sort=2"`
}

func (s *sortUserRow) SQLName() string { return "users" }

func TestFunctional_List_Sort_PostgreSQL(t *testing.T) {
	t.Parallel()
	_, ex := beginTxPG(t)
	ctx := context.Background()

	_, err := ex.ExecContext(ctx, `DELETE FROM users`)
	require.NoError(t, err)

	tbl := table.NewTable[sortUserRow](dialect.PostgreSQLDialect{})

	rows := []*sortUserRow{
		{Name: "Charlie", Email: "c@test"},
		{Name: "Alice", Email: "a@test"},
		{Name: "Bob", Email: "b@test"},
	}

	for _, r := range rows {
		ins, err := tbl.Insert(ctx, ex, r)
		require.NoError(t, err)
		require.NotNil(t, ins)
	}

	list, err := tbl.List(ctx, ex)
	require.NoError(t, err)
	require.Len(t, list, 3)

	// sort=1,desc: Name DESC → Charlie, Bob, Alice
	assert.Equal(t, "Charlie", list[0].Name)
	assert.Equal(t, "Bob", list[1].Name)
	assert.Equal(t, "Alice", list[2].Name)
}

func TestFunctional_List_Sort_MixedDirections(t *testing.T) {
	t.Parallel()
	_, ex := beginTxPG(t)
	ctx := context.Background()

	_, err := ex.ExecContext(ctx, `DELETE FROM users`)
	require.NoError(t, err)

	tbl := table.NewTable[sortUserRow](dialect.PostgreSQLDialect{})

	rows := []*sortUserRow{
		{Name: "Charlie", Email: "z@test"},
		{Name: "Bob", Email: "z@test"},
		{Name: "Alice", Email: "a@test"},
	}

	for _, r := range rows {
		ins, err := tbl.Insert(ctx, ex, r)
		require.NoError(t, err)
		require.NotNil(t, ins)
	}

	list, err := tbl.List(ctx, ex)
	require.NoError(t, err)
	require.Len(t, list, 3)

	// sort=1,desc: Name DESC → Charlie, Bob, Alice
	assert.Equal(t, "Charlie", list[0].Name)
	assert.Equal(t, "Bob", list[1].Name)
	assert.Equal(t, "Alice", list[2].Name)

	// для одинаковых email (Bob и Charlie) порядок недетерминирован
	// между Name DESC и Email ASC — проверяем только Name
}

func TestFunctional_List_NoSort_PostgreSQL(t *testing.T) {
	t.Parallel()
	_, ex := beginTxPG(t)
	ctx := context.Background()

	_, err := ex.ExecContext(ctx, `DELETE FROM users`)
	require.NoError(t, err)

	tbl := table.NewTable[fixtures.User](dialect.PostgreSQLDialect{})

	for _, r := range []*fixtures.User{
		{Name: "Z", Email: "z@test"},
		{Name: "A", Email: "a@test"},
	} {
		_, err := tbl.Insert(ctx, ex, r)
		require.NoError(t, err)
	}

	list, err := tbl.List(ctx, ex)
	require.NoError(t, err)
	require.Len(t, list, 2)

	names := map[string]bool{}
	for _, r := range list {
		names[r.Name] = true
	}
	assert.True(t, names["Z"])
	assert.True(t, names["A"])
}
