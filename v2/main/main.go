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
	tbl := field_info.NewTable[test_structs.ComplexRow](dialect.PostgreSQLDialect{})
	fmt.Println(tbl.InsertSQL)
	//fmt.Printf("%#v\n", fis)
}
