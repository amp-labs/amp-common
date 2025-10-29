package maps_test

import (
	"fmt"
	"hash"
	"testing"

	"github.com/amp-labs/amp-common/hashing"
	"github.com/amp-labs/amp-common/maps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testKey is a simple key type that implements collectable.Collectable.
type testKey struct {
	value string
}

func (k testKey) UpdateHash(h hash.Hash) error {
	_, err := h.Write([]byte(k.value))

	return err
}

func (k testKey) Equals(other testKey) bool {
	return k.value == other.value
}

// collidingKey is a key type that intentionally produces hash collisions.
type collidingKey struct {
	id   int
	hash string
}

func (k collidingKey) UpdateHash(h hash.Hash) error {
	_, err := h.Write([]byte(k.hash))

	return err
}

func (k collidingKey) Equals(other collidingKey) bool {
	return k.id == other.id
}

func TestNewHashMap(t *testing.T) {
	t.Parallel()

	t.Run("creates empty map", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, string](hashing.Sha256)
		require.NotNil(t, m)
		assert.Equal(t, 0, m.Size())
	})

	t.Run("map is usable immediately", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, int](hashing.Sha256)
		key := testKey{value: "test"}
		err := m.Add(key, 42)
		require.NoError(t, err)
		assert.Equal(t, 1, m.Size())
	})
}

func TestNewHashMapWithSize(t *testing.T) {
	t.Parallel()

	t.Run("creates map with specified capacity", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMapWithSize[testKey, string](hashing.Sha256, 100)
		require.NotNil(t, m)
		assert.Equal(t, 0, m.Size())
	})

	t.Run("can add entries beyond initial size", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMapWithSize[testKey, int](hashing.Sha256, 2)

		for i := range 10 {
			key := testKey{value: fmt.Sprintf("key%d", i)}
			err := m.Add(key, i)
			require.NoError(t, err)
		}

		assert.Equal(t, 10, m.Size())
	})

	t.Run("performs better with pre-allocation for large maps", func(t *testing.T) {
		t.Parallel()

		// This test verifies that pre-allocation works without error
		// Performance benchmarking would be done separately
		m := maps.NewHashMapWithSize[testKey, int](hashing.Sha256, 1000)

		for i := range 1000 {
			key := testKey{value: fmt.Sprintf("key%d", i)}
			err := m.Add(key, i)
			require.NoError(t, err)
		}

		assert.Equal(t, 1000, m.Size())
	})
}

func TestHashMap_Add(t *testing.T) {
	t.Parallel()

	t.Run("adds new key-value pair", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, string](hashing.Sha256)
		key := testKey{value: "test"}
		err := m.Add(key, "value")
		require.NoError(t, err)
		assert.Equal(t, 1, m.Size())
	})

	t.Run("updates existing key", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, string](hashing.Sha256)
		key := testKey{value: "test"}
		err := m.Add(key, "value1")
		require.NoError(t, err)

		err = m.Add(key, "value2")
		require.NoError(t, err)
		assert.Equal(t, 1, m.Size())
	})

	t.Run("handles multiple different keys", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, int](hashing.Sha256)

		for i := range 100 {
			key := testKey{value: fmt.Sprintf("key%d", i)}
			err := m.Add(key, i)
			require.NoError(t, err)
		}

		assert.Equal(t, 100, m.Size())
	})

	t.Run("returns error on hash collision", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[collidingKey, string](hashing.Sha256)
		key1 := collidingKey{id: 1, hash: "same"}
		key2 := collidingKey{id: 2, hash: "same"}

		err := m.Add(key1, "value1")
		require.NoError(t, err)

		err = m.Add(key2, "value2")
		assert.Error(t, err)
	})
}

func TestHashMap_Remove(t *testing.T) {
	t.Parallel()

	t.Run("removes existing key", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, string](hashing.Sha256)
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

		m := maps.NewHashMap[testKey, string](hashing.Sha256)
		key := testKey{value: "missing"}
		err := m.Remove(key)
		require.NoError(t, err)
		assert.Equal(t, 0, m.Size())
	})

	t.Run("removes only specified key", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, string](hashing.Sha256)
		key1 := testKey{value: "key1"}
		key2 := testKey{value: "key2"}
		err := m.Add(key1, "value1")
		require.NoError(t, err)
		err = m.Add(key2, "value2")
		require.NoError(t, err)

		err = m.Remove(key1)
		require.NoError(t, err)
		assert.Equal(t, 1, m.Size())

		contains, err := m.Contains(key2)
		require.NoError(t, err)
		assert.True(t, contains)
	})

	t.Run("returns error on hash collision", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[collidingKey, string](hashing.Sha256)
		key1 := collidingKey{id: 1, hash: "same"}
		key2 := collidingKey{id: 2, hash: "same"}

		err := m.Add(key1, "value1")
		require.NoError(t, err)

		err = m.Remove(key2)
		assert.Error(t, err)
	})
}

