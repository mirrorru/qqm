// Created at 2026-06-28
package meta

import (
	"reflect"
	"sync"
)

// metaCache — thread-safe кэш метаданных на базе sync.Map.
var metaCache sync.Map

// Created at 2026-06-28
// GetOrBuildRowMeta возвращает кэшированную RowMeta или строит новую.
// LoadOrStore гарантирует, что для одного типа построится только один RowMeta.
func GetOrBuildRowMeta(t reflect.Type, tableName string) *RowMeta {
	// разыменовываем указатель для ключа кэша
	key := t
	for key.Kind() == reflect.Pointer {
		key = key.Elem()
	}

	if cached, ok := metaCache.Load(key); ok {
		return cached.(*RowMeta)
	}

	rm := BuildRowMeta(t, tableName)
	actual, _ := metaCache.LoadOrStore(key, rm)
	return actual.(*RowMeta)
}

// Created at 2026-06-28
// ClearCache очищает кэш (для тестов).
func ClearCache() {
	metaCache.Range(func(k, v any) bool {
		metaCache.Delete(k)
		return true
	})
}
