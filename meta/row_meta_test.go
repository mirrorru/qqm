// Created on kilo-host at 2026-06-28
package meta

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUser — тестовая структура ROW
type TestUser struct {
	ID     int64  `qqm:"col=user_id;pk"`
	Name   string `qqm:"col=name"`
	Age    int    `qqm:"col=age"`
	Bio    string `qqm:"col=bio;readonly"`
	Secret string `qqm:"omit"`
}

func (*TestUser) SQLName() string { return "users" }

func TestBuildRowMeta_Basic(t *testing.T) {
	rm := BuildRowMeta(reflect.TypeOf(TestUser{}), "users")

	assert.Equal(t, "users", rm.TableName)
	assert.NotEmpty(t, rm.Fields)
}

func TestBuildRowMeta_PKField(t *testing.T) {
	rm := BuildRowMeta(reflect.TypeOf(TestUser{}), "users")

	require.NotEmpty(t, rm.PKFields)
	assert.Equal(t, "user_id", rm.PKFields[0].Column)
	assert.True(t, rm.PKFields[0].IsPK)
	assert.Equal(t, 1, rm.PKFields[0].PkOrder)
}

func TestBuildRowMeta_Columns(t *testing.T) {
	rm := BuildRowMeta(reflect.TypeOf(TestUser{}), "users")

	for _, col := range rm.Columns {
		assert.NotEqual(t, "secret", col, "omit field should not be in Columns")
	}
}

func TestBuildRowMeta_InsertColumns(t *testing.T) {
	rm := BuildRowMeta(reflect.TypeOf(TestUser{}), "users")
	cols := rm.InsertColumns()

	for _, col := range cols {
		assert.NotEqual(t, "secret", col, "omit should not be in InsertColumns")
	}

	// PK включаются (если не auto), name и age должны быть
	assert.Contains(t, cols, "user_id")
	assert.Contains(t, cols, "name")
	assert.Contains(t, cols, "age")
}

func TestBuildRowMeta_UpdateColumns(t *testing.T) {
	rm := BuildRowMeta(reflect.TypeOf(TestUser{}), "users")
	cols := rm.UpdateColumns()

	for _, col := range cols {
		assert.NotEqual(t, "user_id", col, "PK should not be in UpdateColumns")
		assert.NotEqual(t, "bio", col, "readonly should not be in UpdateColumns")
		assert.NotEqual(t, "secret", col, "omit should not be in UpdateColumns")
	}

	assert.Contains(t, cols, "name")
	assert.Contains(t, cols, "age")
}

func TestScanDest(t *testing.T) {
	rm := BuildRowMeta(reflect.TypeOf(TestUser{}), "users")

	user := &TestUser{
		ID:     42,
		Name:   "Alice",
		Age:    30,
		Bio:    "Developer",
		Secret: "hidden",
	}

	dest := rm.ScanDest(user)

	assert.Equal(t, len(rm.Columns), len(dest))

	for _, d := range dest {
		assert.NotNil(t, d)
	}
}

// Created at 2026-06-28
// TestBuildRowMeta_UnexportedAnonymous проверяет что неэкспортируемые anonymous поля пропускаются
func TestBuildRowMeta_UnexportedAnonymous(t *testing.T) {
	type unexportedKey int64 //nolint:unused
	type RowWithUnexported struct {
		unexportedKey        //nolint:unused // неэкспортируемое anonymous поле
		Name          string `qqm:"col=name"`
	}

	rm := BuildRowMeta(reflect.TypeOf(RowWithUnexported{}), "test")

	for _, f := range rm.Fields {
		assert.NotEqual(t, "unexportedKey", f.Name, "unexported anonymous field should be skipped")
	}
}

// Created at 2026-06-28
// TestBuildRowMeta_DuplicateColumns проверяет валидацию уникальности колонок
func TestBuildRowMeta_DuplicateColumns(t *testing.T) {
	type DuplicateRow struct {
		ID   int64  `qqm:"col=id;pk"`
		Name string `qqm:"col=id"`
	}

	assert.Panics(t, func() {
		BuildRowMeta(reflect.TypeOf(DuplicateRow{}), "test")
	}, "should panic on duplicate column names")
}

