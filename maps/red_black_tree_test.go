package maps_test

import (
	"fmt"
	"testing"

	"github.com/amp-labs/amp-common/maps"
	"github.com/amp-labs/amp-common/sortable"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRedBlackTreeMap(t *testing.T) {
	t.Parallel()

	t.Run("creates empty map", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, string]()
		require.NotNil(t, m)
		assert.Equal(t, 0, m.Size())
	})

	t.Run("map is usable immediately", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, int]()
		err := m.Add(sortable.Int(1), 42)
		require.NoError(t, err)
		assert.Equal(t, 1, m.Size())
	})
}

func TestRedBlackTreeMap_Add(t *testing.T) {
	t.Parallel()

	t.Run("adds new key-value pair", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, string]()
		err := m.Add(sortable.Int(1), "value")
		require.NoError(t, err)
		assert.Equal(t, 1, m.Size())
	})

	t.Run("updates existing key", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, string]()
		err := m.Add(sortable.Int(1), "value1")
		require.NoError(t, err)

		err = m.Add(sortable.Int(1), "value2")
		require.NoError(t, err)
		assert.Equal(t, 1, m.Size())

		val, found, err := m.Get(sortable.Int(1))
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "value2", val)
	})

	t.Run("handles multiple different keys", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, int]()

		for i := range 100 {
			err := m.Add(sortable.Int(i), i)
			require.NoError(t, err)
		}

		assert.Equal(t, 100, m.Size())
	})

	t.Run("maintains sorted order", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, string]()

		// Insert in random order
		keys := []int{5, 2, 8, 1, 9, 3, 7, 4, 6}
		for _, k := range keys {
			err := m.Add(sortable.Int(k), fmt.Sprintf("val%d", k))
			require.NoError(t, err)
		}

		// Verify sorted iteration
		expected := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
		i := 0

		for k := range m.Seq() {
			assert.Equal(t, sortable.Int(expected[i]), k)

			i++
		}
	})

	t.Run("handles adding to root", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, string]()
		err := m.Add(sortable.Int(1), "root")
		require.NoError(t, err)

		val, found, err := m.Get(sortable.Int(1))
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "root", val)
	})
}

func TestRedBlackTreeMap_Get(t *testing.T) {
	t.Parallel()

	t.Run("returns value for existing key", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, string]()
		err := m.Add(sortable.Int(1), "value")
		require.NoError(t, err)

		val, found, err := m.Get(sortable.Int(1))
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "value", val)
	})

	t.Run("returns zero value and false for missing key", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, string]()
		val, found, err := m.Get(sortable.Int(1))
		require.NoError(t, err)
		assert.False(t, found)
		assert.Empty(t, val)
	})

	t.Run("returns most recent value for updated key", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, string]()
		err := m.Add(sortable.Int(1), "value1")
		require.NoError(t, err)

		err = m.Add(sortable.Int(1), "value2")
		require.NoError(t, err)

		val, found, err := m.Get(sortable.Int(1))
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "value2", val)
	})

	t.Run("handles multiple keys correctly", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, int]()

		for i := range 10 {
			err := m.Add(sortable.Int(i), i*10)
			require.NoError(t, err)
		}

		for i := range 10 {
			val, found, err := m.Get(sortable.Int(i))
			require.NoError(t, err)
			assert.True(t, found)
			assert.Equal(t, i*10, val)
		}
	})
}

func TestRedBlackTreeMap_GetOrElse(t *testing.T) {
	t.Parallel()

	t.Run("returns value for existing key", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, string]()
		err := m.Add(sortable.Int(1), "value")
		require.NoError(t, err)

		val, err := m.GetOrElse(sortable.Int(1), "default")
		require.NoError(t, err)
		assert.Equal(t, "value", val)
	})

	t.Run("returns default value for missing key", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, string]()
		val, err := m.GetOrElse(sortable.Int(1), "default")
		require.NoError(t, err)
		assert.Equal(t, "default", val)
	})

	t.Run("returns default value for empty map", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, int]()
		val, err := m.GetOrElse(sortable.Int(1), 999)
		require.NoError(t, err)
		assert.Equal(t, 999, val)
	})
}