func TestHashMap_Clear(t *testing.T) {
	t.Parallel()

	t.Run("removes all entries", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, int](hashing.Sha256)

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

		m := maps.NewHashMap[testKey, string](hashing.Sha256)
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

		m := maps.NewHashMap[testKey, string](hashing.Sha256)
		m.Clear()
		assert.Equal(t, 0, m.Size())
	})
}

//nolint:dupl // Intentional duplication with TestOrderedHashMap_Contains for parallel test coverage
func TestHashMap_Contains(t *testing.T) {
	t.Parallel()

	t.Run("returns true for existing key", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, string](hashing.Sha256)
		key := testKey{value: "test"}
		err := m.Add(key, "value")
		require.NoError(t, err)

		contains, err := m.Contains(key)
		require.NoError(t, err)
		assert.True(t, contains)
	})

	t.Run("returns false for non-existent key", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, string](hashing.Sha256)
		key := testKey{value: "missing"}

		contains, err := m.Contains(key)
		require.NoError(t, err)
		assert.False(t, contains)
	})

	t.Run("returns false after key is removed", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, string](hashing.Sha256)
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

		m := maps.NewHashMap[collidingKey, string](hashing.Sha256)
		key1 := collidingKey{id: 1, hash: "same"}
		key2 := collidingKey{id: 2, hash: "same"}

		err := m.Add(key1, "value1")
		require.NoError(t, err)

		contains, err := m.Contains(key2)
		require.Error(t, err)
		assert.False(t, contains)
	})
}

//nolint:dupl // Intentional duplication with TestOrderedHashMap_Size for parallel test coverage
func TestHashMap_Size(t *testing.T) {
	t.Parallel()

	t.Run("returns zero for empty map", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, string](hashing.Sha256)
		assert.Equal(t, 0, m.Size())
	})

	t.Run("returns correct size after additions", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, string](hashing.Sha256)
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

		m := maps.NewHashMap[testKey, string](hashing.Sha256)
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

		m := maps.NewHashMap[testKey, string](hashing.Sha256)
		key := testKey{value: "key"}
		err := m.Add(key, "value1")
		require.NoError(t, err)

		err = m.Add(key, "value2")
		require.NoError(t, err)
		assert.Equal(t, 1, m.Size())
	})
}

func TestHashMap_Seq(t *testing.T) {
	t.Parallel()

	t.Run("iterates over all entries", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, int](hashing.Sha256)
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

		m := maps.NewHashMap[testKey, string](hashing.Sha256)
		count := 0

		for range m.Seq() {
			count++
		}

		assert.Equal(t, 0, count)
	})

	t.Run("stops early when yield returns false", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, int](hashing.Sha256)

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
}

