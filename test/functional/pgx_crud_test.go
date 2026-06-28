// Created at 2026-06-28
//go:build functional

package functional

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/executor"
	"github.com/mirrorru/qqm/table"
	"github.com/mirrorru/qqm/test/fixtures"
)

// openTestPGX открывает подключение к тестовой БД PostgreSQL через pgx/v5.
func openTestPGX(t *testing.T) *pgx.Conn {
	t.Helper()
	conn, err := pgx.Connect(context.Background(), pgDSN())
	require.NoError(t, err, "failed to connect via pgx")
	t.Cleanup(func() {
		_ = conn.Close(context.Background())
	})
	return conn
}

func TestFunctional_PGX_CRUD_Rooms(t *testing.T) {
	conn := openTestPGX(t)
	ex := executor.NewPGXAdapter(conn)
	ctx := context.Background()

	_, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS rooms (
			id BIGSERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			square DOUBLE PRECISION NOT NULL,
			created_at BIGINT NOT NULL DEFAULT 0
		)
	`)
	require.NoError(t, err)
	defer func() { _, _ = conn.Exec(ctx, `DROP TABLE IF EXISTS rooms`) }()

	_, err = conn.Exec(ctx, `DELETE FROM rooms`)
	require.NoError(t, err)

	tbl := table.NewTable[*fixtures.Rooms](dialect.PostgreSQLDialect{})

	now := int64(1700000000)
	room := &fixtures.Rooms{
		Name:      "PGX Room",
		Square:    75.0,
		CreatedAt: now,
	}

	inserted, err := tbl.Insert(ctx, ex, room)
	require.NoError(t, err)
	assert.Equal(t, room.Name, inserted.Name)
	assert.Equal(t, room.Square, inserted.Square)
	assert.NotZero(t, inserted.ID)

	fetched, err := tbl.GetByKey(ctx, ex, inserted.ID)
	require.NoError(t, err)
	assert.Equal(t, inserted.ID, fetched.ID)
	assert.Equal(t, "PGX Room", fetched.Name)

	fetched.Name = "PGX Room Updated"
	err = tbl.Update(ctx, ex, fetched)
	require.NoError(t, err)

	updated, err := tbl.GetByKey(ctx, ex, inserted.ID)
	require.NoError(t, err)
	assert.Equal(t, "PGX Room Updated", updated.Name)

	list, err := tbl.List(ctx, ex)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	err = tbl.Delete(ctx, ex, inserted.ID)
	require.NoError(t, err)

	_, err = tbl.GetByKey(ctx, ex, inserted.ID)
	assert.Error(t, err)
}

func TestFunctional_PGX_ListWithFilters(t *testing.T) {
	conn := openTestPGX(t)
	ex := executor.NewPGXAdapter(conn)
	ctx := context.Background()

	_, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS user_with_age (
			id BIGINT NOT NULL,
			name TEXT NOT NULL,
			email TEXT NOT NULL,
			age BIGINT NOT NULL
		)
	`)
	require.NoError(t, err)
	defer func() { _, _ = conn.Exec(ctx, `DROP TABLE IF EXISTS user_with_age`) }()

	_, err = conn.Exec(ctx, `DELETE FROM user_with_age`)
	require.NoError(t, err)

	tbl := table.NewTable[*fixtures.UserWithAge](dialect.PostgreSQLDialect{})

	users := []*fixtures.UserWithAge{
		{ID: 1, Name: "Alice", Email: "alice@test.com", Age: 25},
		{ID: 2, Name: "Bob", Email: "bob@test.com", Age: 30},
		{ID: 3, Name: "Charlie", Email: "charlie@test.com", Age: 35},
		{ID: 4, Name: "Diana", Email: "diana@test.com", Age: 40},
	}
	for _, u := range users {
		_, err := tbl.Insert(ctx, ex, u)
		require.NoError(t, err)
	}

	result, err := tbl.List(ctx, ex)
	require.NoError(t, err)
	assert.Len(t, result, 4)

	result, err = tbl.List(ctx, ex, table.AndFilter(
		table.Field("Name", table.And, table.Eq("Alice")),
	))
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "Alice", result[0].Name)

	result, err = tbl.List(ctx, ex, table.AndFilter(
		table.Field("Age", table.And, table.Gt(30)),
	))
	require.NoError(t, err)
	assert.Len(t, result, 2)

	result, err = tbl.List(ctx, ex, table.AndFilter(
		table.Field("Age", table.And, table.Between(30, 40)),
	))
	require.NoError(t, err)
	assert.Len(t, result, 3)
}

