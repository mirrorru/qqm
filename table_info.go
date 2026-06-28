package qqm

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

func getTableName(t reflect.Type) string {
	zero := reflect.New(t).Interface()
	if namer, ok := zero.(SQLNamer); ok {
		return namer.SQLName()
	}
	return meta.ToSnakeCase(t.Name())
}

func isKey(key, val string) bool {
	l := len(key)
	return len(val) >= l && val[:l] == key
}
