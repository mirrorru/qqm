// Created at 2026-06-28
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

func TestFunctional_CRUD_Rooms_PostgreSQL(t *testing.T) {
	t.Parallel()
	_, ex := beginTxPG(t)
	ctx := context.Background()

	_, err := ex.ExecContext(ctx, `DELETE FROM rooms`)
	require.NoError(t, err)

	tbl := table.NewTable[fixtures.Rooms](dialect.PostgreSQLDialect{})

	now := int64(1700000000)
	room := &fixtures.Rooms{
		Name:      "Conference Room A",
		Square:    50.5,
		CreatedAt: now,
	}

	inserted, err := tbl.Insert(ctx, ex, room)
	require.NoError(t, err)
	assert.Equal(t, room.Name, inserted.Name)
	assert.Equal(t, room.Square, inserted.Square)
	assert.NotZero(t, inserted.ID, "auto-generated ID should not be zero")

	fetched, err := tbl.GetByPK(ctx, ex, inserted.ID)
	require.NoError(t, err)
	assert.Equal(t, inserted.ID, fetched.ID)
	assert.Equal(t, room.Name, fetched.Name)
	assert.Equal(t, room.Square, fetched.Square)

	fetched.Name = "Conference Room B"
	fetched.Square = 60.0
	err = tbl.Update(ctx, ex, fetched)
	require.NoError(t, err)

	updated, err := tbl.GetByPK(ctx, ex, inserted.ID)
	require.NoError(t, err)
	assert.Equal(t, "Conference Room B", updated.Name)
	assert.Equal(t, 60.0, updated.Square)

	list, err := tbl.List(ctx, ex)
	require.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, inserted.ID, list[0].ID)

	err = tbl.Delete(ctx, ex, inserted.ID)
	require.NoError(t, err)

	_, err = tbl.GetByPK(ctx, ex, inserted.ID)
	assert.Error(t, err)
}

func TestFunctional_CRUD_RoomMapping_PostgreSQL(t *testing.T) {
	t.Parallel()
	_, ex := beginTxPG(t)
	ctx := context.Background()

	tbl := table.NewTable[fixtures.RoomMapping](dialect.PostgreSQLDialect{})

	now := int64(1700000000)
	mapping := &fixtures.RoomMapping{
		MappingRoomID: fixtures.MappingRoomID{ID: 100},
		TeacherKey:    fixtures.TeacherKey{Key: fixtures.TeacherID(200)},
		From:          now,
		To:            now + 7200,
		CreatedAt:     now,
	}

	inserted, err := tbl.Insert(ctx, ex, mapping)
	require.NoError(t, err, "insert failed: %s", tbl.Internals().InsertSQL())
	assert.Equal(t, mapping.MappingRoomID.ID, inserted.MappingRoomID.ID)
	assert.Equal(t, mapping.TeacherKey.Key, inserted.TeacherKey.Key)

	fetched, err := tbl.GetByPK(ctx, ex, int64(100), int64(200))
	require.NoError(t, err)
	assert.Equal(t, mapping.MappingRoomID.ID, fetched.MappingRoomID.ID)
	assert.Equal(t, mapping.TeacherKey.Key, fetched.TeacherKey.Key)

	fetched.To = now + 10800
	err = tbl.Update(ctx, ex, fetched)
	require.NoError(t, err)

	updated, err := tbl.GetByPK(ctx, ex, int64(100), int64(200))
	require.NoError(t, err)
	assert.Equal(t, now+10800, updated.To)

	list, err := tbl.List(ctx, ex)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	err = tbl.Delete(ctx, ex, int64(100), int64(200))
	require.NoError(t, err)

	_, err = tbl.GetByPK(ctx, ex, int64(100), int64(200))
	assert.Error(t, err)
}

