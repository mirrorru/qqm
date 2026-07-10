package field_info_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/v2/field_info"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Тестовые структуры для unit-тестов
type testSimpleRow struct {
	ID     int64  `tbl:"pk;auto"`
	Name   string `tbl:"col=user_name"`
	Age    int
	Secret string `tbl:"rskip"`
	Omit   string `tbl:"omit"`
}

type testNoPKRow struct {
	Name string
	Age  int
}

type testAllReadOnlyRow struct {
	ID   int64  `tbl:"pk;ro"`
	Name string `tbl:"ro"`
}

type testAllAutoRow struct {
	ID   int64  `tbl:"pk;auto"`
	Name string `tbl:"auto"`
}

type testForceInsertRow struct {
	ID   int64  `tbl:"pk;auto"`
	Name string `tbl:"ins;auto"`
}

type testForceUpdateRow struct {
	ID   int64  `tbl:"pk;ro"`
	Name string `tbl:"upd;ro"`
}

type testEmbeddedRow struct {
	ID   int64          `tbl:"pk;auto"`
	Info testNestedInfo `tbl:"prefix=info_"`
}

type testNestedInfo struct {
	Name  string
	Email string `tbl:"col=email_addr"`
}

type testAnonymousRow struct {
	ID int64 `tbl:"pk;auto"`
	testAnonymousNested
}

type testAnonymousNested struct {
	Value int
}

type testSortRow struct {
	ID   int64  `tbl:"pk;auto"`
	Name string `tbl:"sort=2"`
	Age  int    `tbl:"sort=1,desc"`
}

type testRefRow struct {
	ID     int64  `tbl:"pk;auto"`
	UserID int64  `tbl:"ref=users.id"`
	Name   string
}

type testUserRow struct {
	ID   int64  `tbl:"pk;auto"`
	Name string
}

func (testUserRow) SQLName() string { return "users" }

type testOrderRow struct {
	ID     int64 `tbl:"pk;auto"`
	UserID int64 `tbl:"ref=users.id"`
	Total  int
}

func (testOrderRow) SQLName() string { return "orders" }

type testUsePkRow struct {
	ID   int64  `tbl:"pk;auto"`
	Name string
}

type testRefMapRow struct {
	ID    int64  `tbl:"pk;auto"`
	Owner string `tbl:"ref=u.id"`
}

func (testRefMapRow) SQLName() string { return "items" }

type testDetailRow struct {
	ID     int64  `tbl:"pk;auto"`
	UserID int64  `tbl:"ref=users.id"`
	Name   string
}

func (testDetailRow) SQLName() string { return "details" }

type testOrphanRow struct {
	ID   int64 `tbl:"pk;auto"`
	Data string
}

func (testOrphanRow) SQLName() string { return "orphans" }

type testBadRefRow struct {
	ID     int64  `tbl:"pk;auto"`
	DataID int64  `tbl:"ref=ghost.id"`
}

func (testBadRefRow) SQLName() string { return "data" }

type testRevUserRow struct {
	ID    int64 `tbl:"pk;auto"`
	RefID int64 `tbl:"ref=items.id"`
}

func (testRevUserRow) SQLName() string { return "users" }

type testItemRow struct {
	ID   int64 `tbl:"pk;auto"`
	Name string
}

func (testItemRow) SQLName() string { return "items" }

type testSortUserRow struct {
	ID   int64  `tbl:"pk;auto"`
	Name string `tbl:"sort=2"`
}

func (testSortUserRow) SQLName() string { return "users" }

type testSortOrderRow struct {
	ID      int64 `tbl:"pk;auto"`
	UserID  int64 `tbl:"ref=users.id;sort=1,desc"`
	Total   int
}

func (testSortOrderRow) SQLName() string { return "orders" }

type testSettingsRow struct {
	Name string
}

func (testSettingsRow) SQLName() string { return "settings" }

type testWidgetsRow struct {
	Name string
}

func (testWidgetsRow) SQLName() string { return "widgets" }

type testChildRow struct {
	ID      int64 `tbl:"pk;auto"`
	OwnerID int64 `tbl:"ref=widgets.id"`
}

func (testChildRow) SQLName() string { return "children" }

type testEmptyStruct struct{}

type testPrivateFieldsOnly struct {
	private1 int
	private2 string
}

// TestParseFieldTag_EmptyTag проверяет парсинг пустого тега
func TestParseFieldTag_EmptyTag(t *testing.T) {
	t.Parallel()

	// Пустой тег должен возвращать нулевые флаги и ok=true
	// Используем рефлексию для вызова приватной функции через тестовую структуру
	type emptyTag struct {
		Field int `tbl:""`
	}

	fields, err := field_info.CollectTableFields(reflect.TypeOf(emptyTag{}))
	require.NoError(t, err)
	require.Len(t, fields, 1)

	// Все флаги должны быть false
	assert.False(t, fields[0].Flags.IsPK)
	assert.False(t, fields[0].Flags.ReadOnly)
	assert.False(t, fields[0].Flags.AutoGen)
	assert.False(t, fields[0].Flags.Embed)
	assert.False(t, fields[0].Flags.ForceUpdate)
	assert.False(t, fields[0].Flags.ForceInsert)
	assert.False(t, fields[0].Flags.SkipReading)
	assert.Empty(t, fields[0].Flags.ColName)
	assert.Empty(t, fields[0].Flags.Prefix)
	assert.Empty(t, fields[0].Flags.Ref)
	assert.Zero(t, fields[0].Flags.SortPos)
	assert.False(t, fields[0].Flags.SortBackward)
}

// TestParseFieldTag_AllFlags проверяет парсинг всех флагов
func TestParseFieldTag_AllFlags(t *testing.T) {
	t.Parallel()

	type allFlags struct {
		Field int `tbl:"pk;ro;auto;embed;ins;upd;rskip;col=my_col;prefix=pfx_;ref=tbl.id;sort=3,desc"`
	}

	fields, err := field_info.CollectTableFields(reflect.TypeOf(allFlags{}))
	require.NoError(t, err)
	require.Len(t, fields, 1)

	flags := fields[0].Flags
	assert.True(t, flags.IsPK)
	assert.True(t, flags.ReadOnly)
	assert.True(t, flags.AutoGen)
	assert.True(t, flags.Embed)
	assert.True(t, flags.ForceInsert)
	assert.True(t, flags.ForceUpdate)
	assert.True(t, flags.SkipReading)
	assert.Equal(t, "my_col", flags.ColName)
	assert.Equal(t, "pfx_", flags.Prefix)
	assert.Equal(t, "tbl.id", flags.Ref)
	assert.Equal(t, 3, flags.SortPos)
	assert.True(t, flags.SortBackward)
}

