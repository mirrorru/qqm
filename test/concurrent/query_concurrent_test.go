//go:build concurrent

package concurrent

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"math/big"
	"sync"
	"testing"

	"github.com/mirrorru/qqm"
	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/test/fixtures"
	"github.com/mirrorru/qqm/txproc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

func TestConcurrent_Query_NewQuery_NoRace(t *testing.T) {
	t.Parallel()

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			q := qqm.NewQuery[fixtures.UserWithOrder](dialect.SQLiteDialect{})
			_ = q.SQLs()
			_ = q.FlatFields()
		}()
	}

	wg.Wait()
}

func TestConcurrent_Query_NewQuery_MixedTypes(t *testing.T) {
	t.Parallel()

	const goroutines = 50
	var wg sync.WaitGroup

	run := func() {
		defer wg.Done()
		_ = qqm.NewQuery[fixtures.UserWithOrder](dialect.SQLiteDialect{}).SQLs()
		_ = qqm.NewQuery[fixtures.UserWithOrderLeft](dialect.SQLiteDialect{}).SQLs()
		_ = qqm.NewQuery[fixtures.UserWithSortAndOrder](dialect.SQLiteDialect{}).SQLs()
	}

	wg.Add(goroutines)
	for range goroutines {
		go run()
	}

	wg.Wait()
}

