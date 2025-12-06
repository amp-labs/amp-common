package maps_test

import (
	"fmt"
	"testing"

	"github.com/amp-labs/amp-common/hashing"
	"github.com/amp-labs/amp-common/maps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOrderedHashMap(t *testing.T) {
	t.Parallel()

	t.Run("creates empty map", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		require.NotNil(t, m)
		assert.Equal(t, 0, m.Size())
	})

	t.Run("map is usable immediately", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)
		key := testKey{value: "test"}
		err := m.Add(key, 42)
		require.NoError(t, err)
		assert.Equal(t, 1, m.Size())
	})

	t.Run("preserves insertion order from start", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)
		expectedOrder := []string{"first", "second", "third"}

		for i, key := range expectedOrder {
			err := m.Add(testKey{value: key}, i)
			require.NoError(t, err)
		}

		// Verify order
		idx := 0
		for i, entry := range m.Seq() {
			assert.Equal(t, idx, i)
			assert.Equal(t, expectedOrder[idx], entry.Key.value)

			idx++
		}
	})
}

func TestOrderedHashMap_Add(t *testing.T) {
	t.Parallel()

	t.Run("adds new key-value pair", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		key := testKey{value: "test"}
		err := m.Add(key, "value")
		require.NoError(t, err)
		assert.Equal(t, 1, m.Size())
	})

	t.Run("updates existing key without changing order", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		key := testKey{value: "test"}
		err := m.Add(key, "value1")
		require.NoError(t, err)

		err = m.Add(key, "value2")
		require.NoError(t, err)
		assert.Equal(t, 1, m.Size())

		// Verify order is maintained (key should still be first)
		count := 0

		for i, entry := range m.Seq() {
			assert.Equal(t, 0, i)
			assert.Equal(t, "test", entry.Key.value)
			assert.Equal(t, "value2", entry.Value)

			count++
		}

		assert.Equal(t, 1, count)
	})

	t.Run("handles multiple different keys in order", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)
		keys := []string{"key1", "key2", "key3", "key4", "key5"}

		for i, k := range keys {
			key := testKey{value: k}
			err := m.Add(key, i)
			require.NoError(t, err)
		}

		assert.Equal(t, len(keys), m.Size())

		// Verify insertion order
		idx := 0
		for i, entry := range m.Seq() {
			assert.Equal(t, idx, i)
			assert.Equal(t, keys[idx], entry.Key.value)
			assert.Equal(t, idx, entry.Value)

			idx++
		}
	})

	t.Run("returns error on hash collision", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[collidingKey, string](hashing.Sha256)
		key1 := collidingKey{id: 1, hash: "same"}
		key2 := collidingKey{id: 2, hash: "same"}

		err := m.Add(key1, "value1")
		require.NoError(t, err)

		err = m.Add(key2, "value2")
		assert.Error(t, err)
	})
}

func TestOrderedHashMap_Remove(t *testing.T) {
	t.Parallel()

	t.Run("removes existing key", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
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

		m := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		key := testKey{value: "missing"}
		err := m.Remove(key)
		require.NoError(t, err)
		assert.Equal(t, 0, m.Size())
	})

	t.Run("removes only specified key and maintains order", func(t *testing.T) {
		t.Parallel()

		orderedMap := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		key1 := testKey{value: "key1"}
		key2 := testKey{value: "key2"}
		key3 := testKey{value: "key3"}
		err := orderedMap.Add(key1, "value1")
		require.NoError(t, err)
		err = orderedMap.Add(key2, "value2")
		require.NoError(t, err)
		err = orderedMap.Add(key3, "value3")
		require.NoError(t, err)

		err = orderedMap.Remove(key2)
		require.NoError(t, err)
		assert.Equal(t, 2, orderedMap.Size())

		// Verify order is maintained
		expectedKeys := []string{"key1", "key3"}

		idx := 0
		for i, entry := range orderedMap.Seq() {
			assert.Equal(t, idx, i)
			assert.Equal(t, expectedKeys[idx], entry.Key.value)

			idx++
		}
	})

	t.Run("removes key from middle preserves order", func(t *testing.T) {
		t.Parallel()

		orderedMap := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)
		keys := []string{"a", "b", "c", "d", "e"}

		for i, k := range keys {
			err := orderedMap.Add(testKey{value: k}, i)
			require.NoError(t, err)
		}

		// Remove middle key
		err := orderedMap.Remove(testKey{value: "c"})
		require.NoError(t, err)

		// Verify order
		expected := []string{"a", "b", "d", "e"}

		idx := 0
		for i, entry := range orderedMap.Seq() {
			assert.Equal(t, idx, i)
			assert.Equal(t, expected[idx], entry.Key.value)

			idx++
		}
	})

	t.Run("returns error on hash collision", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[collidingKey, string](hashing.Sha256)
		key1 := collidingKey{id: 1, hash: "same"}
		key2 := collidingKey{id: 2, hash: "same"}

		err := m.Add(key1, "value1")
		require.NoError(t, err)

		err = m.Remove(key2)
		assert.Error(t, err)
	})
}

