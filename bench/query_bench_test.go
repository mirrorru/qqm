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

func setupQueryDB(b *testing.B) (*sql.DB, qqm.TxProcessor) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		b.Fatal(err)
	}

	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT NOT NULL
		);
		CREATE TABLE orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL REFERENCES users(id),
			amount REAL NOT NULL
		)
	`)
	if err != nil {
		b.Fatal(err)
	}

	ex := txproc.NewDBAdapterVal(db)
	return db, ex
}

func insertQueryTestData(b *testing.B, db *sql.DB, ex qqm.TxProcessor) {
	ctx := context.Background()
	userTbl := qqm.NewTable[fixtures.User](dialect.SQLiteDialect{})
	orderTbl := qqm.NewTable[fixtures.Order](dialect.SQLiteDialect{})

	for i := 1; i <= 10; i++ {
		_, _, _ = userTbl.Ins(ctx, ex, &fixtures.User{ID: int64(i), Name: "User", Email: "user@test.com"})
	}

	for i := 1; i <= 10; i++ {
		for j := 1; j <= 5; j++ {
			_, _, _ = orderTbl.Ins(ctx, ex, &fixtures.Order{UserID: int64(i), Amount: float64(j * 10)})
		}
	}
}

func BenchmarkQuery_New(b *testing.B) {
	d := dialect.SQLiteDialect{}
	for i := 0; i < b.N; i++ {
		_ = qqm.NewQuery[fixtures.UserWithOrder](d)
	}
}

func BenchmarkQuery_New_ThreeTables(b *testing.B) {
	d := dialect.SQLiteDialect{}
	for i := 0; i < b.N; i++ {
		_ = qqm.NewQuery[fixtures.UserWithSortAndOrder](d)
	}
}

func BenchmarkQuery_One(b *testing.B) {
	db, ex := setupQueryDB(b)
	defer db.Close()
	insertQueryTestData(b, db, ex)

	query := qqm.NewQuery[fixtures.UserWithOrder](dialect.SQLiteDialect{})
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = query.One(ctx, ex, int64(1))
	}
}

func BenchmarkQuery_Many_Small(b *testing.B) {
	db, ex := setupQueryDB(b)
	defer db.Close()
	insertQueryTestData(b, db, ex)

	query := qqm.NewQuery[fixtures.UserWithOrder](dialect.SQLiteDialect{})
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = query.Many(ctx, ex, nil)
	}
}

func BenchmarkQuery_Many_Large(b *testing.B) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT NOT NULL
		);
		CREATE TABLE orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL REFERENCES users(id),
			amount REAL NOT NULL
		)
	`)
	if err != nil {
		b.Fatal(err)
	}

	ex := txproc.NewDBAdapterVal(db)
	ctx := context.Background()
	userTbl := qqm.NewTable[fixtures.User](dialect.SQLiteDialect{})
	orderTbl := qqm.NewTable[fixtures.Order](dialect.SQLiteDialect{})

	for i := 1; i <= 100; i++ {
		_, _, _ = userTbl.Ins(ctx, ex, &fixtures.User{ID: int64(i), Name: "User", Email: "user@test.com"})
	}

	for i := 1; i <= 100; i++ {
		for j := 1; j <= 10; j++ {
			_, _, _ = orderTbl.Ins(ctx, ex, &fixtures.Order{UserID: int64(i), Amount: float64(j * 10)})
		}
	}

	query := qqm.NewQuery[fixtures.UserWithOrder](dialect.SQLiteDialect{})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = query.Many(ctx, ex, nil)
	}
}

func BenchmarkQuery_Many_WithFilter(b *testing.B) {
	db, ex := setupQueryDB(b)
	defer db.Close()
	insertQueryTestData(b, db, ex)

	query := qqm.NewQuery[fixtures.UserWithOrder](dialect.SQLiteDialect{})
	ctx := context.Background()

	filter := &qqm.Filter{
		Range: qqm.And(qqm.Cond(1, qqm.CmdEq, "User")),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = query.Many(ctx, ex, filter)
	}
}

func BenchmarkQuery_Many_LEFT_JOIN(b *testing.B) {
	db, ex := setupQueryDB(b)
	defer db.Close()

	exLocal := txproc.NewDBAdapterVal(db)
	ctx := context.Background()
	userTbl := qqm.NewTable[fixtures.User](dialect.SQLiteDialect{})

	for i := 1; i <= 10; i++ {
		_, _, _ = userTbl.Ins(ctx, exLocal, &fixtures.User{ID: int64(i), Name: "User", Email: "user@test.com"})
	}

	query := qqm.NewQuery[fixtures.UserWithOrderLeft](dialect.SQLiteDialect{})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = query.Many(ctx, ex, nil)
	}
}
