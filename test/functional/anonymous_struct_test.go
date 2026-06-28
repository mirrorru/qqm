// Created at 2026-06-28
//go:build functional

package functional

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/table"
	"github.com/mirrorru/qqm/test/fixtures"
)

func TestFunctional_AnonymousStruct_TagParsing(t *testing.T) {
	tbl := table.NewTable[*fixtures.RowWithEmbeddedPK](dialect.PostgreSQLDialect{})
	m := tbl.Internals().Meta()

	assert.Equal(t, "row_with_embedded_pk", m.TableName)

	require.Len(t, m.PKFields, 1)
	assert.Equal(t, "id", m.PKFields[0].Column)
	assert.True(t, m.PKFields[0].IsPK)

	assert.Contains(t, m.Columns, "usr_name")
	assert.Contains(t, m.Columns, "usr_email")
	assert.Contains(t, m.Columns, "status")
	assert.Contains(t, m.Columns, "id")
}

func TestFunctional_AnonymousStruct_DeepNesting(t *testing.T) {
	tbl := table.NewTable[*fixtures.RowWithDeepEmbed](dialect.PostgreSQLDialect{})
	m := tbl.Internals().Meta()

	assert.Contains(t, m.Columns, "nested_deep_name")
	assert.Contains(t, m.Columns, "nested_deep_email")
	assert.Contains(t, m.Columns, "nested_extra")
	assert.Contains(t, m.Columns, "top_field")
}

func TestFunctional_AnonymousStruct_AutoReadonly(t *testing.T) {
	tbl := table.NewTable[*fixtures.RowWithAutoEmbed](dialect.PostgreSQLDialect{})
	m := tbl.Internals().Meta()

	insertCols := m.InsertColumns()
	assert.Contains(t, insertCols, "id")
	assert.Contains(t, insertCols, "value")
	assert.NotContains(t, insertCols, "created_at", "auto field from embedded struct should not be in InsertColumns")
	assert.Contains(t, insertCols, "updated_at", "readonly field should be in InsertColumns (set once on insert)")

	updateCols := m.UpdateColumns()
	assert.Contains(t, updateCols, "value")
	assert.NotContains(t, updateCols, "id", "PK should not be in UpdateColumns")
	assert.NotContains(t, updateCols, "created_at", "auto field should not be in UpdateColumns")
	assert.NotContains(t, updateCols, "updated_at", "readonly field should not be in UpdateColumns")
}

func TestFunctional_AnonymousStruct_PKAuto(t *testing.T) {
	tbl := table.NewTable[*fixtures.RowWithPKAuto](dialect.PostgreSQLDialect{})
	m := tbl.Internals().Meta()

	require.Len(t, m.PKFields, 1)
	assert.Equal(t, "id", m.PKFields[0].Column)
	assert.True(t, m.PKFields[0].IsPK)
	assert.True(t, m.PKFields[0].IsAuto)

	insertCols := m.InsertColumns()
	assert.NotContains(t, insertCols, "id", "auto PK should not be in InsertColumns")
	assert.Contains(t, insertCols, "name")
}

func TestFunctional_NamedStructPrefix_TagParsing(t *testing.T) {
	tbl := table.NewTable[*fixtures.PersonWithAddress](dialect.PostgreSQLDialect{})
	m := tbl.Internals().Meta()

	assert.Equal(t, "person_with_address", m.TableName)

	assert.Contains(t, m.Columns, "home_city")
	assert.Contains(t, m.Columns, "home_street")
	assert.Contains(t, m.Columns, "home_zip")
	assert.Contains(t, m.Columns, "work_city")
	assert.Contains(t, m.Columns, "work_street")
	assert.Contains(t, m.Columns, "work_zip")
	assert.Contains(t, m.Columns, "name")
	assert.Contains(t, m.Columns, "id")

	require.Len(t, m.PKFields, 1)
	assert.Equal(t, "id", m.PKFields[0].Column)
}
