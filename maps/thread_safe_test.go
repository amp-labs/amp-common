package maps_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/amp-labs/amp-common/hashing"
	"github.com/amp-labs/amp-common/maps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:dupl // Intentional duplication with thread_safe_ordered_test.go for parallel test coverage
func TestNewThreadSafeMap(t *testing.T) {
	t.Parallel()

	t.Run("wraps existing map", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, string](hashing.Sha256)
		tsm := maps.NewThreadSafeMap(m)
		require.NotNil(t, tsm)
		assert.Equal(t, 0, tsm.Size())
	})

	t.Run("wrapped map is usable immediately", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, int](hashing.Sha256)
		tsm := maps.NewThreadSafeMap(m)
		key := testKey{value: "test"}
		err := tsm.Add(key, 42)
		require.NoError(t, err)
		assert.Equal(t, 1, tsm.Size())
	})

	t.Run("returns nil when given nil map", func(t *testing.T) {
		t.Parallel()

		var m maps.Map[testKey, string]
		tsm := maps.NewThreadSafeMap(m)
		assert.Nil(t, tsm)
	})

	t.Run("returns existing thread-safe map as-is", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, string](hashing.Sha256)
		tsm1 := maps.NewThreadSafeMap(m)
		tsm2 := maps.NewThreadSafeMap(tsm1)

		// Should be the same instance, not double-wrapped
		assert.Equal(t, fmt.Sprintf("%p", tsm1), fmt.Sprintf("%p", tsm2))
	})
}

func TestThreadSafeMap_Add(t *testing.T) {
	t.Parallel()

	t.Run("adds new key-value pair", func(t *testing.T) {
		t.Parallel()

		m := maps.NewThreadSafeMap(maps.NewHashMap[testKey, string](hashing.Sha256))
		key := testKey{value: "test"}
		err := m.Add(key, "value")
		require.NoError(t, err)
		assert.Equal(t, 1, m.Size())
	})

	t.Run("updates existing key", func(t *testing.T) {
		t.Parallel()

		m := maps.NewThreadSafeMap(maps.NewHashMap[testKey, string](hashing.Sha256))
		key := testKey{value: "test"}
		err := m.Add(key, "value1")
		require.NoError(t, err)

		err = m.Add(key, "value2")
		require.NoError(t, err)
		assert.Equal(t, 1, m.Size())
	})

	t.Run("concurrent adds are safe", func(t *testing.T) {
		t.Parallel()

		threadSafeMap := maps.NewThreadSafeMap(maps.NewHashMap[testKey, int](hashing.Sha256))
		numGoroutines := 10
		addsPerGoroutine := 100

		var waitGroup sync.WaitGroup

		for goroutineIndex := range numGoroutines {
			waitGroup.Add(1)

			go func(offset int) {
				defer waitGroup.Done()

				for addIndex := range addsPerGoroutine {
					key := testKey{value: fmt.Sprintf("key-%d-%d", offset, addIndex)}
					err := threadSafeMap.Add(key, offset*1000+addIndex)
					assert.NoError(t, err)
				}
			}(goroutineIndex)
		}

		waitGroup.Wait()
		assert.Equal(t, numGoroutines*addsPerGoroutine, threadSafeMap.Size())
	})

	t.Run("concurrent updates to same key are safe", func(t *testing.T) {
		t.Parallel()

		threadSafeMap := maps.NewThreadSafeMap(maps.NewHashMap[testKey, int](hashing.Sha256))
		key := testKey{value: "shared"}
		numGoroutines := 100

		var waitGroup sync.WaitGroup

		for goroutineIndex := range numGoroutines {
			waitGroup.Add(1)

			go func(value int) {
				defer waitGroup.Done()

				err := threadSafeMap.Add(key, value)
				assert.NoError(t, err)
			}(goroutineIndex)
		}

		waitGroup.Wait()
		assert.Equal(t, 1, threadSafeMap.Size())
		contains, err := threadSafeMap.Contains(key)
		require.NoError(t, err)
		assert.True(t, contains)
	})
}

