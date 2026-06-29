// Updated at 2026-06-29
package meta

import (
	"reflect"
	"sync"
)

// cacheKey — ключ кэша, включающий тип и имя таблицы.
type cacheKey struct {
	t reflect.Type
	n string
}

// metaCache — thread-safe кэш метаданных.
var (
	metaMu  sync.RWMutex
	metaMap = make(map[cacheKey]*RowMeta)
)

// Updated at 2026-06-29
// GetOrBuildRowMeta возвращает кэшированную RowMeta или строит новую.
func GetOrBuildRowMeta(t reflect.Type, tableName string) *RowMeta {
	key := cacheKey{t: t, n: tableName}
	for key.t.Kind() == reflect.Pointer {
		key.t = key.t.Elem()
	}

	metaMu.RLock()
	cached, ok := metaMap[key]
	metaMu.RUnlock()
	if ok {
		return cached
	}

	rm := BuildRowMeta(t, tableName)

	metaMu.Lock()
	if cached, ok := metaMap[key]; ok {
		metaMu.Unlock()
		return cached
	}
	metaMap[key] = rm
	metaMu.Unlock()
	return rm
}

// Updated at 2026-06-29
// ClearCache очищает кэш (для тестов).
func ClearCache() {
	metaMu.Lock()
	metaMap = make(map[cacheKey]*RowMeta)
	metaMu.Unlock()
}