func TestConcurrent_Query_Many_INNER_JOIN(t *testing.T) {
	t.Parallel()

	db, initErr := sql.Open("sqlite", "file:test_q_inner?mode=memory&cache=shared")
	require.NoError(t, initErr)
	defer func() { _ = db.Close() }()

	db.SetMaxOpenConns(1)

	_, initErr = db.Exec(`
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
	require.NoError(t, initErr)

	ctx := context.Background()
	ex := txproc.NewDBAdapterVal(db)

	userTbl := qqm.NewTable[fixtures.User](dialect.SQLiteDialect{})
	orderTbl := qqm.NewTable[fixtures.Order](dialect.SQLiteDialect{})

	for i := range 10 {
		_, _, insErr := userTbl.Ins(ctx, ex, &fixtures.User{
			ID:    int64(i + 1),
			Name:  fmt.Sprintf("User_%d", i),
			Email: fmt.Sprintf("user%d@test.com", i),
		})
		require.NoError(t, insErr)
	}

	for i := range 10 {
		_, _, insErr := orderTbl.Ins(ctx, ex, &fixtures.Order{
			UserID: int64(i + 1),
			Amount: float64(100 + i*10),
		})
		require.NoError(t, insErr)
	}

	query := qqm.NewQuery[fixtures.UserWithOrder](dialect.SQLiteDialect{})

	const numWorkers = 10
	const queriesPerWorker = 20

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for range numWorkers {
		go func() {
			defer wg.Done()
			for range queriesPerWorker {
				results, manyErr := query.Many(ctx, ex, nil)
				assert.NoError(t, manyErr)
				assert.Len(t, results, 10)
				for _, r := range results {
					assert.NotZero(t, r.User.ID)
					assert.NotEmpty(t, r.User.Name)
					assert.NotZero(t, r.Order.ID)
				}
			}
		}()
	}

	wg.Wait()
}

func TestConcurrent_Query_Many_LEFT_JOIN(t *testing.T) {
	t.Parallel()

	db, initErr := sql.Open("sqlite", "file:test_q_left?mode=memory&cache=shared")
	require.NoError(t, initErr)
	defer func() { _ = db.Close() }()

	db.SetMaxOpenConns(1)

	_, initErr = db.Exec(`
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
	require.NoError(t, initErr)

	ctx := context.Background()
	ex := txproc.NewDBAdapterVal(db)

	userTbl := qqm.NewTable[fixtures.User](dialect.SQLiteDialect{})
	orderTbl := qqm.NewTable[fixtures.Order](dialect.SQLiteDialect{})

	for i := range 10 {
		_, _, insErr := userTbl.Ins(ctx, ex, &fixtures.User{
			ID:    int64(i + 1),
			Name:  fmt.Sprintf("User_%d", i),
			Email: fmt.Sprintf("user%d@test.com", i),
		})
		require.NoError(t, insErr)
	}

	// Добавляем заказы только первым 5 пользователям (остальные без заказов)
	for i := range 5 {
		_, _, insErr := orderTbl.Ins(ctx, ex, &fixtures.Order{
			UserID: int64(i + 1),
			Amount: float64(100 + i*10),
		})
		require.NoError(t, insErr)
	}

	query := qqm.NewQuery[fixtures.UserWithOrderLeft](dialect.SQLiteDialect{})

	const numWorkers = 10
	const queriesPerWorker = 20

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for range numWorkers {
		go func() {
			defer wg.Done()
			for range queriesPerWorker {
				results, manyErr := query.Many(ctx, ex, nil)
				assert.NoError(t, manyErr)
				assert.Len(t, results, 10)

				withOrder := 0
				withoutOrder := 0
				for _, r := range results {
					if r.Order.ID != 0 {
						withOrder++
					} else {
						withoutOrder++
					}
				}
				assert.Equal(t, 5, withOrder)
				assert.Equal(t, 5, withoutOrder)
			}
		}()
	}

	wg.Wait()
}

func TestConcurrent_Query_One_INNER_JOIN(t *testing.T) {
	t.Parallel()

	db, initErr := sql.Open("sqlite", "file:test_q_one?mode=memory&cache=shared")
	require.NoError(t, initErr)
	defer func() { _ = db.Close() }()

	db.SetMaxOpenConns(1)

	_, initErr = db.Exec(`
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
	require.NoError(t, initErr)

	ctx := context.Background()
	ex := txproc.NewDBAdapterVal(db)

	userTbl := qqm.NewTable[fixtures.User](dialect.SQLiteDialect{})
	orderTbl := qqm.NewTable[fixtures.Order](dialect.SQLiteDialect{})

	const numUsers = 20
	for i := range numUsers {
		_, _, insErr := userTbl.Ins(ctx, ex, &fixtures.User{
			ID:    int64(i + 1),
			Name:  fmt.Sprintf("User_%d", i),
			Email: fmt.Sprintf("user%d@test.com", i),
		})
		require.NoError(t, insErr)

		_, _, insErr = orderTbl.Ins(ctx, ex, &fixtures.Order{
			UserID: int64(i + 1),
			Amount: float64(100 + i*10),
		})
		require.NoError(t, insErr)
	}

	query := qqm.NewQuery[fixtures.UserWithOrder](dialect.SQLiteDialect{})

	const numWorkers = 10
	const queriesPerWorker = 20

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for range numWorkers {
		go func() {
			defer wg.Done()
			for range queriesPerWorker {
				nBig, randErr := rand.Int(rand.Reader, big.NewInt(numUsers))
				if randErr != nil {
					continue
				}
				userID := nBig.Int64() + 1

				row, oneErr := query.One(ctx, ex, userID)
				assert.NoError(t, oneErr)
				assert.Equal(t, userID, row.User.ID)
				assert.NotEmpty(t, row.User.Name)
				assert.NotZero(t, row.Order.ID)
				assert.Equal(t, userID, row.Order.UserID)
			}
		}()
	}

	wg.Wait()
}

func TestConcurrent_Query_One_LEFT_JOIN(t *testing.T) {
	t.Parallel()

	db, initErr := sql.Open("sqlite", "file:test_q_one_left?mode=memory&cache=shared")
	require.NoError(t, initErr)
	defer func() { _ = db.Close() }()

	db.SetMaxOpenConns(1)

	_, initErr = db.Exec(`
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
	require.NoError(t, initErr)

	ctx := context.Background()
	ex := txproc.NewDBAdapterVal(db)

	userTbl := qqm.NewTable[fixtures.User](dialect.SQLiteDialect{})

	const numUsers = 20
	for i := range numUsers {
		_, _, insErr := userTbl.Ins(ctx, ex, &fixtures.User{
			ID:    int64(i + 1),
			Name:  fmt.Sprintf("User_%d", i),
			Email: fmt.Sprintf("user%d@test.com", i),
		})
		require.NoError(t, insErr)
	}

	query := qqm.NewQuery[fixtures.UserWithOrderLeft](dialect.SQLiteDialect{})

	const numWorkers = 10
	const queriesPerWorker = 20

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for range numWorkers {
		go func() {
			defer wg.Done()
			for range queriesPerWorker {
				nBig, randErr := rand.Int(rand.Reader, big.NewInt(numUsers))
				if randErr != nil {
					continue
				}
				userID := nBig.Int64() + 1

				row, oneErr := query.One(ctx, ex, userID)
				assert.NoError(t, oneErr)
				assert.Equal(t, userID, row.User.ID)
				assert.NotEmpty(t, row.User.Name)
				// При LEFT JOIN без заказов Order должен быть zero-value
				assert.Zero(t, row.Order.ID)
				assert.Zero(t, row.Order.UserID)
			}
		}()
	}

	wg.Wait()
}

func TestConcurrent_Query_Many_WithFilter(t *testing.T) {
	t.Parallel()

	db, initErr := sql.Open("sqlite", "file:test_q_filter?mode=memory&cache=shared")
	require.NoError(t, initErr)
	defer func() { _ = db.Close() }()

	db.SetMaxOpenConns(1)

	_, initErr = db.Exec(`
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
	require.NoError(t, initErr)

	ctx := context.Background()
	ex := txproc.NewDBAdapterVal(db)

	userTbl := qqm.NewTable[fixtures.User](dialect.SQLiteDialect{})
	orderTbl := qqm.NewTable[fixtures.Order](dialect.SQLiteDialect{})

	for i := range 20 {
		_, _, insErr := userTbl.Ins(ctx, ex, &fixtures.User{
			ID:    int64(i + 1),
			Name:  fmt.Sprintf("User_%d", i),
			Email: fmt.Sprintf("user%d@test.com", i),
		})
		require.NoError(t, insErr)

		amount := 100.0
		if i%2 == 0 {
			amount = 500.0
		}
		_, _, insErr = orderTbl.Ins(ctx, ex, &fixtures.Order{
			UserID: int64(i + 1),
			Amount: amount,
		})
		require.NoError(t, insErr)
	}

	query := qqm.NewQuery[fixtures.UserWithOrder](dialect.SQLiteDialect{})

	// FlatFields: idx=1=users.name, idx=5=orders.amount

	const numWorkers = 8
	const queriesPerWorker = 15

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for range numWorkers {
		go func() {
			defer wg.Done()
			for range queriesPerWorker {
				// Фильтр по имени (idx=1)
				nBig, _ := rand.Int(rand.Reader, big.NewInt(20))
				nameFilter := fmt.Sprintf("User_%d", nBig.Int64())

				filter := &qqm.Filter{
					Range: qqm.And(qqm.Cond(1, qqm.CmdEq, nameFilter)),
				}
				results, manyErr := query.Many(ctx, ex, filter)
				assert.NoError(t, manyErr)
				assert.Len(t, results, 1)
				assert.Equal(t, nameFilter, results[0].User.Name)

				// Фильтр по amount (idx=5)
				filter2 := &qqm.Filter{
					Range: qqm.And(qqm.Cond(5, qqm.CmdGt, 300.0)),
				}
				results2, manyErr2 := query.Many(ctx, ex, filter2)
				assert.NoError(t, manyErr2)
				assert.Len(t, results2, 10) // ровно 10 пользователей с amount=500
				for _, r := range results2 {
					assert.InEpsilon(t, 500.0, r.Order.Amount, 0.001)
				}
			}
		}()
	}

	wg.Wait()
}

func TestConcurrent_Query_Many_LimitOffset(t *testing.T) {
	t.Parallel()

	db, initErr := sql.Open("sqlite", "file:test_q_limit?mode=memory&cache=shared")
	require.NoError(t, initErr)
	defer func() { _ = db.Close() }()

	db.SetMaxOpenConns(1)

	_, initErr = db.Exec(`
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
	require.NoError(t, initErr)

	ctx := context.Background()
	ex := txproc.NewDBAdapterVal(db)

	userTbl := qqm.NewTable[fixtures.User](dialect.SQLiteDialect{})
	orderTbl := qqm.NewTable[fixtures.Order](dialect.SQLiteDialect{})

	for i := range 30 {
		_, _, insErr := userTbl.Ins(ctx, ex, &fixtures.User{
			ID:    int64(i + 1),
			Name:  fmt.Sprintf("User_%02d", i),
			Email: fmt.Sprintf("u%02d@test.com", i),
		})
		require.NoError(t, insErr)
		_, _, insErr = orderTbl.Ins(ctx, ex, &fixtures.Order{
			UserID: int64(i + 1),
			Amount: 100.0,
		})
		require.NoError(t, insErr)
	}

	query := qqm.NewQuery[fixtures.UserWithOrder](dialect.SQLiteDialect{})

	const numWorkers = 8
	const queriesPerWorker = 15

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for range numWorkers {
		go func() {
			defer wg.Done()
			for range queriesPerWorker {
				filter := &qqm.Filter{Limit: 10}
				results, manyErr := query.Many(ctx, ex, filter)
				assert.NoError(t, manyErr)
				assert.Len(t, results, 10)

				filter2 := &qqm.Filter{Offset: 5, Limit: 15}
				results2, manyErr2 := query.Many(ctx, ex, filter2)
				assert.NoError(t, manyErr2)
				assert.Len(t, results2, 15)
			}
		}()
	}

	wg.Wait()
}

func TestConcurrent_Query_SQLs_Access(t *testing.T) {
	t.Parallel()

	query := qqm.NewQuery[fixtures.UserWithOrder](dialect.SQLiteDialect{})

	const goroutines = 200
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			sqls := query.SQLs()
			assert.NotEmpty(t, sqls.GetOneCmd)
			assert.NotEmpty(t, sqls.ListCmdStart)

			flatFields := query.FlatFields()
			assert.NotEmpty(t, flatFields)
			// UserWithOrder: 3 поля users + 3 поля orders = 6
			assert.Len(t, flatFields, 6)
		}()
	}

	wg.Wait()
}

func TestConcurrent_Query_Many_Sort(t *testing.T) {
	t.Parallel()

	db, initErr := sql.Open("sqlite", "file:test_q_sort?mode=memory&cache=shared")
	require.NoError(t, initErr)
	defer func() { _ = db.Close() }()

	db.SetMaxOpenConns(1)

	_, initErr = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT NOT NULL,
			age INTEGER NOT NULL DEFAULT 0
		);
		CREATE TABLE orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL REFERENCES users(id),
			amount REAL NOT NULL
		)
	`)
	require.NoError(t, initErr)

	ctx := context.Background()
	ex := txproc.NewDBAdapterVal(db)

	userTbl := qqm.NewTable[fixtures.UserWithSort](dialect.SQLiteDialect{})
	orderTbl := qqm.NewTable[fixtures.Order](dialect.SQLiteDialect{})

	_, _, insErr := userTbl.Ins(ctx, ex, &fixtures.UserWithSort{
		ID: 1, Name: "Charlie", Email: "c@test.com", Age: 30,
	})
	require.NoError(t, insErr)
	_, _, insErr = userTbl.Ins(ctx, ex, &fixtures.UserWithSort{
		ID: 2, Name: "Alice", Email: "a@test.com", Age: 25,
	})
	require.NoError(t, insErr)
	_, _, insErr = userTbl.Ins(ctx, ex, &fixtures.UserWithSort{
		ID: 3, Name: "Bob", Email: "b@test.com", Age: 35,
	})
	require.NoError(t, insErr)

	for _, uid := range []int64{1, 2, 3} {
		_, _, insErr = orderTbl.Ins(ctx, ex, &fixtures.Order{
			UserID: uid, Amount: float64(uid * 100),
		})
		require.NoError(t, insErr)
	}

	query := qqm.NewQuery[fixtures.UserWithSortAndOrder](dialect.SQLiteDialect{})

	const numWorkers = 10
	const queriesPerWorker = 20

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for range numWorkers {
		go func() {
			defer wg.Done()
			for range queriesPerWorker {
				results, manyErr := query.Many(ctx, ex, nil)
				assert.NoError(t, manyErr)
				assert.Len(t, results, 3)
				// Сортировка: Name ASC (sort=1), Email DESC (sort=2)
				assert.Equal(t, "Alice", results[0].User.Name)
				assert.Equal(t, "Bob", results[1].User.Name)
				assert.Equal(t, "Charlie", results[2].User.Name)
			}
		}()
	}

	wg.Wait()
}

func TestConcurrent_Query_NewQuery_Postgres(t *testing.T) {
	t.Parallel()

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			q := qqm.NewQuery[fixtures.UserWithOrder](dialect.PostgreSQLDialect{})
			sqls := q.SQLs()
			assert.NotEmpty(t, sqls.GetOneCmd)
			assert.NotEmpty(t, sqls.ListCmdStart)

			q2 := qqm.NewQuery[fixtures.UserWithOrderLeft](dialect.PostgreSQLDialect{})
			sqls2 := q2.SQLs()
			assert.NotEmpty(t, sqls2.GetOneCmd)
			assert.NotEmpty(t, sqls2.ListCmdStart)
		}()
	}

	wg.Wait()
}
