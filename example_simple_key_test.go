package qqm_test

import (
	"fmt"

	"github.com/mirrorru/qqm"
	"github.com/mirrorru/qqm/dialect"
)

// User — структура с простым ключом
type User struct {
	ID    int64 `qqm:"pk"`
	Name  string
	Email string
	Age   int
}

// Example_simpleKey demonstrates usage with a simple key
func Example_simpleKey() {
	userTable := qqm.NewTable[User](dialect.SQLiteDialect{})

	fmt.Println("INSERT:", userTable.Internals().InsertSQL())
	fmt.Println("UPDATE:", userTable.Internals().UpdateSQL())
	fmt.Println("SELECT:", userTable.Internals().SelectSQL())
	fmt.Println("DELETE:", userTable.Internals().DeleteSQL())

	meta := userTable.Internals().Meta()
	fmt.Println("Table:", meta.TableName)
	fmt.Println("Columns:", meta.Columns)

	// Output:
	// INSERT: INSERT INTO user (id, name, email, age) VALUES (?, ?, ?, ?) RETURNING id, name, email, age
	// UPDATE: UPDATE user SET name = ?, email = ?, age = ? WHERE id = ?
	// SELECT: SELECT id, name, email, age FROM user WHERE id = ?
	// DELETE: DELETE FROM user WHERE id = ?
	// Table: user
	// Columns: [id name email age]
}
