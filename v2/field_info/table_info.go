package field_info

import (
	"reflect"

	"github.com/mirrorru/qqm/meta"
)

type SQLNamer interface {
	SQLName() string
}

type sqlTexts struct {
	InsertCmd      string
	UpdateCmd      string
	DeleteCmd      string
	GetOneCmd      string
	ListCmdStart   string
	ListSortString string
}

type Filter interface{}

func getTableName(t reflect.Type) string {
	zero := reflect.New(t).Interface()
	if namer, ok := zero.(SQLNamer); ok {
		return namer.SQLName()
	}
	return meta.ToSnakeCase(t.Name())
}