func TestThreadSafeMap_Remove(t *testing.T) {
	t.Parallel()

	t.Run("removes existing key", func(t *testing.T) {
		t.Parallel()

		m := maps.NewThreadSafeMap(maps.NewHashMap[testKey, string](hashing.Sha256))
		key := testKey{value: "test"}
		err := m.Add(key, "value")
		require.NoError(t, err)

		err = m.Remove(key)
		require.NoError(t, err)
		assert.Equal(t, 0, m.Size())

		contains, err := m.Contains(key)
		require.NoError(t, err)
		assert.False(t, contains)
	})

	t.Run("no-op for non-existent key", func(t *testing.T) {
		t.Parallel()

		m := maps.NewThreadSafeMap(maps.NewHashMap[testKey, string](hashing.Sha256))
		key := testKey{value: "missing"}
		err := m.Remove(key)
		require.NoError(t, err)
		assert.Equal(t, 0, m.Size())
	})

	t.Run("concurrent removes are safe", func(t *testing.T) {
		t.Parallel()

		threadSafeMap := maps.NewThreadSafeMap(maps.NewHashMap[testKey, int](hashing.Sha256))
		numKeys := 1000

		// Populate map
		for keyIndex := range numKeys {
			key := testKey{value: fmt.Sprintf("key-%d", keyIndex)}
			err := threadSafeMap.Add(key, keyIndex)
			require.NoError(t, err)
		}

		// Concurrent removal
		var waitGroup sync.WaitGroup
		for keyIndex := range numKeys {
			waitGroup.Add(1)

			go func(index int) {
				defer waitGroup.Done()

				key := testKey{value: fmt.Sprintf("key-%d", index)}
				err := threadSafeMap.Remove(key)
				assert.NoError(t, err)
			}(keyIndex)
		}

		waitGroup.Wait()
		assert.Equal(t, 0, threadSafeMap.Size())
	})

	t.Run("concurrent add and remove are safe", func(t *testing.T) {
		t.Parallel()

		threadSafeMap := maps.NewThreadSafeMap(maps.NewHashMap[testKey, int](hashing.Sha256))
		numOperations := 1000

		var waitGroup sync.WaitGroup

		// Half the goroutines add, half remove
		for opIndex := range numOperations {
			waitGroup.Add(1)

			go func(index int) {
				defer waitGroup.Done()

				key := testKey{value: fmt.Sprintf("key-%d", index%100)}
				if index%2 == 0 {
					_ = threadSafeMap.Add(key, index)
				} else {
					_ = threadSafeMap.Remove(key)
				}
			}(opIndex)
		}

		waitGroup.Wait()
		// Should complete without panics or race conditions
		_ = threadSafeMap.Size() // Just verify it's accessible
	})
}

func TestThreadSafeMap_Clear(t *testing.T) {
	t.Parallel()

	t.Run("removes all entries", func(t *testing.T) {
		t.Parallel()

		m := maps.NewThreadSafeMap(maps.NewHashMap[testKey, int](hashing.Sha256))

		for i := range 10 {
			key := testKey{value: fmt.Sprintf("key%d", i)}
			err := m.Add(key, i)
			require.NoError(t, err)
		}

		m.Clear()
		assert.Equal(t, 0, m.Size())
	})

	t.Run("map is usable after clear", func(t *testing.T) {
		t.Parallel()

		m := maps.NewThreadSafeMap(maps.NewHashMap[testKey, string](hashing.Sha256))
		key1 := testKey{value: "key1"}
		err := m.Add(key1, "value1")
		require.NoError(t, err)

		m.Clear()

		key2 := testKey{value: "key2"}
		err = m.Add(key2, "value2")
		require.NoError(t, err)
		assert.Equal(t, 1, m.Size())
	})

	t.Run("concurrent clear and add are safe", func(t *testing.T) {
		t.Parallel()

		threadSafeMap := maps.NewThreadSafeMap(maps.NewHashMap[testKey, int](hashing.Sha256))

		var waitGroup sync.WaitGroup

		// Add items in background
		for addIndex := range 100 {
			waitGroup.Add(1)

			go func(index int) {
				defer waitGroup.Done()

				key := testKey{value: fmt.Sprintf("key-%d", index)}
				_ = threadSafeMap.Add(key, index)
			}(addIndex)
		}

		// Clear concurrently
		waitGroup.Add(1)

		go func() {
			defer waitGroup.Done()
			time.Sleep(5 * time.Millisecond)
			threadSafeMap.Clear()
		}()

		waitGroup.Wait()
		// Should complete without panics
		_ = threadSafeMap.Size()
	})
}