func TestOrderedHashMap_Clear(t *testing.T) {
	t.Parallel()

	t.Run("removes all entries", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)

		for i := range 10 {
			key := testKey{value: fmt.Sprintf("key%d", i)}
			err := m.Add(key, i)
			require.NoError(t, err)
		}

		m.Clear()
		assert.Equal(t, 0, m.Size())

		// Verify iteration yields nothing
		count := 0
		for range m.Seq() {
			count++
		}

		assert.Equal(t, 0, count)
	})

	t.Run("map is usable after clear", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		key1 := testKey{value: "key1"}
		err := m.Add(key1, "value1")
		require.NoError(t, err)

		m.Clear()

		key2 := testKey{value: "key2"}
		err = m.Add(key2, "value2")
		require.NoError(t, err)
		assert.Equal(t, 1, m.Size())
	})

	t.Run("clear on empty map is safe", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		m.Clear()
		assert.Equal(t, 0, m.Size())
	})
}

//nolint:dupl // Intentional duplication with TestHashMap_Contains for parallel test coverage
func TestOrderedHashMap_Contains(t *testing.T) {
	t.Parallel()

	t.Run("returns true for existing key", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		key := testKey{value: "test"}
		err := m.Add(key, "value")
		require.NoError(t, err)

		contains, err := m.Contains(key)
		require.NoError(t, err)
		assert.True(t, contains)
	})

	t.Run("returns false for non-existent key", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		key := testKey{value: "missing"}

		contains, err := m.Contains(key)
		require.NoError(t, err)
		assert.False(t, contains)
	})

	t.Run("returns false after key is removed", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		key := testKey{value: "test"}
		err := m.Add(key, "value")
		require.NoError(t, err)

		err = m.Remove(key)
		require.NoError(t, err)

		contains, err := m.Contains(key)
		require.NoError(t, err)
		assert.False(t, contains)
	})

	t.Run("returns error on hash collision", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[collidingKey, string](hashing.Sha256)
		key1 := collidingKey{id: 1, hash: "same"}
		key2 := collidingKey{id: 2, hash: "same"}

		err := m.Add(key1, "value1")
		require.NoError(t, err)

		contains, err := m.Contains(key2)
		require.Error(t, err)
		assert.False(t, contains)
	})
}

//nolint:dupl // Intentional duplication with TestHashMap_Size for parallel test coverage
func TestOrderedHashMap_Size(t *testing.T) {
	t.Parallel()

	t.Run("returns zero for empty map", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		assert.Equal(t, 0, m.Size())
	})

	t.Run("returns correct size after additions", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
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

	t.Run("returns correct size after removals", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		key1 := testKey{value: "key1"}
		key2 := testKey{value: "key2"}
		err := m.Add(key1, "value1")
		require.NoError(t, err)
		err = m.Add(key2, "value2")
		require.NoError(t, err)

		err = m.Remove(key1)
		require.NoError(t, err)
		assert.Equal(t, 1, m.Size())
	})

	t.Run("updating existing key doesn't change size", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		key := testKey{value: "key"}
		err := m.Add(key, "value1")
		require.NoError(t, err)

		err = m.Add(key, "value2")
		require.NoError(t, err)
		assert.Equal(t, 1, m.Size())
	})
}