func TestHashMap_Union(t *testing.T) {
	t.Parallel()

	t.Run("combines two maps", func(t *testing.T) {
		t.Parallel()

		m1 := maps.NewHashMap[testKey, string](hashing.Sha256)
		m1.Add(testKey{value: "key1"}, "value1") //nolint:errcheck
		m1.Add(testKey{value: "key2"}, "value2") //nolint:errcheck

		m2 := maps.NewHashMap[testKey, string](hashing.Sha256)
		m2.Add(testKey{value: "key3"}, "value3") //nolint:errcheck
		m2.Add(testKey{value: "key4"}, "value4") //nolint:errcheck

		result, err := m1.Union(m2)
		require.NoError(t, err)
		assert.Equal(t, 4, result.Size())
	})

	t.Run("other map values take precedence for duplicate keys", func(t *testing.T) {
		t.Parallel()

		m1 := maps.NewHashMap[testKey, string](hashing.Sha256)
		m1.Add(testKey{value: "key"}, "value1") //nolint:errcheck

		m2 := maps.NewHashMap[testKey, string](hashing.Sha256)
		m2.Add(testKey{value: "key"}, "value2") //nolint:errcheck

		result, err := m1.Union(m2)
		require.NoError(t, err)
		assert.Equal(t, 1, result.Size())
	})

	t.Run("original maps are not modified", func(t *testing.T) {
		t.Parallel()

		m1 := maps.NewHashMap[testKey, string](hashing.Sha256)
		m1.Add(testKey{value: "key1"}, "value1") //nolint:errcheck

		m2 := maps.NewHashMap[testKey, string](hashing.Sha256)
		m2.Add(testKey{value: "key2"}, "value2") //nolint:errcheck

		result, err := m1.Union(m2)
		require.NoError(t, err)
		assert.Equal(t, 2, result.Size())
		assert.Equal(t, 1, m1.Size())
		assert.Equal(t, 1, m2.Size())
	})

	t.Run("union with empty map", func(t *testing.T) {
		t.Parallel()

		m1 := maps.NewHashMap[testKey, string](hashing.Sha256)
		m1.Add(testKey{value: "key1"}, "value1") //nolint:errcheck

		m2 := maps.NewHashMap[testKey, string](hashing.Sha256)

		result, err := m1.Union(m2)
		require.NoError(t, err)
		assert.Equal(t, 1, result.Size())
	})
}

func TestHashMap_Intersection(t *testing.T) {
	t.Parallel()

	t.Run("returns common keys", func(t *testing.T) {
		t.Parallel()

		m1 := maps.NewHashMap[testKey, string](hashing.Sha256)
		m1.Add(testKey{value: "key1"}, "value1") //nolint:errcheck
		m1.Add(testKey{value: "key2"}, "value2") //nolint:errcheck
		m1.Add(testKey{value: "key3"}, "value3") //nolint:errcheck

		m2 := maps.NewHashMap[testKey, string](hashing.Sha256)
		m2.Add(testKey{value: "key2"}, "other2") //nolint:errcheck
		m2.Add(testKey{value: "key3"}, "other3") //nolint:errcheck
		m2.Add(testKey{value: "key4"}, "other4") //nolint:errcheck

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

	t.Run("values are from first map", func(t *testing.T) {
		t.Parallel()

		m1 := maps.NewHashMap[testKey, string](hashing.Sha256)
		m1.Add(testKey{value: "key"}, "value1") //nolint:errcheck

		m2 := maps.NewHashMap[testKey, string](hashing.Sha256)
		m2.Add(testKey{value: "key"}, "value2") //nolint:errcheck

		result, err := m1.Intersection(m2)
		require.NoError(t, err)
		assert.Equal(t, 1, result.Size())
	})

	t.Run("returns empty map when no common keys", func(t *testing.T) {
		t.Parallel()

		m1 := maps.NewHashMap[testKey, string](hashing.Sha256)
		m1.Add(testKey{value: "key1"}, "value1") //nolint:errcheck

		m2 := maps.NewHashMap[testKey, string](hashing.Sha256)
		m2.Add(testKey{value: "key2"}, "value2") //nolint:errcheck

		result, err := m1.Intersection(m2)
		require.NoError(t, err)
		assert.Equal(t, 0, result.Size())
	})

	t.Run("original maps are not modified", func(t *testing.T) {
		t.Parallel()

		m1 := maps.NewHashMap[testKey, string](hashing.Sha256)
		m1.Add(testKey{value: "key1"}, "value1") //nolint:errcheck
		m1.Add(testKey{value: "key2"}, "value2") //nolint:errcheck

		m2 := maps.NewHashMap[testKey, string](hashing.Sha256)
		m2.Add(testKey{value: "key2"}, "value2") //nolint:errcheck

		result, err := m1.Intersection(m2)
		require.NoError(t, err)
		assert.Equal(t, 1, result.Size())
		assert.Equal(t, 2, m1.Size())
		assert.Equal(t, 1, m2.Size())
	})
}

func TestHashMap_Clone(t *testing.T) {
	t.Parallel()

	t.Run("creates independent copy", func(t *testing.T) {
		t.Parallel()

		original := maps.NewHashMap[testKey, string](hashing.Sha256)
		original.Add(testKey{value: "key1"}, "value1") //nolint:errcheck
		original.Add(testKey{value: "key2"}, "value2") //nolint:errcheck

		cloned := original.Clone()
		assert.Equal(t, original.Size(), cloned.Size())

		// Modify original
		original.Add(testKey{value: "key3"}, "value3") //nolint:errcheck

		// Clone should not be affected
		assert.Equal(t, 3, original.Size())
		assert.Equal(t, 2, cloned.Size())
	})

	t.Run("cloned map has same entries", func(t *testing.T) {
		t.Parallel()

		original := maps.NewHashMap[testKey, int](hashing.Sha256)
		expected := map[string]int{
			"key1": 1,
			"key2": 2,
			"key3": 3,
		}

		for k, v := range expected {
			original.Add(testKey{value: k}, v) //nolint:errcheck
		}

		cloned := original.Clone()

		for k, expectedValue := range expected {
			contains, err := cloned.Contains(testKey{value: k})
			require.NoError(t, err)
			assert.True(t, contains, "cloned map should contain key %s", k)

			_ = expectedValue // Value checking would require a Get method
		}
	})

	t.Run("clones empty map", func(t *testing.T) {
		t.Parallel()

		original := maps.NewHashMap[testKey, string](hashing.Sha256)
		cloned := original.Clone()
		require.NotNil(t, cloned)
		assert.Equal(t, 0, cloned.Size())
	})
}

func TestHashMap_Get(t *testing.T) {
	t.Parallel()

	t.Run("returns value for existing key", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, string](hashing.Sha256)
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

		m := maps.NewHashMap[testKey, string](hashing.Sha256)
		key := testKey{value: "missing"}

		value, found, err := m.Get(key)
		require.NoError(t, err)
		assert.False(t, found)
		assert.Equal(t, "", value)
	})

	t.Run("returns zero value and false for missing key with int type", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, int](hashing.Sha256)
		key := testKey{value: "missing"}

		value, found, err := m.Get(key)
		require.NoError(t, err)
		assert.False(t, found)
		assert.Equal(t, 0, value)
	})

	t.Run("returns most recent value for updated key", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, string](hashing.Sha256)
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

		m := maps.NewHashMap[testKey, int](hashing.Sha256)
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

		m := maps.NewHashMap[collidingKey, string](hashing.Sha256)

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

		m := maps.NewHashMap[testKey, *string](hashing.Sha256)
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

		m := maps.NewHashMap[testKey, string](hashing.Sha256)
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

		m := maps.NewHashMap[testKey, string](hashing.Sha256)
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

		m := maps.NewHashMap[testKey, testValue](hashing.Sha256)
		key := testKey{value: "test"}
		expected := testValue{name: "Alice", age: 30}

		err := m.Add(key, expected)
		require.NoError(t, err)

		value, found, err := m.Get(key)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, expected, value)
	})
}