func TestThreadSafeMap_Contains(t *testing.T) {
	t.Parallel()

	t.Run("returns true for existing key", func(t *testing.T) {
		t.Parallel()

		m := maps.NewThreadSafeMap(maps.NewHashMap[testKey, string](hashing.Sha256))
		key := testKey{value: "test"}
		err := m.Add(key, "value")
		require.NoError(t, err)

		contains, err := m.Contains(key)
		require.NoError(t, err)
		assert.True(t, contains)
	})

	t.Run("returns false for non-existent key", func(t *testing.T) {
		t.Parallel()

		m := maps.NewThreadSafeMap(maps.NewHashMap[testKey, string](hashing.Sha256))
		key := testKey{value: "missing"}

		contains, err := m.Contains(key)
		require.NoError(t, err)
		assert.False(t, contains)
	})

	t.Run("concurrent reads are safe", func(t *testing.T) {
		t.Parallel()

		threadSafeMap := maps.NewThreadSafeMap(maps.NewHashMap[testKey, int](hashing.Sha256))
		numKeys := 100

		// Populate map
		for keyIndex := range numKeys {
			key := testKey{value: fmt.Sprintf("key-%d", keyIndex)}
			err := threadSafeMap.Add(key, keyIndex)
			require.NoError(t, err)
		}

		// Concurrent reads
		numReaders := 50

		var waitGroup sync.WaitGroup

		for range numReaders {
			waitGroup.Add(1)

			go func() {
				defer waitGroup.Done()

				for keyIdx := range numKeys {
					key := testKey{value: fmt.Sprintf("key-%d", keyIdx)}
					contains, err := threadSafeMap.Contains(key)
					assert.NoError(t, err)
					assert.True(t, contains)
				}
			}()
		}

		waitGroup.Wait()
	})

	t.Run("concurrent reads and writes are safe", func(t *testing.T) {
		t.Parallel()

		threadSafeMap := maps.NewThreadSafeMap(maps.NewHashMap[testKey, int](hashing.Sha256))
		numOperations := 1000

		var waitGroup sync.WaitGroup

		for opIndex := range numOperations {
			waitGroup.Add(1)

			go func(index int) {
				defer waitGroup.Done()

				key := testKey{value: fmt.Sprintf("key-%d", index%100)}
				if index%2 == 0 {
					_ = threadSafeMap.Add(key, index)
				} else {
					_, _ = threadSafeMap.Contains(key)
				}
			}(opIndex)
		}

		waitGroup.Wait()
	})
}

func TestThreadSafeMap_Size(t *testing.T) {
	t.Parallel()

	t.Run("returns zero for empty map", func(t *testing.T) {
		t.Parallel()

		m := maps.NewThreadSafeMap(maps.NewHashMap[testKey, string](hashing.Sha256))
		assert.Equal(t, 0, m.Size())
	})

	t.Run("returns correct size after additions", func(t *testing.T) {
		t.Parallel()

		m := maps.NewThreadSafeMap(maps.NewHashMap[testKey, string](hashing.Sha256))
		assert.Equal(t, 0, m.Size())

		key1 := testKey{value: "key1"}
		err := m.Add(key1, "value1")
		require.NoError(t, err)
		assert.Equal(t, 1, m.Size())

		key2 := testKey{value: "key2"}
		err = m.Add(key2, "value2")
		require.NoError(t, err)
		assert.Equal(t, 2, m.Size())
	})

	t.Run("concurrent size checks are safe", func(t *testing.T) {
		t.Parallel()

		threadSafeMap := maps.NewThreadSafeMap(maps.NewHashMap[testKey, int](hashing.Sha256))

		var waitGroup sync.WaitGroup

		// Add items in background
		for addIndex := range 100 {
			waitGroup.Add(1)

			go func(index int) {
				defer waitGroup.Done()

				key := testKey{value: fmt.Sprintf("key-%d", index)}
				_ = threadSafeMap.Add(key, index)
			}(addIndex)
		}

		// Check size concurrently
		for range 50 {
			waitGroup.Add(1)

			go func() {
				defer waitGroup.Done()

				_ = threadSafeMap.Size()
			}()
		}

		waitGroup.Wait()
		assert.Equal(t, 100, threadSafeMap.Size())
	})
}