func TestOrderedHashMap_Seq(t *testing.T) {
	t.Parallel()

	t.Run("iterates over all entries in insertion order", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)
		keys := []string{"key1", "key2", "key3", "key4", "key5"}

		for i, k := range keys {
			key := testKey{value: k}
			err := m.Add(key, i)
			require.NoError(t, err)
		}

		// Verify order and index
		idx := 0
		for i, entry := range m.Seq() {
			assert.Equal(t, idx, i, "index should match iteration order")
			assert.Equal(t, keys[idx], entry.Key.value, "key order should match insertion order")
			assert.Equal(t, idx, entry.Value)

			idx++
		}

		assert.Equal(t, len(keys), idx, "should iterate over all entries")
	})

	t.Run("maintains order after updates", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		keys := []string{"a", "b", "c"}

		for _, k := range keys {
			err := m.Add(testKey{value: k}, k)
			require.NoError(t, err)
		}

		// Update middle key
		err := m.Add(testKey{value: "b"}, "updated")
		require.NoError(t, err)

		// Verify order unchanged
		idx := 0
		for i, entry := range m.Seq() {
			assert.Equal(t, idx, i)
			assert.Equal(t, keys[idx], entry.Key.value)

			idx++
		}
	})

	t.Run("handles empty map", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		count := 0

		for range m.Seq() {
			count++
		}

		assert.Equal(t, 0, count)
	})

	t.Run("stops early when yield returns false", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)

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

	t.Run("preserves order after removals", func(t *testing.T) {
		t.Parallel()

		orderedMap := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)
		keys := []string{"a", "b", "c", "d", "e"}

		for i, k := range keys {
			err := orderedMap.Add(testKey{value: k}, i)
			require.NoError(t, err)
		}

		// Remove some keys
		_ = orderedMap.Remove(testKey{value: "b"}) //nolint:errcheck
		_ = orderedMap.Remove(testKey{value: "d"}) //nolint:errcheck

		// Verify remaining order
		expected := []string{"a", "c", "e"}

		idx := 0
		for i, entry := range orderedMap.Seq() {
			assert.Equal(t, idx, i)
			assert.Equal(t, expected[idx], entry.Key.value)

			idx++
		}
	})
}

