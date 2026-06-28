// Created at 2026-06-28
package meta

import (
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetOrBuildRowMeta_CacheHit(t *testing.T) {
	ClearCache()

	t1 := reflect.TypeOf(TestUser{})
	rm1 := GetOrBuildRowMeta(t1, "users")
	rm2 := GetOrBuildRowMeta(t1, "users")

	assert.Same(t, rm1, rm2, "should return same cached instance")
}

func TestGetOrBuildRowMeta_Concurrent(t *testing.T) {
	ClearCache()

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	results := make([]*RowMeta, goroutines)

	for i := range goroutines {
		go func(idx int) {
			defer wg.Done()
			t1 := reflect.TypeOf(TestUser{})
			results[idx] = GetOrBuildRowMeta(t1, "users")
		}(i)
	}

	wg.Wait()

	// все должны получить один и тот же экземпляр
	for i := 1; i < goroutines; i++ {
		assert.Same(t, results[0], results[i], "all goroutines should get same RowMeta")
	}
}

func TestGetOrBuildRowMeta_DifferentTypes(t *testing.T) {
	ClearCache()

	type Other struct {
		ID int `qqm:"pk=1"`
	}

	rm1 := GetOrBuildRowMeta(reflect.TypeOf(TestUser{}), "users")
	rm2 := GetOrBuildRowMeta(reflect.TypeOf(Other{}), "others")

	assert.NotSame(t, rm1, rm2, "different types should have different RowMeta")
}
