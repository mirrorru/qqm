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
	"github.com/mirrorru/qqm/test/fixtures"
	"github.com/mirrorru/qqm/txproc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

func TestConcurrent_NewTable_NoRace(t *testing.T) {
	t.Parallel()

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			tbl := qqm.NewTable[fixtures.User](qqm.SQLiteDialect)
			_ = tbl.SQLs()
			_ = tbl.Defs()
		}()
	}

	wg.Wait()
}

func TestConcurrent_NewTable_MixedTypes(t *testing.T) {
	t.Parallel()

	const goroutines = 50
	var wg sync.WaitGroup

	run := func() {
		defer wg.Done()
		_ = qqm.NewTable[fixtures.User](qqm.SQLiteDialect).SQLs()
		_ = qqm.NewTable[fixtures.OrgUser](qqm.SQLiteDialect).SQLs()
		_ = qqm.NewTable[fixtures.Rooms](qqm.SQLiteDialect).SQLs()
		_ = qqm.NewTable[fixtures.RowWithEmbeddedPK](qqm.SQLiteDialect).SQLs()
		_ = qqm.NewTable[fixtures.RowWithDeepEmbed](qqm.SQLiteDialect).SQLs()
		_ = qqm.NewTable[fixtures.PersonWithAddress](qqm.SQLiteDialect).SQLs()
	}

	wg.Add(goroutines)
	for range goroutines {
		go run()
	}

	wg.Wait()
}

//nolint:gocognit
func TestConcurrent_CRUD_SharedTable(t *testing.T) {
	t.Parallel()

	db, initErr := sql.Open("sqlite", "file:test_crud_shared?mode=memory&cache=shared")
	require.NoError(t, initErr)
	defer func() { _ = db.Close() }()

	db.SetMaxOpenConns(1)

	ctx := context.Background()
	ex := txproc.NewDBAdapterVal(db)
	tbl := qqm.NewTable[fixtures.User](qqm.SQLiteDialect)

	_, initErr = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			email TEXT NOT NULL
		)
	`)
	require.NoError(t, initErr)

	const numRows = 30
	const numWorkers = 8
	const opsPerWorker = 30

	ids := make([]int64, numRows)
	for i := range numRows {
		u, _, insErr := tbl.Ins(ctx, ex, &fixtures.User{
			Name:  fmt.Sprintf("User_%d", i),
			Email: fmt.Sprintf("user%d@test.com", i),
		})
		require.NoError(t, insErr)
		ids[i] = u.ID
	}

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for range numWorkers {
		go func() {
			defer wg.Done()
			for op := range opsPerWorker {
				nBig, randErr := rand.Int(rand.Reader, big.NewInt(int64(len(ids))))
				if randErr != nil {
					continue
				}
				idx := int(nBig.Int64())
				rowID := ids[idx]

				switch op % 5 {
				case 0:
					_, oneErr := tbl.One(ctx, ex, rowID)
					assert.NoError(t, oneErr)
				case 1:
					fetched, getErr := tbl.One(ctx, ex, rowID)
					if getErr == nil {
						fetched.Name = "updated_concurrent"
						_, _, _ = tbl.Upd(ctx, ex, fetched)
					}
				case 2:
					u, _, insErr := tbl.Ins(ctx, ex, &fixtures.User{
						Name:  "concurrent_insert",
						Email: "concurrent@test.com",
					})
					if insErr == nil && u.ID != 0 {
						_, _ = tbl.Del(ctx, ex, u.ID)
					}
				case 3:
					results, manyErr := tbl.Many(ctx, ex, nil)
					assert.NoError(t, manyErr)
					assert.NotEmpty(t, results)
				case 4:
					results, _ := tbl.Many(ctx, ex, &qqm.Filter{Limit: 5})
					_ = results
				}
			}
		}()
	}

	wg.Wait()

	result, err := tbl.Many(ctx, ex, nil)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(result), numRows)
}

func TestConcurrent_InsDelete_ManyWorkers(t *testing.T) {
	t.Parallel()

	db, initErr := sql.Open("sqlite", "file:test_ins_del?mode=memory&cache=shared")
	require.NoError(t, initErr)
	defer func() { _ = db.Close() }()

	db.SetMaxOpenConns(1)

	ctx := context.Background()
	ex := txproc.NewDBAdapterVal(db)
	tbl := qqm.NewTable[fixtures.User](qqm.SQLiteDialect)

	_, initErr = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			email TEXT NOT NULL
		)
	`)
	require.NoError(t, initErr)

	const numWorkers = 10
	const insertsPerWorker = 10

	var mu sync.Mutex
	var allIDs []int64

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for w := range numWorkers {
		go func(workerID int) {
			defer wg.Done()
			for i := range insertsPerWorker {
				u, _, insErr := tbl.Ins(ctx, ex, &fixtures.User{
					Name:  fmt.Sprintf("Worker_%d_Insert_%d", workerID, i),
					Email: fmt.Sprintf("w%d_i%d@test.com", workerID, i),
				})
				if insErr == nil {
					mu.Lock()
					allIDs = append(allIDs, u.ID)
					mu.Unlock()
				}
			}
		}(w)
	}

	wg.Wait()

	result, err := tbl.Many(ctx, ex, nil)
	require.NoError(t, err)
	assert.Len(t, result, numWorkers*insertsPerWorker)
	assert.Len(t, allIDs, numWorkers*insertsPerWorker)

	for _, id := range allIDs {
		row, oneErr := tbl.One(ctx, ex, id)
		require.NoError(t, oneErr)
		assert.NotZero(t, row.ID)
		assert.NotEmpty(t, row.Name)
		assert.NotEmpty(t, row.Email)
	}
}