func TestOrderedHashMap_Union(t *testing.T) {
	t.Parallel()

	t.Run("combines two maps in order", func(t *testing.T) {
		t.Parallel()

		m1 := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		_ = m1.Add(testKey{value: "key1"}, "value1") //nolint:errcheck
		_ = m1.Add(testKey{value: "key2"}, "value2") //nolint:errcheck

		m2 := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		_ = m2.Add(testKey{value: "key3"}, "value3") //nolint:errcheck
		_ = m2.Add(testKey{value: "key4"}, "value4") //nolint:errcheck

		result, err := m1.Union(m2)
		require.NoError(t, err)
		assert.Equal(t, 4, result.Size())

		// Verify order: m1 keys first, then m2 keys
		expectedOrder := []string{"key1", "key2", "key3", "key4"}

		idx := 0
		for i, entry := range result.Seq() {
			assert.Equal(t, idx, i)
			assert.Equal(t, expectedOrder[idx], entry.Key.value)

			idx++
		}
	})

	t.Run("other map values take precedence but maintain first map position", func(t *testing.T) {
		t.Parallel()

		m1 := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		_ = m1.Add(testKey{value: "key1"}, "value1") //nolint:errcheck
		_ = m1.Add(testKey{value: "key2"}, "value2") //nolint:errcheck
		_ = m1.Add(testKey{value: "key3"}, "value3") //nolint:errcheck

		m2 := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		_ = m2.Add(testKey{value: "key2"}, "updated2") //nolint:errcheck
		_ = m2.Add(testKey{value: "key4"}, "value4")   //nolint:errcheck

		result, err := m1.Union(m2)
		require.NoError(t, err)
		assert.Equal(t, 4, result.Size())

		// Verify order: key2 stays in its original position from m1
		expectedOrder := []string{"key1", "key2", "key3", "key4"}

		idx := 0
		for i, entry := range result.Seq() {
			assert.Equal(t, idx, i)
			assert.Equal(t, expectedOrder[idx], entry.Key.value)

			if entry.Key.value == "key2" {
				assert.Equal(t, "updated2", entry.Value, "value should be from second map")
			}

			idx++
		}
	})

	t.Run("original maps are not modified", func(t *testing.T) {
		t.Parallel()

		m1 := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		_ = m1.Add(testKey{value: "key1"}, "value1") //nolint:errcheck

		m2 := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		_ = m2.Add(testKey{value: "key2"}, "value2") //nolint:errcheck

		result, err := m1.Union(m2)
		require.NoError(t, err)
		assert.Equal(t, 2, result.Size())
		assert.Equal(t, 1, m1.Size())
		assert.Equal(t, 1, m2.Size())
	})

	t.Run("union with empty map", func(t *testing.T) {
		t.Parallel()

		m1 := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		_ = m1.Add(testKey{value: "key1"}, "value1") //nolint:errcheck

		m2 := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)

		result, err := m1.Union(m2)
		require.NoError(t, err)
		assert.Equal(t, 1, result.Size())
	})
}

func TestOrderedHashMap_Intersection(t *testing.T) {
	t.Parallel()

	t.Run("returns common keys in first map order", func(t *testing.T) {
		t.Parallel()

		m1 := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		_ = m1.Add(testKey{value: "key1"}, "value1") //nolint:errcheck
		_ = m1.Add(testKey{value: "key2"}, "value2") //nolint:errcheck
		_ = m1.Add(testKey{value: "key3"}, "value3") //nolint:errcheck
		_ = m1.Add(testKey{value: "key4"}, "value4") //nolint:errcheck

		m2 := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		_ = m2.Add(testKey{value: "key4"}, "other4") //nolint:errcheck
		_ = m2.Add(testKey{value: "key2"}, "other2") //nolint:errcheck
		_ = m2.Add(testKey{value: "key5"}, "other5") //nolint:errcheck

		result, err := m1.Intersection(m2)
		require.NoError(t, err)
		assert.Equal(t, 2, result.Size())

		// Verify order from first map (key2, key4)
		expectedOrder := []string{"key2", "key4"}

		idx := 0
		for i, entry := range result.Seq() {
			assert.Equal(t, idx, i)
			assert.Equal(t, expectedOrder[idx], entry.Key.value)

			idx++
		}
	})

	t.Run("values are from first map", func(t *testing.T) {
		t.Parallel()

		m1 := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		_ = m1.Add(testKey{value: "key"}, "value1") //nolint:errcheck

		m2 := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		_ = m2.Add(testKey{value: "key"}, "value2") //nolint:errcheck

		result, err := m1.Intersection(m2)
		require.NoError(t, err)
		assert.Equal(t, 1, result.Size())

		for _, entry := range result.Seq() {
			assert.Equal(t, "value1", entry.Value)
		}
	})

	t.Run("returns empty map when no common keys", func(t *testing.T) {
		t.Parallel()

		m1 := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		_ = m1.Add(testKey{value: "key1"}, "value1") //nolint:errcheck

		m2 := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		_ = m2.Add(testKey{value: "key2"}, "value2") //nolint:errcheck

		result, err := m1.Intersection(m2)
		require.NoError(t, err)
		assert.Equal(t, 0, result.Size())
	})

	t.Run("original maps are not modified", func(t *testing.T) {
		t.Parallel()

		m1 := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		_ = m1.Add(testKey{value: "key1"}, "value1") //nolint:errcheck
		_ = m1.Add(testKey{value: "key2"}, "value2") //nolint:errcheck

		m2 := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		_ = m2.Add(testKey{value: "key2"}, "value2") //nolint:errcheck

		result, err := m1.Intersection(m2)
		require.NoError(t, err)
		assert.Equal(t, 1, result.Size())
		assert.Equal(t, 2, m1.Size())
		assert.Equal(t, 1, m2.Size())
	})
}