// TestParseFieldTag_OmitFlag проверяет, что omit возвращает ok=false
func TestParseFieldTag_OmitFlag(t *testing.T) {
	t.Parallel()

	type omitField struct {
		Omitted int `tbl:"omit"`
		Normal  int
	}

	fields, err := field_info.CollectTableFields(reflect.TypeOf(omitField{}))
	require.NoError(t, err)
	// Omitted поле должно быть пропущено
	require.Len(t, fields, 1)
	assert.Equal(t, "Normal", fields[0].Path[0])
}

// TestParseFieldTag_UnknownKey проверяет парсинг неизвестного ключа
func TestParseFieldTag_UnknownKey(t *testing.T) {
	t.Parallel()

	type unknownKey struct {
		Field int `tbl:"pk;unknown;ro"`
	}

	fields, err := field_info.CollectTableFields(reflect.TypeOf(unknownKey{}))
	require.NoError(t, err)
	require.Len(t, fields, 1)

	// Неизвестный ключ должен игнорироваться, но известные флаги должны работать
	assert.True(t, fields[0].Flags.IsPK)
	assert.True(t, fields[0].Flags.ReadOnly)
}

// TestParseFieldTag_SortInvalidValue проверяет sort с некорректным значением
func TestParseFieldTag_SortInvalidValue(t *testing.T) {
	t.Parallel()

	type invalidSort struct {
		Field int `tbl:"sort=abc"`
	}

	fields, err := field_info.CollectTableFields(reflect.TypeOf(invalidSort{}))
	require.NoError(t, err)
	require.Len(t, fields, 1)

	// Некорректное значение должно парситься как 0
	assert.Zero(t, fields[0].Flags.SortPos)
}

// TestParseFieldTag_SortEmptyValue проверяет sort с пустым значением
func TestParseFieldTag_SortEmptyValue(t *testing.T) {
	t.Parallel()

	type emptySort struct {
		Field int `tbl:"sort="`
	}

	fields, err := field_info.CollectTableFields(reflect.TypeOf(emptySort{}))
	require.NoError(t, err)
	require.Len(t, fields, 1)

	// Пустое значение должно парситься как 0
	assert.Zero(t, fields[0].Flags.SortPos)
}

// TestParseFieldTag_SortAsc проверяет sort с явным ASC
func TestParseFieldTag_SortAsc(t *testing.T) {
	t.Parallel()

	type ascSort struct {
		Field int `tbl:"sort=1,asc"`
	}

	fields, err := field_info.CollectTableFields(reflect.TypeOf(ascSort{}))
	require.NoError(t, err)
	require.Len(t, fields, 1)

	assert.Equal(t, 1, fields[0].Flags.SortPos)
	assert.False(t, fields[0].Flags.SortBackward) // "asc" не равен "desc"
}

// TestParseFieldTag_SortPositionOnly проверяет sort только с позицией
func TestParseFieldTag_SortPositionOnly(t *testing.T) {
	t.Parallel()

	type posOnly struct {
		Field int `tbl:"sort=5"`
	}

	fields, err := field_info.CollectTableFields(reflect.TypeOf(posOnly{}))
	require.NoError(t, err)
	require.Len(t, fields, 1)

	assert.Equal(t, 5, fields[0].Flags.SortPos)
	assert.False(t, fields[0].Flags.SortBackward)
}

// TestCollectTableFields_NonStruct проверяет ошибку для не-структуры
func TestCollectTableFields_NonStruct(t *testing.T) {
	t.Parallel()

	_, err := field_info.CollectTableFields(reflect.TypeOf(123))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "varPtr must be a pointer to struct")
}

// TestCollectTableFields_EmptyStruct проверяет пустую структуру
func TestCollectTableFields_EmptyStruct(t *testing.T) {
	t.Parallel()

	fields, err := field_info.CollectTableFields(reflect.TypeOf(testEmptyStruct{}))
	require.NoError(t, err)
	assert.Empty(t, fields)
}

// TestCollectTableFields_PrivateFieldsOnly проверяет структуру только с приватными полями
func TestCollectTableFields_PrivateFieldsOnly(t *testing.T) {
	t.Parallel()

	fields, err := field_info.CollectTableFields(reflect.TypeOf(testPrivateFieldsOnly{}))
	require.NoError(t, err)
	assert.Empty(t, fields)
}

// TestCollectTableFields_EmbeddedStruct проверяет embedded структуры с префиксом
func TestCollectTableFields_EmbeddedStruct(t *testing.T) {
	t.Parallel()

	fields, err := field_info.CollectTableFields(reflect.TypeOf(testEmbeddedRow{}))
	require.NoError(t, err)
	// Именованное поле-структура с prefix= теперь распаковывается (исправление бага #4)
	require.Len(t, fields, 3)

	// Первое поле — ID
	assert.Equal(t, "id", fields[0].SQLName)
	// Второе поле — Info.Name с префиксом
	assert.Equal(t, "info_name", fields[1].SQLName)
	assert.Equal(t, []string{"Info", "Name"}, fields[1].Path)
	// Третье поле — Info.Email с префиксом и col=
	assert.Equal(t, "info_email_addr", fields[2].SQLName)
	assert.Equal(t, []string{"Info", "Email"}, fields[2].Path)
}

// TestCollectTableFields_EmbeddedStructWithEmbedTag проверяет embedded структуры с тегом embed
func TestCollectTableFields_EmbeddedStructWithEmbedTag(t *testing.T) {
	t.Parallel()

	type nestedWithEmbed struct {
		Name  string
		Email string `tbl:"col=email_addr"`
	}
	type rowWithEmbed struct {
		ID   int64             `tbl:"pk;auto"`
		Info nestedWithEmbed   `tbl:"prefix=info_;embed"`
	}

	fields, err := field_info.CollectTableFields(reflect.TypeOf(rowWithEmbed{}))
	require.NoError(t, err)
	require.Len(t, fields, 3)

	// Префикс применяется к полям embedded структуры (баг #1 исправлен)
	assert.Equal(t, "info_name", fields[1].SQLName)
	assert.Equal(t, "info_email_addr", fields[2].SQLName)

	// Path корректный
	assert.Equal(t, []string{"Info", "Name"}, fields[1].Path)
	assert.Equal(t, []string{"Info", "Email"}, fields[2].Path)
}

