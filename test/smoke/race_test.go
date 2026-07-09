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
	"github.com/mirrorru/qqm/txproc"
	"github.com/stretchr/testify/require"

	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/test/fixtures"
	_ "modernc.org/sqlite"
)

func TestSmoke_DataRace_ConcurrentCRUDAndList(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	db.SetMaxOpenConns(1)

	ctx := context.Background()
	ex := txproc.NewDBAdapterVal(db)
	tbl := qqm.NewTable[fixtures.Rooms](dialect.SQLiteDialect{})

	_, err = db.Exec(`
		CREATE TABLE rooms (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			square REAL NOT NULL,
			created_at INTEGER NOT NULL DEFAULT 0
		)
	`)
	require.NoError(t, err)

	const numRows = 30
	const numWorkers = 10
	const opsPerWorker = 30

	ids := make([]int64, numRows)
	for i := range numRows {
		inserted, err := tbl.Insert(ctx, ex, &fixtures.Rooms{
			Name:   fmt.Sprintf("Room_%d", i),
			Square: float64(i) * 10.0,
		})
		require.NoError(t, err)
		ids[i] = inserted.ID
	}

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for w := range numWorkers {
		go func(workerID int) {
			defer wg.Done()
			for op := range opsPerWorker {
				nBig, err := rand.Int(rand.Reader, big.NewInt(int64(len(ids))))
				if err != nil {
					continue
				}
				idx := int(nBig.Int64())
				rowID := ids[idx]

				switch op % 5 {
				case 0:
					_, _ = tbl.GetByPK(ctx, ex, rowID)
				case 1:
					fetched, getErr := tbl.GetByPK(ctx, ex, rowID)
					if getErr == nil {
						fetched.Name = fmt.Sprintf("W%d_O%d", workerID, op)
						_, _ = tbl.Update(ctx, ex, fetched)
					}
				case 2:
					inserted, insErr := tbl.Insert(ctx, ex, &fixtures.Rooms{
						Name:   fmt.Sprintf("New_W%d_O%d", workerID, op),
						Square: float64(workerID*100 + op),
					})
					if insErr == nil && inserted.ID != 0 {
						_ = tbl.Delete(ctx, ex, inserted.ID)
					}
				case 3:
					_, _ = tbl.List(ctx, ex)
				case 4:
					_, _ = tbl.List(ctx, ex, qqm.AndFilter(
						qqm.Field("Square", qqm.And, qqm.Gt(float64(workerID*5))),
					))
				}
			}
		}(w)
	}

	wg.Wait()

	result, err := tbl.List(ctx, ex)
	require.NoError(t, err)
	require.Len(t, result, numRows)
}
