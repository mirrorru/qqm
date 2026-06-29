# PROJECT-ARCHITECTURE.md

## Общая архитектура

qqm — ORM-подобная Go-библиотека для типизированной работы с SQL-базами данных.
Архитектура построена на слоях абстракции:

```
Пользовательский код
       ↓
   qqm.go (public API)
       ↓
   table.Table[ROW] / table.Query[QROW] (CRUD + JOIN)
       ↓
   meta.RowMeta (метаданные структуры)
   dialect.DialectProvider (диалект БД)
       ↓
   executor.Executor (абстракция выполнения SQL)
       ↓
   database/sql  или  pgx
```

## Ключевые концепции

### 1. Public API (qqm.go)

Единая точка входа:

- `qqm.NewTable[ROW](dialect)` — создание типизированной таблицы
- `qqm.NewQuery[QROW](dialect)` — создание multi-table запроса с JOIN
- `qqm.SQLiteDialect`, `qqm.PostgreSQLDialect` — предопределённые диалекты
- `qqm.And`, `qqm.Or` — операторы комбинирования фильтров
- `qqm.Field()`, `qqm.Eq()`, `qqm.Gt()`, ... — конструкторы фильтров
- `qqm.AndFilter()`, `qqm.OrFilter()` — комбинирование фильтров

Все типы и функции — алиасы на реализацию в пакете `table/`.

### 2. Table[ROW] (table/table.go)

Generic-структура, параметризованная типом строки ROW.

```go
type Table[ROW any] struct {
    internal *TableInternals
}
```

**Интерфейс CRUD[ROW]:**
- `Insert(ctx, ex, src *ROW) (*ROW, error)` — вставка с RETURNING
- `GetByKey(ctx, ex, keys...) (*ROW, error)` — SELECT по PK
- `Update(ctx, ex, src *ROW) error` — UPDATE по PK
- `Delete(ctx, ex, keys...) error` — DELETE по PK
- `List(ctx, ex, filter) ([]*ROW, error)` — SELECT с фильтрацией

**Тип ROW:** структура передаётся по значению (value‑type), не указатель.
`NewTable[ROW]` принимает только ROW без `*` — при попытке передать указатель будет panic.

**Внутренняя оптимизация сканирования:**
- `scanDestHelper` — кеширует индексы полей и переиспользует срез `[]any` для дестинаций
- `resetForRow` — обновляет указатели в `dest` для текущей строки за O(1) на поле
- Insert, GetByKey, List используют `scanHelper` вместо `Meta().ScanDest()`

### 3. Query[QROW] (table/multi_query.go)

Generic-структура для SELECT-запросов по нескольким таблицам с JOIN.

```go
type Query[QROW any] struct {
    dialect      dialect.DialectProvider
    qmeta        *queryMeta
    qrowType     reflect.Type
    scanTemplate *scanContext
}
```

**Метод:**
- `List(ctx, ex, filter) ([]*QROW, error)` — SELECT с JOIN и фильтрацией

**Оптимизация сканирования:**
- `scanTemplate` создаётся один раз в `NewQuery` (вместо `newScanContext` на каждую строку)
- `buildScanTemplate` — строит шаблон дестинаций и кеширует индексы полей
- `resetForRow` — обновляет указатели в `dest` для очередной строки
- Для pointer-полей (`*Order`) значения сканируются во временный `[]any`
  Если все PK-колонки NULL → указатель устанавливается в nil
  Иначе — создаётся структура, значения копируются с приведением типов
- Для value-полей (`Order`) адреса полей напрямую помещаются в `dest`

**Результат:** одна аллокация `dest` и одна `apply` на строку вместо полного пересоздания контекста.

**Авто-вывод JOIN:**
- Поля QROW — существующие ROW-структуры (User, Order и т.д.)
- Тип JOIN определяется по типу поля: `*Order` → LEFT, `Order` → INNER
- Условие ON выводится из тега `ref=table.col` на полях ROW-структур
- Явное переопределение через теги поля: `join=LEFT`, `on=...`, `table=...`, `primary`

### 4. Метаданные (meta/)

`meta.BuildRowMeta(type, tableName)` — сбор метаданных через reflect.

**Процесс:**
- Проход по всем публичным полям структуры
- Парсинг тегов `qqm:"..."`
- Для embedded и именованных полей-структур с `prefix=...` — рекурсивный обход
- Валидация дубликатов имён колонок
- Кеширование через `meta.Cache` (sync.Map, ленивая инициализация)

**Правила маппинга:**
| Сценарий | Результат |
|----------|-----------|
| Поле без тегов | Колонка = snake_case от имени поля |
| `col=name` | Колонка = name |
| `pk` | Поле — первичный ключ |
| `auto` | Пропускается в INSERT |
| `readonly` | Пропускается в UPDATE |
| `omit` | Полностью исключается из SQL |
| `prefix=...` на embedded struct | Колонки с префиксом |
| `prefix=...` на именованной struct | Колонки с префиксом (новая возможность) |

### 5. Генерация SQL (table/query.go)

SQL-запросы генерируются один раз и кешируются в `TableInternals`:

