package main

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/mirrorru/dot"
	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/txproc"
	"github.com/mirrorru/qqm/v2/field_info"
	_ "modernc.org/sqlite"
)

type Users struct {
	ID   int64 `tbl:"pk;auto"`
	Name string
	Age  int
}

type Orders struct {
	ID     int64 `tbl:"pk;auto"`
	UserID int64 `tbl:"ref=users.id"`
	Item   string
	Qty    int
}

type UsersOrders struct {
	User  Users
	Order Orders `tbl:"join=left"`
}

func main() {
	pg := dot.MustMake(sql.Open("pgx", "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"))
	defer func() { _ = pg.Close() }()

	mem := dot.MustMake(sql.Open("sqlite", ":memory:"))
	defer func() { _ = mem.Close() }()

	dot.MustMake(mem.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			age int NOT NULL
		)
	`))
	dot.MustMake(mem.Exec(`
		CREATE TABLE orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL REFERENCES user(id) ON DELETE CASCADE,
			item TEXT NOT NULL,
			qty int NOT NULL
		)
	`))

	ctx := context.Background()

	var err error
	tUser := field_info.NewTable[Users](dialect.SQLiteDialect{})
	fmt.Println(tUser.SQLs().GetOneCmd)
	tOrder := field_info.NewTable[Orders](dialect.SQLiteDialect{})
	fmt.Println(tOrder.SQLs().GetOneCmd)
	qUserOrder := field_info.NewQuery[UsersOrders](dialect.SQLiteDialect{})
	fmt.Println(qUserOrder.SQLs().GetOneCmd)
	tx := txproc.NewDBAdapterVal(mem)

	alice, _, err := tUser.Ins(ctx, tx, &Users{Name: "Alice", Age: 11})
	fmt.Println(alice, err)
	fmt.Println(tOrder.Ins(ctx, tx, &Orders{UserID: alice.ID, Item: "Item01", Qty: 1}))
	fmt.Println(tOrder.Ins(ctx, tx, &Orders{UserID: alice.ID, Item: "Item02", Qty: 2}))

	bob, _, err := tUser.Ins(ctx, tx, &Users{Name: "Bob", Age: 22})
	fmt.Println(bob, err)
	fmt.Println(tOrder.Ins(ctx, tx, &Orders{UserID: bob.ID, Item: "Item21", Qty: 21}))

	clare, _, err := tUser.Ins(ctx, tx, &Users{Name: "Clare", Age: 33})
	fmt.Println(clare, err)
	rows := dot.MustMake(qUserOrder.Many(ctx, tx, nil))
	for _, row := range rows {
		fmt.Println("row:", *row)
	}
	fmt.Println()

	fmt.Println(qUserOrder.One(ctx, tx, alice.ID))
	fmt.Println(qUserOrder.One(ctx, tx, bob.ID))
	fmt.Println(qUserOrder.One(ctx, tx, clare.ID))
	fmt.Println(qUserOrder.One(ctx, tx, -1))

}
