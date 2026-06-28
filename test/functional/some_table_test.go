// Created at 2026-06-28
//go:build functional

package functional

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/executor"
	"github.com/mirrorru/qqm/table"
	"github.com/mirrorru/qqm/test/fixtures"
)

func TestFunctional_SomeTable_Meta(t *testing.T) {
	tbl := table.NewTable[*fixtures.SomeTable](dialect.PostgreSQLDialect{})
	m := tbl.Internals().Meta()

	assert.Equal(t, "some_table", m.TableName)
	require.Len(t, m.PKFields, 1)
	assert.Equal(t, "some_id", m.PKFields[0].Column)
	assert.Equal(t, 1, m.PKFields[0].PkOrder)

	assert.Contains(t, m.Columns, "some_id")
	assert.Contains(t, m.Columns, "field_rw")
	assert.Contains(t, m.Columns, "field_ro")

	insertCols := m.InsertColumns()
	assert.Contains(t, insertCols, "some_id")
	assert.Contains(t, insertCols, "field_rw")
	assert.NotContains(t, insertCols, "field_ro", "auto field should not be in InsertColumns")

	updateCols := m.UpdateColumns()
	assert.Contains(t, updateCols, "field_rw")
	assert.NotContains(t, updateCols, "some_id", "PK should not be in UpdateColumns")
	assert.NotContains(t, updateCols, "field_ro", "auto field should not be in UpdateColumns")
}

func TestFunctional_SomeTable_CRUD_PostgreSQL(t *testing.T) {
	db := openTestPG(t)
	defer func() { _ = db.Close() }()

	ex := executor.NewDBAdapter(db)
	ctx := context.Background()

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS some_table (
			some_id BIGINT PRIMARY KEY,
			field_rw TEXT NOT NULL,
			field_ro TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)
	require.NoError(t, err)
	defer func() {
		_, _ = db.Exec(`DROP TABLE IF EXISTS some_table`)
	}()

	_, err = db.Exec(`DELETE FROM some_table`)
	require.NoError(t, err)

	tbl := table.NewTable[*fixtures.SomeTable](dialect.PostgreSQLDialect{})

	now := time.Now().Truncate(time.Second)
	row := &fixtures.SomeTable{
		SomeID:  fixtures.SomeID(1),
		FieldRW: "hello",
		FieldRO: now,
	}

	inserted, err := tbl.Insert(ctx, ex, row)
	require.NoError(t, err)
	assert.Equal(t, fixtures.SomeID(1), inserted.SomeID)
	assert.Equal(t, "hello", inserted.FieldRW)

	fetched, err := tbl.GetByKey(ctx, ex, int64(1))
	require.NoError(t, err)
	assert.Equal(t, fixtures.SomeID(1), fetched.SomeID)
	assert.Equal(t, "hello", fetched.FieldRW)

	fetched.FieldRW = "world"
	err = tbl.Update(ctx, ex, fetched)
	require.NoError(t, err)

	updated, err := tbl.GetByKey(ctx, ex, int64(1))
	require.NoError(t, err)
	assert.Equal(t, "world", updated.FieldRW)

	list, err := tbl.List(ctx, ex)
	require.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, fixtures.SomeID(1), list[0].SomeID)

	err = tbl.Delete(ctx, ex, int64(1))
	require.NoError(t, err)

	_, err = tbl.GetByKey(ctx, ex, int64(1))
	assert.Error(t, err)
}

func TestFunctional_SomeTable_QueryRow_PostgreSQL(t *testing.T) {
	db := openTestPG(t)
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS some_table (
			some_id BIGINT PRIMARY KEY,
			field_rw TEXT NOT NULL,
			field_ro TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)
	require.NoError(t, err)
	defer func() {
		_, _ = db.Exec(`DROP TABLE IF EXISTS some_table`)
	}()

	_, err = db.Exec(`DELETE FROM some_table`)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx,
		`INSERT INTO some_table (some_id, field_rw) VALUES ($1, $2)`,
		int64(42), "queryrow-test",
	)
	require.NoError(t, err)

	var id int64
	var rw string
	var ro time.Time
	row := db.QueryRowContext(ctx,
		`SELECT some_id, field_rw, field_ro FROM some_table WHERE some_id = $1`,
		int64(42),
	)
	err = row.Scan(&id, &rw, &ro)
	require.NoError(t, err)
	assert.Equal(t, int64(42), id)
	assert.Equal(t, "queryrow-test", rw)
	assert.False(t, ro.IsZero())
}
