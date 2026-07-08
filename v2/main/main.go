package main

import (
	"fmt"

	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/v2/field_info"
	"github.com/mirrorru/qqm/v2/test_structs"
)

func main() {
	//	var ptr *test_structs.ComplexRow
	//	t := reflect.TypeOf(ptr)
	//	fis, err := field_info.CollectTableFields(t)
	//	if err != nil {
	//		panic(err)
	//	}
	//	for _, fi := range fis {
	//		fmt.Println(fi.SQLName, fi.Path)
	//	}
	tbl := field_info.NewTable[test_structs.SimpleRow](dialect.PostgreSQLDialect{})
	fmt.Println("Fields")
	for pos, fld := range tbl.Defs().Fields {
		fmt.Printf("%d: %+v\n", pos, fld)
	}
	sqls := tbl.Defs().SQL
	fmt.Println("insert:", sqls.InsertCmd)
	fmt.Println("getOne:", sqls.GetOneCmd)
	fmt.Println("update:", sqls.UpdateCmd)
	fmt.Println("delete:", sqls.DeleteCmd)
	fmt.Println("list:", sqls.ListCmdStart)
	fmt.Println("sort:", sqls.ListSortString)
	//fmt.Printf("%#v\n", fis)
}
