package set_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/amp-labs/amp-common/hashing"
	"github.com/amp-labs/amp-common/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:dupl // Intentional duplication with thread_safe_ordered_test.go for parallel test coverage
func TestNewThreadSafeSet(t *testing.T) {
	t.Parallel()

	t.Run("wraps existing set", func(t *testing.T) {
		t.Parallel()

		s := set.NewSet[hashing.HashableString](hashing.Sha256)
		tss := set.NewThreadSafeSet(s)
		require.NotNil(t, tss)
		assert.Equal(t, 0, tss.Size())
	})

	t.Run("wrapped set is usable immediately", func(t *testing.T) {
		t.Parallel()

		s := set.NewSet[hashing.HashableString](hashing.Sha256)
		tss := set.NewThreadSafeSet(s)
		err := tss.Add(hashing.HashableString("test"))
		require.NoError(t, err)
		assert.Equal(t, 1, tss.Size())
	})

	t.Run("returns nil when given nil set", func(t *testing.T) {
		t.Parallel()

		var s set.Set[hashing.HashableString]

		tss := set.NewThreadSafeSet(s)
		assert.Nil(t, tss)
	})

	t.Run("returns existing thread-safe set as-is", func(t *testing.T) {
		t.Parallel()

		s := set.NewSet[hashing.HashableString](hashing.Sha256)
		tss1 := set.NewThreadSafeSet(s)
		tss2 := set.NewThreadSafeSet(tss1)

		// Should be the same instance, not double-wrapped
		assert.Equal(t, fmt.Sprintf("%p", tss1), fmt.Sprintf("%p", tss2))
	})
}

//nolint:dupl // Intentional duplication with thread_safe_ordered_test.go for parallel test coverage
func TestThreadSafeSet_Add(t *testing.T) {
	t.Parallel()

	t.Run("adds new element", func(t *testing.T) {
		t.Parallel()

		s := set.NewThreadSafeSet(set.NewSet[hashing.HashableString](hashing.Sha256))
		err := s.Add(hashing.HashableString("test"))
		require.NoError(t, err)
		assert.Equal(t, 1, s.Size())
	})

	t.Run("no error for duplicate element", func(t *testing.T) {
		t.Parallel()

		s := set.NewThreadSafeSet(set.NewSet[hashing.HashableString](hashing.Sha256))
		err := s.Add(hashing.HashableString("test"))
		require.NoError(t, err)

		err = s.Add(hashing.HashableString("test"))
		require.NoError(t, err)
		assert.Equal(t, 1, s.Size())
	})

	t.Run("concurrent adds are safe", func(t *testing.T) {
		t.Parallel()

		threadSafeSet := set.NewThreadSafeSet(set.NewSet[hashing.HashableString](hashing.Sha256))
		numGoroutines := 10
		addsPerGoroutine := 100

		var waitGroup sync.WaitGroup

		for goroutineIndex := range numGoroutines {
			waitGroup.Add(1)

			go func(offset int) {
				defer waitGroup.Done()

				for addIndex := range addsPerGoroutine {
					element := hashing.HashableString(fmt.Sprintf("element-%d-%d", offset, addIndex))
					err := threadSafeSet.Add(element)
					assert.NoError(t, err)
				}
			}(goroutineIndex)
		}

		waitGroup.Wait()
		assert.Equal(t, numGoroutines*addsPerGoroutine, threadSafeSet.Size())
	})

	t.Run("concurrent adds to same element are safe", func(t *testing.T) {
		t.Parallel()

		threadSafeSet := set.NewThreadSafeSet(set.NewSet[hashing.HashableString](hashing.Sha256))
		element := hashing.HashableString("shared")
		numGoroutines := 100

		var waitGroup sync.WaitGroup

		for range numGoroutines {
			waitGroup.Go(func() {
				err := threadSafeSet.Add(element)
				assert.NoError(t, err)
			})
		}

		waitGroup.Wait()
		assert.Equal(t, 1, threadSafeSet.Size())
		contains, err := threadSafeSet.Contains(element)
		require.NoError(t, err)
		assert.True(t, contains)
	})
}

