# qqm — Quick Query Maker

**qqm** is an ORM-like Go library for type-safe work with SQL databases through Go structs. It automatically generates SQL queries based on struct field tags and provides a simple CRUD interface, including multi-table SELECT with JOIN.

## Features

- **Typed tables** — `Table[ROW]` parameterized by your struct.
- **Multi-table queries** — `Query[QROW]` for SELECT with JOIN via ref-relationships.
- **Auto-generated SQL** — INSERT, UPDATE, SELECT, DELETE are built from struct metadata.
- **Dialect support** — SQLite (`?`) and PostgreSQL (`$1`, `$2`, …).
- **CRUD interface** — Insert, Update, GetByPK, Delete, List.
- **Flexible filtering** — And/Or combinations, operators Eq, Gt, Lt, Gte, Lte, Between, In.
- **Qualified names** — filters on joined table fields (`"Order.Amount"`).
- **LEFT JOIN with nil** — `*ROW` pointer fields automatically become nil when no matching row exists.
- **Field tags** — column, primary key, foreign key, update, auto, omit, join, table, primary, sort, create.
- **Embedded structs** — support for embedding with column prefix.
- **Named struct fields** — prefix for non-anonymous structs (e.g., multiple addresses).
- **Composite keys** — variable number of fields in PK.
- **SQL caching** — queries are generated once on first access.
- **No runtime reflection** — metadata is lazily collected and cached.

## Installation

```bash
go get github.com/mirrorru/qqm
```

## Quick Start

### Model Definition

```go
type User struct {
    ID    int64  `qqm:"pk"`
    Name  string
    Email string
    Age   int
}
```

Default naming rules:
- Table name — snake_case from struct name: `user`.
- Column name — snake_case from field name: `name`, `email`, `age`.

### Table Creation and SQL

```go
import "github.com/mirrorru/qqm"

userTable := qqm.NewTable[User](qqm.SQLiteDialect)

fmt.Println(userTable.Internals().InsertSQL())
// INSERT INTO user (id, name, email, age) VALUES (?, ?, ?, ?) RETURNING id, name, email, age

fmt.Println(userTable.Internals().SelectSQL())
// SELECT id, name, email, age FROM user WHERE id = ?
```

### Full CRUD

```go
import (
    "context"
    "database/sql"
    "github.com/mirrorru/qqm"
)

func Example() {
    db, _ := sql.Open("sqlite", ":memory:")
    db.Exec(`CREATE TABLE users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT, email TEXT, age INTEGER
    )`)

    ex := qqm.NewDBAdapterVal(db)
    ctx := context.Background()
    tbl := qqm.NewTable[User](qqm.SQLiteDialect)

    // Create
    u, _ := tbl.Insert(ctx, ex, &User{Name: "Alice", Email: "alice@test.com", Age: 25})

    // Read
    alice, _ := tbl.GetByPK(ctx, ex, u.ID)

    // Update
    alice.Age = 26
    tbl.Update(ctx, ex, alice)

    // List with filter
    result, _ := tbl.List(ctx, ex, qqm.Field("Age", qqm.And, qqm.Gt(20)))

    // Delete
    tbl.Delete(ctx, ex, u.ID)
}
```

## Column Configuration via Tags

Tag format: `qqm:"col=name;pk;ref=table.col;update;auto;omit;prefix=...;join=TYPE;table=...;primary;sort=<pos>[,dir];create=..."`

| Option | Description |
|--------|-------------|
| `col=name` | Column name in DB (default: snake_case from field name) |
| `pk` | Field is a primary key |
| `ref=table.col` | Foreign key reference |
| `prefix=...` | Prefix for columns from embedded or named struct |
| `update` | Allows UPDATE on auto field |
| `auto` | Excluded from INSERT (e.g., SERIAL) |
| `omit` | Fully excluded from SQL generation |
| `join=TYPE` | JOIN type for Query: LEFT, INNER, RIGHT, FULL |
| `table=...` | Override table name for Query field |
| `primary` | Explicit primary table marker in Query |
| `sort=<pos>[,dir]` | Position in ORDER BY for List() (1-based), direction ASC/DESC |
| `create=...` | Column definition string in CREATE TABLE (DEFAULT, UNIQUE, etc.) |

### Prefix for Named Struct Fields

The `prefix` tag works not only for embedded (anonymous) structs but also for named struct fields. This allows reusing the same Go struct for different table columns:

```go
type Address struct {
    City   string
    Street string
    Zip    string
}

type Person struct {
    ID          int64   `qqm:"pk"`
    Name        string
    HomeAddress Address `qqm:"prefix=home_"`
    WorkAddress Address `qqm:"prefix=work_"`
}
// Columns: id, name, home_city, home_street, home_zip, work_city, work_street, work_zip
```

## Multi-table Queries (JOIN)

`Query[QROW]` — typed SELECT with JOIN. QROW fields are existing ROW structs. JOIN conditions are auto-inferred from `ref=` tags.

### Query Struct Definition