func TestRedBlackTreeMap_Remove(t *testing.T) {
	t.Parallel()

	t.Run("removes existing key", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, string]()
		err := m.Add(sortable.Int(1), "value")
		require.NoError(t, err)

		err = m.Remove(sortable.Int(1))
		require.NoError(t, err)
		assert.Equal(t, 0, m.Size())

		found, err := m.Contains(sortable.Int(1))
		require.NoError(t, err)
		assert.False(t, found)
	})

	t.Run("no-op for non-existent key", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, string]()
		err := m.Remove(sortable.Int(1))
		require.NoError(t, err)
		assert.Equal(t, 0, m.Size())
	})

	t.Run("removes from multiple keys", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, int]() //nolint:varnamelen // Short variable name acceptable in tests

		for i := range 10 {
			err := m.Add(sortable.Int(i), i)
			require.NoError(t, err)
		}

		for i := 0; i < 10; i += 2 {
			err := m.Remove(sortable.Int(i))
			require.NoError(t, err)
		}

		assert.Equal(t, 5, m.Size())

		// Verify remaining keys
		for i := 1; i < 10; i += 2 {
			found, err := m.Contains(sortable.Int(i))
			require.NoError(t, err)
			assert.True(t, found)
		}
	})

	t.Run("maintains tree balance after removal", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, int]() //nolint:varnamelen // Short variable name acceptable in tests

		// Insert many elements
		for i := range 100 {
			err := m.Add(sortable.Int(i), i)
			require.NoError(t, err)
		}

		// Remove half of them
		for i := 0; i < 100; i += 2 {
			err := m.Remove(sortable.Int(i))
			require.NoError(t, err)
		}

		assert.Equal(t, 50, m.Size())

		// Verify all remaining elements are accessible
		for i := 1; i < 100; i += 2 {
			val, found, err := m.Get(sortable.Int(i))
			require.NoError(t, err)
			assert.True(t, found)
			assert.Equal(t, i, val)
		}
	})
}

func TestRedBlackTreeMap_Clear(t *testing.T) {
	t.Parallel()

	t.Run("removes all entries", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, string]()

		for i := range 10 {
			err := m.Add(sortable.Int(i), fmt.Sprintf("val%d", i))
			require.NoError(t, err)
		}

		m.Clear()
		assert.Equal(t, 0, m.Size())
	})

	t.Run("map is usable after clear", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, int]()
		err := m.Add(sortable.Int(1), 10)
		require.NoError(t, err)

		m.Clear()

		err = m.Add(sortable.Int(2), 20)
		require.NoError(t, err)
		assert.Equal(t, 1, m.Size())
	})
}

func TestRedBlackTreeMap_Contains(t *testing.T) {
	t.Parallel()

	t.Run("returns true for existing key", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, string]()
		err := m.Add(sortable.Int(1), "value")
		require.NoError(t, err)

		found, err := m.Contains(sortable.Int(1))
		require.NoError(t, err)
		assert.True(t, found)
	})

	t.Run("returns false for non-existent key", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, string]()
		found, err := m.Contains(sortable.Int(1))
		require.NoError(t, err)
		assert.False(t, found)
	})

	t.Run("returns false after removal", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, string]()
		err := m.Add(sortable.Int(1), "value")
		require.NoError(t, err)

		err = m.Remove(sortable.Int(1))
		require.NoError(t, err)

		found, err := m.Contains(sortable.Int(1))
		require.NoError(t, err)
		assert.False(t, found)
	})
}

func TestRedBlackTreeMap_Size(t *testing.T) {
	t.Parallel()

	t.Run("returns zero for empty map", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, string]()
		assert.Equal(t, 0, m.Size())
	})

	t.Run("returns correct size after additions", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, int]()

		for i := range 5 {
			err := m.Add(sortable.Int(i), i)
			require.NoError(t, err)
			assert.Equal(t, i+1, m.Size())
		}
	})

	t.Run("size decreases after removals", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, int]()

		for i := range 5 {
			err := m.Add(sortable.Int(i), i)
			require.NoError(t, err)
		}

		for i := range 3 {
			err := m.Remove(sortable.Int(i))
			require.NoError(t, err)
		}

		assert.Equal(t, 2, m.Size())
	})
}

