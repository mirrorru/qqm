# qqm — Quick Query Maker

[English version](README.en.md)

**qqm** — ORM-подобная Go-библиотека для типизированной работы с SQL-базами данных. Автоматически генерирует SQL-запросы на основе тегов `tbl` в полях структур и предоставляет CRUD-интерфейс, включая multi-table SELECT с JOIN.

## Возможности

- **Типизированные таблицы** — `Table[ROW]` параметризуется вашей структурой.
- **Multi-table запросы** — `Query[QROW]` для SELECT с JOIN по ref-связям.
- **Автогенерация SQL** — INSERT, UPDATE, SELECT, DELETE строятся по метаданным структуры.
- **Поддержка диалектов** — SQLite (`?`) и PostgreSQL (`$1`, `$2`, …).
- **CRUD-интерфейс** — `Ins`, `Upd`, `One`, `Del`, `Many`.
- **Гибкая фильтрация** — дерево условий: `And`/`Or`/`Not`-группы, операторы Eq, Gt, Lt, Like, ILike, In, IsNull.
- **LEFT JOIN с обнулением** — поля присоединённых таблиц обнуляются при отсутствии строки.
- **Теги `tbl`** — PK, FK, read-only, auto-генерация, префиксы, сортировка.
- **Вложенные структуры** — embedded и именованные поля-структуры с префиксами.
- **Составные ключи** — произвольное количество PK-полей.
- **Кеширование SQL** — запросы генерируются один раз в `NewTable`/`NewQuery`.
- **Без рефлексии в рантайме** — метаданные собираются лениво и кешируются.

## Установка

```bash
go get github.com/mirrorru/qqm
```

## Быстрый старт

### Определение модели

```go
type User struct {
    ID    int64  `tbl:"pk;auto"`
    Name  string
    Email string
    Age   int
}

func (u *User) SQLName() string { return "users" }
```

Правила именования по умолчанию:
- Имя таблицы — `SQLName()`, если реализован, иначе snake_case от имени структуры.
- Имя колонки — snake_case от имени поля: `name`, `email`, `age`.

### Полный CRUD

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

    // Create — возвращает вставленную строку (RETURNING)
    inserted, _, err := tbl.Ins(ctx, ex, &User{Name: "Alice", Email: "alice@test.com"})

    // Read — по PK
    alice, err := tbl.One(ctx, ex, inserted.ID)

    // Update — возвращает обновлённую строку (RETURNING)
    alice.Name = "Alice Updated"
    returned, _, err := tbl.Upd(ctx, ex, alice)

    // Delete — по PK
    delResult, err := tbl.Del(ctx, ex, alice.ID)

    // Many — SELECT с фильтром и сортировкой
    filter := &qqm.Filter{
        Range: qqm.And(qqm.Cond(1, qqm.CmdGt, 20)),
    }
    results, err := tbl.Many(ctx, ex, filter)
}
```

## Настройка колонок через теги

Формат тега: `tbl:"pk;ro;auto;embed;omit;ins;upd;rskip;col=name;prefix=...;ref=...;sort=<pos>[,dir]"`

| Опция | Описание |
|-------|----------|
| `pk` | Поле — первичный ключ |
| `ro` | Read-only (только SELECT, исключается из INSERT/UPDATE) |
| `auto` | Автогенерируемое поле (исключается из INSERT, если нет `ins`) |
| `embed` | Принудительная распаковка вложенной структуры |
| `omit` | Полное игнорирование поля |
| `ins` | Принудительное включение в INSERT (даже для auto) |
| `upd` | Принудительное включение в UPDATE (даже для ro/auto) |
| `rskip` | Исключение из SELECT (read skip) |
| `col=name` | Имя колонки в БД (по умолчанию: snake_case от имени поля) |
| `prefix=...` | Префикс колонок для embedded или именованной структуры |
| `ref=table.col` | Внешний ключ |
| `sort=<pos>[,dir]` | Позиция в ORDER BY (1-based), направление ASC/DESC |

### Префикс для именованных полей-структур

Тег `prefix` работает для embedded и именованных полей-структур:

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
// Колонки: id, name, home_city, home_street, home_zip, work_city, work_street, work_zip
```

Флаги наследуются от родительских полей-структур: `ro`, `auto`, `ins`, `upd`, `rskip`, `prefix`, `sort`.

## Multi-table запросы (JOIN)

`Query[QROW]` — типизированный SELECT с JOIN. JOIN-условия выводятся автоматически из тегов `ref=` на полях ROW-структур.

### Определение Query-структуры

```go
type User struct {
    ID    int64  `tbl:"pk"`
    Name  string
    Email string
}

func (u *User) SQLName() string { return "users" }

type Order struct {
    ID     int64   `tbl:"pk;auto"`
    UserID int64   `tbl:"ref=users.id"`
    Amount float64
}

func (o *Order) SQLName() string { return "orders" }

// Query-структура
type UserWithOrder struct {
    User  User  `tbl:"from"`       // FROM users (первичная таблица)
    Order Order `tbl:"join=left"`  // LEFT JOIN orders ON orders.user_id = users.id
}
```

### Использование Query

```go
query := qqm.NewQuery[UserWithOrder](qqm.SQLiteDialect)

// Many — SELECT с JOIN и фильтром
results, err := query.Many(ctx, ex, &qqm.Filter{
    Range: qqm.And(
        qqm.Cond(1, qqm.CmdEq, "Alice"),      // users.name = ?
        qqm.Cond(5, qqm.CmdGt, 200.0),         // orders.amount > ?
    ),
})

// One — SELECT с JOIN по PK первичной таблицы
row, err := query.One(ctx, ex, int64(1))
```

