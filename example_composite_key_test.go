package qqm_test

import (
	"fmt"

	"github.com/mirrorru/qqm"
	"github.com/mirrorru/qqm/dialect"
)

// OrgUser — структура с составным ключом
type OrgUser struct {
	OrgID  int64 `qqm:"pk"`
	UserID int64 `qqm:"pk"`
	Name   string
	Email  string
}

func (o *OrgUser) SQLName() string { return "org_users" }

// Example_compositeKey demonstrates usage with a composite key
func Example_compositeKey() {
	orgUserTable := qqm.NewTable[OrgUser](dialect.SQLiteDialect{})

	fmt.Println("INSERT:", orgUserTable.Internals().InsertSQL())
	fmt.Println("UPDATE:", orgUserTable.Internals().UpdateSQL())
	fmt.Println("SELECT:", orgUserTable.Internals().SelectSQL())
	fmt.Println("DELETE:", orgUserTable.Internals().DeleteSQL())

	meta := orgUserTable.Internals().Meta()
	fmt.Println("Table:", meta.TableName)
	fmt.Println("Columns:", meta.Columns)
	fmt.Println("PK count:", len(meta.PKFields))
	for _, pk := range meta.PKFields {
		fmt.Printf("  PK: %s (order: %d)\n", pk.Column, pk.PkOrder)
	}

	// Output:
	// INSERT: INSERT INTO org_users (org_id, user_id, name, email) VALUES (?, ?, ?, ?) RETURNING org_id, user_id, name, email
	// UPDATE: UPDATE org_users SET name = ?, email = ? WHERE org_id = ? AND user_id = ? RETURNING org_id, user_id, name, email
	// SELECT: SELECT org_id, user_id, name, email FROM org_users WHERE org_id = ? AND user_id = ?
	// DELETE: DELETE FROM org_users WHERE org_id = ? AND user_id = ?
	// Table: org_users
	// Columns: [org_id user_id name email]
	// PK count: 2
	//   PK: org_id (order: 1)
	//   PK: user_id (order: 2)
}
