package qqm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/test/fixtures"
)

func TestTable_SQLite_SimpleKeySQL(t *testing.T) {
	tbl := NewTable[fixtures.User](dialect.SQLiteDialect{})

	assert.Equal(t, "users", tbl.Internals().Meta().TableName)
	assert.Contains(t, tbl.Internals().Meta().Columns, "name")
	assert.Contains(t, tbl.Internals().Meta().Columns, "email")

	insertSQL := tbl.Internals().InsertSQL()
	assert.NotEmpty(t, insertSQL)
	assert.Contains(t, insertSQL, `INSERT INTO users`)
	assert.Contains(t, insertSQL, `id`)
	assert.Contains(t, insertSQL, `name`)
	assert.Contains(t, insertSQL, `email`)
	assert.Contains(t, insertSQL, "RETURNING")

	updateSQL := tbl.Internals().UpdateSQL()
	assert.NotEmpty(t, updateSQL)
	assert.Contains(t, updateSQL, `UPDATE users`)
	assert.Contains(t, updateSQL, `SET name = ?`)
	assert.Contains(t, updateSQL, `WHERE id = ?`)

	selectSQL := tbl.Internals().SelectSQL()
	assert.NotEmpty(t, selectSQL)
	assert.Contains(t, selectSQL, `SELECT id, name, email`)
	assert.Contains(t, selectSQL, `FROM users`)
	assert.Contains(t, selectSQL, `WHERE id = ?`)

	deleteSQL := tbl.Internals().DeleteSQL()
	assert.NotEmpty(t, deleteSQL)
	assert.Contains(t, deleteSQL, `DELETE FROM users`)
	assert.Contains(t, deleteSQL, `WHERE id = ?`)
}

func TestTable_SQLite_QueryCaching(t *testing.T) {
	tbl := NewTable[fixtures.User](dialect.SQLiteDialect{})

	sql1 := tbl.Internals().InsertSQL()
	sql2 := tbl.Internals().InsertSQL()

	assert.Equal(t, sql1, sql2, "cached queries should be identical")
}

func TestTable_SQLite_CompositeKeySQL(t *testing.T) {
	tbl := NewTable[fixtures.OrgUser](dialect.SQLiteDialect{})

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
	tbl := NewTable[fixtures.UserWithAge](dialect.SQLiteDialect{})

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
	tbl := NewTable[fixtures.UserWithAge](dialect.PostgreSQLDialect{})

	insertSQL := tbl.Internals().InsertSQL()
	assert.Contains(t, insertSQL, "$1")
	assert.Contains(t, insertSQL, "$2")
	assert.Contains(t, insertSQL, "$3")

	selectSQL := tbl.Internals().SelectSQL()
	assert.Contains(t, selectSQL, "$1")
}

