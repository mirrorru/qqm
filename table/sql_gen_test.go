// Created at 2026-06-28
package table

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/test/fixtures"
)

func TestTable_SQLite_SimpleKeySQL(t *testing.T) {
	tbl := NewTable[*fixtures.User](dialect.SQLiteDialect{})

	assert.Equal(t, "user", tbl.Internals().Meta().TableName)
	assert.Contains(t, tbl.Internals().Meta().Columns, "name")
	assert.Contains(t, tbl.Internals().Meta().Columns, "email")

	insertSQL := tbl.Internals().InsertSQL()
	assert.NotEmpty(t, insertSQL)
	assert.Contains(t, insertSQL, `INSERT INTO user`)
	assert.Contains(t, insertSQL, `id`)
	assert.Contains(t, insertSQL, `name`)
	assert.Contains(t, insertSQL, `email`)
	assert.Contains(t, insertSQL, "RETURNING")

	updateSQL := tbl.Internals().UpdateSQL()
	assert.NotEmpty(t, updateSQL)
	assert.Contains(t, updateSQL, `UPDATE user`)
	assert.Contains(t, updateSQL, `SET name = ?`)
	assert.Contains(t, updateSQL, `WHERE id = ?`)

	selectSQL := tbl.Internals().SelectSQL()
	assert.NotEmpty(t, selectSQL)
	assert.Contains(t, selectSQL, `SELECT id, name, email`)
	assert.Contains(t, selectSQL, `FROM user`)
	assert.Contains(t, selectSQL, `WHERE id = ?`)

	deleteSQL := tbl.Internals().DeleteSQL()
	assert.NotEmpty(t, deleteSQL)
	assert.Contains(t, deleteSQL, `DELETE FROM user`)
	assert.Contains(t, deleteSQL, `WHERE id = ?`)
}

func TestTable_SQLite_QueryCaching(t *testing.T) {
	tbl := NewTable[*fixtures.User](dialect.SQLiteDialect{})

	sql1 := tbl.Internals().InsertSQL()
	sql2 := tbl.Internals().InsertSQL()

	assert.Equal(t, sql1, sql2, "cached queries should be identical")
}

func TestTable_SQLite_CompositeKeySQL(t *testing.T) {
	tbl := NewTable[*fixtures.OrgUser](dialect.SQLiteDialect{})

	assert.Equal(t, "org_users", tbl.Internals().Meta().TableName)
	require.Len(t, tbl.Internals().Meta().PKFields, 2)
	assert.Equal(t, "org_id", tbl.Internals().Meta().PKFields[0].Column)
	assert.Equal(t, "user_id", tbl.Internals().Meta().PKFields[1].Column)

	insertSQL := tbl.Internals().InsertSQL()
	assert.Contains(t, insertSQL, `INSERT INTO org_users`)
	assert.Contains(t, insertSQL, `(org_id, user_id, name, email)`)
	assert.Contains(t, insertSQL, `VALUES (?, ?, ?, ?)`)

	updateSQL := tbl.Internals().UpdateSQL()
	assert.Contains(t, updateSQL, `SET name = ?, email = ?`)
	assert.Contains(t, updateSQL, `org_id = ?`)
	assert.Contains(t, updateSQL, `user_id = ?`)

	selectSQL := tbl.Internals().SelectSQL()
	assert.Contains(t, selectSQL, `SELECT org_id, user_id, name, email`)
	assert.Contains(t, selectSQL, `WHERE org_id = ? AND user_id = ?`)

	deleteSQL := tbl.Internals().DeleteSQL()
	assert.Contains(t, deleteSQL, `DELETE FROM org_users`)
	assert.Contains(t, deleteSQL, `WHERE org_id = ? AND user_id = ?`)
}