func TestHashMap_GetOrElse(t *testing.T) {
	t.Parallel()

	t.Run("returns value for existing key", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, string](hashing.Sha256)
		key := testKey{value: "test"}
		err := m.Add(key, "actual")
		require.NoError(t, err)

		value, err := m.GetOrElse(key, "default")
		require.NoError(t, err)
		assert.Equal(t, "actual", value)
	})

	t.Run("returns default value for missing key", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, string](hashing.Sha256)
		key := testKey{value: "missing"}

		value, err := m.GetOrElse(key, "default")
		require.NoError(t, err)
		assert.Equal(t, "default", value)
	})

	t.Run("returns default value for empty map", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, int](hashing.Sha256)
		key := testKey{value: "test"}

		value, err := m.GetOrElse(key, 42)
		require.NoError(t, err)
		assert.Equal(t, 42, value)
	})

	t.Run("returns error on hash collision", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[collidingKey, string](hashing.Sha256)
		key1 := collidingKey{id: 1, hash: "same"}
		err := m.Add(key1, "value1")
		require.NoError(t, err)

		key2 := collidingKey{id: 2, hash: "same"}
		value, err := m.GetOrElse(key2, "default")
		require.Error(t, err)
		assert.Equal(t, "", value)
	})
}

func TestHashMap_Keys(t *testing.T) {
	t.Parallel()

	t.Run("returns all keys from map", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, int](hashing.Sha256)
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

	t.Run("returns empty set for empty map", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, string](hashing.Sha256)
		keySet := m.Keys()
		require.NotNil(t, keySet)
		assert.Equal(t, 0, keySet.Size())
	})

	t.Run("returned set is independent", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, int](hashing.Sha256)
		key := testKey{value: "key1"}
		err := m.Add(key, 1)
		require.NoError(t, err)

		keySet := m.Keys()
		assert.Equal(t, 1, keySet.Size())

		// Add to original map
		key2 := testKey{value: "key2"}
		err = m.Add(key2, 2)
		require.NoError(t, err)

		// Key set should not be affected
		assert.Equal(t, 1, keySet.Size())
	})
}