//nolint:gocognit
func TestConcurrent_UpdDelete_Interleaved(t *testing.T) {
	t.Parallel()

	db, initErr := sql.Open("sqlite", "file:test_upd_del?mode=memory&cache=shared")
	require.NoError(t, initErr)
	defer func() { _ = db.Close() }()

	db.SetMaxOpenConns(1)

	ctx := context.Background()
	ex := txproc.NewDBAdapterVal(db)
	tbl := qqm.NewTable[fixtures.UserWithAge](qqm.SQLiteDialect)

	_, initErr = db.Exec(`
		CREATE TABLE user_with_age (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			email TEXT NOT NULL,
			age INTEGER NOT NULL DEFAULT 0
		)
	`)
	require.NoError(t, initErr)

	const numRows = 20
	ids := make([]int64, numRows)
	for i := range numRows {
		u, _, insErr := tbl.Ins(ctx, ex, &fixtures.UserWithAge{
			Name:  fmt.Sprintf("User_%d", i),
			Email: fmt.Sprintf("user%d@test.com", i),
			Age:   0,
		})
		require.NoError(t, insErr)
		ids[i] = u.ID
	}

	const numWorkers = 5
	const iters = 10

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	// Каждый воркер отвечает за свой диапазон строк, чтобы гарантировать покрытие всех ID.
	for w := range numWorkers {
		go func(workerID int) {
			defer wg.Done()
			// Делим ids на диапазоны между воркерами
			chunkSize := (len(ids) + numWorkers - 1) / numWorkers
			start := workerID * chunkSize
			end := min(start+chunkSize, len(ids))
			myIDs := ids[start:end]

			for iter := range iters {
				rowID := myIDs[iter%len(myIDs)]

				fetched, getErr := tbl.One(ctx, ex, rowID)
				if getErr != nil {
					continue
				}

				fetched.Age++
				_, _, updErr := tbl.Upd(ctx, ex, fetched)
				if updErr != nil {
					continue
				}

				verify, verErr := tbl.One(ctx, ex, rowID)
				if verErr == nil {
					assert.GreaterOrEqual(t, verify.Age, 1)
				}
			}
		}(w)
	}

	wg.Wait()

	for _, id := range ids {
		row, oneErr := tbl.One(ctx, ex, id)
		require.NoError(t, oneErr)
		assert.Positive(t, row.Age, "row %d should have been updated at least once", id)
	}
}

func TestConcurrent_CompositeKey_CRUD(t *testing.T) {
	t.Parallel()

	db, initErr := sql.Open("sqlite", "file:test_composite?mode=memory&cache=shared")
	require.NoError(t, initErr)
	defer func() { _ = db.Close() }()

	db.SetMaxOpenConns(1)

	ctx := context.Background()
	ex := txproc.NewDBAdapterVal(db)
	tbl := qqm.NewTable[fixtures.OrgUser](qqm.SQLiteDialect)

	_, initErr = db.Exec(`
		CREATE TABLE org_users (
			org_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			email TEXT NOT NULL,
			PRIMARY KEY (org_id, user_id)
		)
	`)
	require.NoError(t, initErr)

	const numOrgs = 5
	const usersPerOrg = 10
	const numWorkers = 8
	const iters = 15

	var mu sync.Mutex
	inserted := make(map[[2]int64]bool)

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for range numWorkers {
		go func() {
			defer wg.Done()
			for range iters {
				nBigOrg, _ := rand.Int(rand.Reader, big.NewInt(numOrgs))
				nBigUser, _ := rand.Int(rand.Reader, big.NewInt(usersPerOrg))
				orgID := nBigOrg.Int64() + 1
				userID := nBigUser.Int64() + 1

				_, _, insErr := tbl.Ins(ctx, ex, &fixtures.OrgUser{
					OrgID:  orgID,
					UserID: userID,
					Name:   fmt.Sprintf("org%d_user%d", orgID, userID),
					Email:  fmt.Sprintf("o%d_u%d@test.com", orgID, userID),
				})
				if insErr == nil {
					mu.Lock()
					inserted[[2]int64{orgID, userID}] = true
					mu.Unlock()
				}
			}
		}()
	}

	wg.Wait()

	for key := range inserted {
		row, oneErr := tbl.One(ctx, ex, key[0], key[1])
		require.NoError(t, oneErr)
		assert.Equal(t, key[0], row.OrgID)
		assert.Equal(t, key[1], row.UserID)
	}
}