// Created at 2026-06-28
// TestBuildRowMeta_AnonymousStructFieldGroup проверяет anonymous struct как набор полей
func TestBuildRowMeta_AnonymousStructFieldGroup(t *testing.T) {
	type AuditFields struct {
		CreatedAt string `qqm:"col=created_at"`
		UpdatedAt string `qqm:"col=updated_at"`
	}
	type RowWithEmbed struct {
		ID int64 `qqm:"col=id;pk"`
		AuditFields
	}

	rm := BuildRowMeta(reflect.TypeOf(RowWithEmbed{}), "test")

	cols := rm.Columns
	assert.Contains(t, cols, "created_at")
	assert.Contains(t, cols, "updated_at")
	assert.Contains(t, cols, "id")
}

// Created at 2026-06-28
// TestBuildRowMeta_CompositeKey проверяет составной ключ (порядок по объявлению)
func TestBuildRowMeta_CompositeKey(t *testing.T) {
	type CompositeKey struct {
		OrgID  int64 `qqm:"col=org_id;pk"`
		UserID int64 `qqm:"col=user_id;pk"`
	}
	type RowWithCompositeKey struct {
		CompositeKey
		Name string `qqm:"col=name"`
	}

	rm := BuildRowMeta(reflect.TypeOf(RowWithCompositeKey{}), "test")

	assert.Len(t, rm.PKFields, 2)
	assert.Equal(t, "org_id", rm.PKFields[0].Column)
	assert.Equal(t, 1, rm.PKFields[0].PkOrder)
	assert.Equal(t, "user_id", rm.PKFields[1].Column)
	assert.Equal(t, 2, rm.PKFields[1].PkOrder)
}

// Created at 2026-06-28
// TestBuildRowMeta_AnonymousNonStructNotPK проверяет что anonymous non-struct не является PK
func TestBuildRowMeta_AnonymousNonStructNotPK(t *testing.T) {
	type EmbeddedID int64
	type Row struct {
		EmbeddedID
		Name string `qqm:"col=name"`
	}

	rm := BuildRowMeta(reflect.TypeOf(Row{}), "test")

	assert.Empty(t, rm.PKFields, "anonymous non-struct should not be PK without pk tag")
}

// Created at 2026-06-28
// TestBuildRowMeta_NamedStructPrefix проверяет префикс на именованных полях-структурах
func TestBuildRowMeta_NamedStructPrefix(t *testing.T) {
	type Address struct {
		City   string `qqm:"col=city"`
		Street string `qqm:"col=street"`
		Zip    string `qqm:"col=zip"`
	}
	type Person struct {
		ID          int64 `qqm:"pk"`
		Name        string
		HomeAddress Address `qqm:"prefix=home_"`
		WorkAddress Address `qqm:"prefix=work_"`
	}

	rm := BuildRowMeta(reflect.TypeOf(Person{}), "person")

	assert.Contains(t, rm.Columns, "home_city")
	assert.Contains(t, rm.Columns, "home_street")
	assert.Contains(t, rm.Columns, "home_zip")
	assert.Contains(t, rm.Columns, "work_city")
	assert.Contains(t, rm.Columns, "work_street")
	assert.Contains(t, rm.Columns, "work_zip")
	assert.Contains(t, rm.Columns, "id")
	assert.Contains(t, rm.Columns, "name")

	assert.Len(t, rm.PKFields, 1)
	assert.Equal(t, "id", rm.PKFields[0].Column)

	// home_ и work_ поля не должны быть PK
	for _, f := range rm.Fields {
		if f.Name == "City" || f.Name == "Street" || f.Name == "Zip" {
			assert.False(t, f.IsPK, "field %s should not be PK", f.Name)
		}
	}
}

// Created at 2026-06-28
// TestBuildRowMeta_NamedStructPrefixWithoutPrefix проверяет, что без префикса поля-структуры создаются как обычные поля
func TestBuildRowMeta_NamedStructPrefixWithoutPrefix(t *testing.T) {
	type Address struct {
		City string
	}
	type Person struct {
		ID      int64 `qqm:"pk"`
		Address Address
	}

	rm := BuildRowMeta(reflect.TypeOf(Person{}), "person")

	assert.Contains(t, rm.Columns, "address")
	assert.NotContains(t, rm.Columns, "city")
}