func TestThreadSafeMap_Seq(t *testing.T) {
	t.Parallel()

	t.Run("iterates over all entries", func(t *testing.T) {
		t.Parallel()

		m := maps.NewThreadSafeMap(maps.NewHashMap[testKey, int](hashing.Sha256))
		expected := map[string]int{
			"key1": 1,
			"key2": 2,
			"key3": 3,
		}

		for k, v := range expected {
			key := testKey{value: k}
			err := m.Add(key, v)
			require.NoError(t, err)
		}

		visited := make(map[string]int)
		for key, value := range m.Seq() {
			visited[key.value] = value
		}

		assert.Equal(t, expected, visited)
	})

	t.Run("handles empty map", func(t *testing.T) {
		t.Parallel()

		m := maps.NewThreadSafeMap(maps.NewHashMap[testKey, string](hashing.Sha256))
		count := 0

		for range m.Seq() {
			count++
		}

		assert.Equal(t, 0, count)
	})

	t.Run("stops early when yield returns false", func(t *testing.T) {
		t.Parallel()

		m := maps.NewThreadSafeMap(maps.NewHashMap[testKey, int](hashing.Sha256))

		for i := range 10 {
			key := testKey{value: fmt.Sprintf("key%d", i)}
			err := m.Add(key, i)
			require.NoError(t, err)
		}

		count := 0
		for range m.Seq() {
			count++
			if count >= 5 {
				break
			}
		}

		assert.Equal(t, 5, count)
	})

	t.Run("iteration sees snapshot of map at call time", func(t *testing.T) {
		t.Parallel()

		threadSafeMap := maps.NewThreadSafeMap(maps.NewHashMap[testKey, int](hashing.Sha256))

		// Add initial entries
		for keyIndex := range 5 {
			key := testKey{value: fmt.Sprintf("key%d", keyIndex)}
			err := threadSafeMap.Add(key, keyIndex)
			require.NoError(t, err)
		}

		// Get iterator
		seq := threadSafeMap.Seq()

		// Modify map after getting iterator
		for keyIndex := 5; keyIndex < 10; keyIndex++ {
			key := testKey{value: fmt.Sprintf("key%d", keyIndex)}
			_ = threadSafeMap.Add(key, keyIndex)
		}

		// Iterator should only see first 5 entries
		count := 0
		for range seq {
			count++
		}

		assert.Equal(t, 5, count)
		assert.Equal(t, 10, threadSafeMap.Size()) // But map has 10 entries
	})

	t.Run("concurrent iteration and modification are safe", func(t *testing.T) {
		t.Parallel()

		threadSafeMap := maps.NewThreadSafeMap(maps.NewHashMap[testKey, int](hashing.Sha256))

		// Populate map
		for keyIndex := range 100 {
			key := testKey{value: fmt.Sprintf("key-%d", keyIndex)}
			err := threadSafeMap.Add(key, keyIndex)
			require.NoError(t, err)
		}

		var waitGroup sync.WaitGroup

		// Multiple concurrent iterators
		for range 10 {
			waitGroup.Add(1)

			go func() {
				defer waitGroup.Done()

				count := 0
				for range threadSafeMap.Seq() {
					count++
				}

				assert.Positive(t, count)
			}()
		}

		// Modify map while iterating
		for modifyIndex := range 50 {
			waitGroup.Add(1)

			go func(index int) {
				defer waitGroup.Done()

				key := testKey{value: fmt.Sprintf("new-key-%d", index)}
				_ = threadSafeMap.Add(key, index)
			}(modifyIndex)
		}

		waitGroup.Wait()
	})

	t.Run("multiple concurrent iterators don't block each other", func(t *testing.T) {
		t.Parallel()

		threadSafeMap := maps.NewThreadSafeMap(maps.NewHashMap[testKey, int](hashing.Sha256))

		// Large map for slower iteration
		for keyIndex := range 1000 {
			key := testKey{value: fmt.Sprintf("key-%d", keyIndex)}
			err := threadSafeMap.Add(key, keyIndex)
			require.NoError(t, err)
		}

		start := time.Now()

		var waitGroup sync.WaitGroup

		// Multiple concurrent iterators
		numIterators := 10
		for range numIterators {
			waitGroup.Add(1)

			go func() {
				defer waitGroup.Done()

				for range threadSafeMap.Seq() {
					// Simulate slow iteration
					time.Sleep(10 * time.Microsecond)
				}
			}()
		}

		waitGroup.Wait()

		elapsed := time.Since(start)

		// If iterators were blocking each other, this would take much longer
		// With concurrent iteration (snapshot approach), they should run in parallel
		// This is a rough check - exact timing depends on system
		assert.Less(t, elapsed, 5*time.Second, "iterators appear to be blocking each other")
	})
}

