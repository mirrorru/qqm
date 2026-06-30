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
	"github.com/mirrorru/qqm/test/fixtures"
	_ "modernc.org/sqlite"
)

func TestSmoke_CRUD_Rooms(t *testing.T) {
	t.Parallel()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	ex := qqm.NewDBAdapterVal(db)
	ctx := context.Background()

	_, err = db.Exec(`
		CREATE TABLE rooms (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			square REAL NOT NULL,
			created_at INTEGER NOT NULL DEFAULT 123
		)
	`)
	require.NoError(t, err)

	tbl := qqm.NewTable[fixtures.Rooms](dialect.SQLiteDialect{})

	createdAtExpect := int64(123)
	room := &fixtures.Rooms{
		Name:   "Conference Room A",
		Square: 50.5,
	}

	inserted, err := tbl.Insert(ctx, ex, room)
	require.NoError(t, err)
	assert.Equal(t, room.Name, inserted.Name)
	assert.Equal(t, room.Square, inserted.Square)
	assert.NotZero(t, inserted.ID, "auto-generated ID should not be zero")
	assert.Equal(t, createdAtExpect, inserted.CreatedAt)

	fetched, err := tbl.GetByPK(ctx, ex, inserted.ID)
	require.NoError(t, err)
	assert.Equal(t, inserted.ID, fetched.ID)
	assert.Equal(t, room.Name, fetched.Name)
	assert.Equal(t, room.Square, fetched.Square)
	assert.Equal(t, createdAtExpect, fetched.CreatedAt)

	fetched.Name = "Conference Room B"
	fetched.Square = 60.0
	fetched.CreatedAt = -createdAtExpect
	returned, err := tbl.Update(ctx, ex, fetched)
	require.NoError(t, err)
	assert.NotNil(t, returned)
	assert.Equal(t, "Conference Room B", returned.Name)
	assert.Equal(t, 60.0, returned.Square)
	assert.Equal(t, createdAtExpect, returned.CreatedAt, tbl.Internals().UpdateSQL())

	updated, err := tbl.GetByPK(ctx, ex, inserted.ID)
	require.NoError(t, err)
	assert.Equal(t, "Conference Room B", updated.Name)
	assert.Equal(t, 60.0, updated.Square)
	assert.Equal(t, createdAtExpect, updated.CreatedAt)

	list, err := tbl.List(ctx, ex)
	require.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, inserted.ID, list[0].ID)

	err = tbl.Delete(ctx, ex, inserted.ID)
	require.NoError(t, err)

	_, err = tbl.GetByPK(ctx, ex, inserted.ID)
	assert.Error(t, err)
}