func TestFunctional_PGX_CRUD_RoomMapping(t *testing.T) {
	conn := openTestPGX(t)
	ex := executor.NewPGXAdapter(conn)
	ctx := context.Background()

	_, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS room_mapping (
			room_id BIGINT NOT NULL,
			teacher_ID BIGINT NOT NULL,
			time_from BIGINT NOT NULL,
			time_to BIGINT NOT NULL,
			created_at BIGINT NOT NULL DEFAULT 0,
			PRIMARY KEY (room_id, teacher_ID)
		)
	`)
	require.NoError(t, err)
	defer func() { _, _ = conn.Exec(ctx, `DROP TABLE IF EXISTS room_mapping`) }()

	_, err = conn.Exec(ctx, `DELETE FROM room_mapping`)
	require.NoError(t, err)

	tbl := table.NewTable[*fixtures.RoomMapping](dialect.PostgreSQLDialect{})

	now := int64(1700000000)
	mapping := &fixtures.RoomMapping{
		MappingRoomID: fixtures.MappingRoomID{ID: 100},
		TeacherKey:    fixtures.TeacherKey{Key: fixtures.TeacherID(200)},
		From:          now,
		To:            now + 7200,
		CreatedAt:     now,
	}

	inserted, err := tbl.Insert(ctx, ex, mapping)
	require.NoError(t, err)
	assert.Equal(t, mapping.MappingRoomID.ID, inserted.MappingRoomID.ID)
	assert.Equal(t, mapping.TeacherKey.Key, inserted.TeacherKey.Key)

	fetched, err := tbl.GetByKey(ctx, ex, int64(100), int64(200))
	require.NoError(t, err)
	assert.Equal(t, mapping.MappingRoomID.ID, fetched.MappingRoomID.ID)

	fetched.To = now + 10800
	err = tbl.Update(ctx, ex, fetched)
	require.NoError(t, err)

	updated, err := tbl.GetByKey(ctx, ex, int64(100), int64(200))
	require.NoError(t, err)
	assert.Equal(t, now+10800, updated.To)

	list, err := tbl.List(ctx, ex)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	err = tbl.Delete(ctx, ex, int64(100), int64(200))
	require.NoError(t, err)

	_, err = tbl.GetByKey(ctx, ex, int64(100), int64(200))
	assert.Error(t, err)
}

func TestFunctional_PGX_CRUD_WithTx(t *testing.T) {
	conn := openTestPGX(t)
	ctx := context.Background()

	_, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS rooms (
			id BIGSERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			square DOUBLE PRECISION NOT NULL,
			created_at BIGINT NOT NULL DEFAULT 0
		)
	`)
	require.NoError(t, err)
	defer func() { _, _ = conn.Exec(ctx, `DROP TABLE IF EXISTS rooms`) }()

	_, err = conn.Exec(ctx, `DELETE FROM rooms`)
	require.NoError(t, err)

	tbl := table.NewTable[*fixtures.Rooms](dialect.PostgreSQLDialect{})

	t.Run("commit transaction via pgx", func(t *testing.T) {
		tx, err := conn.Begin(ctx)
		require.NoError(t, err)

		ex := executor.NewPGXTxAdapter(tx)

		inserted, err := tbl.Insert(ctx, ex, &fixtures.Rooms{
			Name:   "PGX Tx Room",
			Square: 100.0,
		})
		require.NoError(t, err)
		assert.NotZero(t, inserted.ID)

		err = tx.Commit(ctx)
		require.NoError(t, err)

		fetched, err := tbl.GetByKey(ctx, executor.NewPGXAdapter(conn), inserted.ID)
		require.NoError(t, err)
		assert.Equal(t, "PGX Tx Room", fetched.Name)
	})

	t.Run("rollback transaction via pgx", func(t *testing.T) {
		tx, err := conn.Begin(ctx)
		require.NoError(t, err)

		ex := executor.NewPGXTxAdapter(tx)

		inserted, err := tbl.Insert(ctx, ex, &fixtures.Rooms{
			Name:   "PGX Rollback Room",
			Square: 200.0,
		})
		require.NoError(t, err)
		assert.NotZero(t, inserted.ID)

		err = tx.Rollback(ctx)
		require.NoError(t, err)

		_, err = tbl.GetByKey(ctx, executor.NewPGXAdapter(conn), inserted.ID)
		assert.Error(t, err, "should not find rolled-back row")
	})

	t.Run("GetByKey within pgx transaction", func(t *testing.T) {
		ex := executor.NewPGXAdapter(conn)
		inserted, err := tbl.Insert(ctx, ex, &fixtures.Rooms{
			Name:   "PGX Tx GetByKey",
			Square: 300.0,
		})
		require.NoError(t, err)

		tx, err := conn.Begin(ctx)
		require.NoError(t, err)

		txEx := executor.NewPGXTxAdapter(tx)
		fetched, err := tbl.GetByKey(ctx, txEx, inserted.ID)
		require.NoError(t, err)
		assert.Equal(t, "PGX Tx GetByKey", fetched.Name)

		err = tx.Commit(ctx)
		require.NoError(t, err)
	})
}