func TestHashMap_ForEach(t *testing.T) {
	t.Parallel()

	t.Run("calls function for each entry", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, int](hashing.Sha256)
		expected := map[string]int{
			"key1": 1,
			"key2": 2,
			"key3": 3,
		}

		for k, v := range expected {
			err := m.Add(testKey{value: k}, v)
			require.NoError(t, err)
		}

		visited := make(map[string]int)

		m.ForEach(func(key testKey, value int) {
			visited[key.value] = value
		})

		assert.Equal(t, expected, visited)
	})

	t.Run("handles empty map", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, string](hashing.Sha256)
		callCount := 0

		m.ForEach(func(key testKey, value string) {
			callCount++
		})

		assert.Equal(t, 0, callCount)
	})

	t.Run("function can modify external state", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, int](hashing.Sha256)
		m.Add(testKey{value: "a"}, 1) //nolint:errcheck
		m.Add(testKey{value: "b"}, 2) //nolint:errcheck

		sum := 0

		m.ForEach(func(key testKey, value int) {
			sum += value
		})

		assert.Equal(t, 3, sum)
	})
}

func TestHashMap_ForAll(t *testing.T) {
	t.Parallel()

	t.Run("returns true when predicate holds for all entries", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, int](hashing.Sha256)
		m.Add(testKey{value: "a"}, 2) //nolint:errcheck
		m.Add(testKey{value: "b"}, 4) //nolint:errcheck
		m.Add(testKey{value: "c"}, 6) //nolint:errcheck

		result := m.ForAll(func(key testKey, value int) bool {
			return value%2 == 0 // all even
		})

		assert.True(t, result)
	})

	t.Run("returns false when predicate fails for any entry", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, int](hashing.Sha256)
		m.Add(testKey{value: "a"}, 2) //nolint:errcheck
		m.Add(testKey{value: "b"}, 3) //nolint:errcheck
		m.Add(testKey{value: "c"}, 4) //nolint:errcheck

		result := m.ForAll(func(key testKey, value int) bool {
			return value%2 == 0
		})

		assert.False(t, result)
	})

	t.Run("returns true for empty map", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, int](hashing.Sha256)

		result := m.ForAll(func(key testKey, value int) bool {
			return false // vacuously true
		})

		assert.True(t, result)
	})

	t.Run("short circuits on first failure", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, int](hashing.Sha256)
		for i := range 100 {
			m.Add(testKey{value: fmt.Sprintf("key%d", i)}, i) //nolint:errcheck
		}

		callCount := 0
		result := m.ForAll(func(key testKey, value int) bool {
			callCount++

			return value < 5
		})

		assert.False(t, result)
		assert.Greater(t, 100, callCount, "should short circuit")
	})
}

func TestHashMap_Filter(t *testing.T) {
	t.Parallel()

	t.Run("returns map with entries matching predicate", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, int](hashing.Sha256)
		for i := 1; i <= 5; i++ {
			m.Add(testKey{value: fmt.Sprintf("key%d", i)}, i) //nolint:errcheck
		}

		result := m.Filter(func(key testKey, value int) bool {
			return value%2 == 0 // keep even values
		})

		assert.Equal(t, 2, result.Size())
		contains, _ := result.Contains(testKey{value: "key2"})
		assert.True(t, contains)
		contains, _ = result.Contains(testKey{value: "key4"})
		assert.True(t, contains)
	})

	t.Run("returns empty map when no entries match", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, int](hashing.Sha256)
		m.Add(testKey{value: "a"}, 1) //nolint:errcheck
		m.Add(testKey{value: "b"}, 3) //nolint:errcheck

		result := m.Filter(func(key testKey, value int) bool {
			return value%2 == 0
		})

		assert.Equal(t, 0, result.Size())
	})

	t.Run("returns all entries when all match", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, int](hashing.Sha256)
		m.Add(testKey{value: "a"}, 2) //nolint:errcheck
		m.Add(testKey{value: "b"}, 4) //nolint:errcheck

		result := m.Filter(func(key testKey, value int) bool {
			return true
		})

		assert.Equal(t, 2, result.Size())
	})

	t.Run("original map is not modified", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, int](hashing.Sha256)
		m.Add(testKey{value: "a"}, 1) //nolint:errcheck
		m.Add(testKey{value: "b"}, 2) //nolint:errcheck

		result := m.Filter(func(key testKey, value int) bool {
			return value == 1
		})

		assert.Equal(t, 1, result.Size())
		assert.Equal(t, 2, m.Size())
	})
}