### Теги Query-полей

Формат: `tbl:"from;join=left;alias=...;map=k1:v1;pk;omit;sort=<pos>"`

| Опция | Описание |
|-------|----------|
| `from` | Первичная таблица (FROM). Должна быть ровно одна. |
| `join=left\|right\|inner` | Тип JOIN. По умолчанию: inner. |
| `alias=...` | Алиас таблицы в SQL |
| `map=k1:v1,k2:v2` | Маппинг имён ref-таблиц для JOIN ON |
| `pk` | Использовать PK этой таблицы в WHERE для Query.One |
| `omit` | Полностью исключить таблицу из Query |
| `sort=<pos>` | Приоритет сортировки таблицы в ORDER BY |

### LEFT JOIN и обнуление

Если в LEFT JOIN нет совпадений, все поля присоединённой структуры обнуляются (zero value):

```go
// Для пользователя без заказов
row, _ := query.One(ctx, ex, userWithoutOrdersID)
// row.Order.ID == 0, row.Order.Amount == 0.0
```

## Фильтрация

Фильтры строятся как дерево узлов: `And`/`Or`/`Not`-группы с `ConditionNode`-листьями.

```go
type Filter struct {
    Offset uint32      // OFFSET
    Limit  uint32      // LIMIT
    Range  FilterNode  // дерево условий
}
```

### Конструкторы

```go
// Условие: Cond(fieldIdx, CommandOp, value)
nameEq := qqm.Cond(1, qqm.CmdEq, "Alice")

// Группы
qqm.And(nameEq, qqm.Cond(2, qqm.CmdGt, 18))   // AND
qqm.Or(qqm.Cond(3, qqm.CmdEq, "admin"), ...)   // OR
qqm.Not(qqm.Cond(1, qqm.CmdIsNull))             // NOT
```

### Операторы

| Константа | SQL |
|-----------|-----|
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

### Индексы полей

Индексы для `Cond()` — позиция поля в плоском списке `TableDefinition.Fields` или `Query.FlatFields()`. Порядок соответствует порядку полей в структуре (с учётом распаковки embedded и пропуска omit/rskip).

### Примеры

```go
// Простой фильтр: name = "Alice" AND age > 18
filter := &qqm.Filter{
    Range: qqm.And(
        qqm.Cond(1, qqm.CmdEq, "Alice"),
        qqm.Cond(2, qqm.CmdGt, 18),
    ),
}

// OR с BETWEEN: role = "admin" OR role = "moderator"
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

// Пагинация: OFFSET 10 LIMIT 20
filter := &qqm.Filter{
    Offset: 10,
    Limit:  20,
}
```

## Диалекты

| Диалект | Плейсхолдер | RETURNING | ILIKE |
|---------|-------------|-----------|-------|
| `qqm.SQLiteDialect` | `?` | Да | `LOWER() LIKE LOWER()` |
| `qqm.PostgreSQLDialect` | `$1`, `$2`, … | Да | `ILIKE` |

## Адаптеры БД

Адаптеры в пакете `txproc`:

| Адаптер | Конструктор | Для чего |
|---------|------------|----------|
| `txproc.DBAdapter` | `txproc.NewDBAdapterVal(db)` | `*sql.DB` |
| `txproc.TxAdapter` | `txproc.NewTxAdapterVal(tx)` | `*sql.Tx` |
| `txproc.PGXAdapter` | `txproc.NewPGXAdapterVal(conn)` | `*pgx.Conn` |
| `txproc.PGXTxAdapter` | `txproc.NewPGXTxAdapterVal(tx)` | `pgx.Tx` |

### Транзакции

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

Все CRUD-методы (`Ins`, `Upd`, `One`, `Del`, `Many`) работают с любым `txproc.TxProcessor`.

## Примеры

### Составной ключ

```go
type OrgUser struct {
    OrgID  int64 `tbl:"pk"`
    UserID int64 `tbl:"pk"`
    Role   string
}

func (o *OrgUser) SQLName() string { return "org_users" }

// Использование
tbl := qqm.NewTable[OrgUser](qqm.SQLiteDialect)
row, err := tbl.One(ctx, ex, int64(1), int64(42))
```

### Кастомное имя таблицы

```go
func (u *OrgUser) SQLName() string { return "org_members" }
```

### Embedded структуры с префиксом

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
// Колонки: id, title, audit_created_at, audit_updated_at
```

### Сортировка

```go
type UserWithSort struct {
    ID    int64  `tbl:"pk;auto"`
    Name  string `tbl:"sort=1"`       // ORDER BY name ASC
    Email string `tbl:"sort=2,desc"`  // затем email DESC
    Age   int
}
```

### Auto-поля

```go
type Timestamps struct {
    CreatedAt string `tbl:"col=created_at;auto"`      // не в INSERT
    UpdatedAt string `tbl:"col=updated_at;auto;upd"`  // только в UPDATE
}
```

## Интерфейс TxProcessor

```go
type TxProcessor interface {
    ExecContext(ctx context.Context, query string, args ...any) (Result, error)
    QueryContext(ctx context.Context, query string, args ...any) (Rows, error)
    QueryRowContext(ctx context.Context, query string, args ...any) Row
}
```

- `QueryRowContext` — для запросов, возвращающих одну строку (Ins/Upd с RETURNING, One).
- `QueryContext` — для `Many` (несколько строк).
- `ExecContext` — для `Del` и Ins/Upd без RETURNING.