// Created at 2026-06-28
// TestBuildRowMeta_PKOrderDeclaration проверяет что порядок PK определяется порядком объявления
func TestBuildRowMeta_PKOrderDeclaration(t *testing.T) {
	type Row struct {
		Second int64 `qqm:"pk"`
		First  int64 `qqm:"pk"`
		Third  int64 `qqm:"pk"`
	}

	rm := BuildRowMeta(reflect.TypeOf(Row{}), "test")

	require.Len(t, rm.PKFields, 3)
	assert.Equal(t, 1, rm.PKFields[0].PkOrder)
	assert.Equal(t, "second", rm.PKFields[0].Column)
	assert.Equal(t, 2, rm.PKFields[1].PkOrder)
	assert.Equal(t, "first", rm.PKFields[1].Column)
	assert.Equal(t, 3, rm.PKFields[2].PkOrder)
	assert.Equal(t, "third", rm.PKFields[2].Column)
}

// Created at 2026-06-29
func TestBuildRowMeta_SortFields_Basic(t *testing.T) {
	type Row struct {
		ID   int64  `qqm:"pk"`
		Name string `qqm:"sort=1"`
		Age  int    `qqm:"sort=2,desc"`
	}

	rm := BuildRowMeta(reflect.TypeOf(Row{}), "test")

	require.Len(t, rm.SortFields, 2)
	assert.Equal(t, "name", rm.SortFields[0].Column)
	assert.Equal(t, 1, rm.SortFields[0].SortPosition)
	assert.Equal(t, "ASC", rm.SortFields[0].SortDirection)

	assert.Equal(t, "age", rm.SortFields[1].Column)
	assert.Equal(t, 2, rm.SortFields[1].SortPosition)
	assert.Equal(t, "DESC", rm.SortFields[1].SortDirection)
}

// Created at 2026-06-29
func TestBuildRowMeta_SortFields_OrderedByPosition(t *testing.T) {
	type Row struct {
		ID     int64  `qqm:"pk"`
		Second string `qqm:"sort=2"`
		First  string `qqm:"sort=1"`
	}

	rm := BuildRowMeta(reflect.TypeOf(Row{}), "test")

	require.Len(t, rm.SortFields, 2)
	assert.Equal(t, "first", rm.SortFields[0].Column)
	assert.Equal(t, 1, rm.SortFields[0].SortPosition)

	assert.Equal(t, "second", rm.SortFields[1].Column)
	assert.Equal(t, 2, rm.SortFields[1].SortPosition)
}

// Created at 2026-06-29
func TestBuildRowMeta_SortFields_NoSort(t *testing.T) {
	type Row struct {
		ID   int64 `qqm:"pk"`
		Name string
	}

	rm := BuildRowMeta(reflect.TypeOf(Row{}), "test")

	assert.Len(t, rm.SortFields, 0)
}

// Created at 2026-06-29
func TestBuildRowMeta_SortFields_Embedded(t *testing.T) {
	type Embedded struct {
		Name string `qqm:"sort=1"`
	}

	type Row struct {
		ID int64 `qqm:"pk"`
		Embedded
		Age int `qqm:"sort=2,desc"`
	}

	rm := BuildRowMeta(reflect.TypeOf(Row{}), "test")

	require.Len(t, rm.SortFields, 2)
	assert.Equal(t, "name", rm.SortFields[0].Column)
	assert.Equal(t, 1, rm.SortFields[0].SortPosition)
	assert.Equal(t, "age", rm.SortFields[1].Column)
	assert.Equal(t, 2, rm.SortFields[1].SortPosition)
}

// Created at 2026-06-29
func TestBuildRowMeta_SortFields_WithPrefix(t *testing.T) {
	type Addr struct {
		City string `qqm:"sort=1"`
		Zip  string `qqm:"sort=2,desc"`
	}
	type Row struct {
		ID          int64 `qqm:"pk"`
		HomeAddress Addr  `qqm:"prefix=home_"`
	}

	rm := BuildRowMeta(reflect.TypeOf(Row{}), "test")

	require.Len(t, rm.SortFields, 2)
	assert.Equal(t, "home_city", rm.SortFields[0].Column)
	assert.Equal(t, 1, rm.SortFields[0].SortPosition)
	assert.Equal(t, "home_zip", rm.SortFields[1].Column)
	assert.Equal(t, 2, rm.SortFields[1].SortPosition)
	assert.Equal(t, "DESC", rm.SortFields[1].SortDirection)
}