//nolint:dupl // Intentional duplication with thread_safe_ordered_test.go for parallel test coverage
func TestThreadSafeSet_AddAll(t *testing.T) {
	t.Parallel()

	t.Run("adds multiple elements", func(t *testing.T) {
		t.Parallel()

		s := set.NewThreadSafeSet(set.NewSet[hashing.HashableString](hashing.Sha256))
		err := s.AddAll(
			hashing.HashableString("elem1"),
			hashing.HashableString("elem2"),
			hashing.HashableString("elem3"),
		)
		require.NoError(t, err)
		assert.Equal(t, 3, s.Size())
	})

	t.Run("concurrent AddAll operations are safe", func(t *testing.T) {
		t.Parallel()

		threadSafeSet := set.NewThreadSafeSet(set.NewSet[hashing.HashableString](hashing.Sha256))
		numGoroutines := 10

		var waitGroup sync.WaitGroup

		for goroutineIndex := range numGoroutines {
			waitGroup.Add(1)

			go func(offset int) {
				defer waitGroup.Done()

				elements := make([]hashing.HashableString, 10)
				for i := range elements {
					elements[i] = hashing.HashableString(fmt.Sprintf("elem-%d-%d", offset, i))
				}

				err := threadSafeSet.AddAll(elements...)
				assert.NoError(t, err)
			}(goroutineIndex)
		}

		waitGroup.Wait()
		assert.Equal(t, numGoroutines*10, threadSafeSet.Size())
	})
}

//nolint:dupl // Intentional duplication with thread_safe_ordered_test.go for parallel test coverage
func TestThreadSafeSet_Remove(t *testing.T) {
	t.Parallel()

	t.Run("removes existing element", func(t *testing.T) {
		t.Parallel()

		s := set.NewThreadSafeSet(set.NewSet[hashing.HashableString](hashing.Sha256))
		element := hashing.HashableString("test")
		err := s.Add(element)
		require.NoError(t, err)

		err = s.Remove(element)
		require.NoError(t, err)
		assert.Equal(t, 0, s.Size())

		contains, err := s.Contains(element)
		require.NoError(t, err)
		assert.False(t, contains)
	})

	t.Run("no-op for non-existent element", func(t *testing.T) {
		t.Parallel()

		s := set.NewThreadSafeSet(set.NewSet[hashing.HashableString](hashing.Sha256))
		element := hashing.HashableString("missing")
		err := s.Remove(element)
		require.NoError(t, err)
		assert.Equal(t, 0, s.Size())
	})

	t.Run("concurrent removes are safe", func(t *testing.T) {
		t.Parallel()

		threadSafeSet := set.NewThreadSafeSet(set.NewSet[hashing.HashableString](hashing.Sha256))
		numElements := 1000

		// Populate set
		for elemIndex := range numElements {
			element := hashing.HashableString(fmt.Sprintf("elem-%d", elemIndex))
			err := threadSafeSet.Add(element)
			require.NoError(t, err)
		}

		// Concurrent removal
		var waitGroup sync.WaitGroup
		for elemIndex := range numElements {
			waitGroup.Add(1)

			go func(index int) {
				defer waitGroup.Done()

				element := hashing.HashableString(fmt.Sprintf("elem-%d", index))
				err := threadSafeSet.Remove(element)
				assert.NoError(t, err)
			}(elemIndex)
		}

		waitGroup.Wait()
		assert.Equal(t, 0, threadSafeSet.Size())
	})

	t.Run("concurrent add and remove are safe", func(t *testing.T) {
		t.Parallel()

		threadSafeSet := set.NewThreadSafeSet(set.NewSet[hashing.HashableString](hashing.Sha256))
		numOperations := 1000

		var waitGroup sync.WaitGroup

		// Half the goroutines add, half remove
		for opIndex := range numOperations {
			waitGroup.Add(1)

			go func(index int) {
				defer waitGroup.Done()

				element := hashing.HashableString(fmt.Sprintf("elem-%d", index%100))
				if index%2 == 0 {
					_ = threadSafeSet.Add(element)
				} else {
					_ = threadSafeSet.Remove(element)
				}
			}(opIndex)
		}

		waitGroup.Wait()
		// Should complete without panics or race conditions
		_ = threadSafeSet.Size() // Just verify it's accessible
	})
}

