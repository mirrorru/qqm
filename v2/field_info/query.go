package field_info

import (
	"reflect"

	"github.com/mirrorru/dot"
)

type Query[QROW any] struct {
	tables     []TableDefinition
	names      []string
	flags      []TableFlags
	primaryIdx int
}

func NewQuery[QROW any]() *Query[QROW] {
	var (
		ptr *QROW
	)
	t := reflect.TypeOf(ptr).Elem()

	if t.Kind() != reflect.Struct {
		panic("QROW must be a struct")
	}
	tables := make([]TableDefinition, 0, t.NumField())
	flags := make([]TableFlags, 0, t.NumField())
	names := make([]string, 0, t.NumField())
	primaryIdx := -1
	for idx := range t.NumField() {
		sf := t.Field(idx)
		if !sf.IsExported() || sf.Anonymous {
			continue
		}
		tFlags, ok := parseTableTag(sf.Tag.Get(tagName))
		if !ok {
			continue
		}
		if tFlags.IsFrom {
			if primaryIdx != -1 {
				panic("multiple primary tag fields found")
			}
			primaryIdx = idx
		}
		sqlName := getTableName(sf.Type)

		flags = append(flags, tFlags)
		names = append(names, sqlName)
		tables = append(tables, TableDefinition{
			TableName: sqlName,
			Fields:    dot.MustMake(CollectTableFields(t.Field(idx).Type)),
		})
		_ = sf.Tag.Get(tagName)
	}

	return &Query[QROW]{
		tables:     tables,
		names:      names,
		flags:      flags,
		primaryIdx: max(primaryIdx, 0),
	}
}
