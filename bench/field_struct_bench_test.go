package bench

import (
	"reflect"
	"testing"

	"github.com/mirrorru/qqm"
	"github.com/mirrorru/qqm/test/fixtures"
)

func BenchmarkCollectTableFields_Simple(b *testing.B) {
	t := reflect.TypeOf(fixtures.User{})
	for i := 0; i < b.N; i++ {
		_, _ = qqm.CollectTableFields(t)
	}
}

func BenchmarkCollectTableFields_Nested(b *testing.B) {
	t := reflect.TypeOf(fixtures.PersonWithAddress{})
	for i := 0; i < b.N; i++ {
		_, _ = qqm.CollectTableFields(t)
	}
}

func BenchmarkCollectTableFields_Cached(b *testing.B) {
	t := reflect.TypeOf(fixtures.User{})
	qqm.CollectTableFields(t)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = qqm.CollectTableFields(t)
	}
}

func BenchmarkCollectTableFields_DeepNested(b *testing.B) {
	t := reflect.TypeOf(fixtures.RowWithDeepEmbed{})
	for i := 0; i < b.N; i++ {
		_, _ = qqm.CollectTableFields(t)
	}
}

func BenchmarkTable_New_WithFields(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = qqm.NewTable[fixtures.User](qqm.SQLiteDialect)
	}
}