//nolint:dupl // Intentional duplication with thread_safe_ordered_test.go for parallel test coverage
func TestThreadSafeSet_Clear(t *testing.T) {
	t.Parallel()

	t.Run("removes all entries", func(t *testing.T) {
		t.Parallel()

		s := set.NewThreadSafeSet(set.NewSet[hashing.HashableString](hashing.Sha256))

		for i := range 10 {
			element := hashing.HashableString(fmt.Sprintf("elem%d", i))
			err := s.Add(element)
			require.NoError(t, err)
		}

		s.Clear()
		assert.Equal(t, 0, s.Size())
	})

	t.Run("set is usable after clear", func(t *testing.T) {
		t.Parallel()

		s := set.NewThreadSafeSet(set.NewSet[hashing.HashableString](hashing.Sha256))
		elem1 := hashing.HashableString("elem1")
		err := s.Add(elem1)
		require.NoError(t, err)

		s.Clear()

		elem2 := hashing.HashableString("elem2")
		err = s.Add(elem2)
		require.NoError(t, err)
		assert.Equal(t, 1, s.Size())
	})

	t.Run("concurrent clear and add are safe", func(t *testing.T) {
		t.Parallel()

		threadSafeSet := set.NewThreadSafeSet(set.NewSet[hashing.HashableString](hashing.Sha256))

		var waitGroup sync.WaitGroup

		// Add items in background
		for addIndex := range 100 {
			waitGroup.Add(1)

			go func(index int) {
				defer waitGroup.Done()

				element := hashing.HashableString(fmt.Sprintf("elem-%d", index))
				_ = threadSafeSet.Add(element)
			}(addIndex)
		}

		// Clear concurrently

		waitGroup.Go(func() {
			time.Sleep(5 * time.Millisecond)
			threadSafeSet.Clear()
		})

		waitGroup.Wait()
		// Should complete without panics
		_ = threadSafeSet.Size()
	})
}

//nolint:dupl // Intentional duplication with thread_safe_ordered_test.go for parallel test coverage
func TestThreadSafeSet_Contains(t *testing.T) {
	t.Parallel()

	t.Run("returns true for existing element", func(t *testing.T) {
		t.Parallel()

		s := set.NewThreadSafeSet(set.NewSet[hashing.HashableString](hashing.Sha256))
		element := hashing.HashableString("test")
		err := s.Add(element)
		require.NoError(t, err)

		contains, err := s.Contains(element)
		require.NoError(t, err)
		assert.True(t, contains)
	})

	t.Run("returns false for non-existent element", func(t *testing.T) {
		t.Parallel()

		s := set.NewThreadSafeSet(set.NewSet[hashing.HashableString](hashing.Sha256))
		element := hashing.HashableString("missing")

		contains, err := s.Contains(element)
		require.NoError(t, err)
		assert.False(t, contains)
	})

	t.Run("concurrent reads are safe", func(t *testing.T) {
		t.Parallel()

		threadSafeSet := set.NewThreadSafeSet(set.NewSet[hashing.HashableString](hashing.Sha256))
		numElements := 100

		// Populate set
		for elemIndex := range numElements {
			element := hashing.HashableString(fmt.Sprintf("elem-%d", elemIndex))
			err := threadSafeSet.Add(element)
			require.NoError(t, err)
		}

		// Concurrent reads
		numReaders := 50

		var waitGroup sync.WaitGroup

		for range numReaders {
			waitGroup.Go(func() {
				for elemIdx := range numElements {
					element := hashing.HashableString(fmt.Sprintf("elem-%d", elemIdx))
					contains, err := threadSafeSet.Contains(element)
					require.NoError(t, err)
					assert.True(t, contains)
				}
			})
		}

		waitGroup.Wait()
	})

	t.Run("concurrent reads and writes are safe", func(t *testing.T) {
		t.Parallel()

		threadSafeSet := set.NewThreadSafeSet(set.NewSet[hashing.HashableString](hashing.Sha256))
		numOperations := 1000

		var waitGroup sync.WaitGroup

		for opIndex := range numOperations {
			waitGroup.Add(1)

			go func(index int) {
				defer waitGroup.Done()

				element := hashing.HashableString(fmt.Sprintf("elem-%d", index%100))
				if index%2 == 0 {
					_ = threadSafeSet.Add(element)
				} else {
					_, _ = threadSafeSet.Contains(element)
				}
			}(opIndex)
		}

		waitGroup.Wait()
	})
}

