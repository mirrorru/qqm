# PROJECT-ARCHITECTURE.md

## Общая архитектура

qqm — ORM-подобная Go-библиотека для типизированной работы с SQL-базами данных.
Архитектура построена на слоях абстракции:

```
Пользовательский код
       ↓
    qqm.go (public API)
       ↓
    Table[ROW] / Query[QROW] (CRUD + JOIN)
       ↓
    TableFields / FieldFlags (метаданные структуры)
    dialect.DialectProvider (диалект БД)
       ↓
    txproc.TxProcessor (абстракция выполнения SQL)
       ↓
    database/sql  или  pgx
```

## Ключевые концепции

### 1. Public API (qqm.go)

Единая точка входа:

- `NewTable[ROW](dialect)` — создание типизированной таблицы
- `NewQuery[QROW](dialect)` — создание multi-table запроса с JOIN
- `SQLiteDialect`, `PostgreSQLDialect` — предопределённые диалекты
- `Cond()`, `And()`, `Or()`, `Not()` — конструкторы фильтров (filter.go)

Все типы и функции — в корневом пакете `qqm`.

### 2. Table[ROW] (table_row.go)

Generic-структура, параметризованная типом строки ROW.

```go
type Table[ROW any] struct {
    tableDef TableDefinition
    sql      sqlTexts
    dialect  dialect.DialectProvider
}
```

**Методы:**
- `Ins(ctx, tx, *ROW) (*ROW, Result, error)` — вставка с RETURNING
- `Upd(ctx, tx, *ROW) (*ROW, Result, error)` — UPDATE по PK с RETURNING
- `One(ctx, tx, keys...) (*ROW, error)` — SELECT по PK
- `Del(ctx, tx, keys...) (Result, error)` — DELETE по PK
- `Many(ctx, tx, filter) ([]*ROW, error)` — SELECT с фильтрацией и сортировкой

**Тип ROW:** структура передаётся по значению (value‑type), не указатель.
`NewTable[ROW]` принимает только ROW без `*`.

**Доступ к SQL:** `tbl.SQLs()` возвращает `sqlTexts` с полями: `InsertCmd`, `UpdateCmd`, `DeleteCmd`, `GetOneCmd`, `ListCmdStart`, `ListSortString`.

**Доступ к метаданным:** `tbl.Defs()` возвращает `TableDefinition` (TableName, Fields, Indexes).

### 3. Query[QROW] (query.go)

Generic-структура для SELECT-запросов по нескольким таблицам с JOIN.

```go
type Query[QROW any] struct {
    dialect    dialect.DialectProvider
    tables     []queryTableEntry
    primaryIdx int
    flatFields TableFields
    idxMapping map[string]int
    sql        sqlTexts
    qrowType   reflect.Type
}
```

**Методы:**
- `One(ctx, tx, keys...) (*QROW, error)` — SELECT с JOIN по PK первичной таблицы
- `Many(ctx, tx, filter) ([]*QROW, error)` — SELECT с JOIN, фильтрацией и сортировкой

**Оптимизация сканирования:**
- `newScanState` создаёт один буфер dest и массив tempDests на строку
- `clearTempDests` обнуляет временные дестинации между строками
- `applyNulls` после Scan проверяет ref-поля на NULL: если все NULL — обнуляет всю структуру таблицы (LEFT JOIN)

**Авто-вывод JOIN:**
- Поля QROW — существующие ROW-структуры (User, Order и т.д.)
- JOIN-условия выводятся из тегов `ref=table:col` на полях ROW-структур
- Тип JOIN задаётся тегом `join=left|right|inner` на поле QROW
- Первичная таблица помечается тегом `from`
- `buildFlatFields` создаёт плоский список полей с квалифицированными именами (`alias.col`) для фильтрации

### 4. Метаданные (field_struct.go)

`CollectTableFields(reflect.Type)` — сбор метаданных через reflect.