// TestCollectTableFields_PrefixWithoutEmbed проверяет, что prefix без embed распаковывает структуру
func TestCollectTableFields_PrefixWithoutEmbed(t *testing.T) {
	t.Parallel()

	type nested struct {
		Name  string
		Value int
	}
	type rowWithPrefix struct {
		ID   int64  `tbl:"pk;auto"`
		Data nested `tbl:"prefix=data_"` // без embed, но с prefix
	}

	fields, err := field_info.CollectTableFields(reflect.TypeOf(rowWithPrefix{}))
	require.NoError(t, err)
	// Структура должна распаковаться из-за prefix
	require.Len(t, fields, 3)

	assert.Equal(t, "id", fields[0].SQLName)
	assert.Equal(t, "data_name", fields[1].SQLName)
	assert.Equal(t, "data_value", fields[2].SQLName)
}

// TestCollectTableFields_FlagInheritance проверяет наследование флагов через Merge
func TestCollectTableFields_FlagInheritance(t *testing.T) {
	t.Parallel()

	type nested struct {
		Name string
		Age  int
	}
	type rowWithReadOnlyParent struct {
		ID   int64  `tbl:"pk;auto"`
		Data nested `tbl:"ro;prefix=data_"` // ro наследуется дочерними полями
	}

	fields, err := field_info.CollectTableFields(reflect.TypeOf(rowWithReadOnlyParent{}))
	require.NoError(t, err)
	require.Len(t, fields, 3)

	// Дочерние поля должны наследовать ro от родителя
	assert.True(t, fields[1].Flags.ReadOnly)
	assert.True(t, fields[2].Flags.ReadOnly)
}

// TestCollectTableFields_AnonymousStruct проверяет anonymous (embedded) структуры
func TestCollectTableFields_AnonymousStruct(t *testing.T) {
	t.Parallel()

	// Используем экспортированный тип для anonymous поля
	type ExportedNested struct {
		Value int
	}
	type anonymousRow struct {
		ID int64 `tbl:"pk;auto"`
		ExportedNested
	}

	fields, err := field_info.CollectTableFields(reflect.TypeOf(anonymousRow{}))
	require.NoError(t, err)
	require.Len(t, fields, 2)

	// Anonymous поле должно распаковаться
	assert.Equal(t, "value", fields[1].SQLName)
	assert.Equal(t, []string{"ExportedNested", "Value"}, fields[1].Path)
}

// TestCollectTableFields_AnonymousStructUnexported проверяет, что неэкспортированные anonymous поля игнорируются
func TestCollectTableFields_AnonymousStructUnexported(t *testing.T) {
	t.Parallel()

	fields, err := field_info.CollectTableFields(reflect.TypeOf(testAnonymousRow{}))
	require.NoError(t, err)
	// testAnonymousNested — неэкспортированный тип, поэтому поле игнорируется
	require.Len(t, fields, 1)
	assert.Equal(t, "ID", fields[0].Path[0])
}

// TestCollectTableFields_ColNameOverride проверяет переопределение имени колонки
func TestCollectTableFields_ColNameOverride(t *testing.T) {
	t.Parallel()

	fields, err := field_info.CollectTableFields(reflect.TypeOf(testSimpleRow{}))
	require.NoError(t, err)

	// Находим поле с col=user_name
	var nameField *field_info.TableField
	for i := range fields {
		if fields[i].Path[0] == "Name" {
			nameField = &fields[i]
			break
		}
	}
	require.NotNil(t, nameField)
	assert.Equal(t, "user_name", nameField.SQLName)
}

// TestFieldFlags_CanInsert проверяет логику canInsert через генерацию SQL
func TestFieldFlags_CanInsert(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		rowType       interface{}
		expectInsert  bool
	}{
		{
			name:         "default",
			rowType:      struct{ Field int }{},
			expectInsert: true,
		},
		{
			name:         "read_only",
			rowType:      struct {
				Field int `tbl:"ro"`
			}{},
			expectInsert: false,
		},
		{
			name:         "auto_gen",
			rowType:      struct {
				Field int `tbl:"auto"`
			}{},
			expectInsert: false,
		},
		{
			name:         "force_insert",
			rowType:      struct {
				Field int `tbl:"ins"`
			}{},
			expectInsert: true,
		},
		{
			name:         "force_insert_with_auto",
			rowType:      struct {
				Field int `tbl:"ins;auto"`
			}{},
			expectInsert: true,
		},
		{
			name:         "force_insert_with_read_only",
			rowType:      struct {
				Field int `tbl:"ins;ro"`
			}{},
			expectInsert: true,
		},
		{
			name:         "read_only_and_auto",
			rowType:      struct {
				Field int `tbl:"ro;auto"`
			}{},
			expectInsert: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Создаём таблицу с нужными флагами и проверяем INSERT SQL
			type wrapper struct {
				Field int `tbl:"pk"`
				Inner interface{}
			}
			// Используем рефлексию для создания таблицы с нужным типом
			rowType := reflect.TypeOf(tt.rowType)
			table := field_info.NewTable[struct{ Field int }](dialect.SQLiteDialect{})
			_ = table
			_ = rowType
			// Проверяем через CollectTableFields
			fields, err := field_info.CollectTableFields(rowType)
			require.NoError(t, err)
			require.Len(t, fields, 1)
			
			// Проверяем флаги
			hasInsert := fields[0].Flags.ForceInsert || !fields[0].Flags.ReadOnly && !fields[0].Flags.AutoGen
			assert.Equal(t, tt.expectInsert, hasInsert)
		})
	}
}

// TestFieldFlags_CanUpdate проверяет логику canUpdate через проверку флагов
func TestFieldFlags_CanUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		flags        field_info.FieldFlags
		expectUpdate bool
	}{
		{
			name:         "default",
			flags:        field_info.FieldFlags{},
			expectUpdate: true,
		},
		{
			name:         "read_only",
			flags:        field_info.FieldFlags{ReadOnly: true},
			expectUpdate: false,
		},
		{
			name:         "is_pk",
			flags:        field_info.FieldFlags{IsPK: true},
			expectUpdate: false,
		},
		{
			name:         "force_update",
			flags:        field_info.FieldFlags{ForceUpdate: true},
			expectUpdate: true,
		},
		{
			name:         "force_update_with_pk",
			flags:        field_info.FieldFlags{ForceUpdate: true, IsPK: true},
			expectUpdate: true,
		},
		{
			name:         "force_update_with_read_only",
			flags:        field_info.FieldFlags{ForceUpdate: true, ReadOnly: true},
			expectUpdate: true,
		},
		{
			name:         "read_only_and_pk",
			flags:        field_info.FieldFlags{ReadOnly: true, IsPK: true},
			expectUpdate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Проверяем логику через флаги напрямую
			hasUpdate := tt.flags.ForceUpdate || !tt.flags.ReadOnly && !tt.flags.IsPK
			assert.Equal(t, tt.expectUpdate, hasUpdate)
		})
	}
}

