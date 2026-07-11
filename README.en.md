# qqm — Quick Query Maker

**qqm** is an ORM-like Go library for type-safe work with SQL databases. It automatically generates SQL queries based on `tbl` struct field tags and provides a CRUD interface, including multi-table SELECT with JOIN.

## Features

- **Typed tables** — `Table[ROW]` parameterized by your struct.
- **Multi-table queries** — `Query[QROW]` for SELECT with JOIN via ref-relationships.
- **Auto-generated SQL** — INSERT, UPDATE, SELECT, DELETE are built from struct metadata.
- **Dialect support** — SQLite (`?`) and PostgreSQL (`$1`, `$2`, …).
- **CRUD interface** — `Ins`, `Upd`, `One`, `Del`, `Many`.
- **Flexible filtering** — tree of conditions: `And`/`Or`/`Not` groups, operators Eq, Gt, Lt, Like, ILike, In, IsNull.
- **LEFT JOIN nulling** — joined table fields are zeroed when no matching row exists.
- **`tbl` tags** — PK, FK, read-only, auto-generated, prefixes, sorting.
- **Nested structs** — embedded and named struct fields with prefixes.
- **Composite keys** — arbitrary number of PK fields.
- **SQL caching** — queries are generated once in `NewTable`/`NewQuery`.
- **No runtime reflection** — metadata is lazily collected and cached.

## Installation

```bash
go get github.com/mirrorru/qqm
```

## Quick Start

### Model Definition

```go
type User struct {
    ID    int64  `tbl:"pk;auto"`
    Name  string
    Email string
    Age   int
}

func (u *User) SQLName() string { return "users" }
```

Default naming rules:
- Table name — `SQLName()` if implemented, otherwise snake_case from struct name.
- Column name — snake_case from field name: `name`, `email`, `age`.

### Full CRUD

```go
import (
    "context"
    "database/sql"
    "github.com/mirrorru/qqm"
    "github.com/mirrorru/qqm/txproc"
)

func Example() {
    db, _ := sql.Open("sqlite", ":memory:")
    db.Exec(`CREATE TABLE users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL, email TEXT NOT NULL
    )`)

    ex := txproc.NewDBAdapterVal(db)
    ctx := context.Background()
    tbl := qqm.NewTable[User](qqm.SQLiteDialect)

    // Create — returns inserted row (RETURNING)
    inserted, _, err := tbl.Ins(ctx, ex, &User{Name: "Alice", Email: "alice@test.com"})

    // Read — by PK
    alice, err := tbl.One(ctx, ex, inserted.ID)

    // Update — returns updated row (RETURNING)
    alice.Name = "Alice Updated"
    returned, _, err := tbl.Upd(ctx, ex, alice)

    // Delete — by PK
    delResult, err := tbl.Del(ctx, ex, alice.ID)

    // Many — SELECT with filter and sorting
    filter := &qqm.Filter{
        Range: qqm.And(qqm.Cond(1, qqm.CmdGt, 20)),
    }
    results, err := tbl.Many(ctx, ex, filter)
}
```

## Column Configuration via Tags

Tag format: `tbl:"pk;ro;auto;embed;omit;ins;upd;rskip;col=name;prefix=...;ref=...;sort=<pos>[:dir]"`

| Option | Description |
|--------|-------------|
| `pk` | Primary key |
| `ro` | Read-only (SELECT only, excluded from INSERT/UPDATE) |
| `auto` | Auto-generated (excluded from INSERT unless `ins`) |
| `embed` | Force struct unpacking |
| `omit` | Fully excluded from SQL |
| `ins` | Force inclusion in INSERT (even for auto) |
| `upd` | Force inclusion in UPDATE (even for ro/auto) |
| `rskip` | Exclude from SELECT (read skip) |
| `col=name` | Column name in DB (default: snake_case from field name) |
| `prefix=...` | Prefix for columns from embedded or named struct |
| `ref=table:col` | Foreign key reference |
| `sort=<pos>[:dir]` | Position in ORDER BY (1-based), direction ASC/DESC |

### Prefix for Named Struct Fields

The `prefix` tag works for both embedded and named struct fields:

```go
type Address struct {
    City   string
    Street string
    Zip    string
}

type Person struct {
    ID          int64   `tbl:"pk"`
    Name        string
    HomeAddress Address `tbl:"prefix=home_"`
    WorkAddress Address `tbl:"prefix=work_"`
}
// Columns: id, name, home_city, home_street, home_zip, work_city, work_street, work_zip
```

Flags are inherited from parent struct fields: `ro`, `auto`, `ins`, `upd`, `rskip`, `prefix`, `sort`.

## Multi-table Queries (JOIN)

`Query[QROW]` — typed SELECT with JOIN. JOIN conditions are auto-inferred from `ref=` tags on ROW struct fields.

### Query Struct Definition

```go
type User struct {
    ID    int64  `tbl:"pk"`
    Name  string
    Email string
}

func (u *User) SQLName() string { return "users" }

type Order struct {
    ID     int64   `tbl:"pk;auto"`
    UserID int64   `tbl:"ref=users:id"`
    Amount float64
}

func (o *Order) SQLName() string { return "orders" }

// Query struct
type UserWithOrder struct {
    User  User  `tbl:"from"`       // FROM users (primary table)
    Order Order `tbl:"join=left"`  // LEFT JOIN orders ON orders.user_id = users.id
}
```

### Using Query

```go
query := qqm.NewQuery[UserWithOrder](qqm.SQLiteDialect)

// Many — SELECT with JOIN and filter
results, err := query.Many(ctx, ex, &qqm.Filter{
    Range: qqm.And(
        qqm.Cond(1, qqm.CmdEq, "Alice"),      // users.name = ?
        qqm.Cond(5, qqm.CmdGt, 200.0),         // orders.amount > ?
    ),
})

// One — SELECT with JOIN by primary table PK
row, err := query.One(ctx, ex, int64(1))
```

### Query Field Tags

Format: `tbl:"from;join=left;alias=...;map=k1:v1;pk;omit;sort=<pos>"`

| Option | Description |
|--------|-------------|
| `from` | Primary table (FROM). Must have exactly one. |
| `join=left\|right\|inner` | JOIN type. Default: inner. |
| `alias=...` | Table alias in SQL |
| `map=k1:v1,k2:v2` | Mapping of ref-table names for JOIN ON |
| `pk` | Use this table's PK in WHERE for Query.One |
| `omit` | Fully exclude table from Query |
| `sort=<pos>` | Table sort priority in ORDER BY |

### LEFT JOIN and Nulling

When LEFT JOIN finds no match, all joined struct fields are zeroed:

```go
// For a user without orders
row, _ := query.One(ctx, ex, userWithoutOrdersID)
// row.Order.ID == 0, row.Order.Amount == 0.0
```

## Filtering

Filters are built as a tree of nodes: `And`/`Or`/`Not` groups with `ConditionNode` leaves.

```go
type Filter struct {
    Offset uint32      // OFFSET
    Limit  uint32      // LIMIT
    Range  FilterNode  // condition tree
}
```

### Constructors

```go
// Condition: Cond(fieldIdx, CommandOp, value)
nameEq := qqm.Cond(1, qqm.CmdEq, "Alice")

// Groups
qqm.And(nameEq, qqm.Cond(2, qqm.CmdGt, 18))     // AND
qqm.Or(qqm.Cond(3, qqm.CmdEq, "admin"), ...)     // OR
qqm.Not(qqm.Cond(1, qqm.CmdIsNull))              // NOT
```

### Operators

| Constant | SQL |
|----------|-----|
| `CmdEq` | `= ?` |
| `CmdNotEq` | `<> ?` |
| `CmdGt` | `> ?` |
| `CmdGte` | `>= ?` |
| `CmdLt` | `< ?` |
| `CmdLte` | `<= ?` |
| `CmdLike` | `LIKE ?` |
| `CmdILike` | `ILIKE ?` (PG) / `LOWER() LIKE LOWER()` (SQLite) |
| `CmdIn` | `IN (?, ?, ...)` |
| `CmdIsNull` | `IS NULL` |
| `CmdIsNotNull` | `IS NOT NULL` |

### Field Indexes

Indexes for `Cond()` are the position of the field in the flat list `TableDefinition.Fields` or `Query.FlatFields()`. The order corresponds to the field order in the struct (accounting for embedded unpacking and skipping omit/rskip).

### Examples

