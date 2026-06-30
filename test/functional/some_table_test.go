//go:build functional

package functional

import (
	"context"
	"testing"
	"time"

	"github.com/mirrorru/qqm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/test/fixtures"
)

func TestFunctional_SomeTable_Meta(t *testing.T) {
	t.Parallel()
	tbl := qqm.NewTable[fixtures.SomeTable](dialect.PostgreSQLDialect{})
	m := tbl.Internals().Meta()

	assert.Equal(t, "some_table", m.TableName)
	require.Len(t, m.PKFields, 1)
	assert.Equal(t, "some_id", m.PKFields[0].Column)
	assert.Equal(t, 1, m.PKFields[0].PkOrder)

	assert.Contains(t, m.Columns, "some_id")
	assert.Contains(t, m.Columns, "field_rw")
	assert.Contains(t, m.Columns, "field_ro")

	insertCols := m.InsertColumns()
	assert.NotContains(t, insertCols, "some_id", "auto PK should not be in InsertColumns")
	assert.Contains(t, insertCols, "field_rw")
	assert.NotContains(t, insertCols, "field_ro", "auto field should not be in InsertColumns")

	updateCols := m.UpdateColumns()
	assert.Contains(t, updateCols, "field_rw")
	assert.NotContains(t, updateCols, "some_id", "PK should not be in UpdateColumns")
	assert.NotContains(t, updateCols, "field_ro", "auto field should not be in UpdateColumns")
}

func TestFunctional_SomeTable_CRUD_PostgreSQL(t *testing.T) {
	t.Parallel()
	_, ex := beginTxPG(t)
	ctx := context.Background()

	tbl := qqm.NewTable[fixtures.SomeTable](dialect.PostgreSQLDialect{})

	now := time.Now().Truncate(time.Second)
	row := &fixtures.SomeTable{
		FieldRW: "hello",
		FieldRO: now,
	}

	inserted, err := tbl.Insert(ctx, ex, row)
	require.NoError(t, err)

	fetched, err := tbl.GetByPK(ctx, ex, int64(inserted.SomeID))
	require.NoError(t, err)
	assert.Equal(t, inserted.SomeID, fetched.SomeID)
	assert.Equal(t, "hello", fetched.FieldRW)

	fetched.FieldRW = "world"
	returned, err := tbl.Update(ctx, ex, fetched)
	require.NoError(t, err)
	assert.NotNil(t, returned)
	assert.Equal(t, "world", returned.FieldRW)

	updated, err := tbl.GetByPK(ctx, ex, int64(inserted.SomeID))
	require.NoError(t, err)
	assert.Equal(t, "world", updated.FieldRW)

	list, err := tbl.List(ctx, ex)
	require.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, inserted.SomeID, list[0].SomeID)

	err = tbl.Delete(ctx, ex, int64(inserted.SomeID))
	require.NoError(t, err)

	_, err = tbl.GetByPK(ctx, ex, int64(inserted.SomeID))
	assert.Error(t, err)
}

func TestFunctional_SomeTable_QueryRow_PostgreSQL(t *testing.T) {
	t.Parallel()
	tx, _ := beginTxPG(t)
	ctx := context.Background()

	_, err := tx.ExecContext(ctx,
		`INSERT INTO some_table (some_id, field_rw) VALUES ($1, $2)`,
		int64(42), "queryrow-test",
	)
	require.NoError(t, err)

	var id int64
	var rw string
	var ro time.Time
	row := tx.QueryRowContext(ctx,
		`SELECT some_id, field_rw, field_ro FROM some_table WHERE some_id = $1`,
		int64(42),
	)
	err = row.Scan(&id, &rw, &ro)
	require.NoError(t, err)
	assert.Equal(t, int64(42), id)
	assert.Equal(t, "queryrow-test", rw)
	assert.False(t, ro.IsZero())
}