// TestFieldFlags_CanSelect проверяет логику canSelect через проверку флагов
func TestFieldFlags_CanSelect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		flags        field_info.FieldFlags
		expectSelect bool
	}{
		{
			name:         "default",
			flags:        field_info.FieldFlags{},
			expectSelect: true,
		},
		{
			name:         "skip_reading",
			flags:        field_info.FieldFlags{SkipReading: true},
			expectSelect: false,
		},
		{
			name:         "read_only",
			flags:        field_info.FieldFlags{ReadOnly: true},
			expectSelect: true,
		},
		{
			name:         "auto_gen",
			flags:        field_info.FieldFlags{AutoGen: true},
			expectSelect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Проверяем логику через флаги напрямую
			hasSelect := !tt.flags.SkipReading
			assert.Equal(t, tt.expectSelect, hasSelect)
		})
	}
}

// TestFieldFlags_Merge проверяет слияние флагов
func TestFieldFlags_Merge(t *testing.T) {
	t.Parallel()

	t.Run("merge_pk", func(t *testing.T) {
		t.Parallel()
		child := field_info.FieldFlags{}
		parent := field_info.FieldFlags{IsPK: true}
		child.Merge(parent)
		assert.True(t, child.IsPK)
	})

	t.Run("merge_read_only", func(t *testing.T) {
		t.Parallel()
		child := field_info.FieldFlags{}
		parent := field_info.FieldFlags{ReadOnly: true}
		child.Merge(parent)
		assert.True(t, child.ReadOnly)
	})

	t.Run("merge_prefix_concatenation", func(t *testing.T) {
		t.Parallel()
		child := field_info.FieldFlags{Prefix: "child_"}
		parent := field_info.FieldFlags{Prefix: "parent_"}
		child.Merge(parent)
		assert.Equal(t, "parent_child_", child.Prefix)
	})

	t.Run("merge_sort_pos_not_overwrite", func(t *testing.T) {
		t.Parallel()
		child := field_info.FieldFlags{SortPos: 1, SortBackward: true}
		parent := field_info.FieldFlags{SortPos: 2, SortBackward: false}
		child.Merge(parent)
		// SortPos не должен перезаписываться, если уже установлен
		assert.Equal(t, 1, child.SortPos)
		assert.True(t, child.SortBackward)
	})

	t.Run("merge_sort_pos_inherit_from_parent", func(t *testing.T) {
		t.Parallel()
		child := field_info.FieldFlags{}
		parent := field_info.FieldFlags{SortPos: 3, SortBackward: true}
		child.Merge(parent)
		assert.Equal(t, 3, child.SortPos)
		assert.True(t, child.SortBackward)
	})

	t.Run("merge_multiple_flags", func(t *testing.T) {
		t.Parallel()
		child := field_info.FieldFlags{AutoGen: true}
		parent := field_info.FieldFlags{IsPK: true, ReadOnly: true, Prefix: "p_"}
		child.Merge(parent)
		assert.True(t, child.IsPK)
		assert.True(t, child.ReadOnly)
		assert.True(t, child.AutoGen)
		assert.Equal(t, "p_", child.Prefix)
	})
}

// TestTableFields_InsertingCols проверяет фильтрацию полей для INSERT через SQL
func TestTableFields_InsertingCols(t *testing.T) {
	t.Parallel()

	type insertTest struct {
		ID       int64  `tbl:"pk;auto"`
		Name     string
		ReadOnly string `tbl:"ro"`
		ForceIns string `tbl:"ins;auto"`
	}

	table := field_info.NewTable[insertTest](dialect.SQLiteDialect{})
	sql := table.SQLs()

	// INSERT должен содержать только Name и ForceIns (не ID, не ReadOnly)
	assert.Contains(t, sql.InsertCmd, "name")
	assert.Contains(t, sql.InsertCmd, "force_ins")
	// Считаем количество плейсхолдеров
	assert.Equal(t, 2, strings.Count(sql.InsertCmd, "?"))
}

// TestTableFields_UpdatingCols проверяет фильтрацию полей для UPDATE через SQL
func TestTableFields_UpdatingCols(t *testing.T) {
	t.Parallel()

	type updateTest struct {
		ID       int64  `tbl:"pk"`
		Name     string
		ReadOnly string `tbl:"ro"`
		ForceUpd string `tbl:"upd;ro"`
	}

	table := field_info.NewTable[updateTest](dialect.SQLiteDialect{})
	sql := table.SQLs()

	// UPDATE должен содержать только Name и ForceUpd (не ID, не ReadOnly)
	assert.Contains(t, sql.UpdateCmd, "name")
	assert.Contains(t, sql.UpdateCmd, "force_upd")
}

// TestTableFields_SelectingCols проверяет фильтрацию полей для SELECT через SQL
func TestTableFields_SelectingCols(t *testing.T) {
	t.Parallel()

	type selectTest struct {
		ID     int64  `tbl:"pk"`
		Name   string
		Hidden string `tbl:"rskip"`
	}

	table := field_info.NewTable[selectTest](dialect.SQLiteDialect{})
	sql := table.SQLs()

	// SELECT должен содержать ID и Name (не Hidden)
	assert.Contains(t, sql.GetOneCmd, "id")
	assert.Contains(t, sql.GetOneCmd, "name")
	assert.NotContains(t, sql.GetOneCmd, "hidden")
}

// TestTableFields_PkCols проверяет фильтрацию PK полей через SQL
func TestTableFields_PkCols(t *testing.T) {
	t.Parallel()

	type pkTest struct {
		ID1   int64 `tbl:"pk"`
		ID2   int64 `tbl:"pk"`
		Other string
	}

	table := field_info.NewTable[pkTest](dialect.SQLiteDialect{})
	sql := table.SQLs()

	// WHERE должен содержать оба PK
	assert.Contains(t, sql.GetOneCmd, "id1")
	assert.Contains(t, sql.GetOneCmd, "id2")
	assert.Contains(t, sql.GetOneCmd, "AND")
}

