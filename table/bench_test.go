// Created at 2026-06-29
// Benchmark-тесты для table-пакета

//go:build bench

package table

import (
	"context"
	"reflect"
	"testing"

	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/executor"
	"github.com/mirrorru/qqm/meta"
	"github.com/mirrorru/qqm/test/fixtures"
)

var (
	benchRowsSink []*fixtures.UserWithAge
	benchStrSink  string
	benchArgsSink []any
)

type benchRows struct {
	count int
	pos   int
}

func (r *benchRows) Next() bool {
	if r.pos < r.count {
		r.pos++
		return true
	}
	return false
}

func (r *benchRows) Scan(dest ...any) error {
	for _, d := range dest {
		if v := reflect.ValueOf(d); v.Kind() == reflect.Pointer && !v.IsNil() {
			elem := v.Elem()
			switch elem.Kind() {
			case reflect.Int, reflect.Int64:
				elem.SetInt(1)
			case reflect.String:
				elem.SetString("test")
			case reflect.Float64:
				elem.SetFloat(1.0)
			}
		}
	}
	return nil
}

func (r *benchRows) Close() error { return nil }

type benchExecutor struct {
	sql string
}

func (e *benchExecutor) ExecContext(_ context.Context, _ string, _ ...any) (executor.Result, error) {
	return mockResult{}, nil
}

func (e *benchExecutor) QueryContext(_ context.Context, query string, _ ...any) (executor.Rows, error) {
	e.sql = query
	return &benchRows{count: 10}, nil
}

func (e *benchExecutor) QueryRowContext(_ context.Context, _ string, _ ...any) executor.Row {
	return &benchRows{count: 1}
}

func BenchmarkNewTable(b *testing.B) {
	b.Run("SQLite", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = NewTable[fixtures.UserWithAge](dialect.SQLiteDialect{})
		}
	})

	b.Run("PostgreSQL", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = NewTable[fixtures.UserWithAge](dialect.PostgreSQLDialect{})
		}
	})
}

func BenchmarkInsertSQL(b *testing.B) {
	tbl := NewTable[fixtures.UserWithAge](dialect.SQLiteDialect{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchStrSink = tbl.Internals().InsertSQL()
	}
}

func BenchmarkUpdateSQL(b *testing.B) {
	tbl := NewTable[fixtures.UserWithAge](dialect.SQLiteDialect{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchStrSink = tbl.Internals().UpdateSQL()
	}
}

func BenchmarkSelectSQL(b *testing.B) {
	tbl := NewTable[fixtures.UserWithAge](dialect.SQLiteDialect{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchStrSink = tbl.Internals().SelectSQL()
	}
}

func BenchmarkBuildFilterWhereClause(b *testing.B) {
	tbl := NewTable[fixtures.UserWithAge](dialect.SQLiteDialect{})

	b.Run("single condition", func(b *testing.B) {
		filters := []Filter{
			AndFilter(Field("Age", And, Gt(18))),
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			benchStrSink, benchArgsSink, _ = tbl.buildFilterWhereClause(filters)
		}
	})

	b.Run("multiple conditions", func(b *testing.B) {
		filters := []Filter{
			AndFilter(
				Field("Age", And, Gt(18), Lt(65)),
				Field("Name", And, Eq("Alice")),
			),
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			benchStrSink, benchArgsSink, _ = tbl.buildFilterWhereClause(filters)
		}
	})

	b.Run("OR filter", func(b *testing.B) {
		filters := []Filter{
			OrFilter(
				Field("Name", And, Eq("Alice")),
				Field("Name", And, Eq("Bob")),
			),
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			benchStrSink, benchArgsSink, _ = tbl.buildFilterWhereClause(filters)
		}
	})
}

func BenchmarkList_NoFilter(b *testing.B) {
	tbl := NewTable[fixtures.UserWithAge](dialect.SQLiteDialect{})
	ctx := context.Background()
	exe := &benchExecutor{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchRowsSink, _ = tbl.List(ctx, exe)
	}
}

func BenchmarkList_WithFilter(b *testing.B) {
	tbl := NewTable[fixtures.UserWithAge](dialect.SQLiteDialect{})
	ctx := context.Background()
	exe := &benchExecutor{}
	filter := AndFilter(Field("Age", And, Gt(18)))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchRowsSink, _ = tbl.List(ctx, exe, filter)
	}
}

func BenchmarkScanDest(b *testing.B) {
	tbl := NewTable[fixtures.UserWithAge](dialect.SQLiteDialect{})
	row := &fixtures.UserWithAge{ID: 1, Name: "test", Email: "test@test.com", Age: 25}

	b.Run("ScanDest (public API)", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			benchSink = tbl.Internals().Meta().ScanDest(row)
		}
	})

	b.Run("scanHelper.resetForRow", func(b *testing.B) {
		helper := tbl.internal.scanHelper
		rv := tbl.rowValue(row)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			benchSink = helper.resetForRow(rv)
		}
	})
}

var benchSink any

func BenchmarkNewScanContext(b *testing.B) {
	meta.ClearCache()

	q, err := NewQuery[fixtures.UserWithOrder](dialect.SQLiteDialect{})
	if err != nil {
		b.Fatal(err)
	}

	b.Run("buildTemplate", func(b *testing.B) {
		qm := q.qmeta
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = buildScanTemplate(qm)
		}
	})

	b.Run("resetForRow", func(b *testing.B) {
		qrow := reflect.New(reflect.TypeOf(fixtures.UserWithOrder{})).Elem()
		tmpl := q.scanTemplate
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			tmpl.resetForRow(qrow)
		}
	})
}