func TestRedBlackTreeMap_Seq(t *testing.T) {
	t.Parallel()

	t.Run("iterates over all entries in sorted order", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, string]()

		keys := []int{5, 2, 8, 1, 9}
		for _, k := range keys {
			err := m.Add(sortable.Int(k), fmt.Sprintf("val%d", k))
			require.NoError(t, err)
		}

		result := make(map[int]string)
		for k, v := range m.Seq() {
			result[int(k)] = v
		}

		assert.Len(t, result, 5)
		assert.Equal(t, "val1", result[1])
		assert.Equal(t, "val5", result[5])
	})

	t.Run("handles empty map", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, string]()

		count := 0
		for range m.Seq() {
			count++
		}

		assert.Equal(t, 0, count)
	})

	t.Run("stops early when yield returns false", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, int]()

		for i := range 10 {
			err := m.Add(sortable.Int(i), i)
			require.NoError(t, err)
		}

		count := 0

		for k := range m.Seq() {
			count++

			if k == sortable.Int(5) {
				break
			}
		}

		assert.LessOrEqual(t, count, 6)
	})
}

func TestRedBlackTreeMap_Union(t *testing.T) {
	t.Parallel()

	t.Run("combines two maps", func(t *testing.T) {
		t.Parallel()

		m1 := maps.NewRedBlackTreeMap[sortable.Int, string]()
		err := m1.Add(sortable.Int(1), "a")
		require.NoError(t, err)
		err = m1.Add(sortable.Int(2), "b")
		require.NoError(t, err)

		m2 := maps.NewRedBlackTreeMap[sortable.Int, string]()
		err = m2.Add(sortable.Int(3), "c")
		require.NoError(t, err)

		result, err := m1.Union(m2)
		require.NoError(t, err)
		assert.Equal(t, 3, result.Size())
	})

	t.Run("overlapping keys use second map value", func(t *testing.T) {
		t.Parallel()

		m1 := maps.NewRedBlackTreeMap[sortable.Int, string]()
		err := m1.Add(sortable.Int(1), "a")
		require.NoError(t, err)

		m2 := maps.NewRedBlackTreeMap[sortable.Int, string]()
		err = m2.Add(sortable.Int(1), "b")
		require.NoError(t, err)

		result, err := m1.Union(m2)
		require.NoError(t, err)

		val, found, err := result.Get(sortable.Int(1))
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "b", val)
	})

	t.Run("original maps are not modified", func(t *testing.T) {
		t.Parallel()

		m1 := maps.NewRedBlackTreeMap[sortable.Int, string]()
		err := m1.Add(sortable.Int(1), "a")
		require.NoError(t, err)

		m2 := maps.NewRedBlackTreeMap[sortable.Int, string]()
		err = m2.Add(sortable.Int(2), "b")
		require.NoError(t, err)

		_, err = m1.Union(m2)
		require.NoError(t, err)

		assert.Equal(t, 1, m1.Size())
		assert.Equal(t, 1, m2.Size())
	})
}

func TestRedBlackTreeMap_Intersection(t *testing.T) {
	t.Parallel()

	t.Run("returns common keys", func(t *testing.T) {
		t.Parallel()

		m1 := maps.NewRedBlackTreeMap[sortable.Int, string]()
		err := m1.Add(sortable.Int(1), "a")
		require.NoError(t, err)
		err = m1.Add(sortable.Int(2), "b")
		require.NoError(t, err)

		m2 := maps.NewRedBlackTreeMap[sortable.Int, string]()
		err = m2.Add(sortable.Int(2), "c")
		require.NoError(t, err)
		err = m2.Add(sortable.Int(3), "d")
		require.NoError(t, err)

		result, err := m1.Intersection(m2)
		require.NoError(t, err)
		assert.Equal(t, 1, result.Size())

		found, err := result.Contains(sortable.Int(2))
		require.NoError(t, err)
		assert.True(t, found)
	})

	t.Run("returns empty map when no common keys", func(t *testing.T) {
		t.Parallel()

		m1 := maps.NewRedBlackTreeMap[sortable.Int, string]()
		err := m1.Add(sortable.Int(1), "a")
		require.NoError(t, err)

		m2 := maps.NewRedBlackTreeMap[sortable.Int, string]()
		err = m2.Add(sortable.Int(2), "b")
		require.NoError(t, err)

		result, err := m1.Intersection(m2)
		require.NoError(t, err)
		assert.Equal(t, 0, result.Size())
	})
}

func TestRedBlackTreeMap_Clone(t *testing.T) {
	t.Parallel()

	t.Run("creates independent copy", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, string]()
		err := m.Add(sortable.Int(1), "a")
		require.NoError(t, err)

		cloned := m.Clone()
		assert.Equal(t, m.Size(), cloned.Size())

		err = cloned.Add(sortable.Int(2), "b")
		require.NoError(t, err)

		assert.Equal(t, 1, m.Size())
		assert.Equal(t, 2, cloned.Size())
	})

	t.Run("clones empty map", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, string]()
		cloned := m.Clone()
		assert.Equal(t, 0, cloned.Size())
	})
}