func TestTable_SQLite_UserWithAgeSQL(t *testing.T) {
	tbl := NewTable[*fixtures.UserWithAge](dialect.SQLiteDialect{})

	insertSQL := tbl.Internals().InsertSQL()
	assert.Contains(t, insertSQL, `INSERT INTO user_with_age`)
	assert.Contains(t, insertSQL, `id`)
	assert.Contains(t, insertSQL, `name`)
	assert.Contains(t, insertSQL, `email`)
	assert.Contains(t, insertSQL, `age`)
	assert.Contains(t, insertSQL, "RETURNING")

	updateSQL := tbl.Internals().UpdateSQL()
	assert.Contains(t, updateSQL, `UPDATE user_with_age`)
	assert.Contains(t, updateSQL, `SET name = ?`)
	assert.Contains(t, updateSQL, `WHERE id = ?`)

	selectSQL := tbl.Internals().SelectSQL()
	assert.Contains(t, selectSQL, `SELECT id, name, email, age`)
	assert.Contains(t, selectSQL, `FROM user_with_age`)
	assert.Contains(t, selectSQL, `WHERE id = ?`)

	deleteSQL := tbl.Internals().DeleteSQL()
	assert.Contains(t, deleteSQL, `DELETE FROM user_with_age`)
	assert.Contains(t, deleteSQL, `WHERE id = ?`)
}

func TestTable_PostgreSQL_SimpleKeySQL(t *testing.T) {
	tbl := NewTable[*fixtures.UserWithAge](dialect.PostgreSQLDialect{})

	insertSQL := tbl.Internals().InsertSQL()
	assert.Contains(t, insertSQL, "$1")
	assert.Contains(t, insertSQL, "$2")
	assert.Contains(t, insertSQL, "$3")

	selectSQL := tbl.Internals().SelectSQL()
	assert.Contains(t, selectSQL, "$1")
}

func TestTable_PostgreSQL_AnonymousStructSQL(t *testing.T) {
	tbl := NewTable[*fixtures.RowWithEmbeddedPK](dialect.PostgreSQLDialect{})

	insertSQL := tbl.Internals().InsertSQL()
	assert.Contains(t, insertSQL, `id`)
	assert.Contains(t, insertSQL, `usr_name`)
	assert.Contains(t, insertSQL, `usr_email`)
	assert.Contains(t, insertSQL, `status`)

	selectSQL := tbl.Internals().SelectSQL()
	assert.Contains(t, selectSQL, `usr_name`)
	assert.Contains(t, selectSQL, `usr_email`)
	assert.Contains(t, selectSQL, `WHERE id = $1`)

	updateSQL := tbl.Internals().UpdateSQL()
	assert.Contains(t, updateSQL, `SET usr_name = $1, usr_email = $2, status = $3`)
	assert.Contains(t, updateSQL, `WHERE id = $4`)
}

func TestTable_PostgreSQL_SomeTableSQL(t *testing.T) {
	tbl := NewTable[*fixtures.SomeTable](dialect.PostgreSQLDialect{})

	insertSQL := tbl.Internals().InsertSQL()
	assert.Contains(t, insertSQL, `INSERT INTO some_table`)
	assert.Contains(t, insertSQL, `some_id`)
	assert.Contains(t, insertSQL, `field_rw`)
	assert.NotRegexp(t, `field_ro.*VALUES`, insertSQL, "auto field should not be in INSERT columns")
	assert.Contains(t, insertSQL, "$1")
	assert.Contains(t, insertSQL, "$2")
	assert.Contains(t, insertSQL, "RETURNING")

	updateSQL := tbl.Internals().UpdateSQL()
	assert.Contains(t, updateSQL, `UPDATE some_table`)
	assert.Contains(t, updateSQL, `SET field_rw = $1`)
	assert.Contains(t, updateSQL, `WHERE some_id = $2`)

	selectSQL := tbl.Internals().SelectSQL()
	assert.Contains(t, selectSQL, `SELECT some_id, field_rw, field_ro`)
	assert.Contains(t, selectSQL, `FROM some_table`)
	assert.Contains(t, selectSQL, `WHERE some_id = $1`)

	deleteSQL := tbl.Internals().DeleteSQL()
	assert.Contains(t, deleteSQL, `DELETE FROM some_table`)
	assert.Contains(t, deleteSQL, `WHERE some_id = $1`)
}
