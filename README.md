# qqm — Quick Query Maker

**qqm** — это ORM-подобная Go-библиотека для типизированной работы с SQL-базами данных через структуры. Она автоматически генерирует SQL-запросы на основе тегов в полях структуры и предоставляет простой CRUD-интерфейс.

## Возможности

- **Типизированные таблицы** — `Table[ROW]` параметризуется вашей структурой.
- **Автогенерация SQL** — INSERT, UPDATE, SELECT, DELETE строятся по метаданным структуры.
- **Поддержка диалектов** — SQLite (`?`) и PostgreSQL (`$1`, `$2`, …).
- **CRUD-интерфейс** — Insert, Update, GetByKey, Delete, List.
- **Гибкая фильтрация** — And/Or-комбинации, операторы Eq, Gt, Lt, Gte, Lte, Between, In.
- **Теги полей** — колонка, первичный ключ, внешний ключ, readonly, auto, omit.
- **Embedded структуры** — поддержка встраивания с префиксом колонок.
- **Именованные поля-структуры** — префикс для неанонимных структур (например, несколько адресов).
- **Составные ключи** —任意ное количество полей в PK.
- **Кеширование SQL** — запросы генерируются один раз при первом обращении.
- **Без рефлексии в рантайме** — метаданные собираются лениво и кешируются.

## Установка

```bash
go get github.com/mirrorru/qqm
```

## Быстрый старт

### Определение модели

```go
type User struct {
    ID    int64  `qqm:"pk"`
    Name  string
    Email string
    Age   int
}
```

Правила именования по умолчанию:
- Имя таблицы — snake_case от имени структуры: `user`.
- Имя колонки — snake_case от имени поля: `name`, `email`, `age`.

### Создание таблицы и SQL

```go
import "github.com/mirrorru/qqm"

userTable := qqm.NewTable[*User](qqm.SQLiteDialect)

fmt.Println(userTable.Internals().InsertSQL())
// INSERT INTO user (id, name, email, age) VALUES (?, ?, ?, ?) RETURNING id, name, email, age

fmt.Println(userTable.Internals().SelectSQL())
// SELECT id, name, email, age FROM user WHERE id = ?
```

### Полный CRUD

```go
import (
    "context"
    "database/sql"
    "github.com/mirrorru/qqm"
    "github.com/mirrorru/qqm/executor"
)

func Example() {
    db, _ := sql.Open("sqlite", ":memory:")
    db.Exec(`CREATE TABLE users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT, email TEXT, age INTEGER
    )`)

    ex := executor.NewDBAdapter(db)
    ctx := context.Background()
    tbl := qqm.NewTable[*User](qqm.SQLiteDialect)

    // Create
    u, _ := tbl.Insert(ctx, ex, &User{Name: "Alice", Email: "alice@test.com", Age: 25})

    // Read
    alice, _ := tbl.GetByKey(ctx, ex, u.ID)

    // Update
    alice.Age = 26
    tbl.Update(ctx, ex, alice)

    // List with filter
    result, _ := tbl.List(ctx, ex, qqm.Field("Age", qqm.And, qqm.Gt(20)))

    // Delete
    tbl.Delete(ctx, ex, u.ID)
}
```

## Настройка колонок через теги

Формат тега: `qqm:"col=name;pk;ref=table.col;readonly;auto;omit;prefix=..."`

| Опция | Описание |
|-------|----------|
| `col=name` | Имя колонки в БД (по умолчанию: snake_case от имени поля) |
| `pk` | Поле является первичным ключом |
| `ref=table.col` | Внешний ключ |
| `prefix=...` | Префикс для колонок из embedded или именованной структуры |
| `readonly` | Не участвует в UPDATE |
| `auto` | Не участвует в INSERT (например, SERIAL) |
| `omit` | Полностью исключается из SQL |

### Префикс для именованных полей-структур

Тег `prefix` работает не только для embedded (анонимных) структур, но и для именованных полей-структур. Это позволяет переиспользовать одну и ту же Go-структуру для разных табличных колонок:

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
// Колонки: id, name, home_city, home_street, home_zip, work_city, work_street, work_zip
```

## Примеры

### Составной ключ

```go
type OrgUser struct {
    OrgID  int64  `qqm:"pk"`
    UserID int64  `qqm:"pk"`
    Role   string
}
```

### Кастомное имя таблицы

Реализуйте интерфейс `table.SQLNamer`:

```go
func (u *OrgUser) SQLName() string { return "org_members" }
```

### Embedded структуры с префиксом

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
// Колонки: id, title, audit_created_at, audit_updated_at
```


```go
// AND-условия
andFilter := qqm.AndFilter(
    qqm.Field("Age", qqm.And, qqm.Gte(18), qqm.Lte(60)),
    qqm.Field("Status", qqm.And, qqm.Eq("active")),
)

// OR-условия
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

## Диалекты

| Диалект | Плейсхолдер | RETURNING |
|---------|-------------|-----------|
| `qqm.SQLiteDialect` | `?` | Да |
| `qqm.PostgreSQLDialect` | `$1`, `$2`, … | Да |

## Адаптеры БД

Для передачи в CRUD-методы используйте адаптеры из `executor`:

| Адаптер | Конструктор | Для чего |
|---------|------------|----------|
| `executor.DBAdapter` | `executor.NewDBAdapter(db)` | `*sql.DB` |
| `executor.TxAdapter` | `executor.NewTxAdapter(tx)` | `*sql.Tx` |
| `executor.PGXAdapter` | `executor.NewPGXAdapter(conn)` | `*pgx.Conn` |
| `executor.PGXTxAdapter` | `executor.NewPGXTxAdapter(tx)` | `pgx.Tx` |

### Транзакции

```go
tx, _ := db.BeginTx(ctx, nil)
ex := executor.NewTxAdapter(tx)

inserted, err := tbl.Insert(ctx, ex, &User{Name: "Alice"})
if err != nil {
    _ = tx.Rollback()
    return err
}
_ = tx.Commit()
```

Все CRUD-методы (`Insert`, `Update`, `GetByKey`, `Delete`, `List`) работают как с `DBAdapter`, так и с `TxAdapter`.

## Интерфейс Executor

Пакет `executor` определяет интерфейс для абстракции SQL-выполнения:

```go
type Executor interface {
    ExecContext(ctx context.Context, query string, args ...any) (Result, error)
    QueryContext(ctx context.Context, query string, args ...any) (Rows, error)
    QueryRowContext(ctx context.Context, query string, args ...any) Row
}
```

- `QueryRowContext` — для запросов, возвращающих одну строку (Insert RETURNING, GetByKey).