- `InsertSQL()` — INSERT INTO ... VALUES (...) RETURNING ...
- `UpdateSQL()` — UPDATE ... SET ... WHERE pk = ?
- `SelectSQL()` — SELECT ... FROM ... WHERE pk = ?
- `DeleteSQL()` — DELETE FROM ... WHERE pk = ?
- `ListSQL(filter)` — SELECT ... FROM ... WHERE conditions

Плейсхолдеры зависят от диалекта: `?` для SQLite, `$N` для PostgreSQL.

### 6. Диалекты (dialect/)

```go
type DialectProvider interface {
    Name() string
    QuoteIdent(name string) string
    Placeholder(n int) string
    SupportsReturning() bool
}
```

Реализации:
- `SQLiteDialect` — `?`, RETURNING поддерживается
- `PostgreSQLDialect` — `$1, $2, ...`, RETURNING поддерживается

### 7. Executor (executor/)

Абстракция выполнения SQL-запросов.

```go
type Executor interface {
    ExecContext(ctx, query, args...) (Result, error)
    QueryContext(ctx, query, args...) (Rows, error)
    QueryRowContext(ctx, query, args...) Row
}
```

**Адаптеры:**
| Адаптер | Конструктор | Базовый тип |
|---------|------------|-------------|
| `DBAdapter` | `NewDBAdapter(db)` | `*sql.DB` |
| `TxAdapter` | `NewTxAdapter(tx)` | `*sql.Tx` |
| `PGXAdapter` | `NewPGXAdapter(conn)` | `*pgx.Conn` |
| `PGXTxAdapter` | `NewPGXTxAdapter(tx)` | `pgx.Tx` |

Адаптеры pgx добавлены для поддержки нативного PostgreSQL-драйвера.
Поля адаптеров сделаны приватными, создание — только через конструкторы.

### 8. Фильтрация (table/filter.go)

Для single-table запросов (`Table.List`) — `whereBuilder` с простыми именами полей.
Для multi-table запросов (`Query.List`) — `multiWhereBuilder` с квалифицированными именами (`"Order.Amount"`).

```go
Filter       = AndFilter/OrFilter из FieldFilter[]
FieldFilter  = имя_поля + FilterOp + []Condition
Condition    = ConditionOp + value
```

**Операторы:** `OpEq`, `OpGt`, `OpLt`, `OpGte`, `OpLte`, `OpBetween`, `OpIn`
**Комбинаторы:** `And`, `Or`

Хелперы: `Eq()`, `Gt()`, `Lt()`, `Gte()`, `Lte()`, `Between()`, `In()`,
`Field()`, `AndFilter()`, `OrFilter()`

### 9. Тестирование

- **Unit-тесты** — в каждом пакете (meta, dialect, table)
- **Smoke** — `test/smoke/` (build tag: smoke), быстрая проверка на SQLite
- **Functional** — `test/functional/` (build tag: functional), полные сценарии на PostgreSQL
- **pgx functional** — `test/functional/pgx_crud_test.go` (build tag: functional), тесты через pgx

Smoke-тесты включают проверку транзакций (commit, rollback, GetByKey внутри транзакции).

## Поток вызова Insert

```
tbl.Insert(ctx, ex, &User{...})
  → Internals().InsertSQL()         // сгенерированный SQL (кеш)
  → meta.InsertValues(row)          // значения полей для INSERT
  → scanHelper.resetForRow(buf)     // дестинации для Scan (кеш)
  → ex.QueryRowContext(sql, args)   // RETURNING
  → row.Scan(dest...)               // заполнение buf
  → копирование *result = *buf      // возврат отдельной строки
```

Аналогично для GetByKey — тоже через `QueryRowContext`.

В List для каждой строки: `resetForRow(buf)` → `Scan(dest)` → копирование `*row = *buf`.

## Изменения по сравнению с предыдущей архитектурой

1. **Value-type ROW (breaking change)** — `NewTable[ROW]` принимает только value-тип (`NewTable[User]`), не указатель. Panic при попытке передать `*User`.
2. **Pointer-результаты** — все CRUD-методы возвращают `*ROW`/`[]*ROW` вместо `ROW`/`[]ROW`; Insert/Update принимают `*ROW`.
3. **scanDestHelper** — новый внутренний кеш дестинаций для Table, `resetForRow` переиспользует `dest`.
4. **scanTemplate в Query** — `buildScanTemplate` создаётся один раз в `NewQuery`, `resetForRow` сбрасывает дестинации на каждую строку. Вместо `newScanContext` на строку.
5. **rowValue** — вспомогательный метод для получения `reflect.Value` из `*ROW`.
6. **Executor.QueryRowContext + Row** — новый метод/интерфейс для single-row запросов
7. **DBAdapter/TxAdapter** — поля приватные, конструкторы `NewDBAdapter`/`NewTxAdapter`
8. **PGXAdapter/PGXTxAdapter** — поддержка pgx (jackc/pgx)
9. **Named struct prefix** — префикс работает на именованных (неанонимных) полях-структурах
10. **Public API (qqm.go)** — единая точка входа с реэкспортами фильтров
11. **errNoRowsReturned удалён** — ошибки обрабатываются через `QueryRowContext`
12. **Multi-table запросы (Query[QROW])** — SELECT с JOIN, авто-вывод ON из ref=,
    квалифицированные имена в фильтрах, NULL-безопасное сканирование для LEFT JOIN
