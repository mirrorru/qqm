package bench

import (
	"context"
	"database/sql"
	"testing"

	"github.com/mirrorru/qqm"
	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/test/fixtures"
	"github.com/mirrorru/qqm/txproc"

	_ "modernc.org/sqlite"
)

func setupTableDB(b *testing.B) (*sql.DB, qqm.TxProcessor) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		b.Fatal(err)
	}

	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			email TEXT NOT NULL,
			age INTEGER NOT NULL DEFAULT 0
		)
	`)
	if err != nil {
		b.Fatal(err)
	}

	ex := txproc.NewDBAdapterVal(db)
	return db, ex
}

func BenchmarkTable_New(b *testing.B) {
	d := dialect.SQLiteDialect{}
	for i := 0; i < b.N; i++ {
		_ = qqm.NewTable[fixtures.User](d)
	}
}

func BenchmarkTable_New_Nested(b *testing.B) {
	d := dialect.SQLiteDialect{}
	for i := 0; i < b.N; i++ {
		_ = qqm.NewTable[fixtures.PersonWithAddress](d)
	}
}

func BenchmarkTable_Ins(b *testing.B) {
	db, ex := setupTableDB(b)
	defer db.Close()

	tbl := qqm.NewTable[fixtures.User](dialect.SQLiteDialect{})
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		user := &fixtures.User{
			Name:  "Test",
			Email: "test@example.com",
		}
		_, _, _ = tbl.Ins(ctx, ex, user)
	}
}

func BenchmarkTable_One(b *testing.B) {
	db, ex := setupTableDB(b)
	defer db.Close()

	tbl := qqm.NewTable[fixtures.User](dialect.SQLiteDialect{})
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		_, _, _ = tbl.Ins(ctx, ex, &fixtures.User{Name: "Test", Email: "test@example.com"})
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = tbl.One(ctx, ex, int64(1))
	}
}

func BenchmarkTable_Many_Small(b *testing.B) {
	db, ex := setupTableDB(b)
	defer db.Close()

	tbl := qqm.NewTable[fixtures.User](dialect.SQLiteDialect{})
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		_, _, _ = tbl.Ins(ctx, ex, &fixtures.User{Name: "Test", Email: "test@example.com"})
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = tbl.Many(ctx, ex, nil)
	}
}

func BenchmarkTable_Many_Large(b *testing.B) {
	db, ex := setupTableDB(b)
	defer db.Close()

	tbl := qqm.NewTable[fixtures.User](dialect.SQLiteDialect{})
	ctx := context.Background()

	for i := 0; i < 1000; i++ {
		_, _, _ = tbl.Ins(ctx, ex, &fixtures.User{Name: "Test", Email: "test@example.com"})
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = tbl.Many(ctx, ex, nil)
	}
}

func BenchmarkTable_Many_WithFilter(b *testing.B) {
	db, ex := setupTableDB(b)
	defer db.Close()

	tbl := qqm.NewTable[fixtures.User](dialect.SQLiteDialect{})
	ctx := context.Background()

	for i := 0; i < 100; i++ {
		_, _, _ = tbl.Ins(ctx, ex, &fixtures.User{Name: "Test", Email: "test@example.com"})
	}

	filter := &qqm.Filter{
		Range: qqm.And(qqm.Cond(1, qqm.CmdEq, "Test")),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = tbl.Many(ctx, ex, filter)
	}
}

func BenchmarkTable_Upd(b *testing.B) {
	db, ex := setupTableDB(b)
	defer db.Close()

	tbl := qqm.NewTable[fixtures.User](dialect.SQLiteDialect{})
	ctx := context.Background()

	_, _, err := tbl.Ins(ctx, ex, &fixtures.User{Name: "Test", Email: "test@example.com"})
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		user := &fixtures.User{ID: 1, Name: "Updated", Email: "updated@example.com"}
		_, _, _ = tbl.Upd(ctx, ex, user)
	}
}