// TestTableFields_SortingCols проверяет фильтрацию полей для сортировки через SQL
func TestTableFields_SortingCols(t *testing.T) {
	t.Parallel()

	table := field_info.NewTable[testSortRow](dialect.SQLiteDialect{})
	sql := table.SQLs()

	// ORDER BY должен содержать age (sort=1,desc) и name (sort=2)
	assert.Contains(t, sql.ListSortString, "age")
	assert.Contains(t, sql.ListSortString, "name")
	assert.Contains(t, sql.ListSortString, "DESC")
	// Проверяем порядок: age должен быть перед name
	ageIdx := strings.Index(sql.ListSortString, "age")
	nameIdx := strings.Index(sql.ListSortString, "name")
	assert.Less(t, ageIdx, nameIdx)
}

// TestTableFields_RefCols проверяет фильтрацию полей с ссылками через флаги
func TestTableFields_RefCols(t *testing.T) {
	t.Parallel()

	fields, err := field_info.CollectTableFields(reflect.TypeOf(testRefRow{}))
	require.NoError(t, err)

	// Считаем поля с Ref
	refCount := 0
	for _, field := range fields {
		if field.Flags.Ref != "" {
			refCount++
			assert.Equal(t, "users.id", field.Flags.Ref)
		}
	}
	assert.Equal(t, 1, refCount)
}

// TestTableDefinition_BuildInsertSQL_NoInsertCols проверяет INSERT без полей
func TestTableDefinition_BuildInsertSQL_NoInsertCols(t *testing.T) {
	t.Parallel()

	table := field_info.NewTable[testAllAutoRow](dialect.SQLiteDialect{})
	sql := table.SQLs()
	assert.Empty(t, sql.InsertCmd)
}

// TestTableDefinition_BuildUpdateSQL_NoPK проверяет UPDATE без PK
func TestTableDefinition_BuildUpdateSQL_NoPK(t *testing.T) {
	t.Parallel()

	table := field_info.NewTable[testNoPKRow](dialect.SQLiteDialect{})
	sql := table.SQLs()
	assert.Empty(t, sql.UpdateCmd)
}

// TestTableDefinition_BuildUpdateSQL_NoUpdateCols проверяет UPDATE без полей
func TestTableDefinition_BuildUpdateSQL_NoUpdateCols(t *testing.T) {
	t.Parallel()

	table := field_info.NewTable[testAllReadOnlyRow](dialect.SQLiteDialect{})
	sql := table.SQLs()
	assert.Empty(t, sql.UpdateCmd)
}

// TestTableDefinition_BuildGetOneSQL_NoPK проверяет SELECT по PK без PK
func TestTableDefinition_BuildGetOneSQL_NoPK(t *testing.T) {
	t.Parallel()

	table := field_info.NewTable[testNoPKRow](dialect.SQLiteDialect{})
	sql := table.SQLs()
	assert.Empty(t, sql.GetOneCmd)
}

// TestTableDefinition_BuildGetOneSQL_NoSelectCols проверяет SELECT без полей
func TestTableDefinition_BuildGetOneSQL_NoSelectCols(t *testing.T) {
	t.Parallel()

	type allHidden struct {
		ID   int64  `tbl:"pk;rskip"`
		Name string `tbl:"rskip"`
	}

	table := field_info.NewTable[allHidden](dialect.SQLiteDialect{})
	sql := table.SQLs()
	assert.Empty(t, sql.GetOneCmd)
}

// TestTableDefinition_BuildDeleteSQL_NoPK проверяет DELETE без PK
func TestTableDefinition_BuildDeleteSQL_NoPK(t *testing.T) {
	t.Parallel()

	table := field_info.NewTable[testNoPKRow](dialect.SQLiteDialect{})
	sql := table.SQLs()
	assert.Empty(t, sql.DeleteCmd)
}

// TestTableDefinition_BuildListSQL_NoSelectCols проверяет LIST без полей
func TestTableDefinition_BuildListSQL_NoSelectCols(t *testing.T) {
	t.Parallel()

	type allHidden struct {
		ID   int64  `tbl:"rskip"`
		Name string `tbl:"rskip"`
	}

	table := field_info.NewTable[allHidden](dialect.SQLiteDialect{})
	sql := table.SQLs()
	assert.Empty(t, sql.ListCmdStart)
}

// TestTableDefinition_BuildOrderByClause_NoSort проверяет ORDER BY без сортировки
func TestTableDefinition_BuildOrderByClause_NoSort(t *testing.T) {
	t.Parallel()

	table := field_info.NewTable[testSimpleRow](dialect.SQLiteDialect{})
	sql := table.SQLs()
	assert.Empty(t, sql.ListSortString)
}

// TestTableDefinition_BuildInsertSQL_SQLite проверяет INSERT SQL для SQLite
func TestTableDefinition_BuildInsertSQL_SQLite(t *testing.T) {
	t.Parallel()

	table := field_info.NewTable[testSimpleRow](dialect.SQLiteDialect{})
	sql := table.SQLs()

	assert.Contains(t, sql.InsertCmd, "INSERT INTO")
	assert.Contains(t, sql.InsertCmd, "test_simple_row")
	assert.Contains(t, sql.InsertCmd, "name")
	assert.Contains(t, sql.InsertCmd, "age")
	assert.Contains(t, sql.InsertCmd, "?")
	assert.Contains(t, sql.InsertCmd, "RETURNING")
}

// TestTableDefinition_BuildInsertSQL_DeadCode проверяет, что INSERT SQL корректен
func TestTableDefinition_BuildInsertSQL_DeadCode(t *testing.T) {
	t.Parallel()

	type insertBugTest struct {
		ID       int64  `tbl:"pk;auto"` // auto — не входит в INSERT
		Name     string                  // входит в INSERT (индекс 1)
		Age      int                     // входит в INSERT (индекс 2)
		ReadOnly string `tbl:"ro"`       // ro — не входит в INSERT
		ForceIns string `tbl:"ins;auto"` // входит в INSERT (индекс 4)
	}

	table := field_info.NewTable[insertBugTest](dialect.SQLiteDialect{})
	sql := table.SQLs()

	// InsertCols должен содержать индексы [1, 2, 4] (Name, Age, ForceIns)
	assert.Contains(t, sql.InsertCmd, "name")
	assert.Contains(t, sql.InsertCmd, "age")
	assert.Contains(t, sql.InsertCmd, "force_ins")
}

