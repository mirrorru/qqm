//go:build functional

package functional

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

func TestMain(m *testing.M) {
	db, err := sql.Open("postgres", pgDSN())
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open PostgreSQL for table setup: %v\n", err)
		os.Exit(m.Run())
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		fmt.Fprintf(os.Stderr, "PostgreSQL not available, skipping DB-dependent tests: %v\n", err)
		os.Exit(m.Run())
	}

	setupTables(db)
	code := m.Run()
	tearDownTables(db)
	_ = db.Close()
	os.Exit(code)
}

func setupTables(db *sql.DB) {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS rooms (
			id BIGSERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			square DOUBLE PRECISION NOT NULL,
			created_at BIGINT NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS room_mapping (
			room_id BIGINT NOT NULL,
			teacher_ID BIGINT NOT NULL,
			time_from BIGINT NOT NULL,
			time_to BIGINT NOT NULL,
			created_at BIGINT NOT NULL DEFAULT 0,
			PRIMARY KEY (room_id, teacher_ID)
		)`,
		`CREATE TABLE IF NOT EXISTS user_with_age (
			id BIGSERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT NOT NULL,
			age BIGINT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS full_room_mapping (
			room_id BIGINT NOT NULL,
			teacher_ID BIGINT NOT NULL,
			time_from BIGINT NOT NULL,
			time_to BIGINT NOT NULL,
			created_at BIGINT NOT NULL DEFAULT 0,
			author_name TEXT NOT NULL,
			PRIMARY KEY (room_id, teacher_ID)
		)`,
		`CREATE TABLE IF NOT EXISTS users (
			id BIGSERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS orders (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id),
			amount DOUBLE PRECISION NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS order_items (
			id BIGSERIAL PRIMARY KEY,
			order_id BIGINT NOT NULL REFERENCES orders(id),
			quantity INTEGER NOT NULL,
			price DOUBLE PRECISION NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS some_table (
			some_id BIGSERIAL PRIMARY KEY,
			field_rw TEXT NOT NULL,
			field_ro TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
	}
	for _, ddl := range tables {
		if _, err := db.Exec(ddl); err != nil {
			panic(fmt.Sprintf("failed to create table: %v\nSQL: %s", err, ddl))
		}
	}
}

func tearDownTables(db *sql.DB) {
	tables := []string{
		"order_items",
		"orders",
		"users",
		"some_table",
		"full_room_mapping",
		"user_with_age",
		"room_mapping",
		"rooms",
	}
	for _, t := range tables {
		_, _ = db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", t))
	}
}