func TestFunctional_ListWithFilters_PostgreSQL(t *testing.T) {
	t.Parallel()
	_, ex := beginTxPG(t)
	ctx := context.Background()

	tbl := table.NewTable[fixtures.UserWithAge](dialect.PostgreSQLDialect{})

	users := []*fixtures.UserWithAge{
		{Name: "Alice", Email: "alice@test.com", Age: 25},
		{Name: "Bob", Email: "bob@test.com", Age: 30},
		{Name: "Charlie", Email: "charlie@test.com", Age: 35},
		{Name: "Diana", Email: "diana@test.com", Age: 40},
	}
	for _, u := range users {
		_, err := tbl.Insert(ctx, ex, u)
		require.NoError(t, err)
	}

	t.Run("List without filters returns all rows", func(t *testing.T) {
		result, err := tbl.List(ctx, ex)
		require.NoError(t, err)
		assert.Len(t, result, 4)
	})

	t.Run("List with Eq filter", func(t *testing.T) {
		result, err := tbl.List(ctx, ex, table.AndFilter(
			table.Field("Name", table.And, table.Eq("Alice")),
		))
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "Alice", result[0].Name)
	})

	t.Run("List with Gt filter", func(t *testing.T) {
		result, err := tbl.List(ctx, ex, table.AndFilter(
			table.Field("Age", table.And, table.Gt(30)),
		))
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("List with Lt filter", func(t *testing.T) {
		result, err := tbl.List(ctx, ex, table.AndFilter(
			table.Field("Age", table.And, table.Lt(30)),
		))
		require.NoError(t, err)
		assert.Len(t, result, 1)
	})

	t.Run("List with Between filter", func(t *testing.T) {
		result, err := tbl.List(ctx, ex, table.AndFilter(
			table.Field("Age", table.And, table.Between(30, 40)),
		))
		require.NoError(t, err)
		assert.Len(t, result, 3)
	})

	t.Run("List with In filter", func(t *testing.T) {
		result, err := tbl.List(ctx, ex, table.AndFilter(
			table.Field("Age", table.And, table.In(25, 35)),
		))
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("List with multiple fields AND", func(t *testing.T) {
		result, err := tbl.List(ctx, ex, table.AndFilter(
			table.Field("Name", table.And, table.Eq("Bob")),
			table.Field("Age", table.And, table.Eq(30)),
		))
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "Bob", result[0].Name)
	})

	t.Run("List with OR between fields", func(t *testing.T) {
		result, err := tbl.List(ctx, ex, table.OrFilter(
			table.Field("Name", table.And, table.Eq("Alice")),
			table.Field("Name", table.And, table.Eq("Charlie")),
		))
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})
}

func TestFunctional_CRUD_FullRoomMapping_PostgreSQL(t *testing.T) {
	t.Parallel()
	_, ex := beginTxPG(t)
	ctx := context.Background()

	tbl := table.NewTable[fixtures.FullRoomMapping](dialect.PostgreSQLDialect{})

	now := int64(1700000000)
	fullMapping := &fixtures.FullRoomMapping{
		RoomMapping: fixtures.RoomMapping{
			MappingRoomID: fixtures.MappingRoomID{ID: 300},
			TeacherKey:    fixtures.TeacherKey{Key: fixtures.TeacherID(400)},
			From:          now,
			To:            now + 3600,
			CreatedAt:     now,
		},
		Author: "John Doe",
	}

	tbl.Internals().InsertSQL()
	inserted, err := tbl.Insert(ctx, ex, fullMapping)
	require.NoError(t, err, "insert failed: %s", tbl.Internals().InsertSQL())
	assert.Equal(t, fullMapping.MappingRoomID.ID, inserted.MappingRoomID.ID)
	assert.Equal(t, fullMapping.Author, inserted.Author)

	fetched, err := tbl.GetByPK(ctx, ex, int64(300), int64(400))
	require.NoError(t, err)
	assert.Equal(t, fullMapping.MappingRoomID.ID, fetched.MappingRoomID.ID)
	assert.Equal(t, fullMapping.Author, fetched.Author)

	fetched.Author = "Jane Smith"
	err = tbl.Update(ctx, ex, fetched)
	require.NoError(t, err)

	updated, err := tbl.GetByPK(ctx, ex, int64(300), int64(400))
	require.NoError(t, err)
	assert.Equal(t, "Jane Smith", updated.Author)

	list, err := tbl.List(ctx, ex)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	err = tbl.Delete(ctx, ex, int64(300), int64(400))
	require.NoError(t, err)

	_, err = tbl.GetByPK(ctx, ex, int64(300), int64(400))
	assert.Error(t, err)
}
