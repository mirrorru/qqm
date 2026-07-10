//go:build concurrent

package concurrent

import (
	"sync"
	"testing"

	"github.com/mirrorru/qqm"
	"github.com/mirrorru/qqm/dialect"
	"github.com/stretchr/testify/assert"
)

func TestConcurrent_FilterBuild_NoRace(t *testing.T) {
	t.Parallel()

	tbl := qqm.NewTable[struct {
		ID   int64  `tbl:"pk;auto"`
		Name string `tbl:"sort=1"`
		Age  int
	}](dialect.SQLiteDialect{})

	tf := tbl.Defs().Fields

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			filter := &qqm.Filter{
				Range: qqm.And(
					qqm.Cond(1, qqm.CmdEq, "test"),
					qqm.Cond(2, qqm.CmdGte, 25),
				),
			}
			where, args := filter.BuildWhere(tf, dialect.SQLiteDialect{})
			assert.NotEmpty(t, where)
			assert.Len(t, args, 2)
		}()
	}

	wg.Wait()
}

func TestConcurrent_CondConstruction(t *testing.T) {
	t.Parallel()

	const goroutines = 200
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			eq := qqm.Cond(0, qqm.CmdEq, "value")
			assert.NotNil(t, eq)

			gt := qqm.Cond(1, qqm.CmdGt, 10)
			assert.NotNil(t, gt)

			ilt := qqm.Cond(2, qqm.CmdILike, "%test%")
			assert.NotNil(t, ilt)

			in := qqm.Cond(3, qqm.CmdIn, []any{1, 2, 3})
			assert.NotNil(t, in)

			isNull := qqm.Cond(4, qqm.CmdIsNull, nil)
			assert.NotNil(t, isNull)
		}()
	}

	wg.Wait()
}

func TestConcurrent_GroupNodeBuild(t *testing.T) {
	t.Parallel()

	tbl := qqm.NewTable[struct {
		ID     int64 `tbl:"pk;auto"`
		Name   string
		Status string
		Amount float64
	}](dialect.SQLiteDialect{})

	tf := tbl.Defs().Fields

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			andFilter := &qqm.Filter{
				Range: qqm.And(
					qqm.Cond(0, qqm.CmdGt, 5),
					qqm.Cond(1, qqm.CmdLike, "%prefix%"),
				),
			}
			where, args := andFilter.BuildWhere(tf, dialect.SQLiteDialect{})
			assert.NotEmpty(t, where)
			assert.Len(t, args, 2)

			orFilter := &qqm.Filter{
				Range: qqm.Or(
					qqm.Cond(2, qqm.CmdEq, "active"),
					qqm.Cond(2, qqm.CmdEq, "pending"),
				),
			}
			where2, args2 := orFilter.BuildWhere(tf, dialect.SQLiteDialect{})
			assert.NotEmpty(t, where2)
			assert.Len(t, args2, 2)

			notFilter := &qqm.Filter{
				Range: qqm.Not(
					qqm.Cond(0, qqm.CmdEq, 0),
				),
			}
			where3, args3 := notFilter.BuildWhere(tf, dialect.SQLiteDialect{})
			assert.NotEmpty(t, where3)
			assert.Len(t, args3, 1)
		}()
	}

	wg.Wait()
}

func TestConcurrent_Filter_LimitOffset(t *testing.T) {
	t.Parallel()

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			filters := []*qqm.Filter{
				{Limit: 10},
				{Offset: 20},
				{Limit: 5, Offset: 15},
				nil,
			}

			for _, f := range filters {
				clause := f.BuildOffsetAndLimit(dialect.SQLiteDialect{})
				_ = clause

				clause2 := f.BuildOffsetAndLimit(dialect.PostgreSQLDialect{})
				_ = clause2
			}
		}()
	}

	wg.Wait()
}

func TestConcurrent_FilterCombinations(t *testing.T) {
	t.Parallel()

	tbl := qqm.NewTable[struct {
		ID     int64 `tbl:"pk;auto"`
		Name   string
		Age    int
		Active bool
	}](dialect.SQLiteDialect{})

	tf := tbl.Defs().Fields

	const goroutines = 150
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			complexFilter := &qqm.Filter{
				Limit: 50,
				Range: qqm.And(
					qqm.Cond(2, qqm.CmdGte, 18),
					qqm.Or(
						qqm.Cond(1, qqm.CmdLike, "A%"),
						qqm.Cond(1, qqm.CmdLike, "B%"),
					),
					qqm.Not(
						qqm.Cond(3, qqm.CmdEq, false),
					),
				),
			}

			where, args := complexFilter.BuildWhere(tf, dialect.SQLiteDialect{})
			assert.NotEmpty(t, where)
			assert.Len(t, args, 4)

			limitClause := complexFilter.BuildOffsetAndLimit(dialect.SQLiteDialect{})
			assert.NotEmpty(t, limitClause)
		}()
	}

	wg.Wait()
}