func TestOrderedHashMap_Clone(t *testing.T) {
	t.Parallel()

	t.Run("creates independent copy with same order", func(t *testing.T) {
		t.Parallel()

		original := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		keys := []string{"key1", "key2", "key3"}

		for _, k := range keys {
			_ = original.Add(testKey{value: k}, k) //nolint:errcheck
		}

		cloned := original.Clone()
		assert.Equal(t, original.Size(), cloned.Size())

		// Verify order is preserved
		idx := 0
		for i, entry := range cloned.Seq() {
			assert.Equal(t, idx, i)
			assert.Equal(t, keys[idx], entry.Key.value)

			idx++
		}

		// Modify original
		_ = original.Add(testKey{value: "key4"}, "key4") //nolint:errcheck

		// Clone should not be affected
		assert.Equal(t, 4, original.Size())
		assert.Equal(t, 3, cloned.Size())
	})

	t.Run("cloned map has same entries in same order", func(t *testing.T) {
		t.Parallel()

		original := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)
		expectedOrder := []string{"a", "b", "c", "d"}

		for i, k := range expectedOrder {
			_ = original.Add(testKey{value: k}, i) //nolint:errcheck
		}

		cloned := original.Clone()

		// Verify all entries exist and are in order
		idx := 0
		for i, entry := range cloned.Seq() {
			assert.Equal(t, idx, i)
			assert.Equal(t, expectedOrder[idx], entry.Key.value)
			assert.Equal(t, idx, entry.Value)

			idx++
		}
	})

	t.Run("clones empty map", func(t *testing.T) {
		t.Parallel()

		original := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		cloned := original.Clone()
		require.NotNil(t, cloned)
		assert.Equal(t, 0, cloned.Size())
	})

	t.Run("modifications to clone don't affect original", func(t *testing.T) {
		t.Parallel()

		original := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		_ = original.Add(testKey{value: "key1"}, "value1") //nolint:errcheck
		_ = original.Add(testKey{value: "key2"}, "value2") //nolint:errcheck

		cloned := original.Clone()

		// Modify clone
		_ = cloned.Add(testKey{value: "key3"}, "value3") //nolint:errcheck
		_ = cloned.Remove(testKey{value: "key1"})        //nolint:errcheck

		// Original should be unchanged
		assert.Equal(t, 2, original.Size())

		// Verify original order
		expectedOrder := []string{"key1", "key2"}
		idx := 0

		for _, entry := range original.Seq() {
			assert.Equal(t, expectedOrder[idx], entry.Key.value)

			idx++
		}
	})
}

