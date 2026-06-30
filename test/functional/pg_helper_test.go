//go:build functional

package functional

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	_ "github.com/lib/pq"
	"github.com/mirrorru/qqm"
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

// beginTxPG открывает подключение, начинает транзакцию с уровнем изоляции
// REPEATABLE READ и регистрирует rollback + закрытие в t.Cleanup.
// Каждый тест получает изолированное состояние БД,
// не затирая данные параллельных тестов.
func beginTxPG(t *testing.T) (*sql.Tx, qqm.Executor) {
	t.Helper()
	db := openTestPG(t)
	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	require.NoError(t, err, "failed to begin transaction")
	t.Cleanup(func() {
		_ = tx.Rollback()
		_ = db.Close()
	})
	return tx, qqm.NewTxAdapterVal(tx)
}

// openTestPGX открывает подключение к тестовой БД PostgreSQL через pgx/v5.
func openTestPGX(t *testing.T) *pgx.Conn {
	t.Helper()
	conn, err := pgx.Connect(context.Background(), pgDSN())
	require.NoError(t, err, "failed to connect via pgx")
	t.Cleanup(func() {
		_ = conn.Close(context.Background())
	})
	return conn
}

// beginTxPGX открывает pgx-подключение, начинает транзакцию с уровнем
// изоляции REPEATABLE READ и регистрирует rollback + закрытие в t.Cleanup.
func beginTxPGX(t *testing.T) (pgx.Tx, qqm.Executor) {
	t.Helper()
	conn := openTestPGX(t)
	ctx := context.Background()
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	require.NoError(t, err, "failed to begin pgx transaction")
	t.Cleanup(func() {
		_ = tx.Rollback(ctx)
		_ = conn.Close(ctx)
	})
	return tx, qqm.NewPGXTxAdapterVal(tx)
}