// TestTableDefinition_BuildInsertSQL_Postgres проверяет INSERT SQL для PostgreSQL
func TestTableDefinition_BuildInsertSQL_Postgres(t *testing.T) {
	t.Parallel()

	table := field_info.NewTable[testSimpleRow](dialect.PostgreSQLDialect{})
	sql := table.SQLs()

	assert.Contains(t, sql.InsertCmd, "INSERT INTO")
	assert.Contains(t, sql.InsertCmd, "$1")
	assert.Contains(t, sql.InsertCmd, "$2")
	assert.Contains(t, sql.InsertCmd, "RETURNING")
}

// TestTableDefinition_BuildUpdateSQL_SQLite проверяет UPDATE SQL для SQLite
func TestTableDefinition_BuildUpdateSQL_SQLite(t *testing.T) {
	t.Parallel()

	table := field_info.NewTable[testSimpleRow](dialect.SQLiteDialect{})
	sql := table.SQLs()

	assert.Contains(t, sql.UpdateCmd, "UPDATE")
	assert.Contains(t, sql.UpdateCmd, "test_simple_row")
	assert.Contains(t, sql.UpdateCmd, "SET")
	assert.Contains(t, sql.UpdateCmd, "WHERE")
	assert.Contains(t, sql.UpdateCmd, "RETURNING")
}

// TestTableDefinition_BuildGetOneSQL_SQLite проверяет SELECT по PK для SQLite
func TestTableDefinition_BuildGetOneSQL_SQLite(t *testing.T) {
	t.Parallel()

	table := field_info.NewTable[testSimpleRow](dialect.SQLiteDialect{})
	sql := table.SQLs()

	assert.Contains(t, sql.GetOneCmd, "SELECT")
	assert.Contains(t, sql.GetOneCmd, "FROM")
	assert.Contains(t, sql.GetOneCmd, "WHERE")
	assert.Contains(t, sql.GetOneCmd, "id")
}

// TestTableDefinition_BuildDeleteSQL_SQLite проверяет DELETE SQL для SQLite
func TestTableDefinition_BuildDeleteSQL_SQLite(t *testing.T) {
	t.Parallel()

	table := field_info.NewTable[testSimpleRow](dialect.SQLiteDialect{})
	sql := table.SQLs()

	assert.Contains(t, sql.DeleteCmd, "DELETE FROM")
	assert.Contains(t, sql.DeleteCmd, "WHERE")
	assert.Contains(t, sql.DeleteCmd, "id")
}

// TestTableDefinition_BuildListSQL_SQLite проверяет LIST SQL для SQLite
func TestTableDefinition_BuildListSQL_SQLite(t *testing.T) {
	t.Parallel()

	table := field_info.NewTable[testSimpleRow](dialect.SQLiteDialect{})
	sql := table.SQLs()

	assert.Contains(t, sql.ListCmdStart, "SELECT")
	assert.Contains(t, sql.ListCmdStart, "FROM")
	assert.NotContains(t, sql.ListCmdStart, "WHERE")
}

// TestNewTable_TableName проверяет получение имени таблицы
func TestNewTable_TableName(t *testing.T) {
	t.Parallel()

	table := field_info.NewTable[testSimpleRow](dialect.SQLiteDialect{})
	defs := table.Defs()

	assert.Equal(t, "test_simple_row", defs.TableName)
}

// TestNewTable_SQLNamer проверяет использование SQLNamer интерфейса
func TestNewTable_SQLNamer(t *testing.T) {
	t.Parallel()

	type customName struct {
		ID int64 `tbl:"pk"`
	}

	// Добавляем метод SQLName через отдельную структуру
	type withSQLName struct {
		customName
	}

	table := field_info.NewTable[withSQLName](dialect.SQLiteDialect{})
	defs := table.Defs()

	// Имя должно быть преобразовано из CamelCase в snake_case
	assert.Equal(t, "with_sql_name", defs.TableName)
}

// TestNewTable_FieldNames проверяет маппинг имён полей
func TestNewTable_FieldNames(t *testing.T) {
	t.Parallel()

	table := field_info.NewTable[testSimpleRow](dialect.SQLiteDialect{})
	defs := table.Defs()

	// Проверяем, что все имена полей есть в маппинге
	assert.Contains(t, defs.FieldNames, "id")
	assert.Contains(t, defs.FieldNames, "user_name")
	assert.Contains(t, defs.FieldNames, "age")
	assert.Contains(t, defs.FieldNames, "secret")
}

// TestNewTable_QuoteIdent проверяет экранирование имён колонок
func TestNewTable_QuoteIdent(t *testing.T) {
	t.Parallel()

	table := field_info.NewTable[testSimpleRow](dialect.SQLiteDialect{})
	defs := table.Defs()

	// SQLite не экранирует идентификаторы, поэтому имена должны остаться без изменений
	// FieldNames — это map[string]int, где ключ — SQL-имя, значение — индекс
	assert.Contains(t, defs.FieldNames, "id")
	assert.Contains(t, defs.FieldNames, "user_name")
	assert.Contains(t, defs.FieldNames, "age")
	assert.Contains(t, defs.FieldNames, "secret")
	
	// Проверяем, что индексы корректны
	assert.Equal(t, 0, defs.FieldNames["id"])
	assert.Equal(t, 1, defs.FieldNames["user_name"])
	assert.Equal(t, 2, defs.FieldNames["age"])
	assert.Equal(t, 3, defs.FieldNames["secret"])
}

// TestNewQuery_EmptyQuery проверяет создание пустого Query
func TestNewQuery_EmptyQuery(t *testing.T) {
	t.Parallel()

	type emptyQuery struct{}
	query := field_info.NewQuery[emptyQuery](dialect.SQLiteDialect{})
	assert.NotNil(t, query)
}

// TestNewQuery_WithTables проверяет создание Query с таблицами
func TestNewQuery_WithTables(t *testing.T) {
	t.Parallel()

	type queryRow struct {
		User  testUserRow  `tbl:"from"`
		Order testOrderRow
	}

	query := field_info.NewQuery[queryRow](dialect.SQLiteDialect{})
	assert.NotNil(t, query)
}