func TestOrderedHashMap_Get(t *testing.T) {
	t.Parallel()

	t.Run("returns value for existing key", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
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

		m := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		key := testKey{value: "missing"}

		value, found, err := m.Get(key)
		require.NoError(t, err)
		assert.False(t, found)
		assert.Equal(t, "", value)
	})

	t.Run("returns zero value and false for missing key with int type", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)
		key := testKey{value: "missing"}

		value, found, err := m.Get(key)
		require.NoError(t, err)
		assert.False(t, found)
		assert.Equal(t, 0, value)
	})

	t.Run("returns most recent value for updated key", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
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

		m := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)
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

	t.Run("returns error on hash collision", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[collidingKey, string](hashing.Sha256)

		// Add first key with a specific hash
		key1 := collidingKey{id: 1, hash: "samehash"}
		err := m.Add(key1, "value1")
		require.NoError(t, err)

		// Try to get with a different key but same hash
		key2 := collidingKey{id: 2, hash: "samehash"}
		value, found, err := m.Get(key2)
		require.Error(t, err)
		assert.False(t, found)
		assert.Equal(t, "", value)
	})

	t.Run("handles nil/empty values correctly", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, *string](hashing.Sha256)
		key := testKey{value: "test"}

		err := m.Add(key, nil)
		require.NoError(t, err)

		value, found, err := m.Get(key)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Nil(t, value)
	})

	t.Run("returns false after key removal", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
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

		m := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		key := testKey{value: "test"}

		err := m.Add(key, "value")
		require.NoError(t, err)

		m.Clear()

		value, found, err := m.Get(key)
		require.NoError(t, err)
		assert.False(t, found)
		assert.Equal(t, "", value)
	})

	t.Run("handles struct values correctly", func(t *testing.T) {
		t.Parallel()

		type testValue struct {
			name string
			age  int
		}

		m := maps.NewOrderedHashMap[testKey, testValue](hashing.Sha256)
		key := testKey{value: "test"}
		expected := testValue{name: "Alice", age: 30}

		err := m.Add(key, expected)
		require.NoError(t, err)

		value, found, err := m.Get(key)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, expected, value)
	})

	t.Run("Get does not affect insertion order", func(t *testing.T) {
		t.Parallel()

		//nolint:varnamelen // Short name acceptable in test context
		m := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)
		keys := []testKey{
			{value: "first"},
			{value: "second"},
			{value: "third"},
		}

		for i, key := range keys {
			err := m.Add(key, i)
			require.NoError(t, err)
		}

		// Get middle key
		_, found, err := m.Get(keys[1])
		require.NoError(t, err)
		assert.True(t, found)

		// Verify order is unchanged
		idx := 0
		for _, entry := range m.Seq() {
			assert.Equal(t, keys[idx], entry.Key)

			idx++
		}
	})
}

func TestOrderedHashMap_GetOrElse(t *testing.T) {
	t.Parallel()

	t.Run("returns value for existing key", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		key := testKey{value: "test"}
		err := m.Add(key, "actual")
		require.NoError(t, err)

		value, err := m.GetOrElse(key, "default")
		require.NoError(t, err)
		assert.Equal(t, "actual", value)
	})

	t.Run("returns default value for missing key", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		key := testKey{value: "missing"}

		value, err := m.GetOrElse(key, "default")
		require.NoError(t, err)
		assert.Equal(t, "default", value)
	})
}

func TestOrderedHashMap_Keys(t *testing.T) {
	t.Parallel()

	t.Run("returns all keys from ordered map", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)
		keys := []testKey{
			{value: "key1"},
			{value: "key2"},
			{value: "key3"},
		}

		for i, k := range keys {
			err := m.Add(k, i)
			require.NoError(t, err)
		}

		keySet := m.Keys()
		require.NotNil(t, keySet)
		assert.Equal(t, 3, keySet.Size())

		for _, k := range keys {
			contains, err := keySet.Contains(k)
			require.NoError(t, err)
			assert.True(t, contains)
		}
	})

	t.Run("returns empty set for empty ordered map", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		keySet := m.Keys()
		require.NotNil(t, keySet)
		assert.Equal(t, 0, keySet.Size())
	})
}

func TestOrderedHashMap_ForEach(t *testing.T) {
	t.Parallel()

	t.Run("calls function for each entry in order", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)
		keys := []testKey{
			{value: "first"},
			{value: "second"},
			{value: "third"},
		}

		for i, k := range keys {
			err := m.Add(k, i)
			require.NoError(t, err)
		}

		visitedKeys := []testKey{}

		m.ForEach(func(key testKey, value int) {
			visitedKeys = append(visitedKeys, key)
		})

		assert.Equal(t, keys, visitedKeys)
	})

	t.Run("handles empty map", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		callCount := 0

		m.ForEach(func(key testKey, value string) {
			callCount++
		})

		assert.Equal(t, 0, callCount)
	})
}