func TestRedBlackTreeMap_Keys(t *testing.T) {
	t.Parallel()

	t.Run("returns all keys from map", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, string]()

		for i := range 5 {
			err := m.Add(sortable.Int(i), fmt.Sprintf("val%d", i))
			require.NoError(t, err)
		}

		keys := m.Keys()
		assert.Equal(t, 5, keys.Size())

		for i := range 5 {
			found, err := keys.Contains(sortable.Int(i))
			require.NoError(t, err)
			assert.True(t, found)
		}
	})

	t.Run("returns empty set for empty map", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, string]()
		keys := m.Keys()
		assert.Equal(t, 0, keys.Size())
	})
}

func TestRedBlackTreeMap_ForEach(t *testing.T) {
	t.Parallel()

	t.Run("calls function for each entry", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, int]()

		for i := range 5 {
			err := m.Add(sortable.Int(i), i*10)
			require.NoError(t, err)
		}

		sum := 0

		m.ForEach(func(k sortable.Int, v int) {
			sum += v
		})

		assert.Equal(t, 100, sum) // 0 + 10 + 20 + 30 + 40
	})

	t.Run("handles empty map", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, int]()

		count := 0

		m.ForEach(func(k sortable.Int, v int) {
			count++
		})

		assert.Equal(t, 0, count)
	})
}

func TestRedBlackTreeMap_ForAll(t *testing.T) {
	t.Parallel()

	t.Run("returns true when predicate holds for all entries", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, int]()

		for i := range 5 {
			err := m.Add(sortable.Int(i), i*2)
			require.NoError(t, err)
		}

		result := m.ForAll(func(k sortable.Int, v int) bool {
			return v%2 == 0
		})

		assert.True(t, result)
	})

	t.Run("returns false when predicate fails for any entry", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, int]()

		for i := range 5 {
			err := m.Add(sortable.Int(i), i)
			require.NoError(t, err)
		}

		result := m.ForAll(func(k sortable.Int, v int) bool {
			return v%2 == 0
		})

		assert.False(t, result)
	})

	t.Run("returns true for empty map", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, int]()

		result := m.ForAll(func(k sortable.Int, v int) bool {
			return false
		})

		assert.True(t, result)
	})
}

func TestRedBlackTreeMap_Filter(t *testing.T) {
	t.Parallel()

	t.Run("returns map with entries matching predicate", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, int]()

		for i := range 10 {
			err := m.Add(sortable.Int(i), i)
			require.NoError(t, err)
		}

		result := m.Filter(func(k sortable.Int, v int) bool {
			return v%2 == 0
		})

		assert.Equal(t, 5, result.Size())
	})

	t.Run("original map is not modified", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, int]()

		for i := range 5 {
			err := m.Add(sortable.Int(i), i)
			require.NoError(t, err)
		}

		_ = m.Filter(func(k sortable.Int, v int) bool {
			return v > 2
		})

		assert.Equal(t, 5, m.Size())
	})
}

func TestRedBlackTreeMap_FilterNot(t *testing.T) {
	t.Parallel()

	t.Run("returns map with entries not matching predicate", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, int]()

		for i := range 10 {
			err := m.Add(sortable.Int(i), i)
			require.NoError(t, err)
		}

		result := m.FilterNot(func(k sortable.Int, v int) bool {
			return v%2 == 0
		})

		assert.Equal(t, 5, result.Size())
	})
}

func TestRedBlackTreeMap_Map(t *testing.T) {
	t.Parallel()

	t.Run("transforms all entries", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, int]()

		for i := range 5 {
			err := m.Add(sortable.Int(i), i)
			require.NoError(t, err)
		}

		result := m.Map(func(k sortable.Int, v int) (sortable.Int, int) {
			return k, v * 2
		})

		for i := range 5 {
			val, found, err := result.Get(sortable.Int(i))
			require.NoError(t, err)
			assert.True(t, found)
			assert.Equal(t, i*2, val)
		}
	})
}