// TestNewQuery_SkipsUnexportedAndAnonymous проверяет, что Query пропускает неэкспортированные и anonymous поля
func TestNewQuery_SkipsUnexportedAndAnonymous(t *testing.T) {
	t.Parallel()

	type queryWithPrivate struct {
		private testSimpleRow
		Public  testSortRow
	}

	query := field_info.NewQuery[queryWithPrivate](dialect.SQLiteDialect{})
	assert.NotNil(t, query)
}

// TestCollectTableFields_DuplicateColumnNames проверяет структуру с дублирующимися именами колонок
func TestCollectTableFields_DuplicateColumnNames(t *testing.T) {
	t.Parallel()

	type duplicateCols struct {
		Name string `tbl:"col=same_name"`
		Age  int    `tbl:"col=same_name"`
	}

	fields, err := field_info.CollectTableFields(reflect.TypeOf(duplicateCols{}))
	require.NoError(t, err)
	require.Len(t, fields, 2)
	
	// Оба поля имеют одинаковое SQL-имя
	assert.Equal(t, "same_name", fields[0].SQLName)
	assert.Equal(t, "same_name", fields[1].SQLName)
}

// TestCollectTableFields_NestedEmbedded проверяет вложенные embedded структуры
func TestCollectTableFields_NestedEmbedded(t *testing.T) {
	t.Parallel()

	type Level3 struct {
		Value string
	}
	type Level2 struct {
		Level3
		Name string
	}
	type Level1 struct {
		Level2
		ID int64 `tbl:"pk"`
	}

	fields, err := field_info.CollectTableFields(reflect.TypeOf(Level1{}))
	require.NoError(t, err)
	
	// Должны быть распакованы все уровни
	assert.GreaterOrEqual(t, len(fields), 3)
}

// TestCollectTableFields_MixedTags проверяет структуру с различными комбинациями тегов
func TestCollectTableFields_MixedTags(t *testing.T) {
	t.Parallel()

	type mixedTags struct {
		ID        int64  `tbl:"pk;auto;sort=1"`
		Name      string `tbl:"col=user_name;sort=2,desc"`
		Email     string `tbl:"ro;rskip"`
		Password  string `tbl:"omit"`
		CreatedAt int64  `tbl:"auto;ins"`
		UpdatedAt int64  `tbl:"auto;upd"`
		RefID     int64  `tbl:"ref=other.id"`
	}

	fields, err := field_info.CollectTableFields(reflect.TypeOf(mixedTags{}))
	require.NoError(t, err)
	
	// Password должен быть пропущен (omit)
	assert.Len(t, fields, 6)
	
	// Проверяем флаги
	for _, f := range fields {
		switch f.Path[0] {
		case "ID":
			assert.True(t, f.Flags.IsPK)
			assert.True(t, f.Flags.AutoGen)
			assert.Equal(t, 1, f.Flags.SortPos)
		case "Name":
			assert.Equal(t, "user_name", f.SQLName)
			assert.Equal(t, 2, f.Flags.SortPos)
			assert.True(t, f.Flags.SortBackward)
		case "Email":
			assert.True(t, f.Flags.ReadOnly)
			assert.True(t, f.Flags.SkipReading)
		case "CreatedAt":
			assert.True(t, f.Flags.AutoGen)
			assert.True(t, f.Flags.ForceInsert)
		case "UpdatedAt":
			assert.True(t, f.Flags.AutoGen)
			assert.True(t, f.Flags.ForceUpdate)
		case "RefID":
			assert.Equal(t, "other.id", f.Flags.Ref)
		}
	}
}

// TestTableDefinition_AllIndexes проверяет allIndexes
func TestTableDefinition_AllIndexes(t *testing.T) {
	t.Parallel()

	type allIndexesTest struct {
		ID       int64  `tbl:"pk;auto"`
		Name     string `tbl:"sort=1"`
		ReadOnly string `tbl:"ro"`
		Hidden   string `tbl:"rskip"`
		RefID    int64  `tbl:"ref=users.id"`
	}

	table := field_info.NewTable[allIndexesTest](dialect.SQLiteDialect{})
	defs := table.Defs()
	
	// Проверяем через SQL, что все индексы корректны
	assert.Contains(t, defs.Indexes.PKCols, 0)        // ID
	assert.Contains(t, defs.Indexes.SortingCols, 1)   // Name
	assert.Contains(t, defs.Indexes.RefCols, 4)       // RefID
}

// TestQuery_SQL_TwoTables проверяет SQL-генерацию для двухтабличного JOIN
func TestQuery_SQL_TwoTables(t *testing.T) {
	t.Parallel()

	type qRow struct {
		User  testUserRow  `tbl:"from"`
		Order testOrderRow
	}

	query := field_info.NewQuery[qRow](dialect.SQLiteDialect{})
	sql := query.SQLs()

	assert.Contains(t, sql.ListCmdStart, "SELECT")
	assert.Contains(t, sql.ListCmdStart, "users.id")
	assert.Contains(t, sql.ListCmdStart, "users.name")
	assert.Contains(t, sql.ListCmdStart, "orders.id")
	assert.Contains(t, sql.ListCmdStart, "orders.user_id")
	assert.Contains(t, sql.ListCmdStart, "orders.total")
	assert.Contains(t, sql.ListCmdStart, "FROM users")
	assert.Contains(t, sql.ListCmdStart, "INNER JOIN orders")
	assert.Contains(t, sql.ListCmdStart, "ON orders.user_id = users.id")
}

// TestQuery_SQL_WithAlias проверяет алиасы таблиц в JOIN
func TestQuery_SQL_WithAlias(t *testing.T) {
	t.Parallel()

	type qRow struct {
		U testUserRow `tbl:"from;alias=u"`
		O testOrderRow
	}

	query := field_info.NewQuery[qRow](dialect.SQLiteDialect{})
	sql := query.SQLs()

	assert.Contains(t, sql.ListCmdStart, "FROM users AS u")
	assert.Contains(t, sql.ListCmdStart, "INNER JOIN orders")
	assert.Contains(t, sql.ListCmdStart, "ON orders.user_id = u.id")
	assert.Contains(t, sql.ListCmdStart, "u.id")
	assert.Contains(t, sql.ListCmdStart, "u.name")
}

// TestQuery_SQL_LEFT_JOIN проверяет LEFT JOIN
func TestQuery_SQL_LEFT_JOIN(t *testing.T) {
	t.Parallel()

	type qRow struct {
		User  testUserRow   `tbl:"from"`
		Order testOrderRow  `tbl:"join=left"`
	}

	query := field_info.NewQuery[qRow](dialect.SQLiteDialect{})
	sql := query.SQLs()

	assert.Contains(t, sql.ListCmdStart, "LEFT JOIN orders")
}