**Процесс:**
- Проход по всем публичным полям структуры
- Парсинг тегов `tbl:"..."`
- Для anonymous, embed и полей-структур с `prefix=...` — рекурсивный обход
- Кеширование через `sync.Map` (ленивая инициализация)

**Правила маппинга:**
| Сценарий | Результат |
|----------|-----------|
| Поле без тегов | Колонка = snake_case от имени поля |
| `col=name` | Колонка = name |
| `pk` | Поле — первичный ключ |
| `ro` | Read-only: только SELECT |
| `auto` | Автогенерация: пропускается в INSERT |
| `ins` | Принудительное включение в INSERT |
| `upd` | Принудительное включение в UPDATE |
| `rskip` | Исключается из SELECT |
| `omit` | Полностью исключается из SQL |
| `embed` | Принудительная распаковка |
| `prefix=...` на структуре | Распаковка колонок с префиксом |
| `ref=table:col` | Внешний ключ |
| `sort=<pos>[:dir]` | Позиция и направление в ORDER BY |

**Наследование флагов (Merge):**
- `ro`, `auto`, `ins`, `upd`, `rskip` — наследуются (OR)
- `prefix` — конкатенируется (`parentPrefix + childPrefix`)
- `sort` — наследуется только при `child.SortPos == 0`

### 5. SQL-индексы полей (field_struct.go)

`fieldsIndexes` содержит заранее вычисленные индексы для разных операций:

| Индекс | Назначение |
|--------|-----------|
| `PKCols` | WHERE для One, Upd, Del |
| `SelectCols` | RETURNING, Scan-дестинации |
| `InsertCols` | INSERT VALUES |
| `UpdateCols` | UPDATE SET |
| `SortingCols` | ORDER BY |
| `RefCols` | Внешние ключи для JOIN ON |

### 6. Генерация SQL (table_definition.go)

SQL-запросы генерируются один раз и хранятся в `sqlTexts`:

```go
type sqlTexts struct {
    InsertCmd      string  // INSERT INTO ... VALUES (...) RETURNING ...
    UpdateCmd      string  // UPDATE ... SET ... WHERE pk = ? RETURNING ...
    DeleteCmd      string  // DELETE FROM ... WHERE pk = ?
    GetOneCmd      string  // SELECT ... FROM ... WHERE pk = ?
    ListCmdStart   string  // SELECT ... FROM ... [WHERE ...]
    ListSortString string  // ORDER BY col1 ASC, col2 DESC
}
```

Генерация в `makeSQLs()` при создании `NewTable`. Плейсхолдеры зависят от диалекта: `?` для SQLite, `$N` для PostgreSQL.

ORDER BY формируется из `SortingCols`: поля сортируются по `SortPos`, выводятся через запятую с ASC/DESC.

### 7. Диалекты (dialect/)

```go
type DialectProvider interface {
    Name() string
    QuoteIdent(name string) string
    Placeholder(n int) string
    SupportsReturning() bool
    OffsetAndLimit(offset, limit uint32) string
    ILIKE(col string, placeholder string) string
}
```

Реализации:
| Диалект | Placeholder | Offset/Limit | ILIKE |
|---------|-------------|--------------|-------|
| `SQLiteDialect` | `?` | `LIMIT n OFFSET m` | `LOWER() LIKE LOWER()` |
| `PostgreSQLDialect` | `$N` | `OFFSET m LIMIT n` | `ILIKE` |

### 8. TxProcessor (txproc/)

Абстракция выполнения SQL-запросов в пакете `txproc`.

```go
type TxProcessor interface {
    ExecContext(ctx, query, args...) (Result, error)
    QueryContext(ctx, query, args...) (Rows, error)
    QueryRowContext(ctx, query, args...) Row
}
```

**Адаптеры:**
| Адаптер | Конструктор | Базовый тип |
|---------|------------|-------------|
| `DBAdapter` | `NewDBAdapterVal(db)` | `*sql.DB` |
| `TxAdapter` | `NewTxAdapterVal(tx)` | `*sql.Tx` |
| `PGXAdapter` | `NewPGXAdapterVal(conn)` | `*pgx.Conn` |
| `PGXTxAdapter` | `NewPGXTxAdapterVal(tx)` | `pgx.Tx` |

