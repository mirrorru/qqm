# PROJECT-STRUCTURE.md

```
qqm/
├── qqm.go                 # Public API: NewTable, Dialects, filter-хелперы
├── table.go               # Table[ROW] — CRUD-методы + внутренности
├── filter.go              # Фильтры: Filter, FieldFilter, Condition, хелперы
├── filter_test.go
├── query.go               # queryBuilder — генерация SQL для одной таблицы
├── query_meta.go          # queryMeta — метаданные multi-table запроса
├── multi_query.go         # Query[QROW] — SELECT с JOIN
├── multi_query_test.go
├── sql_gen_test.go
├── example_simple_key_test.go
├── example_composite_key_test.go
├── bench_test.go
├── bench_list_vs_raw_test.go
├── executor.go               #   Executor, Result, Rows, Row — интерфейсы
├── sql_adapter.go            #   DBAdapter, TxAdapter — для database/sql
├── pgx_adapter.go            #   PGXAdapter, PGXTxAdapter — для pgx
│
├── dialect/               # Диалекты БД
│   ├── dialect.go         #   DialectProvider — интерфейс диалекта
│   ├── sqlite.go          #   SQLiteDialect
│   └── postgres.go        #   PostgreSQLDialect
│
├── meta/                  # Метаданные структур
│   ├── cache.go           #   Кеш метаданных (sync.Map)
│   ├── row_meta.go        #   RowMeta — сбор метаданных по reflect
│   ├── field_meta.go      #   FieldMeta — описание одного поля
│   ├── tag.go             #   Парсинг тегов qqm:"..."
│   ├── tag_test.go
│   ├── row_meta_test.go
│   ├── cache_test.go
│   └── casing.go          #   ToSnakeCase для имён колонок
│
├── test/                  # Интеграционные тесты (build tags)
│   ├── fixtures/          #   Фикстуры: структуры для тестов
│   │   └── fixtures.go
│   ├── smoke/             #   Smoke-тесты (build tag: smoke)
│   │   ├── crud_test.go
│   │   ├── multi_query_test.go
│   │   ├── race_test.go
│   │   └── sort_test.go
│   └── functional/        #   Functional-тесты (build tag: functional)
│       ├── crud_test.go
│       ├── anonymous_struct_test.go
│       ├── some_table_test.go
│       ├── multi_query_test.go
│       ├── sort_test.go
│       └── pgx_crud_test.go
│
├── README.md              # Документация
├── PROJECT-STRUCTURE.md   # Структура проекта (этот файл)
├── PROJECT-ARCHITECTURE.md# Архитектура проекта
├── go.mod
├── go.sum
├── Taskfile.yaml
├── Taskfile-test.yaml
└── .golangci.pipeline.yaml
```
