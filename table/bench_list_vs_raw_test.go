//go:build bench

package table

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/executor"
	"github.com/mirrorru/qqm/test/fixtures"
	_ "modernc.org/sqlite"
)

type user5 struct {
	fixtures.UserWithAge
	Height float64
}

type user25 struct {
	fixtures.UserWithAge
	F1 float64
	F2 float64
	F3 float64
	F4 float64
	F5 float64
	F6 float64
	F7 float64
	I1 int64
	I2 int64
	I3 int64
	I4 int64
	I5 int64
	I6 int64
	I7 int64
	S1 string
	S2 string
	S3 string
	S4 string
	S5 string
	S6 string
	S7 string
}

// sink for raw_sql benchmark
var benchRawListSink5 []*user5
var benchRawListSink25 []*user25

func BenchmarkListVsRawSQL_5_Fld(b *testing.B) {
	rowCounts := []int{10, 100, 1000}

	for _, n := range rowCounts {
		b.Run(fmt.Sprintf("rows=%d", n), func(b *testing.B) {
			db, err := sql.Open("sqlite", ":memory:")
			if err != nil {
				b.Fatal(err)
			}
			defer db.Close()

			_, err = db.Exec(`
				CREATE TABLE user5 (
					id INTEGER PRIMARY KEY,
					name TEXT NOT NULL,
					email TEXT NOT NULL,
					age INTEGER NOT NULL,
				    height FLOAT NOT NULL
				)
			`)
			if err != nil {
				b.Fatal(err)
			}

			for i := 0; i < n; i++ {
				_, err := db.Exec(
					"INSERT INTO user5 (id, name, email, age, height) VALUES (?, ?, ?, ?, ?)",
					i+1, fmt.Sprintf("User%d", i+1), fmt.Sprintf("user%d@test.com", i+1), 20+(i%50), 5.0*i)
				if err != nil {
					b.Fatal(err)
				}
			}

			ex := executor.NewDBAdapter(db)
			ctx := context.Background()
			tbl := NewTable[user5](dialect.SQLiteDialect{})

			b.Run("raw_sql", func(b *testing.B) {
				b.ReportAllocs()
				var sink []*user5
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					rows, err := db.QueryContext(ctx, "SELECT id, name, email, age, height FROM user5")
					if err != nil {
						b.Fatal(err)
					}

					result := make([]*user5, 0, n)
					for rows.Next() {
						var u user5
						if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Age, &u.Height); err != nil {
							b.Fatal(err)
						}
						result = append(result, &u)
					}
					_ = rows.Close()
					sink = result
				}
				benchRawListSink5 = sink
			})

			b.Run("table_list", func(b *testing.B) {
				b.ReportAllocs()
				var sink []*user5
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					result, err := tbl.List(ctx, ex)
					if err != nil {
						b.Fatal(err)
					}
					sink = result
				}
				benchRawListSink5 = sink
			})
		})
	}
}

func BenchmarkListVsRawSQL_25_Fld(b *testing.B) {
	rowCounts := []int{10, 100, 1000}

	for _, n := range rowCounts {
		b.Run(fmt.Sprintf("rows=%d", n), func(b *testing.B) {
			db, err := sql.Open("sqlite", ":memory:")
			if err != nil {
				b.Fatal(err)
			}
			defer db.Close()

			_, err = db.Exec(`
				CREATE TABLE user25 (
					id INTEGER PRIMARY KEY,
					name TEXT NOT NULL,
					email TEXT NOT NULL,
					age INTEGER NOT NULL,
				    F1, F2, F3, F4, F5, F6, F7 FLOAT NOT NULL DEFAULT 10.0,
				    I1, I2, I3, I4, I5, I6, I7 INTEGER NOT NULL DEFAULT 10,  
				    S1, S2, S3, S4, S5, S6, S7 TEXT NOT NULL DEFAULT '-'            
				)
			`)
			if err != nil {
				b.Fatal(err)
			}

			for i := 0; i < n; i++ {
				_, err := db.Exec(
					`INSERT INTO user25 
                           (id, name, email, age,
                            F1, F2, F3, F4, F5, F6, F7,
                            I1, I2, I3, I4, I5, I6, I7,
                            S1, S2, S3, S4, S5, S6, S7)
                           VALUES (?, ?, ?, ?
                                   ,? ,? ,? ,? ,? ,? ,? 
                                   ,? ,? ,? ,? ,? ,? ,? 
                                   ,? ,? ,? ,? ,? ,? ,? 
                                   )`,
					i+1, fmt.Sprintf("User%d", i+1),
					fmt.Sprintf("user%d@test.com", i+1), 20+(i%50), 1.0,
					10.0, 11.0, 12.0, 13.0, 14.0, 15.0, 16.0,
					20, 21, 22, 23, 24, 25, 26,
					"30", "31", "32", "33", "34", "35", "36",
				)
				if err != nil {
					b.Fatal(err)
				}
			}

			ex := executor.NewDBAdapter(db)
			ctx := context.Background()
			tbl := NewTable[user25](dialect.SQLiteDialect{})

			b.Run("raw_sql", func(b *testing.B) {
				b.ReportAllocs()
				var sink []*user25
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					rows, err := db.QueryContext(ctx,
						`SELECT id, name, email, age, 
				    F1, F2, F3, F4, F5, F6, F7, 
				    I1, I2, I3, I4, I5, I6, I7,  
				    S1, S2, S3, S4, S5, S6, S7   
				    FROM user25`)
					if err != nil {
						b.Fatal(err)
					}

					result := make([]*user25, 0, n)
					for rows.Next() {
						var u user25
						if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Age,
							&u.F1, &u.F2, &u.F3, &u.F4, &u.F5, &u.F6, &u.F7,
							&u.I1, &u.I2, &u.I3, &u.I4, &u.I5, &u.I6, &u.I7,
							&u.S1, &u.S2, &u.S3, &u.S4, &u.S5, &u.S6, &u.S7,
						); err != nil {
							b.Fatal(err)
						}
						result = append(result, &u)
					}
					_ = rows.Close()
					sink = result
				}
				benchRawListSink25 = sink
			})

			b.Run("table_list", func(b *testing.B) {
				b.ReportAllocs()
				var sink []*user25
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					result, err := tbl.List(ctx, ex)
					if err != nil {
						b.Fatal(err)
					}
					sink = result
				}
				benchRawListSink25 = sink
			})
		})
	}
}
