# PROJECT-ARCHITECTURE.md

## Общая архитектура

qqm — ORM-подобная Go-библиотека для типизированной работы с SQL-базами данных.
Архитектура построена на слоях абстракции:

```
Пользовательский код
       ↓
   qqm.go (public API)
       ↓
   table.Table[ROW] (CRUD + SQL-генерация)
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
- `Insert(ctx, ex, src) (ROW, error)` — вставка с RETURNING
- `GetByKey(ctx, ex, keys...) (ROW, error)` — SELECT по PK
- `Update(ctx, ex, src) error` — UPDATE по PK
- `Delete(ctx, ex, keys...) error` — DELETE по PK
- `List(ctx, ex, filter) ([]ROW, error)` — SELECT с фильтрацией

Внутри Insert и GetByKey используют `QueryRowContext` (одна строка),
что упрощает код и убирает ручную итерацию по rows.

### 3. Метаданные (meta/)

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

### 4. Генерация SQL (table/query.go)

SQL-запросы генерируются один раз и кешируются в `TableInternals`:

- `InsertSQL()` — INSERT INTO ... VALUES (...) RETURNING ...
- `UpdateSQL()` — UPDATE ... SET ... WHERE pk = ?
- `SelectSQL()` — SELECT ... FROM ... WHERE pk = ?
- `DeleteSQL()` — DELETE FROM ... WHERE pk = ?
- `ListSQL(filter)` — SELECT ... FROM ... WHERE conditions

Плейсхолдеры зависят от диалекта: `?` для SQLite, `$N` для PostgreSQL.

### 5. Диалекты (dialect/)

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

### 6. Executor (executor/)

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

### 7. Фильтрация (table/filter.go)

```go
Filter       = AndFilter/OrFilter из FieldFilter[]
FieldFilter  = имя_поля + FilterOp + []Condition
Condition    = ConditionOp + value
```

**Операторы:** `OpEq`, `OpGt`, `OpLt`, `OpGte`, `OpLte`, `OpBetween`, `OpIn`
**Комбинаторы:** `And`, `Or`

Хелперы: `Eq()`, `Gt()`, `Lt()`, `Gte()`, `Lte()`, `Between()`, `In()`,
`Field()`, `AndFilter()`, `OrFilter()`

### 8. Тестирование

- **Unit-тесты** — в каждом пакете (meta, dialect, table)
- **Smoke** — `test/smoke/` (build tag: smoke), быстрая проверка на SQLite
- **Functional** — `test/functional/` (build tag: functional), полные сценарии на PostgreSQL
- **pgx functional** — `test/functional/pgx_crud_test.go` (build tag: functional), тесты через pgx

Smoke-тесты включают проверку транзакций (commit, rollback, GetByKey внутри транзакции).

## Поток вызова Insert

```
tbl.Insert(ctx, ex, &User{...})
  → Internals().InsertSQL()     // сгенерированный SQL (кеш)
  → meta.InsertValues(row)       // значения полей для INSERT
  → ex.QueryRowContext(sql, args) // RETURNING
  → meta.ScanDest(result)        // дестинации для Scan
  → row.Scan(dest...)            // заполнение структуры
  → возврат готовой ROW
```

Аналогично для GetByKey — тоже через `QueryRowContext`.

## Изменения по сравнению с предыдущей архитектурой

1. **Executor.QueryRowContext + Row** — новый метод/интерфейс для single-row запросов
2. **DBAdapter/TxAdapter** — поля приватные, конструкторы `NewDBAdapter`/`NewTxAdapter`
3. **PGXAdapter/PGXTxAdapter** — поддержка pgx (jackc/pgx)
4. **Named struct prefix** — префикс работает на именованных (неанонимных) полях-структурах
5. **Public API (qqm.go)** — единая точка входа с реэкспортами фильтров
6. **errNoRowsReturned удалён** — ошибки обрабатываются через `QueryRowContext`
