// Created at 2026-06-29
// Benchmark-тесты для meta-пакета

//go:build bench

package meta

import (
	"reflect"
	"testing"
)

var (
	benchSink     interface{}
	benchStrSink  string
	benchColsSink []string
)

func BenchmarkToSnakeCase(b *testing.B) {
	inputs := []string{
		"UserID",
		"UserName",
		"HTTPS",
		"ID",
		"SimpleUser",
		"UserWithLongNameAndWider",
		"",
		"X",
		"ABC",
		"Simple",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, s := range inputs {
			benchStrSink = ToSnakeCase(s)
		}
	}
}

func BenchmarkSplitCamelCase(b *testing.B) {
	inputs := []string{
		"UserID",
		"UserName",
		"HTTPS",
		"ID",
		"SimpleUser",
		"UserWithLongNameAndWider",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, s := range inputs {
			benchStrSink = ToSnakeCase(s)
		}
	}
}

func BenchmarkParseTag(b *testing.B) {
	tags := []string{
		"col=user_name;pk",
		"col=name;pk;ref=users.id;prefix=audit_;readonly;auto;omit",
		"col=name",
		"pk",
		"ref=users.id",
		"prefix=audit_",
		"readonly;auto;omit",
		"join=LEFT;primary;on=users.id=orders.user_id",
		"table=app_users",
		"",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tag := range tags {
			benchSink = ParseTag(tag)
		}
	}
}

type benchUser struct {
	ID        int64  `qqm:"col=user_id;pk"`
	FirstName string `qqm:"col=first_name"`
	LastName  string `qqm:"col=last_name"`
	Email     string
	Age       int
	Bio       string `qqm:"readonly"`
	Secret    string `qqm:"omit"`
	CreatedAt string `qqm:"auto"`
}

func (*benchUser) SQLName() string { return "bench_users" }

type benchAddress struct {
	City   string
	Street string
	Zip    string
}

type benchPerson struct {
	ID          int64 `qqm:"pk"`
	Name        string
	HomeAddress benchAddress `qqm:"prefix=home_"`
	WorkAddress benchAddress `qqm:"prefix=work_"`
	Email       string
	Phone       string
	CreatedAt   string `qqm:"auto"`
	UpdatedAt   string `qqm:"auto"`
}

func BenchmarkBuildRowMeta(b *testing.B) {
	b.Run("simple struct", func(b *testing.B) {
		t := reflect.TypeOf(benchUser{})
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			benchSink = BuildRowMeta(t, "bench_users")
		}
	})

	b.Run("struct with embedded prefix", func(b *testing.B) {
		t := reflect.TypeOf(benchPerson{})
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			benchSink = BuildRowMeta(t, "bench_persons")
		}
	})
}

func BenchmarkGetOrBuildRowMeta(b *testing.B) {
	ClearCache()
	t := reflect.TypeOf(benchUser{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchSink = GetOrBuildRowMeta(t, "bench_users")
	}
}

func BenchmarkScanDest(b *testing.B) {
	rm := BuildRowMeta(reflect.TypeOf(benchUser{}), "bench_users")
	row := &benchUser{
		ID:        1,
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john@example.com",
		Age:       30,
		Bio:       "Developer",
		Secret:    "hidden",
		CreatedAt: "2024-01-01",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchSink = rm.ScanDest(row)
	}
}

func BenchmarkInsertValues(b *testing.B) {
	rm := BuildRowMeta(reflect.TypeOf(benchUser{}), "bench_users")
	row := &benchUser{
		ID:        1,
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john@example.com",
		Age:       30,
		Bio:       "Developer",
		Secret:    "hidden",
		CreatedAt: "2024-01-01",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchSink = rm.InsertValues(row)
	}
}

func BenchmarkInsertColumns(b *testing.B) {
	rm := BuildRowMeta(reflect.TypeOf(benchUser{}), "bench_users")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchColsSink = rm.InsertColumns()
	}
}

func BenchmarkUpdateColumns(b *testing.B) {
	rm := BuildRowMeta(reflect.TypeOf(benchUser{}), "bench_users")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchColsSink = rm.UpdateColumns()
	}
}