func TestOrderedHashMap_ForAll(t *testing.T) {
	t.Parallel()

	t.Run("returns true when predicate holds for all entries", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)
		_ = m.Add(testKey{value: "a"}, 2) //nolint:errcheck
		_ = m.Add(testKey{value: "b"}, 4) //nolint:errcheck
		_ = m.Add(testKey{value: "c"}, 6) //nolint:errcheck

		result := m.ForAll(func(key testKey, value int) bool {
			return value%2 == 0
		})

		assert.True(t, result)
	})

	t.Run("returns false when predicate fails for any entry", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)
		_ = m.Add(testKey{value: "a"}, 2) //nolint:errcheck
		_ = m.Add(testKey{value: "b"}, 3) //nolint:errcheck
		_ = m.Add(testKey{value: "c"}, 4) //nolint:errcheck

		result := m.ForAll(func(key testKey, value int) bool {
			return value%2 == 0
		})

		assert.False(t, result)
	})

	t.Run("returns true for empty map", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)

		result := m.ForAll(func(key testKey, value int) bool {
			return false
		})

		assert.True(t, result)
	})
}

func TestOrderedHashMap_Filter(t *testing.T) {
	t.Parallel()

	t.Run("returns ordered map with entries matching predicate", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)
		for i := 1; i <= 5; i++ {
			_ = m.Add(testKey{value: fmt.Sprintf("key%d", i)}, i) //nolint:errcheck
		}

		result := m.Filter(func(key testKey, value int) bool {
			return value%2 == 0
		})

		assert.Equal(t, 2, result.Size())

		// Verify order is preserved
		expected := []testKey{
			{value: "key2"},
			{value: "key4"},
		}
		idx := 0

		for _, entry := range result.Seq() {
			assert.Equal(t, expected[idx], entry.Key)

			idx++
		}
	})

	t.Run("original map is not modified", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)
		_ = m.Add(testKey{value: "a"}, 1) //nolint:errcheck
		_ = m.Add(testKey{value: "b"}, 2) //nolint:errcheck

		result := m.Filter(func(key testKey, value int) bool {
			return value == 1
		})

		assert.Equal(t, 1, result.Size())
		assert.Equal(t, 2, m.Size())
	})
}

func TestOrderedHashMap_FilterNot(t *testing.T) {
	t.Parallel()

	t.Run("returns ordered map with entries not matching predicate", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)
		for i := 1; i <= 5; i++ {
			_ = m.Add(testKey{value: fmt.Sprintf("key%d", i)}, i) //nolint:errcheck
		}

		result := m.FilterNot(func(key testKey, value int) bool {
			return value%2 == 0
		})

		assert.Equal(t, 3, result.Size())

		// Verify order is preserved
		expected := []testKey{
			{value: "key1"},
			{value: "key3"},
			{value: "key5"},
		}
		idx := 0

		for _, entry := range result.Seq() {
			assert.Equal(t, expected[idx], entry.Key)

			idx++
		}
	})
}

func TestOrderedHashMap_Map(t *testing.T) {
	t.Parallel()

	t.Run("transforms all entries preserving order", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)
		_ = m.Add(testKey{value: "a"}, 1) //nolint:errcheck
		_ = m.Add(testKey{value: "b"}, 2) //nolint:errcheck
		_ = m.Add(testKey{value: "c"}, 3) //nolint:errcheck

		result := m.Map(func(key testKey, value int) (testKey, int) {
			return testKey{value: key.value + "_new"}, value * 2
		})

		assert.Equal(t, 3, result.Size())

		// Verify order is preserved
		expected := []testKey{
			{value: "a_new"},
			{value: "b_new"},
			{value: "c_new"},
		}
		idx := 0

		for _, entry := range result.Seq() {
			assert.Equal(t, expected[idx], entry.Key)
			assert.Equal(t, (idx+1)*2, entry.Value)

			idx++
		}
	})

	t.Run("original map is not modified", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)
		_ = m.Add(testKey{value: "a"}, 1) //nolint:errcheck

		result := m.Map(func(key testKey, value int) (testKey, int) {
			return key, value * 10
		})

		val, found, _ := m.Get(testKey{value: "a"})
		assert.True(t, found)
		assert.Equal(t, 1, val)

		val, found, _ = result.Get(testKey{value: "a"})
		assert.True(t, found)
		assert.Equal(t, 10, val)
	})
}