func TestConcurrent_Many_WithFilter(t *testing.T) {
	t.Parallel()

	db, initErr := sql.Open("sqlite", "file:test_filter_many?mode=memory&cache=shared")
	require.NoError(t, initErr)
	defer func() { _ = db.Close() }()

	db.SetMaxOpenConns(1)

	ctx := context.Background()
	ex := txproc.NewDBAdapterVal(db)
	tbl := qqm.NewTable[fixtures.UserWithAge](qqm.SQLiteDialect)

	_, initErr = db.Exec(`
		CREATE TABLE user_with_age (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			email TEXT NOT NULL,
			age INTEGER NOT NULL DEFAULT 0
		)
	`)
	require.NoError(t, initErr)

	for i := range 50 {
		_, _, insErr := tbl.Ins(ctx, ex, &fixtures.UserWithAge{
			Name:  fmt.Sprintf("User_%d", i),
			Email: fmt.Sprintf("user%d@test.com", i),
			Age:   i,
		})
		require.NoError(t, insErr)
	}

	const numWorkers = 10
	const queriesPerWorker = 20

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for range numWorkers {
		go func() {
			defer wg.Done()
			for range queriesPerWorker {
				nBig, _ := rand.Int(rand.Reader, big.NewInt(50))
				minAge := int(nBig.Int64())

				filter := &qqm.Filter{
					Range: qqm.And(
						qqm.Cond(3, qqm.CmdGte, minAge),
					),
				}

				result, manyErr := tbl.Many(ctx, ex, filter)
				assert.NoError(t, manyErr)
				for _, r := range result {
					assert.GreaterOrEqual(t, r.Age, minAge)
				}
			}
		}()
	}

	wg.Wait()
}

func TestConcurrent_SQLs_Access(t *testing.T) {
	t.Parallel()

	tbl := qqm.NewTable[fixtures.User](qqm.SQLiteDialect)

	const goroutines = 200
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			sqls := tbl.SQLs()
			assert.NotEmpty(t, sqls.InsertCmd)
			assert.NotEmpty(t, sqls.UpdateCmd)
			assert.NotEmpty(t, sqls.GetOneCmd)
			assert.NotEmpty(t, sqls.DeleteCmd)
			assert.NotEmpty(t, sqls.ListCmdStart)

			defs := tbl.Defs()
			assert.NotEmpty(t, defs.TableName)
			assert.NotEmpty(t, defs.Fields)
		}()
	}

	wg.Wait()
}

func TestConcurrent_Table_WithEmbeddedFields(t *testing.T) {
	t.Parallel()

	db, initErr := sql.Open("sqlite", "file:test_embedded?mode=memory&cache=shared")
	require.NoError(t, initErr)
	defer func() { _ = db.Close() }()

	db.SetMaxOpenConns(1)

	ctx := context.Background()
	ex := txproc.NewDBAdapterVal(db)
	tbl := qqm.NewTable[fixtures.RoomMapping](qqm.SQLiteDialect)

	_, initErr = db.Exec(`
		CREATE TABLE room_mapping (
			room_id INTEGER NOT NULL,
			teacher_id INTEGER NOT NULL,
			time_from INTEGER NOT NULL,
			time_to INTEGER NOT NULL,
			created_at INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (room_id, teacher_id)
		)
	`)
	require.NoError(t, initErr)

	const numWorkers = 8
	const opsPerWorker = 15

	var mu sync.Mutex
	inserted := make(map[[2]int64]bool)

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for range numWorkers {
		go func() {
			defer wg.Done()
			for range opsPerWorker {
				nBigRoom, _ := rand.Int(rand.Reader, big.NewInt(20))
				nBigTeacher, _ := rand.Int(rand.Reader, big.NewInt(10))
				roomID := nBigRoom.Int64() + 1
				teacherID := nBigTeacher.Int64() + 1

				_, _, insErr := tbl.Ins(ctx, ex, &fixtures.RoomMapping{
					MappingRoomID: fixtures.MappingRoomID{ID: roomID},
					TeacherKey:    fixtures.TeacherKey{Key: fixtures.TeacherID(teacherID)},
					From:          100,
					To:            200,
				})
				if insErr == nil {
					mu.Lock()
					inserted[[2]int64{roomID, teacherID}] = true
					mu.Unlock()
				}
			}
		}()
	}

	wg.Wait()

	assert.NotEmpty(t, inserted)

	for key := range inserted {
		row, oneErr := tbl.One(ctx, ex, key[0], key[1])
		require.NoError(t, oneErr)
		assert.Equal(t, key[0], row.MappingRoomID.ID)
		assert.Equal(t, fixtures.TeacherID(key[1]), row.TeacherKey.Key)
		assert.Equal(t, int64(100), row.From)
		assert.Equal(t, int64(200), row.To)
	}
}

func TestConcurrent_NewTable_Postgres(t *testing.T) {
	t.Parallel()

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			tbl := qqm.NewTable[fixtures.User](qqm.PostgreSQLDialect)
			sqls := tbl.SQLs()
			assert.NotEmpty(t, sqls.InsertCmd)
			assert.NotEmpty(t, sqls.UpdateCmd)
		}()
	}

	wg.Wait()
}
