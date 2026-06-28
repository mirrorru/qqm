// Created at 2026-06-28
//go:build functional

package functional

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

const (
	envDSN = "TEST_PG_DSN"

	defaultDSN = "noDSNsepecified"
)

// pgDSN формирует DSN для подключения к тестовой БД PostgreSQL.
func pgDSN() string {
	return envOrDefault(envDSN, defaultDSN)
}

// envOrDefault возвращает значение переменной окружения или default.
func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// openTestPG открывает подключение к тестовой БД PostgreSQL.
func openTestPG(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("postgres", pgDSN())
	require.NoError(t, err, "failed to open PostgreSQL connection")

	err = db.Ping()
	require.NoError(t, err, "failed to ping PostgreSQL")

	return db
}
