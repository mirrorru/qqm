package field_info

import (
	"reflect"

	"github.com/mirrorru/dot"
)

type Query[QROW any] struct {
	tables []TableDefinition
	names  []string
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
	names := make([]string, 0, t.NumField())
	for idx := range t.NumField() {
		sf := t.Field(idx)
		if !sf.IsExported() || sf.Anonymous {
			continue
		}

		sqlName := getTableName(sf.Type)
		names = append(names, sqlName)

		tables = append(tables, TableDefinition{
			TableName: sqlName,
			Fields:    dot.MustMake(CollectTableFields(t.Field(idx).Type)),
		})
		_ = sf.Tag.Get(tagName)
	}
	return &Query[QROW]{
		tables: tables,
		names:  names,
	}
}
