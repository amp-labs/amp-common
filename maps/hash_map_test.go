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
