//go:build smoke

package smoke

import (
	"context"
	"database/sql"
	"testing"

	"github.com/mirrorru/qqm/txproc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mirrorru/qqm"
	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/test/fixtures"
	_ "modernc.org/sqlite"
)

func TestSmoke_CreateTable_CreateClause(t *testing.T) {
	t.Parallel()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	ex := txproc.NewDBAdapterVal(db)
	ctx := context.Background()

	tbl := qqm.NewTable[fixtures.RowWithCreate](dialect.SQLiteDialect{})
	ddl := tbl.Internals().CreateTableSQL()
	_, err = db.Exec(ddl)
	require.NoError(t, err, "CREATE TABLE via CreateTableSQL() с create=")

	// проверяем DDL: DEFAULT-значения должны быть в схеме
	assert.Contains(t, ddl, `DEFAULT 'unknown'`)
	assert.Contains(t, ddl, `DEFAULT 'active'`)
	assert.Contains(t, ddl, `DEFAULT 0`)

	// Insert с явными значениями
	r := &fixtures.RowWithCreate{
		Name:   "Bob",
		Status: "inactive",
		Count:  42,
	}
	ins, err := tbl.Insert(ctx, ex, r)
	require.NoError(t, err)
	assert.Equal(t, "Bob", ins.Name)
	assert.Equal(t, "inactive", ins.Status)
	assert.Equal(t, 42, ins.Count)

	// raw SQL: вставка только PK — defaults должны сработать
	_, err = db.ExecContext(ctx, `INSERT INTO row_with_create (id) VALUES (NULL)`)
	require.NoError(t, err)

	list, err := tbl.List(ctx, ex)
	require.NoError(t, err)
	assert.Len(t, list, 2)

	// вторая строка (из raw SQL) должна иметь default-значения
	// ищем строку с Name='unknown'
	found := false
	for _, row := range list {
		if row.Name == "unknown" {
			found = true
			assert.Equal(t, "active", row.Status)
			assert.Equal(t, 0, row.Count)
		}
	}
	assert.True(t, found, "default values should be applied via raw SQL insert")
}

func TestSmoke_CreateTable_RefFK(t *testing.T) {
	t.Parallel()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	_, err = db.Exec(`PRAGMA foreign_keys = ON`)
	require.NoError(t, err)

	ex := txproc.NewDBAdapterVal(db)
	ctx := context.Background()

	// создаём таблицу users (родитель)
	usersTbl := qqm.NewTable[fixtures.User](dialect.SQLiteDialect{})
	_, err = db.Exec(usersTbl.Internals().CreateTableSQL())
	require.NoError(t, err)

	// создаём таблицу с ref=users.id
	ordersTbl := qqm.NewTable[fixtures.Order](dialect.SQLiteDialect{})
	ddl := ordersTbl.Internals().CreateTableSQL()
	_, err = db.Exec(ddl)
	require.NoError(t, err, "CREATE TABLE с REFERENCES users(id)")

	// проверяем DDL: REFERENCES должен быть в схеме
	assert.Contains(t, ddl, `REFERENCES users(id)`)

	// вставляем пользователя
	user, err := usersTbl.Insert(ctx, ex, &fixtures.User{Name: "Alice", Email: "alice@test"})
	require.NoError(t, err)

	// вставляем заказ с существующим user_id
	order, err := ordersTbl.Insert(ctx, ex, &fixtures.Order{UserID: user.ID, Amount: 99.99})
	require.NoError(t, err)
	assert.Equal(t, user.ID, order.UserID)
	assert.Equal(t, 99.99, order.Amount)

	// FK constraint: вставка с несуществующим user_id должна упасть
	_, err = ordersTbl.Insert(ctx, ex, &fixtures.Order{UserID: 99999, Amount: 1.0})
	assert.Error(t, err, "FK constraint should reject invalid user_id")

	// GetByPK
	fetched, err := ordersTbl.GetByPK(ctx, ex, order.ID)
	require.NoError(t, err)
	assert.Equal(t, user.ID, fetched.UserID)
}

func TestSmoke_CreateTable_CompositePK(t *testing.T) {
	t.Parallel()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	ex := txproc.NewDBAdapterVal(db)
	ctx := context.Background()

	tbl := qqm.NewTable[fixtures.OrgUser](dialect.SQLiteDialect{})
	ddl := tbl.Internals().CreateTableSQL()
	_, err = db.Exec(ddl)
	require.NoError(t, err, "CREATE TABLE с составным PK")

	// Insert с составным ключом
	r1 := &fixtures.OrgUser{OrgID: 1, UserID: 100, Name: "u1", Email: "u1@test"}
	ins1, err := tbl.Insert(ctx, ex, r1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), ins1.OrgID)
	assert.Equal(t, int64(100), ins1.UserID)

	// вторая строка с другим составным ключом
	r2 := &fixtures.OrgUser{OrgID: 1, UserID: 200, Name: "u2", Email: "u2@test"}
	ins2, err := tbl.Insert(ctx, ex, r2)
	require.NoError(t, err)
	assert.Equal(t, int64(1), ins2.OrgID)
	assert.Equal(t, int64(200), ins2.UserID)

	// дубликат составного PK должен упасть
	_, err = tbl.Insert(ctx, ex, &fixtures.OrgUser{OrgID: 1, UserID: 100, Name: "dup", Email: "dup@test"})
	assert.Error(t, err, "duplicate composite PK should be rejected")

	// GetByPK по составному ключу
	fetched, err := tbl.GetByPK(ctx, ex, int64(1), int64(100))
	require.NoError(t, err)
	assert.Equal(t, "u1", fetched.Name)

	// List
	list, err := tbl.List(ctx, ex)
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestSmoke_CreateTable_CreateClauseAndSort(t *testing.T) {
	t.Parallel()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	ex := txproc.NewDBAdapterVal(db)
	ctx := context.Background()

	type sortCreateRow struct {
		ID     int64  `qqm:"pk;auto"`
		Name   string `qqm:"sort=1;create=DEFAULT 'unnamed'"`
		Amount int    `qqm:"sort=2,desc;create=DEFAULT 0"`
	}

	tbl := qqm.NewTable[sortCreateRow](dialect.SQLiteDialect{})
	ddl := tbl.Internals().CreateTableSQL()
	_, err = db.Exec(ddl)
	require.NoError(t, err, "CREATE TABLE с create= и sort")

	assert.Contains(t, ddl, `DEFAULT 'unnamed'`)
	assert.Contains(t, ddl, `DEFAULT 0`)

	// Insert с явными значениями
	r1 := &sortCreateRow{Name: "Bob", Amount: 500}
	ins1, err := tbl.Insert(ctx, ex, r1)
	require.NoError(t, err)
	assert.Equal(t, "Bob", ins1.Name)
	assert.Equal(t, 500, ins1.Amount)

	r2 := &sortCreateRow{Name: "Alice", Amount: 100}
	_, err = tbl.Insert(ctx, ex, r2)
	require.NoError(t, err)

	r3 := &sortCreateRow{Name: "Charlie", Amount: 300}
	_, err = tbl.Insert(ctx, ex, r3)
	require.NoError(t, err)

	// List: ORDER BY name ASC, amount DESC
	list, err := tbl.List(ctx, ex)
	require.NoError(t, err)
	require.Len(t, list, 3)

	// Alice (amount=100), Bob (amount=500), Charlie (amount=300) — по Name ASC
	assert.Equal(t, "Alice", list[0].Name)
	assert.Equal(t, "Bob", list[1].Name)
	assert.Equal(t, "Charlie", list[2].Name)
}
