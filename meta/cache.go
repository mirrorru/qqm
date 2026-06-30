package meta

import (
	"reflect"
	"sync"
)

// cacheKey содержит ключ кэша — тип и имя таблицы.
// EN: cacheKey holds the cache key — type and table name.
type cacheKey struct {
	t reflect.Type
	n string
}

// metaMap — thread-safe кэш метаданных (RowMeta).
// EN: metaMap — thread-safe metadata cache (RowMeta).
var (
	metaMu  sync.RWMutex
	metaMap = make(map[cacheKey]*RowMeta)
)

// GetOrBuildRowMeta возвращает кэшированную RowMeta или строит новую.
// EN: GetOrBuildRowMeta returns cached RowMeta or builds a new one.
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

// ClearCache очищает кэш метаданных (для тестов).
// EN: ClearCache clears the metadata cache (for tests).
func ClearCache() {
	metaMu.Lock()
	metaMap = make(map[cacheKey]*RowMeta)
	metaMu.Unlock()
}
