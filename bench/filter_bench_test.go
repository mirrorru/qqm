package bench

import (
	"testing"

	"github.com/mirrorru/qqm"
	"github.com/mirrorru/qqm/dialect"
	"github.com/mirrorru/qqm/test/fixtures"

	_ "modernc.org/sqlite"
)

func BenchmarkFilter_Build_Simple(b *testing.B) {
	d := dialect.SQLiteDialect{}
	tbl := qqm.NewTable[fixtures.User](dialect.SQLiteDialect{})
	fields := tbl.Defs().Fields

	filter := &qqm.Filter{
		Range: qqm.And(qqm.Cond(1, qqm.CmdEq, "test")),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _, _ = filter.BuildWhere(fields, d)
	}
}

func BenchmarkFilter_Build_Complex(b *testing.B) {
	d := dialect.SQLiteDialect{}
	tbl := qqm.NewTable[fixtures.User](dialect.SQLiteDialect{})
	fields := tbl.Defs().Fields

	filter := &qqm.Filter{
		Range: qqm.And(
			qqm.Cond(1, qqm.CmdEq, "test"),
			qqm.Or(
				qqm.Cond(2, qqm.CmdLike, "%example%"),
				qqm.Cond(2, qqm.CmdILike, "%test%"),
			),
			qqm.Cond(3, qqm.CmdGt, 25),
		),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _, _ = filter.BuildWhere(fields, d)
	}
}

func BenchmarkFilter_Build_IN(b *testing.B) {
	d := dialect.SQLiteDialect{}
	tbl := qqm.NewTable[fixtures.User](dialect.SQLiteDialect{})
	fields := tbl.Defs().Fields

	vals := make([]any, 10)
	for i := range vals {
		vals[i] = i
	}

	filter := &qqm.Filter{
		Range: qqm.And(qqm.Cond(0, qqm.CmdIn, vals)),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _, _ = filter.BuildWhere(fields, d)
	}
}

func BenchmarkFilter_Build_OffsetAndLimit(b *testing.B) {
	d := dialect.SQLiteDialect{}

	filter := &qqm.Filter{
		Offset: 10,
		Limit:  20,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = filter.BuildOffsetAndLimit(d)
	}
}