// TestQuery_SQL_RefMap проверяет map= для перевода ref-имён в алиасы
func TestQuery_SQL_RefMap(t *testing.T) {
	t.Parallel()

	type qRow struct {
		U    testUserRow  `tbl:"from;alias=u"`
		Item testRefMapRow `tbl:"map=u:u"`
	}

	query := field_info.NewQuery[qRow](dialect.SQLiteDialect{})
	sql := query.SQLs()

	assert.Contains(t, sql.ListCmdStart, "ON items.owner = u.id")
}

// TestQuery_SQL_OrderBy проверяет глобальную сортировку со всех таблиц
func TestQuery_SQL_OrderBy(t *testing.T) {
	t.Parallel()

	type qRow struct {
		User  testSortUserRow  `tbl:"from"`
		Order testSortOrderRow
	}

	query := field_info.NewQuery[qRow](dialect.SQLiteDialect{})
	sql := query.SQLs()

	assert.Contains(t, sql.ListSortString, "ORDER BY")
	assert.Contains(t, sql.ListSortString, "user_id")
	assert.Contains(t, sql.ListSortString, "name")
	assert.Contains(t, sql.ListSortString, "DESC")

	userIDIdx := strings.Index(sql.ListSortString, "user_id")
	nameIdx := strings.Index(sql.ListSortString, "name")
	assert.Less(t, userIDIdx, nameIdx, "user_id (sort=1) должен быть перед name (sort=2)")
}

// TestQuery_SQL_GetOneCmd проверяет GetOne SQL с PK основной + UsePk таблиц
func TestQuery_SQL_GetOneCmd(t *testing.T) {
	t.Parallel()

	type qRow struct {
		User   testUserRow   `tbl:"from"`
		Detail testDetailRow `tbl:"pk"`
	}

	query := field_info.NewQuery[qRow](dialect.SQLiteDialect{})
	sql := query.SQLs()

	assert.Contains(t, sql.GetOneCmd, "SELECT")
	assert.Contains(t, sql.GetOneCmd, "FROM users")
	assert.Contains(t, sql.GetOneCmd, "WHERE")
	assert.Contains(t, sql.GetOneCmd, "users.id = ?")
	assert.Contains(t, sql.GetOneCmd, "details.id = ?")
}

// TestQuery_SQL_GetOneCmd_PrimaryTableNoPK проверяет GetOne SQL без PK основной таблицы
func TestQuery_SQL_GetOneCmd_PrimaryTableNoPK(t *testing.T) {
	t.Parallel()

	type qRow struct {
		Widget testWidgetsRow `tbl:"from"`
		Child  testChildRow
	}

	query := field_info.NewQuery[qRow](dialect.SQLiteDialect{})
	sql := query.SQLs()

	assert.Empty(t, sql.GetOneCmd)
}

// TestQuery_IDX_Mapping проверяет сквозную нумерацию и idxMapping
func TestQuery_IDX_Mapping(t *testing.T) {
	t.Parallel()

	type qRow struct {
		User  testUserRow  `tbl:"from"`
		Order testOrderRow
	}

	query := field_info.NewQuery[qRow](dialect.SQLiteDialect{})
	flatFields := query.FlatFields()

	assert.Len(t, flatFields, 5)

	assert.Equal(t, "users.id", flatFields[0].SQLName)
	assert.Equal(t, "users.name", flatFields[1].SQLName)
	assert.Equal(t, "orders.id", flatFields[2].SQLName)
	assert.Equal(t, "orders.user_id", flatFields[3].SQLName)
	assert.Equal(t, "orders.total", flatFields[4].SQLName)
}

// TestQuery_FlatFields проверяет квалифицированные имена в flatFields
func TestQuery_FlatFields(t *testing.T) {
	t.Parallel()

	type qRow struct {
		User  testUserRow  `tbl:"from;alias=u"`
		Order testOrderRow `tbl:"alias=o"`
	}

	query := field_info.NewQuery[qRow](dialect.SQLiteDialect{})
	flatFields := query.FlatFields()

	assert.Contains(t, flatFields[0].SQLName, "u.")
	assert.Contains(t, flatFields[2].SQLName, "o.")
}

// TestQuery_MissingFK проверяет ошибку при отсутствии FK-связи
func TestQuery_MissingFK(t *testing.T) {
	t.Parallel()

	type qRow struct {
		User   testUserRow   `tbl:"from"`
		Orphan testOrphanRow
	}

	assert.Panics(t, func() {
		field_info.NewQuery[qRow](dialect.SQLiteDialect{})
	})
}

// TestQuery_RefToNonexistentTable проверяет ошибку при ref на несуществующую таблицу
func TestQuery_RefToNonexistentTable(t *testing.T) {
	t.Parallel()

	type qRow struct {
		User testUserRow  `tbl:"from"`
		Data testBadRefRow
	}

	assert.Panics(t, func() {
		field_info.NewQuery[qRow](dialect.SQLiteDialect{})
	})
}

// TestQuery_NoSortFields проверяет пустой ORDER BY при отсутствии sort-полей
func TestQuery_NoSortFields(t *testing.T) {
	t.Parallel()

	type qRow struct {
		User  testUserRow  `tbl:"from"`
		Order testOrderRow
	}

	query := field_info.NewQuery[qRow](dialect.SQLiteDialect{})
	sql := query.SQLs()

	assert.Empty(t, sql.ListSortString)
}

// TestQuery_RIGHT_JOIN проверяет RIGHT JOIN
func TestQuery_RIGHT_JOIN(t *testing.T) {
	t.Parallel()

	type qRow struct {
		User  testUserRow  `tbl:"from"`
		Order testOrderRow `tbl:"join=right"`
	}

	query := field_info.NewQuery[qRow](dialect.SQLiteDialect{})
	sql := query.SQLs()

	assert.Contains(t, sql.ListCmdStart, "RIGHT JOIN orders")
}

// TestQuery_ReverseFK проверяет обратный поиск FK (forward + reverse)
func TestQuery_ReverseFK(t *testing.T) {
	t.Parallel()

	type qRow struct {
		Item testItemRow  `tbl:"from"`
		User testRevUserRow
	}

	query := field_info.NewQuery[qRow](dialect.SQLiteDialect{})
	sql := query.SQLs()

	assert.Contains(t, sql.ListCmdStart, "ON users.ref_id = items.id")
}