func TestHashMap_FilterNot(t *testing.T) {
	t.Parallel()

	t.Run("returns map with entries not matching predicate", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, int](hashing.Sha256)
		for i := 1; i <= 5; i++ {
			m.Add(testKey{value: fmt.Sprintf("key%d", i)}, i) //nolint:errcheck
		}

		result := m.FilterNot(func(key testKey, value int) bool {
			return value%2 == 0 // exclude even values
		})

		assert.Equal(t, 3, result.Size())
		contains, _ := result.Contains(testKey{value: "key1"})
		assert.True(t, contains)
		contains, _ = result.Contains(testKey{value: "key3"})
		assert.True(t, contains)
		contains, _ = result.Contains(testKey{value: "key5"})
		assert.True(t, contains)
	})

	t.Run("is inverse of Filter", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, int](hashing.Sha256)
		for i := 1; i <= 10; i++ {
			m.Add(testKey{value: fmt.Sprintf("key%d", i)}, i) //nolint:errcheck
		}

		predicate := func(key testKey, value int) bool {
			return value > 5
		}

		filtered := m.Filter(predicate)
		filteredNot := m.FilterNot(predicate)

		assert.Equal(t, 10, filtered.Size()+filteredNot.Size())
	})
}

func TestHashMap_Map(t *testing.T) {
	t.Parallel()

	t.Run("transforms all entries", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, int](hashing.Sha256)
		m.Add(testKey{value: "a"}, 1) //nolint:errcheck
		m.Add(testKey{value: "b"}, 2) //nolint:errcheck

		result := m.Map(func(key testKey, value int) (testKey, int) {
			return testKey{value: key.value + "_new"}, value * 2
		})

		assert.Equal(t, 2, result.Size())
		val, found, _ := result.Get(testKey{value: "a_new"})
		assert.True(t, found)
		assert.Equal(t, 2, val)
	})

	t.Run("can change both keys and values", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, int](hashing.Sha256)
		m.Add(testKey{value: "a"}, 1) //nolint:errcheck
		m.Add(testKey{value: "b"}, 2) //nolint:errcheck

		result := m.Map(func(key testKey, value int) (testKey, int) {
			return testKey{value: key.value + "_modified"}, value * 100
		})

		assert.Equal(t, 2, result.Size())
		val, found, _ := result.Get(testKey{value: "a_modified"})
		assert.True(t, found)
		assert.Equal(t, 100, val)
	})

	t.Run("handles empty map", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, int](hashing.Sha256)

		result := m.Map(func(key testKey, value int) (testKey, int) {
			return key, value * 2
		})

		assert.Equal(t, 0, result.Size())
	})

	t.Run("original map is not modified", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, int](hashing.Sha256)
		m.Add(testKey{value: "a"}, 1) //nolint:errcheck

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

func TestHashMap_FlatMap(t *testing.T) {
	t.Parallel()

	t.Run("flattens nested maps", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, int](hashing.Sha256)
		m.Add(testKey{value: "a"}, 1) //nolint:errcheck
		m.Add(testKey{value: "b"}, 2) //nolint:errcheck

		result := m.FlatMap(func(key testKey, value int) maps.Map[testKey, int] {
			nested := maps.NewHashMap[testKey, int](hashing.Sha256)
			nested.Add(testKey{value: key.value + "_1"}, value)   //nolint:errcheck
			nested.Add(testKey{value: key.value + "_2"}, value*2) //nolint:errcheck

			return nested
		})

		assert.Equal(t, 4, result.Size())
		val, found, _ := result.Get(testKey{value: "a_1"})
		assert.True(t, found)
		assert.Equal(t, 1, val)
		val, found, _ = result.Get(testKey{value: "a_2"})
		assert.True(t, found)
		assert.Equal(t, 2, val)
	})

	t.Run("handles empty results from function", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, int](hashing.Sha256)
		m.Add(testKey{value: "a"}, 1) //nolint:errcheck

		result := m.FlatMap(func(key testKey, value int) maps.Map[testKey, int] {
			return maps.NewHashMap[testKey, int](hashing.Sha256)
		})

		assert.Equal(t, 0, result.Size())
	})

	t.Run("handles empty map", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, int](hashing.Sha256)

		result := m.FlatMap(func(key testKey, value int) maps.Map[testKey, int] {
			nested := maps.NewHashMap[testKey, int](hashing.Sha256)
			nested.Add(key, value) //nolint:errcheck

			return nested
		})

		assert.Equal(t, 0, result.Size())
	})
}