//nolint:dupl // Intentional duplication with thread_safe_ordered_test.go for parallel test coverage
func TestThreadSafeMap_Union(t *testing.T) {
	t.Parallel()

	t.Run("combines two maps", func(t *testing.T) {
		t.Parallel()

		m1 := maps.NewThreadSafeMap(maps.NewHashMap[testKey, string](hashing.Sha256))
		_ = m1.Add(testKey{value: "key1"}, "value1") //nolint:errcheck
		_ = m1.Add(testKey{value: "key2"}, "value2") //nolint:errcheck

		m2 := maps.NewHashMap[testKey, string](hashing.Sha256)
		_ = m2.Add(testKey{value: "key3"}, "value3") //nolint:errcheck
		_ = m2.Add(testKey{value: "key4"}, "value4") //nolint:errcheck

		result, err := m1.Union(m2)
		require.NoError(t, err)
		assert.Equal(t, 4, result.Size())
	})

	t.Run("result is also thread-safe", func(t *testing.T) {
		t.Parallel()

		m1 := maps.NewThreadSafeMap(maps.NewHashMap[testKey, string](hashing.Sha256))
		_ = m1.Add(testKey{value: "key1"}, "value1") //nolint:errcheck

		m2 := maps.NewHashMap[testKey, string](hashing.Sha256)
		_ = m2.Add(testKey{value: "key2"}, "value2") //nolint:errcheck

		result, err := m1.Union(m2)
		require.NoError(t, err)

		// Verify result is thread-safe by doing concurrent operations
		var wg sync.WaitGroup
		for i := range 10 {
			wg.Add(1)

			go func(index int) {
				defer wg.Done()

				key := testKey{value: fmt.Sprintf("new-%d", index)}
				_ = result.Add(key, fmt.Sprintf("value-%d", index))
			}(i)
		}

		wg.Wait()

		assert.Equal(t, 12, result.Size())
	})

	t.Run("original maps are not modified", func(t *testing.T) {
		t.Parallel()

		m1 := maps.NewThreadSafeMap(maps.NewHashMap[testKey, string](hashing.Sha256))
		_ = m1.Add(testKey{value: "key1"}, "value1") //nolint:errcheck

		m2 := maps.NewHashMap[testKey, string](hashing.Sha256)
		_ = m2.Add(testKey{value: "key2"}, "value2") //nolint:errcheck

		result, err := m1.Union(m2)
		require.NoError(t, err)
		assert.Equal(t, 2, result.Size())
		assert.Equal(t, 1, m1.Size())
		assert.Equal(t, 1, m2.Size())
	})

	t.Run("concurrent union operations are safe", func(t *testing.T) {
		t.Parallel()

		firstMap := maps.NewThreadSafeMap(maps.NewHashMap[testKey, int](hashing.Sha256))

		for keyIndex := range 50 {
			key := testKey{value: fmt.Sprintf("key-%d", keyIndex)}
			_ = firstMap.Add(key, keyIndex)
		}

		numOperations := 20

		var waitGroup sync.WaitGroup

		for opIndex := range numOperations {
			waitGroup.Add(1)

			go func(index int) {
				defer waitGroup.Done()

				secondMap := maps.NewHashMap[testKey, int](hashing.Sha256)
				key := testKey{value: fmt.Sprintf("union-key-%d", index)}
				_ = secondMap.Add(key, index)
				_, _ = firstMap.Union(secondMap)
			}(opIndex)
		}

		waitGroup.Wait()
		assert.Equal(t, 50, firstMap.Size()) // Original unchanged
	})
}