```go
// Simple filter: name = "Alice" AND age > 18
filter := &qqm.Filter{
    Range: qqm.And(
        qqm.Cond(1, qqm.CmdEq, "Alice"),
        qqm.Cond(2, qqm.CmdGt, 18),
    ),
}

// OR: role = "admin" OR role = "moderator"
filter := &qqm.Filter{
    Range: qqm.Or(
        qqm.Cond(3, qqm.CmdEq, "admin"),
        qqm.Cond(3, qqm.CmdEq, "moderator"),
    ),
}

// NOT: email IS NOT NULL
filter := &qqm.Filter{
    Range: qqm.Not(qqm.Cond(2, qqm.CmdIsNull)),
}

// IN: name IN ("Alice", "Bob", "Charlie")
filter := &qqm.Filter{
    Range: qqm.And(qqm.Cond(1, qqm.CmdIn, []any{"Alice", "Bob", "Charlie"})),
}

// Pagination: OFFSET 10 LIMIT 20
filter := &qqm.Filter{
    Offset: 10,
    Limit:  20,
}
```

## Dialects

| Dialect | Placeholder | RETURNING | ILIKE |
|---------|-------------|-----------|-------|
| `qqm.SQLiteDialect` | `?` | Yes | `LOWER() LIKE LOWER()` |
| `qqm.PostgreSQLDialect` | `$1`, `$2`, … | Yes | `ILIKE` |

## Database Adapters

Adapters in `txproc` package:

| Adapter | Constructor | For |
|---------|------------|-----|
| `txproc.DBAdapter` | `txproc.NewDBAdapterVal(db)` | `*sql.DB` |
| `txproc.TxAdapter` | `txproc.NewTxAdapterVal(tx)` | `*sql.Tx` |
| `txproc.PGXAdapter` | `txproc.NewPGXAdapterVal(conn)` | `*pgx.Conn` |
| `txproc.PGXTxAdapter` | `txproc.NewPGXTxAdapterVal(tx)` | `pgx.Tx` |

### Transactions

```go
tx, _ := db.BeginTx(ctx, nil)
ex := txproc.NewTxAdapterVal(tx)

inserted, _, err := tbl.Ins(ctx, ex, &User{Name: "Alice"})
if err != nil {
    _ = tx.Rollback()
    return err
}
_ = tx.Commit()
```

All CRUD methods (`Ins`, `Upd`, `One`, `Del`, `Many`) work with any `txproc.TxProcessor`.

## Examples

### Composite Key

```go
type OrgUser struct {
    OrgID  int64 `tbl:"pk"`
    UserID int64 `tbl:"pk"`
    Role   string
}

func (o *OrgUser) SQLName() string { return "org_users" }

// Usage
tbl := qqm.NewTable[OrgUser](qqm.SQLiteDialect)
row, err := tbl.One(ctx, ex, int64(1), int64(42))
```

### Custom Table Name

```go
func (u *OrgUser) SQLName() string { return "org_members" }
```

### Embedded Structs with Prefix

```go
type Audit struct {
    CreatedAt string `tbl:"col=created_at;auto"`
    UpdatedAt string `tbl:"col=updated_at;auto;upd"`
}

type Post struct {
    ID    int64 `tbl:"pk"`
    Title string
    Audit `tbl:"prefix=audit_"`
}
// Columns: id, title, audit_created_at, audit_updated_at
```

### Sorting

```go
type UserWithSort struct {
    ID    int64  `tbl:"pk;auto"`
    Name  string `tbl:"sort=1"`       // ORDER BY name ASC
    Email string `tbl:"sort=2:desc"`  // then email DESC
    Age   int
}
```

### Auto Fields

```go
type Timestamps struct {
    CreatedAt string `tbl:"col=created_at;auto"`      // not in INSERT
    UpdatedAt string `tbl:"col=updated_at;auto;upd"`  // UPDATE only
}
```

## TxProcessor Interface

```go
type TxProcessor interface {
    ExecContext(ctx context.Context, query string, args ...any) (Result, error)
    QueryContext(ctx context.Context, query string, args ...any) (Rows, error)
    QueryRowContext(ctx context.Context, query string, args ...any) Row
}
```

- `QueryRowContext` — for single-row queries (Ins/Upd with RETURNING, One).
- `QueryContext` — for `Many` (multiple rows).
- `ExecContext` — for `Del` and Ins/Upd without RETURNING.