func TestRedBlackTreeMap_FlatMap(t *testing.T) {
	t.Parallel()

	t.Run("flattens nested maps", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, int]()
		err := m.Add(sortable.Int(1), 1)
		require.NoError(t, err)

		result := m.FlatMap(func(k sortable.Int, v int) maps.Map[sortable.Int, int] {
			nested := maps.NewRedBlackTreeMap[sortable.Int, int]()
			_ = nested.Add(sortable.Int(v*10), v*10)
			_ = nested.Add(sortable.Int(v*10+1), v*10+1)

			return nested
		})

		assert.Equal(t, 2, result.Size())
	})

	t.Run("handles nil results from function", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, int]()
		err := m.Add(sortable.Int(1), 1)
		require.NoError(t, err)

		result := m.FlatMap(func(k sortable.Int, v int) maps.Map[sortable.Int, int] {
			return nil
		})

		assert.Equal(t, 0, result.Size())
	})
}

func TestRedBlackTreeMap_Exists(t *testing.T) {
	t.Parallel()

	t.Run("returns true when at least one entry matches", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, int]()

		for i := range 5 {
			err := m.Add(sortable.Int(i), i)
			require.NoError(t, err)
		}

		result := m.Exists(func(k sortable.Int, v int) bool {
			return v == 3
		})

		assert.True(t, result)
	})

	t.Run("returns false when no entries match", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, int]()

		for i := range 5 {
			err := m.Add(sortable.Int(i), i)
			require.NoError(t, err)
		}

		result := m.Exists(func(k sortable.Int, v int) bool {
			return v > 10
		})

		assert.False(t, result)
	})
}

func TestRedBlackTreeMap_FindFirst(t *testing.T) {
	t.Parallel()

	t.Run("returns first matching entry in sorted order", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, int]()

		keys := []int{5, 2, 8, 1, 9}
		for _, k := range keys {
			err := m.Add(sortable.Int(k), k*10)
			require.NoError(t, err)
		}

		result := m.FindFirst(func(k sortable.Int, v int) bool {
			return v > 30
		})

		assert.True(t, result.NonEmpty())
		pair := result.GetOrPanic()
		assert.Equal(t, sortable.Int(5), pair.Key)
		assert.Equal(t, 50, pair.Value)
	})

	t.Run("returns None when no match found", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, int]()

		for i := range 5 {
			err := m.Add(sortable.Int(i), i)
			require.NoError(t, err)
		}

		result := m.FindFirst(func(k sortable.Int, v int) bool {
			return v > 10
		})

		assert.True(t, result.Empty())
	})
}

func TestRedBlackTreeMap_HashFunction(t *testing.T) {
	t.Parallel()

	t.Run("returns nil for red-black tree", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, string]()
		assert.Nil(t, m.HashFunction())
	})
}

func TestRedBlackTreeMap_StressTest(t *testing.T) {
	t.Parallel()

	t.Run("handles large number of insertions and deletions", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.Int, int]() //nolint:varnamelen // Short variable name acceptable in tests

		// Insert many elements
		for i := range 1000 {
			err := m.Add(sortable.Int(i), i*2)
			require.NoError(t, err)
		}

		assert.Equal(t, 1000, m.Size())

		// Remove every other element
		for i := 0; i < 1000; i += 2 {
			err := m.Remove(sortable.Int(i))
			require.NoError(t, err)
		}

		assert.Equal(t, 500, m.Size())

		// Verify remaining elements
		for i := 1; i < 1000; i += 2 {
			val, found, err := m.Get(sortable.Int(i))
			require.NoError(t, err)
			assert.True(t, found)
			assert.Equal(t, i*2, val)
		}

		// Verify deleted elements
		for i := 0; i < 1000; i += 2 {
			found, err := m.Contains(sortable.Int(i))
			require.NoError(t, err)
			assert.False(t, found)
		}
	})
}

func TestRedBlackTreeMap_StringKeys(t *testing.T) {
	t.Parallel()

	t.Run("works with string keys", func(t *testing.T) {
		t.Parallel()

		m := maps.NewRedBlackTreeMap[sortable.String, int]()

		err := m.Add(sortable.String("apple"), 1)
		require.NoError(t, err)
		err = m.Add(sortable.String("banana"), 2)
		require.NoError(t, err)
		err = m.Add(sortable.String("cherry"), 3)
		require.NoError(t, err)

		assert.Equal(t, 3, m.Size())

		// Verify sorted iteration
		keys := []string{"apple", "banana", "cherry"}
		i := 0

		for k := range m.Seq() {
			assert.Equal(t, sortable.String(keys[i]), k)

			i++
		}
	})
}