//nolint:dupl // Intentional duplication with thread_safe_ordered_test.go for parallel test coverage
func TestThreadSafeSet_Size(t *testing.T) {
	t.Parallel()

	t.Run("returns zero for empty set", func(t *testing.T) {
		t.Parallel()

		s := set.NewThreadSafeSet(set.NewSet[hashing.HashableString](hashing.Sha256))
		assert.Equal(t, 0, s.Size())
	})

	t.Run("returns correct size after additions", func(t *testing.T) {
		t.Parallel()

		s := set.NewThreadSafeSet(set.NewSet[hashing.HashableString](hashing.Sha256))
		assert.Equal(t, 0, s.Size())

		elem1 := hashing.HashableString("elem1")
		err := s.Add(elem1)
		require.NoError(t, err)
		assert.Equal(t, 1, s.Size())

		elem2 := hashing.HashableString("elem2")
		err = s.Add(elem2)
		require.NoError(t, err)
		assert.Equal(t, 2, s.Size())
	})

	t.Run("concurrent size checks are safe", func(t *testing.T) {
		t.Parallel()

		threadSafeSet := set.NewThreadSafeSet(set.NewSet[hashing.HashableString](hashing.Sha256))

		var waitGroup sync.WaitGroup

		// Add items in background
		for addIndex := range 100 {
			waitGroup.Add(1)

			go func(index int) {
				defer waitGroup.Done()

				element := hashing.HashableString(fmt.Sprintf("elem-%d", index))
				_ = threadSafeSet.Add(element)
			}(addIndex)
		}

		// Check size concurrently
		for range 50 {
			waitGroup.Go(func() {
				_ = threadSafeSet.Size()
			})
		}

		waitGroup.Wait()
		assert.Equal(t, 100, threadSafeSet.Size())
	})
}

//nolint:dupl // Intentional duplication with thread_safe_ordered_test.go for parallel test coverage
func TestThreadSafeSet_Entries(t *testing.T) {
	t.Parallel()

	t.Run("returns all elements", func(t *testing.T) {
		t.Parallel()

		s := set.NewThreadSafeSet(set.NewSet[hashing.HashableString](hashing.Sha256))
		expected := []string{"elem1", "elem2", "elem3"}

		for _, e := range expected {
			err := s.Add(hashing.HashableString(e))
			require.NoError(t, err)
		}

		entries := s.Entries()
		assert.Len(t, entries, 3)
	})

	t.Run("handles empty set", func(t *testing.T) {
		t.Parallel()

		s := set.NewThreadSafeSet(set.NewSet[hashing.HashableString](hashing.Sha256))
		entries := s.Entries()
		assert.Empty(t, entries)
	})

	t.Run("concurrent Entries calls are safe", func(t *testing.T) {
		t.Parallel()

		threadSafeSet := set.NewThreadSafeSet(set.NewSet[hashing.HashableString](hashing.Sha256))

		// Populate set
		for elemIndex := range 100 {
			element := hashing.HashableString(fmt.Sprintf("elem-%d", elemIndex))
			err := threadSafeSet.Add(element)
			require.NoError(t, err)
		}

		var waitGroup sync.WaitGroup

		// Multiple concurrent Entries calls
		for range 20 {
			waitGroup.Go(func() {
				entries := threadSafeSet.Entries()
				assert.Len(t, entries, 100)
			})
		}

		waitGroup.Wait()
	})
}