func TestThreadSafeMap_Intersection(t *testing.T) {
	t.Parallel()

	t.Run("returns common keys", func(t *testing.T) {
		t.Parallel()

		m1 := maps.NewThreadSafeMap(maps.NewHashMap[testKey, string](hashing.Sha256))
		_ = m1.Add(testKey{value: "key1"}, "value1") //nolint:errcheck
		_ = m1.Add(testKey{value: "key2"}, "value2") //nolint:errcheck
		_ = m1.Add(testKey{value: "key3"}, "value3") //nolint:errcheck

		m2 := maps.NewHashMap[testKey, string](hashing.Sha256)
		_ = m2.Add(testKey{value: "key2"}, "other2") //nolint:errcheck
		_ = m2.Add(testKey{value: "key3"}, "other3") //nolint:errcheck
		_ = m2.Add(testKey{value: "key4"}, "other4") //nolint:errcheck

		result, err := m1.Intersection(m2)
		require.NoError(t, err)
		assert.Equal(t, 2, result.Size())

		contains, err := result.Contains(testKey{value: "key2"})
		require.NoError(t, err)
		assert.True(t, contains)

		contains, err = result.Contains(testKey{value: "key3"})
		require.NoError(t, err)
		assert.True(t, contains)
	})

	t.Run("result is also thread-safe", func(t *testing.T) {
		t.Parallel()

		m1 := maps.NewThreadSafeMap(maps.NewHashMap[testKey, string](hashing.Sha256))
		_ = m1.Add(testKey{value: "key1"}, "value1") //nolint:errcheck

		m2 := maps.NewHashMap[testKey, string](hashing.Sha256)
		_ = m2.Add(testKey{value: "key1"}, "value2") //nolint:errcheck

		result, err := m1.Intersection(m2)
		require.NoError(t, err)

		// Verify result is thread-safe
		var wg sync.WaitGroup
		for range 10 {
			wg.Add(1)

			go func() {
				defer wg.Done()

				contains, _ := result.Contains(testKey{value: "key1"})
				assert.True(t, contains)
			}()
		}

		wg.Wait()
	})

	t.Run("original maps are not modified", func(t *testing.T) {
		t.Parallel()

		m1 := maps.NewThreadSafeMap(maps.NewHashMap[testKey, string](hashing.Sha256))
		_ = m1.Add(testKey{value: "key1"}, "value1") //nolint:errcheck
		_ = m1.Add(testKey{value: "key2"}, "value2") //nolint:errcheck

		m2 := maps.NewHashMap[testKey, string](hashing.Sha256)
		_ = m2.Add(testKey{value: "key2"}, "value2") //nolint:errcheck

		result, err := m1.Intersection(m2)
		require.NoError(t, err)
		assert.Equal(t, 1, result.Size())
		assert.Equal(t, 2, m1.Size())
		assert.Equal(t, 1, m2.Size())
	})
}