```go
// ROW structs
type User struct {
    ID    int64  `qqm:"pk"`
    Name  string
    Email string
}

type Order struct {
    ID     int64   `qqm:"pk;auto"`
    UserID int64   `qqm:"ref=users.id"`
    Amount float64
}

// Query struct
type UserWithOrder struct {
    User  User    // INNER JOIN
    Order *Order  // LEFT JOIN (pointer → nil when no matching row)
}
```

### Usage

```go
q, err := qqm.NewQuery[UserWithOrder](qqm.SQLiteDialect)
// err != nil if no FK found for JOIN

results, err := q.List(ctx, ex,
    qqm.AndFilter(
        qqm.Field("User.Name", qqm.And, qqm.Eq("Alice")),
        qqm.Field("Order.Amount", qqm.And, qqm.Gt(100.0)),
    ),
)

for _, r := range results {
    fmt.Println(r.User.Name, r.Order) // Order == nil for LEFT JOIN with no match
}
```

### JOIN Inference Rules

| Field Type | Default JOIN |
|------------|--------------|
| `Order` (value) | INNER |
| `*Order` (pointer) | LEFT |

The ON clause is built from the `ref=users.id` tag on the `UserID` field of the `Order` struct: `orders.user_id = users.id`.

### Explicit JOIN Control

```go
type CustomQuery struct {
    User  User   `qqm:"table=app_users;primary"`  // name override + primary
    Order *Order `qqm:"join=LEFT"`                 // explicit JOIN type
}
```

### LEFT JOIN and nil

If there is no matching row in the joined table, the pointer field is set to nil:

```go
type UserWithOrderPtr struct {
    User  User
    Order *Order
}

results, _ := q.List(ctx, ex)
for _, r := range results {
    if r.Order == nil {
        fmt.Println(r.User.Name, "has no orders")
    }
}
```

### Filters with Qualified Names

Field names in filters: `"TableName.FieldName"`:

```go
qqm.AndFilter(
    qqm.Field("User.Name", qqm.And, qqm.Eq("Alice")),
    qqm.Field("Order.Amount", qqm.And, qqm.Gt(200.0)),
)
```

## Examples

### Composite Key

```go
type OrgUser struct {
    OrgID  int64  `qqm:"pk"`
    UserID int64  `qqm:"pk"`
    Role   string
}
```

### Custom Table Name

Implement the `qqm.SQLNamer` interface:

```go
func (u *OrgUser) SQLName() string { return "org_members" }
```

### Embedded Structs with Prefix

```go
type Audit struct {
    CreatedAt int64 `qqm:"col=created_at"`
    UpdatedAt int64 `qqm:"col=updated_at"`
}

type Post struct {
    ID    int64 `qqm:"pk"`
    Title string
    Audit `qqm:"prefix=audit_"`
}
// Columns: id, title, audit_created_at, audit_updated_at
```

### Filter Conditions

```go
// AND conditions
andFilter := qqm.AndFilter(
    qqm.Field("Age", qqm.And, qqm.Gte(18), qqm.Lte(60)),
    qqm.Field("Status", qqm.And, qqm.Eq("active")),
)

// OR conditions
orFilter := qqm.OrFilter(
    qqm.Field("Role", qqm.And, qqm.Eq("admin")),
    qqm.Field("Role", qqm.And, qqm.Eq("moderator")),
)

// Between
ageFilter := qqm.AndFilter(
    qqm.Field("Age", qqm.And, qqm.Between(18, 65)),
)

// In
nameFilter := qqm.AndFilter(
    qqm.Field("Name", qqm.And, qqm.In("Alice", "Bob", "Charlie")),
)
```

## Dialects

| Dialect | Placeholder | RETURNING |
|---------|-------------|-----------|
| `qqm.SQLiteDialect` | `?` | Yes |
| `qqm.PostgreSQLDialect` | `$1`, `$2`, … | Yes |

## Database Adapters

Use adapters from the root `qqm` package to pass to CRUD methods:

| Adapter | Constructor | For |
|---------|------------|-----|
| `DBAdapter` | `qqm.NewDBAdapterVal(db)` | `*sql.DB` |
| `TxAdapter` | `qqm.NewTxAdapterVal(tx)` | `*sql.Tx` |
| `PGXAdapter` | `qqm.NewPGXAdapterVal(conn)` | `*pgx.Conn` |
| `PGXTxAdapter` | `qqm.NewPGXTxAdapterVal(tx)` | `pgx.Tx` |

### Transactions

```go
tx, _ := db.BeginTx(ctx, nil)
ex := qqm.NewTxAdapterVal(tx)

inserted, err := tbl.Insert(ctx, ex, &User{Name: "Alice"})
if err != nil {
    _ = tx.Rollback()
    return err
}
_ = tx.Commit()
```

All CRUD methods (`Insert`, `Update`, `GetByPK`, `Delete`, `List`) work with both `DBAdapter` and `TxAdapter`.

## Executor Interface

The `qqm` package defines the interface for SQL execution abstraction:

```go
type Executor interface {
    ExecContext(ctx context.Context, query string, args ...any) (Result, error)
    QueryContext(ctx context.Context, query string, args ...any) (Rows, error)
    QueryRowContext(ctx context.Context, query string, args ...any) Row
}
```

- `QueryRowContext` — for queries returning a single row (Insert RETURNING, GetByPK).