//nolint:dupl // Intentional duplication with thread_safe_ordered_test.go for parallel test coverage
func TestThreadSafeSet_Seq(t *testing.T) {
	t.Parallel()

	t.Run("iterates over all entries", func(t *testing.T) {
		t.Parallel()

		s := set.NewThreadSafeSet(set.NewSet[hashing.HashableString](hashing.Sha256))
		expected := map[string]bool{
			"elem1": true,
			"elem2": true,
			"elem3": true,
		}

		for k := range expected {
			err := s.Add(hashing.HashableString(k))
			require.NoError(t, err)
		}

		visited := make(map[string]bool)
		for element := range s.Seq() {
			visited[string(element)] = true
		}

		assert.Equal(t, expected, visited)
	})

	t.Run("handles empty set", func(t *testing.T) {
		t.Parallel()

		s := set.NewThreadSafeSet(set.NewSet[hashing.HashableString](hashing.Sha256))
		count := 0

		for range s.Seq() {
			count++
		}

		assert.Equal(t, 0, count)
	})

	t.Run("stops early when yield returns false", func(t *testing.T) {
		t.Parallel()

		s := set.NewThreadSafeSet(set.NewSet[hashing.HashableString](hashing.Sha256))

		for i := range 10 {
			element := hashing.HashableString(fmt.Sprintf("elem%d", i))
			err := s.Add(element)
			require.NoError(t, err)
		}

		count := 0
		for range s.Seq() {
			count++
			if count >= 5 {
				break
			}
		}

		assert.Equal(t, 5, count)
	})

	t.Run("iteration sees snapshot of set at call time", func(t *testing.T) {
		t.Parallel()

		threadSafeSet := set.NewThreadSafeSet(set.NewSet[hashing.HashableString](hashing.Sha256))

		// Add initial entries
		for elemIndex := range 5 {
			element := hashing.HashableString(fmt.Sprintf("elem%d", elemIndex))
			err := threadSafeSet.Add(element)
			require.NoError(t, err)
		}

		// Get iterator
		seq := threadSafeSet.Seq()

		// Modify set after getting iterator
		for elemIndex := 5; elemIndex < 10; elemIndex++ {
			element := hashing.HashableString(fmt.Sprintf("elem%d", elemIndex))
			_ = threadSafeSet.Add(element)
		}

		// Iterator should only see first 5 entries
		count := 0
		for range seq {
			count++
		}

		assert.Equal(t, 5, count)
		assert.Equal(t, 10, threadSafeSet.Size()) // But set has 10 entries
	})

	t.Run("concurrent iteration and modification are safe", func(t *testing.T) {
		t.Parallel()

		threadSafeSet := set.NewThreadSafeSet(set.NewSet[hashing.HashableString](hashing.Sha256))

		// Populate set
		for elemIndex := range 100 {
			element := hashing.HashableString(fmt.Sprintf("elem-%d", elemIndex))
			err := threadSafeSet.Add(element)
			require.NoError(t, err)
		}

		var waitGroup sync.WaitGroup

		// Multiple concurrent iterators
		for range 10 {
			waitGroup.Go(func() {
				count := 0
				for range threadSafeSet.Seq() {
					count++
				}

				assert.Positive(t, count)
			})
		}

		// Modify set while iterating
		for modifyIndex := range 50 {
			waitGroup.Add(1)

			go func(index int) {
				defer waitGroup.Done()

				element := hashing.HashableString(fmt.Sprintf("new-elem-%d", index))
				_ = threadSafeSet.Add(element)
			}(modifyIndex)
		}

		waitGroup.Wait()
	})

	t.Run("multiple concurrent iterators don't block each other", func(t *testing.T) {
		t.Parallel()

		threadSafeSet := set.NewThreadSafeSet(set.NewSet[hashing.HashableString](hashing.Sha256))

		// Large set for slower iteration
		for elemIndex := range 1000 {
			element := hashing.HashableString(fmt.Sprintf("elem-%d", elemIndex))
			err := threadSafeSet.Add(element)
			require.NoError(t, err)
		}

		start := time.Now()

		var waitGroup sync.WaitGroup

		// Multiple concurrent iterators
		numIterators := 10
		for range numIterators {
			waitGroup.Go(func() {
				for range threadSafeSet.Seq() {
					// Simulate slow iteration
					time.Sleep(10 * time.Microsecond)
				}
			})
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
func TestThreadSafeSet_Union(t *testing.T) {
	t.Parallel()

	t.Run("combines two sets", func(t *testing.T) {
		t.Parallel()

		s1 := set.NewThreadSafeSet(set.NewSet[hashing.HashableString](hashing.Sha256))
		s1.Add(hashing.HashableString("elem1")) //nolint:errcheck,gosec
		s1.Add(hashing.HashableString("elem2")) //nolint:errcheck,gosec

		s2 := set.NewSet[hashing.HashableString](hashing.Sha256)
		s2.Add(hashing.HashableString("elem3")) //nolint:errcheck,gosec
		s2.Add(hashing.HashableString("elem4")) //nolint:errcheck,gosec

		result, err := s1.Union(s2)
		require.NoError(t, err)
		assert.Equal(t, 4, result.Size())
	})

	t.Run("result is also thread-safe", func(t *testing.T) {
		t.Parallel()

		s1 := set.NewThreadSafeSet(set.NewSet[hashing.HashableString](hashing.Sha256))
		s1.Add(hashing.HashableString("elem1")) //nolint:errcheck,gosec

		s2 := set.NewSet[hashing.HashableString](hashing.Sha256)
		s2.Add(hashing.HashableString("elem2")) //nolint:errcheck,gosec

		result, err := s1.Union(s2)
		require.NoError(t, err)

		// Verify result is thread-safe by doing concurrent operations
		var wg sync.WaitGroup
		for i := range 10 {
			wg.Add(1)

			go func(index int) {
				defer wg.Done()

				element := hashing.HashableString(fmt.Sprintf("new-%d", index))
				_ = result.Add(element)
			}(i)
		}

		wg.Wait()

		assert.Equal(t, 12, result.Size())
	})

	t.Run("original sets are not modified", func(t *testing.T) {
		t.Parallel()

		s1 := set.NewThreadSafeSet(set.NewSet[hashing.HashableString](hashing.Sha256))
		s1.Add(hashing.HashableString("elem1")) //nolint:errcheck,gosec

		s2 := set.NewSet[hashing.HashableString](hashing.Sha256)
		s2.Add(hashing.HashableString("elem2")) //nolint:errcheck,gosec

		result, err := s1.Union(s2)
		require.NoError(t, err)
		assert.Equal(t, 2, result.Size())
		assert.Equal(t, 1, s1.Size())
		assert.Equal(t, 1, s2.Size())
	})

	t.Run("concurrent union operations are safe", func(t *testing.T) {
		t.Parallel()

		firstSet := set.NewThreadSafeSet(set.NewSet[hashing.HashableString](hashing.Sha256))

		for elemIndex := range 50 {
			element := hashing.HashableString(fmt.Sprintf("elem-%d", elemIndex))
			_ = firstSet.Add(element)
		}

		numOperations := 20

		var waitGroup sync.WaitGroup

		for opIndex := range numOperations {
			waitGroup.Add(1)

			go func(index int) {
				defer waitGroup.Done()

				secondSet := set.NewSet[hashing.HashableString](hashing.Sha256)
				element := hashing.HashableString(fmt.Sprintf("union-elem-%d", index))
				_ = secondSet.Add(element)
				_, _ = firstSet.Union(secondSet)
			}(opIndex)
		}

		waitGroup.Wait()
		assert.Equal(t, 50, firstSet.Size()) // Original unchanged
	})
}

//nolint:dupl // Intentional duplication with thread_safe_ordered_test.go for parallel test coverage
func TestThreadSafeSet_Intersection(t *testing.T) {
	t.Parallel()

	t.Run("returns common elements", func(t *testing.T) {
		t.Parallel()

		s1 := set.NewThreadSafeSet(set.NewSet[hashing.HashableString](hashing.Sha256))
		s1.Add(hashing.HashableString("elem1")) //nolint:errcheck,gosec
		s1.Add(hashing.HashableString("elem2")) //nolint:errcheck,gosec
		s1.Add(hashing.HashableString("elem3")) //nolint:errcheck,gosec

		s2 := set.NewSet[hashing.HashableString](hashing.Sha256)
		s2.Add(hashing.HashableString("elem2")) //nolint:errcheck,gosec
		s2.Add(hashing.HashableString("elem3")) //nolint:errcheck,gosec
		s2.Add(hashing.HashableString("elem4")) //nolint:errcheck,gosec

		result, err := s1.Intersection(s2)
		require.NoError(t, err)
		assert.Equal(t, 2, result.Size())

		contains, err := result.Contains(hashing.HashableString("elem2"))
		require.NoError(t, err)
		assert.True(t, contains)

		contains, err = result.Contains(hashing.HashableString("elem3"))
		require.NoError(t, err)
		assert.True(t, contains)
	})

	t.Run("result is also thread-safe", func(t *testing.T) {
		t.Parallel()

		s1 := set.NewThreadSafeSet(set.NewSet[hashing.HashableString](hashing.Sha256))
		s1.Add(hashing.HashableString("elem1")) //nolint:errcheck,gosec

		s2 := set.NewSet[hashing.HashableString](hashing.Sha256)
		s2.Add(hashing.HashableString("elem1")) //nolint:errcheck,gosec

		result, err := s1.Intersection(s2)
		require.NoError(t, err)

		// Verify result is thread-safe
		var wg sync.WaitGroup
		for range 10 {
			wg.Go(func() {
				contains, _ := result.Contains(hashing.HashableString("elem1"))
				assert.True(t, contains)
			})
		}

		wg.Wait()
	})

	t.Run("original sets are not modified", func(t *testing.T) {
		t.Parallel()

		s1 := set.NewThreadSafeSet(set.NewSet[hashing.HashableString](hashing.Sha256))
		s1.Add(hashing.HashableString("elem1")) //nolint:errcheck,gosec
		s1.Add(hashing.HashableString("elem2")) //nolint:errcheck,gosec

		s2 := set.NewSet[hashing.HashableString](hashing.Sha256)
		s2.Add(hashing.HashableString("elem2")) //nolint:errcheck,gosec

		result, err := s1.Intersection(s2)
		require.NoError(t, err)
		assert.Equal(t, 1, result.Size())
		assert.Equal(t, 2, s1.Size())
		assert.Equal(t, 1, s2.Size())
	})
}

// TestThreadSafeSet_RaceConditions uses go test -race to detect race conditions.
//
//nolint:dupl // Intentional duplication with thread_safe_ordered_test.go for parallel test coverage
func TestThreadSafeSet_RaceConditions(t *testing.T) {
	t.Parallel()

	t.Run("stress test with mixed operations", func(t *testing.T) {
		t.Parallel()

		threadSafeSet := set.NewThreadSafeSet(set.NewSet[hashing.HashableString](hashing.Sha256))
		numGoroutines := 50
		operationsPerGoroutine := 100

		var waitGroup sync.WaitGroup

		for goroutineID := range numGoroutines {
			waitGroup.Add(1)

			go func(id int) {
				defer waitGroup.Done()

				for opIndex := range operationsPerGoroutine {
					element := hashing.HashableString(fmt.Sprintf("elem-%d-%d", id, opIndex%10))

					switch opIndex % 6 {
					case 0:
						_ = threadSafeSet.Add(element)
					case 1:
						_ = threadSafeSet.Remove(element)
					case 2:
						_, _ = threadSafeSet.Contains(element)
					case 3:
						_ = threadSafeSet.Size()
					case 4:
						for range threadSafeSet.Seq() {
							break // Just get one element
						}
					case 5:
						if opIndex%20 == 0 {
							threadSafeSet.Clear()
						}
					}
				}
			}(goroutineID)
		}

		waitGroup.Wait()
	})
}
