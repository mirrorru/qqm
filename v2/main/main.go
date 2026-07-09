package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/mirrorru/dot"
	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/txproc"
	"github.com/mirrorru/qqm/v2/field_info"
	"github.com/mirrorru/qqm/v2/test_structs"
)

func main() {
	ctx := context.Background()
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
	sqls := tbl.SQLs()
	fmt.Println("insert:", sqls.InsertCmd)
	fmt.Println("getOne:", sqls.GetOneCmd)
	fmt.Println("update:", sqls.UpdateCmd)
	fmt.Println("delete:", sqls.DeleteCmd)
	fmt.Println("list:", sqls.ListCmdStart)
	fmt.Println("sort:", sqls.ListSortString)
	//fmt.Printf("%#v\n", fis)

	db := dot.MustMake(sql.Open("pgx", "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"))
	defer db.Close()

	//dot.MustMake(db.Exec(test_structs.CreateSimpleTable))
	tx := txproc.NewDBAdapterVal(db)
	x := int(time.Now().Unix())
	row := &test_structs.SimpleRow{
		ID:        0,
		InsFld:    x,
		ReadFld:   x + 1,
		UpdFld:    x + 2,
		SecretFld: x + 3,
		FreeFld:   x + 4,
	}
	newRow, _, err := tbl.Ins(context.Background(), tx, row)
	if err != nil {
		fmt.Println("ins1", err)
		return
	}
	newRow2, _, err := tbl.Ins(context.Background(), tx, row)
	if err != nil {
		fmt.Println("ins2", err)
		return
	}
	newRow2.UpdFld = newRow2.UpdFld / 10
	newRow2, _, err = tbl.Upd(context.Background(), tx, newRow2)
	if err != nil {
		fmt.Println("upd2", err)
		return
	}
	fmt.Println("one1")
	fmt.Println(tbl.One(ctx, tx, newRow.ID))
	fmt.Println("one2")
	fmt.Println(tbl.One(ctx, tx, newRow2.ID))
	fmt.Println("many")
	list, err := tbl.Many(ctx, tx, nil)
	if err != nil {
		fmt.Println("many failed", err)
		return
	}
	for pos, l := range list {
		fmt.Printf("%d: %+v\n", pos, l)
	}
	fmt.Println()
}