func TestThreadSafeMap_Clone(t *testing.T) {
	t.Parallel()

	t.Run("creates independent copy", func(t *testing.T) {
		t.Parallel()

		original := maps.NewThreadSafeMap(maps.NewHashMap[testKey, string](hashing.Sha256))
		_ = original.Add(testKey{value: "key1"}, "value1") //nolint:errcheck
		_ = original.Add(testKey{value: "key2"}, "value2") //nolint:errcheck

		cloned := original.Clone()
		assert.Equal(t, original.Size(), cloned.Size())

		// Modify original
		_ = original.Add(testKey{value: "key3"}, "value3") //nolint:errcheck

		// Clone should not be affected
		assert.Equal(t, 3, original.Size())
		assert.Equal(t, 2, cloned.Size())
	})

	t.Run("cloned map is also thread-safe", func(t *testing.T) {
		t.Parallel()

		original := maps.NewThreadSafeMap(maps.NewHashMap[testKey, int](hashing.Sha256))

		for i := range 10 {
			key := testKey{value: fmt.Sprintf("key%d", i)}
			_ = original.Add(key, i)
		}

		cloned := original.Clone()

		// Verify clone is thread-safe with concurrent operations
		var wg sync.WaitGroup
		for i := range 20 {
			wg.Add(1)

			go func(index int) {
				defer wg.Done()

				key := testKey{value: fmt.Sprintf("new-%d", index)}
				_ = cloned.Add(key, index)
			}(i)
		}

		wg.Wait()

		assert.Equal(t, 30, cloned.Size())
		assert.Equal(t, 10, original.Size())
	})

	t.Run("concurrent cloning is safe", func(t *testing.T) {
		t.Parallel()

		original := maps.NewThreadSafeMap(maps.NewHashMap[testKey, int](hashing.Sha256))

		for keyIndex := range 100 {
			key := testKey{value: fmt.Sprintf("key-%d", keyIndex)}
			_ = original.Add(key, keyIndex)
		}

		clones := make([]maps.Map[testKey, int], 10)

		var waitGroup sync.WaitGroup

		// Create multiple clones concurrently
		for cloneIndex := range 10 {
			waitGroup.Add(1)

			go func(index int) {
				defer waitGroup.Done()

				clones[index] = original.Clone()
			}(cloneIndex)
		}

		waitGroup.Wait()

		// Verify all clones
		for _, clone := range clones {
			assert.Equal(t, 100, clone.Size())
		}
	})
}

// TestThreadSafeMap_RaceConditions uses go test -race to detect race conditions.
func TestThreadSafeMap_RaceConditions(t *testing.T) {
	t.Parallel()

	t.Run("stress test with mixed operations", func(t *testing.T) {
		t.Parallel()

		threadSafeMap := maps.NewThreadSafeMap(maps.NewHashMap[testKey, int](hashing.Sha256))
		numGoroutines := 50
		operationsPerGoroutine := 100

		var waitGroup sync.WaitGroup

		for goroutineID := range numGoroutines {
			waitGroup.Add(1)

			go func(id int) {
				defer waitGroup.Done()

				for opIndex := range operationsPerGoroutine {
					key := testKey{value: fmt.Sprintf("key-%d-%d", id, opIndex%10)}

					switch opIndex % 7 {
					case 0:
						_ = threadSafeMap.Add(key, opIndex)
					case 1:
						_ = threadSafeMap.Remove(key)
					case 2:
						_, _ = threadSafeMap.Contains(key)
					case 3:
						_ = threadSafeMap.Size()
					case 4:
						for range threadSafeMap.Seq() {
							break // Just get one element
						}
					case 5:
						_ = threadSafeMap.Clone()
					case 6:
						if opIndex%20 == 0 {
							threadSafeMap.Clear()
						}
					}
				}
			}(goroutineID)
		}

		waitGroup.Wait()
	})
}

