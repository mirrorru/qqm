# PROJECT-STRUCTURE.md

```
qqm/
├── qqm.go                    # Реэкспорт диалектов: SQLiteDialect, PostgreSQLDialect
├── table_row.go              # Table[ROW] — CRUD: Ins, Upd, One, Del, Many
├── table_definition.go       # TableDefinition, поля, генерация SQL (INSERT/UPDATE/DELETE/SELECT/LIST)
├── table_info.go             # sqlTexts, SQLNamer, getTableName, isKey
├── table_test.go             # Unit-тесты Table, CollectTableFields
├── field_struct.go           # CollectTableFields, FieldFlags, TableField, fieldsIndexes, parseFieldTag
├── field_struct_test.go      # Unit-тесты field_struct и парсинга тегов tbl
├── qtable_flags.go           # TableFlags, JoinMode, parseTableTag — теги для Query-полей
├── query.go                  # Query[QROW] — multi-table SELECT с JOIN: One, Many
├── filter.go                 # Filter, FilterNode, ConditionNode, GroupNode, хелперы Cond/And/Or/Not
├── filter_test.go            # Unit-тесты фильтров
├── executor.go               # TxProcessor, Row, Result, Rows — интерфейсы
│
├── dialect/                  # Диалекты БД
│   ├── dialect.go            #   DialectProvider — интерфейс (Name, QuoteIdent, Placeholder, SupportsReturning, OffsetAndLimit, ILIKE)
│   ├── sqlite.go             #   SQLiteDialect (?, LIMIT/OFFSET, LOWER LIKE)
│   └── postgres.go           #   PostgreSQLDialect ($N, OFFSET/LIMIT, ILIKE)
│
├── defs/                     # SQL-константы
│   └── defs.go               #   INSERT INTO, VALUES, SELECT, FROM, WHERE, JOIN, AND, OR, etc.
│
├── meta/                     # Утилиты
│   ├── casing.go             #   ToSnakeCase (CamelCase → snake_case)
│   └── casing_test.go        #   Unit-тесты casing
│
├── txproc/                   # Абстракция выполнения SQL
│   ├── sql_adapter.go        #   DBAdapter, TxAdapter — адаптеры database/sql
│   └── pgx_adapter.go        #   PGXAdapter, PGXTxAdapter — адаптеры pgx
│
├── test/                     # Интеграционные тесты (build tags)
│   ├── fixtures/             #   Фикстуры: тестовые структуры
│   │   ├── fixtures.go       #     User, OrgUser, Order, OrderItem, Query-структуры
│   │   └── real_case.go      #     DictSubj — реальный кейс
│   ├── smoke/                #   Smoke-тесты (build tag: smoke, SQLite :memory:)
│   │   ├── v2_crud_test.go   #     CRUD Table[ROW] + Query[QROW]
│   │   ├── race_v2_test.go   #     Тесты на гонки данных
│   │   └── dict_subj_with_person_and_legal_test.go  #     Реальный кейс DictSubj
│   ├── functional/           #   Functional-тесты (build tag: functional, PostgreSQL)
│   │   ├── main_test.go      #     TestMain: создание/удаление таблиц
│   │   ├── v2_crud_test.go   #     CRUD Table[ROW] + Query[QROW]
│   │   └── pg_helper_test.go #     Хелперы для PG (beginTxPG)
│   └── concurrent/           #   Concurrent-тесты (build tag: concurrent, race detector)
│       ├── table_concurrent_test.go
│       ├── query_concurrent_test.go
│       └── filter_concurrent_test.go
│
├── README.md                 # Документация (русский)
├── README.en.md              # Документация (English)
├── AGENTS.md                 # Гайд для AI-агентов
├── PROJECT-STRUCTURE.md      # Структура проекта (этот файл)
├── PROJECT-ARCHITECTURE.md   # Архитектура проекта
├── go.mod
├── go.sum
├── Taskfile.yaml             # Основной Taskfile (lint, gen, pg:up/down)
├── Taskfile-test.yaml        # Тестовый Taskfile (unit, smoke, functional, concurrent, coverage)
└── .golangci.pipeline.yaml   # Конфигурация golangci-lint
```