func TestOrderedHashMap_FlatMap(t *testing.T) {
	t.Parallel()

	t.Run("flattens nested maps preserving order", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)
		_ = m.Add(testKey{value: "a"}, 1) //nolint:errcheck
		_ = m.Add(testKey{value: "b"}, 2) //nolint:errcheck

		result := m.FlatMap(func(key testKey, value int) maps.OrderedMap[testKey, int] {
			nested := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)
			_ = nested.Add(testKey{value: key.value + "_1"}, value)   //nolint:errcheck
			_ = nested.Add(testKey{value: key.value + "_2"}, value*2) //nolint:errcheck

			return nested
		})

		assert.Equal(t, 4, result.Size())

		// Verify order: a_1, a_2, b_1, b_2
		expected := []testKey{
			{value: "a_1"},
			{value: "a_2"},
			{value: "b_1"},
			{value: "b_2"},
		}
		idx := 0

		for _, entry := range result.Seq() {
			assert.Equal(t, expected[idx], entry.Key)

			idx++
		}
	})

	t.Run("handles empty map", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)

		result := m.FlatMap(func(key testKey, value int) maps.OrderedMap[testKey, int] {
			nested := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)
			_ = nested.Add(key, value) //nolint:errcheck

			return nested
		})

		assert.Equal(t, 0, result.Size())
	})
}

func TestOrderedHashMap_Exists(t *testing.T) {
	t.Parallel()

	t.Run("returns true when at least one entry matches", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)
		_ = m.Add(testKey{value: "a"}, 1) //nolint:errcheck
		_ = m.Add(testKey{value: "b"}, 2) //nolint:errcheck
		_ = m.Add(testKey{value: "c"}, 3) //nolint:errcheck

		result := m.Exists(func(key testKey, value int) bool {
			return value == 2
		})

		assert.True(t, result)
	})

	t.Run("returns false when no entries match", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)
		_ = m.Add(testKey{value: "a"}, 1) //nolint:errcheck
		_ = m.Add(testKey{value: "b"}, 2) //nolint:errcheck

		result := m.Exists(func(key testKey, value int) bool {
			return value > 10
		})

		assert.False(t, result)
	})

	t.Run("returns false for empty map", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)

		result := m.Exists(func(key testKey, value int) bool {
			return true
		})

		assert.False(t, result)
	})
}

func TestOrderedHashMap_FindFirst(t *testing.T) {
	t.Parallel()

	t.Run("returns first matching entry in insertion order", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)
		_ = m.Add(testKey{value: "a"}, 1) //nolint:errcheck
		_ = m.Add(testKey{value: "b"}, 5) //nolint:errcheck
		_ = m.Add(testKey{value: "c"}, 3) //nolint:errcheck

		result := m.FindFirst(func(key testKey, value int) bool {
			return value > 1
		})

		assert.True(t, result.NonEmpty())
		pair := result.GetOrPanic()
		assert.Equal(t, "b", pair.Key.value)
		assert.Equal(t, 5, pair.Value)
	})

	t.Run("returns None when no match found", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)
		_ = m.Add(testKey{value: "a"}, 1) //nolint:errcheck
		_ = m.Add(testKey{value: "b"}, 2) //nolint:errcheck

		result := m.FindFirst(func(key testKey, value int) bool {
			return value > 10
		})

		assert.True(t, result.Empty())
	})

	t.Run("returns None for empty map", func(t *testing.T) {
		t.Parallel()

		m := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)

		result := m.FindFirst(func(key testKey, value int) bool {
			return true
		})

		assert.True(t, result.Empty())
	})
}