func TestThreadSafeMap_Get(t *testing.T) {
	t.Parallel()

	t.Run("returns value for existing key", func(t *testing.T) {
		t.Parallel()

		base := maps.NewHashMap[testKey, string](hashing.Sha256)
		m := maps.NewThreadSafeMap(base)
		key := testKey{value: "test"}
		err := m.Add(key, "expected")
		require.NoError(t, err)

		value, found, err := m.Get(key)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "expected", value)
	})

	t.Run("returns zero value and false for missing key", func(t *testing.T) {
		t.Parallel()

		base := maps.NewHashMap[testKey, string](hashing.Sha256)
		m := maps.NewThreadSafeMap(base)
		key := testKey{value: "missing"}

		value, found, err := m.Get(key)
		require.NoError(t, err)
		assert.False(t, found)
		assert.Equal(t, "", value)
	})

	t.Run("returns most recent value for updated key", func(t *testing.T) {
		t.Parallel()

		base := maps.NewHashMap[testKey, string](hashing.Sha256)
		m := maps.NewThreadSafeMap(base)
		key := testKey{value: "test"}

		err := m.Add(key, "first")
		require.NoError(t, err)

		err = m.Add(key, "second")
		require.NoError(t, err)

		value, found, err := m.Get(key)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "second", value)
	})

	t.Run("handles multiple keys correctly", func(t *testing.T) {
		t.Parallel()

		base := maps.NewHashMap[testKey, int](hashing.Sha256)
		m := maps.NewThreadSafeMap(base)
		expected := map[string]int{
			"key1": 10,
			"key2": 20,
			"key3": 30,
		}

		for k, v := range expected {
			err := m.Add(testKey{value: k}, v)
			require.NoError(t, err)
		}

		for k, expectedValue := range expected {
			value, found, err := m.Get(testKey{value: k})
			require.NoError(t, err)
			assert.True(t, found)
			assert.Equal(t, expectedValue, value)
		}
	})

	t.Run("concurrent reads are safe", func(t *testing.T) {
		t.Parallel()

		base := maps.NewHashMap[testKey, int](hashing.Sha256)
		threadSafeMap := maps.NewThreadSafeMap(base)

		// Populate map
		for i := range 100 {
			err := threadSafeMap.Add(testKey{value: fmt.Sprintf("key%d", i)}, i)
			require.NoError(t, err)
		}

		// Multiple goroutines reading concurrently
		var waitGroup sync.WaitGroup
		for range 10 {
			waitGroup.Add(1)

			go func() {
				defer waitGroup.Done()

				for i := range 100 {
					value, found, err := threadSafeMap.Get(testKey{value: fmt.Sprintf("key%d", i)})
					assert.NoError(t, err)
					assert.True(t, found)
					assert.Equal(t, i, value)
				}
			}()
		}

		waitGroup.Wait()
	})

	//nolint:dupl // Test structure mirrors thread_safe_ordered_test.go for consistency
	t.Run("concurrent read and write are safe", func(t *testing.T) {
		t.Parallel()

		base := maps.NewHashMap[testKey, int](hashing.Sha256)
		threadSafeMap := maps.NewThreadSafeMap(base)

		// Initialize with some data
		for i := range 50 {
			err := threadSafeMap.Add(testKey{value: fmt.Sprintf("key%d", i)}, i)
			require.NoError(t, err)
		}

		var waitGroup sync.WaitGroup

		// Writer goroutines
		for range 5 {
			waitGroup.Add(1)

			go func() {
				defer waitGroup.Done()

				for i := range 20 {
					key := testKey{value: fmt.Sprintf("key%d", i)}
					_ = threadSafeMap.Add(key, i*100)
				}
			}()
		}

		// Reader goroutines
		for range 10 {
			waitGroup.Add(1)

			go func() {
				defer waitGroup.Done()

				for i := range 50 {
					key := testKey{value: fmt.Sprintf("key%d", i)}
					_, _, _ = threadSafeMap.Get(key)
				}
			}()
		}

		waitGroup.Wait()
	})

	t.Run("returns false after key removal", func(t *testing.T) {
		t.Parallel()

		base := maps.NewHashMap[testKey, string](hashing.Sha256)
		m := maps.NewThreadSafeMap(base)
		key := testKey{value: "test"}

		err := m.Add(key, "value")
		require.NoError(t, err)

		err = m.Remove(key)
		require.NoError(t, err)

		value, found, err := m.Get(key)
		require.NoError(t, err)
		assert.False(t, found)
		assert.Equal(t, "", value)
	})

	t.Run("returns false after clear", func(t *testing.T) {
		t.Parallel()

		base := maps.NewHashMap[testKey, string](hashing.Sha256)
		m := maps.NewThreadSafeMap(base)
		key := testKey{value: "test"}

		err := m.Add(key, "value")
		require.NoError(t, err)

		m.Clear()

		value, found, err := m.Get(key)
		require.NoError(t, err)
		assert.False(t, found)
		assert.Equal(t, "", value)
	})

	t.Run("handles nil/empty values correctly", func(t *testing.T) {
		t.Parallel()

		base := maps.NewHashMap[testKey, *string](hashing.Sha256)
		m := maps.NewThreadSafeMap(base)
		key := testKey{value: "test"}

		err := m.Add(key, nil)
		require.NoError(t, err)

		value, found, err := m.Get(key)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Nil(t, value)
	})
}