Адаптеры — value-типы, создаются через конструкторы.

### 9. Фильтрация (filter.go)

Фильтры строятся как дерево узлов `FilterNode`:

```go
type Filter struct {
    Offset uint32
    Limit  uint32
    Range  FilterNode   // корень дерева условий
}

type ConditionNode struct {  // лист: fieldIdx + op + value
    FieldIdx int
    Op       CommandOp
    Value    any
}

type GroupNode struct {      // группа: LogicOp + Children
    Logic    LogicOp         // LogicAnd / LogicOr / LogicNot
    Children []FilterNode
}
```

**Операторы условий (`CommandOp`):** `CmdEq`, `CmdNotEq`, `CmdGt`, `CmdGte`, `CmdLt`, `CmdLte`, `CmdIsNull`, `CmdIsNotNull`, `CmdLike`, `CmdILike`, `CmdIn`

**Логические операторы (`LogicOp`):** `LogicAnd`, `LogicOr`, `LogicNot`

**Хелперы:** `Cond(fieldIdx, op, value)`, `And(children...)`, `Or(children...)`, `Not(child)`

**fieldIdx** — индекс поля в `TableFields`. Для `Table.Many` — в `tableDef.Fields`. Для `Query.Many` — в `FlatFields()`.

**Построение WHERE:** `Filter.BuildWhere(tableFields, dialect)` → собирает SQL и аргументы, обходя дерево `FilterNode.Build()`.

### 10. Теги таблиц в Query (qtable_flags.go)

```go
type TableFlags struct {
    IsFrom    bool              // tbl:"from" — первичная таблица
    JoinMode  JoinMode          // tbl:"join=left|right|inner"
    Alias     string            // tbl:"alias=..."
    RefMap    map[string]string // tbl:"map=k1:v1,k2:v2" — маппинг имён ref-таблиц
    UsePk     bool              // tbl:"pk" — использовать PK в Query.One
    SortOrder int               // tbl:"sort=..." — приоритет сортировки таблицы
}
```

### 11. Тестирование

| Тип | Пакет | Build tag | БД |
|-----|-------|-----------|----|
| Unit | `qqm`, `qqm_test`, `meta_test` | (нет) | — |
| Smoke | `test/smoke` | `smoke` | SQLite :memory: |
| Functional | `test/functional` | `functional` | PostgreSQL |
| Concurrent | `test/concurrent` | `concurrent` | SQLite :memory: (+ race) |

## Поток вызова Ins

```
tbl.Ins(ctx, ex, &User{...})
  → extractArgs(row, InsertCols)       // значения полей для INSERT
  → ex.QueryRowContext(InsertCmd, args) // RETURNING
  → row.Scan(extractRefs(buf, SelectCols))
  → возврат *buf (вставленная строка)
```

## Поток вызова Many (Table)

```
tbl.Many(ctx, ex, filter)
  → filter.BuildWhere(tableDef.Fields, dialect)  // WHERE + args
  → sql = ListCmdStart + where + ListSortString + offset/limit
  → ex.QueryContext(sql, args)
  → для каждой строки:
      extractRefs(buf, SelectCols) → rows.Scan(refs)
      копирование *rowBuf = *buf
  → возврат []*ROW
```

## Поток вызова Many (Query)

```
query.Many(ctx, ex, filter)
  → filter.BuildWhere(flatFields, dialect)  // WHERE + args
  → sql = ListCmdStart + where + ListSortString + offset/limit
  → ex.QueryContext(sql, args)
  → для каждой строки:
      clearTempDests()
      rows.Scan(dest)                     // primary-поля напрямую, остальные в tempDests
      applyNulls(buf, ss)                 // проверка ref-полей на NULL → обнуление структур
      копирование *rowBuf = *buf
  → возврат []*QROW
```