func TestTable_PostgreSQL_AnonymousStructSQL(t *testing.T) {
	tbl := NewTable[fixtures.RowWithEmbeddedPK](dialect.PostgreSQLDialect{})

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
	tbl := NewTable[fixtures.SomeTable](dialect.PostgreSQLDialect{})

	insertSQL := tbl.Internals().InsertSQL()
	assert.Contains(t, insertSQL, `INSERT INTO some_table`)
	assert.Contains(t, insertSQL, `field_rw`)
	assert.Contains(t, insertSQL, `$1`)
	assert.Contains(t, insertSQL, "RETURNING")
	assert.NotContains(t, insertSQL, `(some_id`, "auto PK should not be in INSERT columns list")
	assert.NotContains(t, insertSQL, `$2`, "auto PK should not have a placeholder")
	assert.NotRegexp(t, `field_ro.*VALUES`, insertSQL, "auto field should not be in INSERT columns")

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

func TestTable_NamedStructPrefix_SQLite(t *testing.T) {
	tbl := NewTable[fixtures.PersonWithAddress](dialect.SQLiteDialect{})

	meta := tbl.Internals().Meta()
	assert.Contains(t, meta.Columns, "home_city")
	assert.Contains(t, meta.Columns, "home_street")
	assert.Contains(t, meta.Columns, "home_zip")
	assert.Contains(t, meta.Columns, "work_city")
	assert.Contains(t, meta.Columns, "work_street")
	assert.Contains(t, meta.Columns, "work_zip")
	assert.Contains(t, meta.Columns, "name")
	assert.Contains(t, meta.Columns, "id")

	insertSQL := tbl.Internals().InsertSQL()
	assert.Contains(t, insertSQL, `home_city`)
	assert.Contains(t, insertSQL, `work_city`)

	selectSQL := tbl.Internals().SelectSQL()
	assert.Contains(t, selectSQL, `home_city, home_street, home_zip, work_city, work_street, work_zip`)
}

func TestTable_ListSQL_WithSort(t *testing.T) {
	tbl := NewTable[fixtures.UserWithSort](dialect.SQLiteDialect{})

	listSQL := tbl.Internals().ListSQL()
	assert.Contains(t, listSQL, `SELECT id, name, email, age FROM users`)
	assert.Contains(t, listSQL, `ORDER BY name ASC, email DESC`)
}

func TestTable_ListSQL_WithSort_MultiplePositions(t *testing.T) {
	tbl := NewTable[fixtures.UserWithSortMulti](dialect.SQLiteDialect{})

	listSQL := tbl.Internals().ListSQL()
	assert.Contains(t, listSQL, `SELECT id, name, email, age FROM user_with_sort_multi`)
	assert.Contains(t, listSQL, `ORDER BY email DESC, name ASC, age ASC`)
}

func TestTable_ListSQL_NoSort(t *testing.T) {
	tbl := NewTable[fixtures.User](dialect.SQLiteDialect{})

	listSQL := tbl.Internals().ListSQL()
	assert.Contains(t, listSQL, `SELECT id, name, email FROM users`)
	assert.NotContains(t, listSQL, `ORDER BY`)
}

func TestTable_ListSQL_WithSort_PostgreSQL(t *testing.T) {
	tbl := NewTable[fixtures.UserWithSort](dialect.PostgreSQLDialect{})

	listSQL := tbl.Internals().ListSQL()
	assert.Contains(t, listSQL, `ORDER BY name ASC, email DESC`)
}

func TestTable_ListSQL_WithSort_CompositeKey(t *testing.T) {
	type Row struct {
		OrgID  int64  `qqm:"pk"`
		UserID int64  `qqm:"pk"`
		Name   string `qqm:"sort=1,desc"`
		Value  int    `qqm:"sort=2"`
	}

	tbl := NewTable[Row](dialect.SQLiteDialect{})

	listSQL := tbl.Internals().ListSQL()
	assert.Contains(t, listSQL, `SELECT org_id, user_id, name, value FROM row`)
	assert.Contains(t, listSQL, `ORDER BY name DESC, value ASC`)
}

func TestTable_QueryListSQL_WithSort(t *testing.T) {
	type QRow struct {
		User  fixtures.UserWithSort
		Order fixtures.OrderWithSort
	}

	q, err := NewQuery[QRow](dialect.SQLiteDialect{})
	require.NoError(t, err)

	listSQL := q.qmeta.listSQL
	assert.Contains(t, listSQL, `ORDER BY t1.name ASC, t1.email DESC`)
}

func TestTable_CreateTableSQL_Basic(t *testing.T) {
	tbl := NewTable[fixtures.User](dialect.SQLiteDialect{})

	ddl := tbl.Internals().CreateTableSQL()
	assert.Contains(t, ddl, `CREATE TABLE users (`)
	assert.Contains(t, ddl, `id INTEGER PRIMARY KEY AUTOINCREMENT`)
	assert.Contains(t, ddl, `name TEXT NOT NULL`)
	assert.Contains(t, ddl, `email TEXT NOT NULL`)
}

func TestTable_CreateTableSQL_WithCreateClause(t *testing.T) {
	tbl := NewTable[fixtures.RowWithCreate](dialect.SQLiteDialect{})

	ddl := tbl.Internals().CreateTableSQL()
	assert.Contains(t, ddl, `CREATE TABLE`)
	assert.Contains(t, ddl, `name TEXT NOT NULL DEFAULT 'unknown'`)
	assert.Contains(t, ddl, `status TEXT NOT NULL DEFAULT 'active'`)
	assert.Contains(t, ddl, `count INTEGER NOT NULL DEFAULT 0`)
}

func TestTable_CreateTableSQL_WithRef(t *testing.T) {
	tbl := NewTable[fixtures.Order](dialect.SQLiteDialect{})

	ddl := tbl.Internals().CreateTableSQL()
	assert.Contains(t, ddl, `CREATE TABLE orders (`)
	assert.Contains(t, ddl, `user_id BIGINT NOT NULL REFERENCES users(id)`)
	assert.Contains(t, ddl, `amount DOUBLE PRECISION NOT NULL`)
}

func TestTable_CreateTableSQL_CompositeKey(t *testing.T) {
	tbl := NewTable[fixtures.OrgUser](dialect.SQLiteDialect{})

	ddl := tbl.Internals().CreateTableSQL()
	assert.Contains(t, ddl, `CREATE TABLE org_users (`)
	assert.Contains(t, ddl, `PRIMARY KEY (org_id, user_id)`)
}

func TestTable_CreateTableSQL_PostgreSQL(t *testing.T) {
	tbl := NewTable[fixtures.RowWithCreate](dialect.PostgreSQLDialect{})

	ddl := tbl.Internals().CreateTableSQL()
	assert.Contains(t, ddl, `DEFAULT 'unknown'`)
	assert.Contains(t, ddl, `DEFAULT 'active'`)
	assert.Contains(t, ddl, `DEFAULT 0`)
}

func TestTable_CreateTableSQL_SomeTable(t *testing.T) {
	tbl := NewTable[fixtures.SomeTable](dialect.SQLiteDialect{})

	ddl := tbl.Internals().CreateTableSQL()
	assert.Contains(t, ddl, `CREATE TABLE some_table (`)
	assert.Contains(t, ddl, `field_rw TEXT NOT NULL`)
	assert.Contains(t, ddl, `field_ro TIMESTAMPTZ NOT NULL`)
}