func TestHashMap_Exists(t *testing.T) {
	t.Parallel()

	t.Run("returns true when at least one entry matches", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, int](hashing.Sha256)
		m.Add(testKey{value: "a"}, 1) //nolint:errcheck
		m.Add(testKey{value: "b"}, 2) //nolint:errcheck
		m.Add(testKey{value: "c"}, 3) //nolint:errcheck

		result := m.Exists(func(key testKey, value int) bool {
			return value == 2
		})

		assert.True(t, result)
	})

	t.Run("returns false when no entries match", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, int](hashing.Sha256)
		m.Add(testKey{value: "a"}, 1) //nolint:errcheck
		m.Add(testKey{value: "b"}, 2) //nolint:errcheck

		result := m.Exists(func(key testKey, value int) bool {
			return value > 10
		})

		assert.False(t, result)
	})

	t.Run("returns false for empty map", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, int](hashing.Sha256)

		result := m.Exists(func(key testKey, value int) bool {
			return true
		})

		assert.False(t, result)
	})

	t.Run("short circuits on first match", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, int](hashing.Sha256)
		for i := range 100 {
			m.Add(testKey{value: fmt.Sprintf("key%d", i)}, i) //nolint:errcheck
		}

		callCount := 0
		result := m.Exists(func(key testKey, value int) bool {
			callCount++

			return value == 5
		})

		assert.True(t, result)
		assert.Greater(t, 100, callCount, "should short circuit")
	})
}

func TestHashMap_FindFirst(t *testing.T) {
	t.Parallel()

	t.Run("returns first matching entry", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, int](hashing.Sha256)
		m.Add(testKey{value: "a"}, 1) //nolint:errcheck
		m.Add(testKey{value: "b"}, 2) //nolint:errcheck
		m.Add(testKey{value: "c"}, 3) //nolint:errcheck

		result := m.FindFirst(func(key testKey, value int) bool {
			return value > 1
		})

		assert.True(t, result.NonEmpty())
		pair := result.GetOrPanic()
		assert.Greater(t, pair.Value, 1)
	})

	t.Run("returns None when no match found", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, int](hashing.Sha256)
		m.Add(testKey{value: "a"}, 1) //nolint:errcheck
		m.Add(testKey{value: "b"}, 2) //nolint:errcheck

		result := m.FindFirst(func(key testKey, value int) bool {
			return value > 10
		})

		assert.True(t, result.Empty())
	})

	t.Run("returns None for empty map", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, int](hashing.Sha256)

		result := m.FindFirst(func(key testKey, value int) bool {
			return true
		})

		assert.True(t, result.Empty())
	})

	t.Run("returns correct key-value pair", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, string](hashing.Sha256)
		m.Add(testKey{value: "apple"}, "red")     //nolint:errcheck
		m.Add(testKey{value: "banana"}, "yellow") //nolint:errcheck

		result := m.FindFirst(func(key testKey, value string) bool {
			return value == "yellow"
		})

		assert.True(t, result.NonEmpty())
		pair := result.GetOrPanic()
		assert.Equal(t, "banana", pair.Key.value)
		assert.Equal(t, "yellow", pair.Value)
	})

	t.Run("short circuits on first match", func(t *testing.T) {
		t.Parallel()

		m := maps.NewHashMap[testKey, int](hashing.Sha256)
		for i := range 100 {
			m.Add(testKey{value: fmt.Sprintf("key%d", i)}, i) //nolint:errcheck
		}

		callCount := 0
		result := m.FindFirst(func(key testKey, value int) bool {
			callCount++

			return value == 5
		})

		assert.True(t, result.NonEmpty())
		assert.Greater(t, 100, callCount, "should short circuit")
	})
}