func TestSmoke_CRUD_RoomMapping(t *testing.T) {
	t.Parallel()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	ex := qqm.NewDBAdapterVal(db)
	ctx := context.Background()

	_, err = db.Exec(`
		CREATE TABLE room_mapping (
			room_id INTEGER NOT NULL,
			teacher_ID INTEGER NOT NULL,
			time_from INTEGER NOT NULL,
			time_to INTEGER NOT NULL,
			created_at INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (room_id, teacher_ID)
		)
	`)
	require.NoError(t, err)

	tbl := qqm.NewTable[fixtures.RoomMapping](dialect.SQLiteDialect{})

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

	fetched, err := tbl.GetByPK(ctx, ex, int64(100), int64(200))
	require.NoError(t, err)
	assert.Equal(t, mapping.MappingRoomID.ID, fetched.MappingRoomID.ID)
	assert.Equal(t, mapping.TeacherKey.Key, fetched.TeacherKey.Key)

	fetched.To = now + 10800
	returned, err := tbl.Update(ctx, ex, fetched)
	require.NoError(t, err)
	assert.NotNil(t, returned)
	assert.Equal(t, now+10800, returned.To)

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

func TestSmoke_ListWithFilters(t *testing.T) {
	t.Parallel()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	_, err = db.Exec(`
		CREATE TABLE user_with_age (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT NOT NULL,
			age INTEGER NOT NULL
		)
	`)
	require.NoError(t, err)

	ex := qqm.NewDBAdapterVal(db)
	ctx := context.Background()
	tbl := qqm.NewTable[fixtures.UserWithAge](dialect.SQLiteDialect{})

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

	t.Run("List without filters returns all rows", func(t *testing.T) {
		result, err := tbl.List(ctx, ex)
		require.NoError(t, err)
		assert.Len(t, result, 4)
	})

	t.Run("List with Eq filter", func(t *testing.T) {
		result, err := tbl.List(ctx, ex, qqm.AndFilter(
			qqm.Field("Name", qqm.And, qqm.Eq("Alice")),
		))
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "Alice", result[0].Name)
	})

	t.Run("List with Gt filter", func(t *testing.T) {
		result, err := tbl.List(ctx, ex, qqm.AndFilter(
			qqm.Field("Age", qqm.And, qqm.Gt(30)),
		))
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("List with Lt filter", func(t *testing.T) {
		result, err := tbl.List(ctx, ex, qqm.AndFilter(
			qqm.Field("Age", qqm.And, qqm.Lt(30)),
		))
		require.NoError(t, err)
		assert.Len(t, result, 1)
	})

	t.Run("List with Between filter", func(t *testing.T) {
		result, err := tbl.List(ctx, ex, qqm.AndFilter(
			qqm.Field("Age", qqm.And, qqm.Between(30, 40)),
		))
		require.NoError(t, err)
		assert.Len(t, result, 3)
	})

	t.Run("List with In filter", func(t *testing.T) {
		result, err := tbl.List(ctx, ex, qqm.AndFilter(
			qqm.Field("Age", qqm.And, qqm.In(25, 35)),
		))
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("List with multiple conditions AND on one field", func(t *testing.T) {
		result, err := tbl.List(ctx, ex, qqm.AndFilter(
			qqm.Field("Age", qqm.And, qqm.Gt(25), qqm.Lt(40)),
		))
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("List with multiple fields AND", func(t *testing.T) {
		result, err := tbl.List(ctx, ex, qqm.AndFilter(
			qqm.Field("Name", qqm.And, qqm.Eq("Bob")),
			qqm.Field("Age", qqm.And, qqm.Eq(30)),
		))
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "Bob", result[0].Name)
	})

	t.Run("List with OR between fields", func(t *testing.T) {
		result, err := tbl.List(ctx, ex, qqm.OrFilter(
			qqm.Field("Name", qqm.And, qqm.Eq("Alice")),
			qqm.Field("Name", qqm.And, qqm.Eq("Charlie")),
		))
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("List with multiple conditions OR on one field", func(t *testing.T) {
		result, err := tbl.List(ctx, ex, qqm.AndFilter(
			qqm.Field("Name", qqm.Or, qqm.Eq("Alice"), qqm.Eq("Diana")),
		))
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("List with Gte and Lte", func(t *testing.T) {
		result, err := tbl.List(ctx, ex, qqm.AndFilter(
			qqm.Field("Age", qqm.And, qqm.Gte(30), qqm.Lte(35)),
		))
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})
}

func TestSmoke_CRUD_FullRoomMapping(t *testing.T) {
	t.Parallel()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	ex := qqm.NewDBAdapterVal(db)
	ctx := context.Background()

	_, err = db.Exec(`
		CREATE TABLE full_room_mapping (
			room_id INTEGER NOT NULL,
			teacher_ID INTEGER NOT NULL,
			time_from INTEGER NOT NULL,
			time_to INTEGER NOT NULL,
			created_at INTEGER NOT NULL DEFAULT 0,
			author_name TEXT NOT NULL,
			PRIMARY KEY (room_id, teacher_ID)
		)
	`)
	require.NoError(t, err)

	tbl := qqm.NewTable[fixtures.FullRoomMapping](dialect.SQLiteDialect{})

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

	inserted, err := tbl.Insert(ctx, ex, fullMapping)
	require.NoError(t, err)
	assert.Equal(t, fullMapping.MappingRoomID.ID, inserted.MappingRoomID.ID)
	assert.Equal(t, fullMapping.Author, inserted.Author)

	fetched, err := tbl.GetByPK(ctx, ex, int64(300), int64(400))
	require.NoError(t, err)
	assert.Equal(t, fullMapping.MappingRoomID.ID, fetched.MappingRoomID.ID)
	assert.Equal(t, fullMapping.Author, fetched.Author)

	fetched.Author = "Jane Smith"
	returned, err := tbl.Update(ctx, ex, fetched)
	require.NoError(t, err)
	assert.NotNil(t, returned)
	assert.Equal(t, "Jane Smith", returned.Author)

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

func TestSmoke_CRUD_WithTx(t *testing.T) {
	t.Parallel()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	_, err = db.Exec(`
		CREATE TABLE rooms (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			square REAL NOT NULL,
			created_at INTEGER NOT NULL DEFAULT 0
		)
	`)
	require.NoError(t, err)

	ctx := context.Background()
	tbl := qqm.NewTable[fixtures.Rooms](dialect.SQLiteDialect{})

	t.Run("commit transaction", func(t *testing.T) {
		tx, err := db.BeginTx(ctx, nil)
		require.NoError(t, err)

		ex := qqm.NewTxAdapterVal(tx)

		inserted, err := tbl.Insert(ctx, ex, &fixtures.Rooms{
			Name:   "Tx Room",
			Square: 100.0,
		})
		require.NoError(t, err)
		assert.NotZero(t, inserted.ID)

		err = tx.Commit()
		require.NoError(t, err)

		fetched, err := tbl.GetByPK(ctx, qqm.NewDBAdapterVal(db), inserted.ID)
		require.NoError(t, err)
		assert.Equal(t, "Tx Room", fetched.Name)
	})

	t.Run("rollback transaction", func(t *testing.T) {
		tx, err := db.BeginTx(ctx, nil)
		require.NoError(t, err)

		ex := qqm.NewTxAdapterVal(tx)

		inserted, err := tbl.Insert(ctx, ex, &fixtures.Rooms{
			Name:   "Rollback Room",
			Square: 200.0,
		})
		require.NoError(t, err)
		assert.NotZero(t, inserted.ID)

		err = tx.Rollback()
		require.NoError(t, err)

		_, err = tbl.GetByPK(ctx, qqm.NewDBAdapterVal(db), inserted.ID)
		assert.Error(t, err, "should not find rolled-back row")
	})

	t.Run("GetByKey within transaction", func(t *testing.T) {
		ex := qqm.NewDBAdapterVal(db)
		inserted, err := tbl.Insert(ctx, ex, &fixtures.Rooms{
			Name:   "Tx GetByKey",
			Square: 300.0,
		})
		require.NoError(t, err)

		tx, err := db.BeginTx(ctx, nil)
		require.NoError(t, err)

		txEx := qqm.NewTxAdapterVal(tx)
		fetched, err := tbl.GetByPK(ctx, txEx, inserted.ID)
		require.NoError(t, err)
		assert.Equal(t, "Tx GetByKey", fetched.Name)

		err = tx.Commit()
		require.NoError(t, err)
	})
}
