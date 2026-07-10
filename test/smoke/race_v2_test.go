//go:build smoke

package smoke

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
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

func TestSmoke_RaceV2_ConcurrentCRUD(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	db.SetMaxOpenConns(1)

	ctx := context.Background()
	ex := txproc.NewDBAdapterVal(db)
	tbl := qqm.NewTable[fixtures.User](dialect.SQLiteDialect{})

	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			email TEXT NOT NULL
		)
	`)
	require.NoError(t, err)

	const numRows = 20
	const numWorkers = 8
	const opsPerWorker = 20

	ids := make([]int64, numRows)
	for i := range numRows {
		u, _, err := tbl.Ins(ctx, ex, &fixtures.User{
			Name:  fmt.Sprintf("User_%d", i),
			Email: "test@test.com",
		})
		require.NoError(t, err)
		ids[i] = u.ID
	}

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for range numWorkers {
		go func() {
			defer wg.Done()
			for op := range opsPerWorker {
				nBig, err := rand.Int(rand.Reader, big.NewInt(int64(len(ids))))
				if err != nil {
					continue
				}
				idx := int(nBig.Int64())
				rowID := ids[idx]

				switch op % 4 {
				case 0:
					_, _ = tbl.One(ctx, ex, rowID)
				case 1:
					fetched, getErr := tbl.One(ctx, ex, rowID)
					if getErr == nil {
						fetched.Name = "updated"
						_, _, _ = tbl.Upd(ctx, ex, fetched)
					}
				case 2:
					u, _, insErr := tbl.Ins(ctx, ex, &fixtures.User{
						Name:  "new_user",
						Email: "new@test.com",
					})
					if insErr == nil && u.ID != 0 {
						_, _ = tbl.Del(ctx, ex, u.ID)
					}
				case 3:
					_, _ = tbl.Many(ctx, ex, nil)
				}
			}
		}()
	}

	wg.Wait()

	result, err := tbl.Many(ctx, ex, nil)
	require.NoError(t, err)
	require.Len(t, result, numRows)
}

func TestSmoke_RaceV2_ParallelNewTable(t *testing.T) {
	t.Parallel()

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			tbl := qqm.NewTable[fixtures.User](dialect.SQLiteDialect{})
			_ = tbl.SQLs()
		}()
	}

	wg.Wait()
}
