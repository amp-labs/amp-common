package maps_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/amp-labs/amp-common/hashing"
	"github.com/amp-labs/amp-common/maps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const defaultValueOrdered = "default"

var errGenerationFailedOrdered = errors.New("generation failed")

//nolint:dupl // Test structure intentionally mirrors DefaultMap tests for consistency
func TestNewDefaultOrderedMap(t *testing.T) {
	t.Parallel()

	t.Run("creates default ordered map with ordered hash map storage", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		m := maps.NewDefaultOrderedMap(baseMap, func(k testKey) (string, error) {
			return defaultValueOrdered, nil
		})

		require.NotNil(t, m)
		assert.Equal(t, 0, m.Size())
	})

	t.Run("wrapping existing default ordered map replaces default function", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		m1 := maps.NewDefaultOrderedMap(baseMap, func(k testKey) (string, error) {
			return "first", nil
		})
		m2 := maps.NewDefaultOrderedMap(m1, func(k testKey) (string, error) {
			return "second", nil
		})

		key := testKey{value: "missing"}
		val, found, err := m2.Get(key)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "second", val)
	})
}

//nolint:dupl // Test structure intentionally mirrors DefaultMapGet tests for consistency
func TestDefaultOrderedMapGet(t *testing.T) {
	t.Parallel()

	t.Run("returns existing value when key exists", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		m := maps.NewDefaultOrderedMap(baseMap, func(k testKey) (string, error) {
			return defaultValueOrdered, nil
		})

		key := testKey{value: "test"}
		err := m.Add(key, "actual")
		require.NoError(t, err)

		val, found, err := m.Get(key)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "actual", val)
	})

	t.Run("generates and adds default value for missing key", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)
		m := maps.NewDefaultOrderedMap(baseMap, func(k testKey) (int, error) {
			return 42, nil
		})

		key := testKey{value: "missing"}
		val, found, err := m.Get(key)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, 42, val)
		assert.Equal(t, 1, m.Size())
	})

	t.Run("returns false when default function returns ErrNoDefaultValue", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		m := maps.NewDefaultOrderedMap(baseMap, func(k testKey) (string, error) {
			return "", maps.ErrNoDefaultValue
		})

		key := testKey{value: "missing"}
		val, found, err := m.Get(key)
		require.NoError(t, err)
		assert.False(t, found)
		assert.Equal(t, "", val)
		assert.Equal(t, 0, m.Size())
	})

	t.Run("returns error when default function fails", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		m := maps.NewDefaultOrderedMap(baseMap, func(k testKey) (string, error) {
			return "", errGenerationFailedOrdered
		})

		key := testKey{value: "missing"}
		val, found, err := m.Get(key)
		require.Error(t, err)
		assert.False(t, found)
		assert.Equal(t, "", val)
		assert.ErrorIs(t, err, errGenerationFailedOrdered)
	})

	t.Run("default function receives correct key", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)

		var receivedKey testKey

		m := maps.NewDefaultOrderedMap(baseMap, func(k testKey) (string, error) {
			receivedKey = k

			return k.value + "-default", nil
		})

		key := testKey{value: "mykey"}
		val, found, err := m.Get(key)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "mykey-default", val)
		assert.Equal(t, key, receivedKey)
	})
}

func TestDefaultOrderedMapAdd(t *testing.T) {
	t.Parallel()

	t.Run("adds new key-value pair", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		m := maps.NewDefaultOrderedMap(baseMap, func(k testKey) (string, error) {
			return defaultValueOrdered, nil
		})

		key := testKey{value: "test"}
		err := m.Add(key, "value")
		require.NoError(t, err)
		assert.Equal(t, 1, m.Size())

		val, found, err := m.Get(key)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "value", val)
	})

	t.Run("updates existing key without changing order", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)
		m := maps.NewDefaultOrderedMap(baseMap, func(k testKey) (int, error) {
			return 0, nil
		})

		key := testKey{value: "test"}
		err := m.Add(key, 10)
		require.NoError(t, err)
		err = m.Add(key, 20)
		require.NoError(t, err)
		assert.Equal(t, 1, m.Size())

		val, found, err := m.Get(key)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, 20, val)
	})

	t.Run("maintains insertion order", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)
		//nolint:varnamelen // Short name acceptable in test context
		m := maps.NewDefaultOrderedMap(baseMap, func(k testKey) (int, error) {
			return 0, nil
		})

		keys := []testKey{
			{value: "first"},
			{value: "second"},
			{value: "third"},
		}

		for i, key := range keys {
			err := m.Add(key, i)
			require.NoError(t, err)
		}

		idx := 0
		for i, entry := range m.Seq() {
			assert.Equal(t, idx, i)
			assert.Equal(t, keys[idx], entry.Key)

			idx++
		}
	})

	t.Run("bypasses default function", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		called := false
		m := maps.NewDefaultOrderedMap(baseMap, func(k testKey) (string, error) {
			called = true

			return defaultValueOrdered, nil
		})

		key := testKey{value: "test"}
		err := m.Add(key, "direct")
		require.NoError(t, err)
		assert.False(t, called)
	})
}

func TestDefaultOrderedMapRemove(t *testing.T) {
	t.Parallel()

	t.Run("removes existing key", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		m := maps.NewDefaultOrderedMap(baseMap, func(k testKey) (string, error) {
			return defaultValueOrdered, nil
		})

		key := testKey{value: "test"}
		err := m.Add(key, "value")
		require.NoError(t, err)
		err = m.Remove(key)
		require.NoError(t, err)
		assert.Equal(t, 0, m.Size())
	})

	t.Run("no-op for non-existent key", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		m := maps.NewDefaultOrderedMap(baseMap, func(k testKey) (string, error) {
			return defaultValueOrdered, nil
		})

		key := testKey{value: "missing"}
		err := m.Remove(key)
		require.NoError(t, err)
		assert.Equal(t, 0, m.Size())
	})
}

func TestDefaultOrderedMapClear(t *testing.T) {
	t.Parallel()

	t.Run("removes all entries", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)
		m := maps.NewDefaultOrderedMap(baseMap, func(k testKey) (int, error) {
			return 0, nil
		})

		for i := range 10 {
			key := testKey{value: fmt.Sprintf("key%d", i)}
			err := m.Add(key, i)
			require.NoError(t, err)
		}

		assert.Equal(t, 10, m.Size())
		m.Clear()
		assert.Equal(t, 0, m.Size())
	})
}

func TestDefaultOrderedMapContains(t *testing.T) {
	t.Parallel()

	t.Run("returns true for existing key", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		m := maps.NewDefaultOrderedMap(baseMap, func(k testKey) (string, error) {
			return defaultValueOrdered, nil
		})

		key := testKey{value: "test"}
		err := m.Add(key, "value")
		require.NoError(t, err)

		contains, err := m.Contains(key)
		require.NoError(t, err)
		assert.True(t, contains)
	})

	t.Run("generates default and returns true for missing key", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		m := maps.NewDefaultOrderedMap(baseMap, func(k testKey) (string, error) {
			return "generated", nil
		})

		key := testKey{value: "missing"}
		contains, err := m.Contains(key)
		require.NoError(t, err)
		assert.True(t, contains)
		assert.Equal(t, 1, m.Size())

		// Verify value was actually added
		val, found, err := m.Get(key)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "generated", val)
	})

	t.Run("returns false when default function returns ErrNoDefaultValue", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		m := maps.NewDefaultOrderedMap(baseMap, func(k testKey) (string, error) {
			return "", maps.ErrNoDefaultValue
		})

		key := testKey{value: "missing"}
		contains, err := m.Contains(key)
		require.NoError(t, err)
		assert.False(t, contains)
		assert.Equal(t, 0, m.Size())
	})

	t.Run("default values added at end of insertion order", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		//nolint:varnamelen // Short name acceptable in test context
		m := maps.NewDefaultOrderedMap(baseMap, func(k testKey) (string, error) {
			return "default-" + k.value, nil
		})

		// Add some explicit entries
		err := m.Add(testKey{value: "first"}, "val1")
		require.NoError(t, err)
		err = m.Add(testKey{value: "second"}, "val2")
		require.NoError(t, err)

		// Trigger default value generation via Contains
		_, err = m.Contains(testKey{value: "third"})
		require.NoError(t, err)

		// Check order
		expectedKeys := []string{"first", "second", "third"}
		idx := 0

		for _, entry := range m.Seq() {
			assert.Equal(t, expectedKeys[idx], entry.Key.value)

			idx++
		}
	})
}

func TestDefaultOrderedMapSize(t *testing.T) {
	t.Parallel()

	t.Run("returns correct size", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)
		m := maps.NewDefaultOrderedMap(baseMap, func(k testKey) (int, error) {
			return 0, nil
		})

		assert.Equal(t, 0, m.Size())

		for i := range 5 {
			key := testKey{value: fmt.Sprintf("key%d", i)}
			err := m.Add(key, i)
			require.NoError(t, err)
			assert.Equal(t, i+1, m.Size())
		}
	})
}

func TestDefaultOrderedMapSeq(t *testing.T) {
	t.Parallel()

	t.Run("iterates in insertion order", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)
		defaultMap := maps.NewDefaultOrderedMap(baseMap, func(k testKey) (int, error) {
			return 0, nil
		})

		keys := []testKey{
			{value: "alpha"},
			{value: "beta"},
			{value: "gamma"},
			{value: "delta"},
		}

		for i, key := range keys {
			err := defaultMap.Add(key, i*10)
			require.NoError(t, err)
		}

		idx := 0
		for i, entry := range defaultMap.Seq() {
			assert.Equal(t, idx, i)
			assert.Equal(t, keys[idx], entry.Key)
			assert.Equal(t, idx*10, entry.Value)

			idx++
		}

		assert.Equal(t, 4, idx)
	})

	t.Run("includes default-generated values in order", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		//nolint:varnamelen // Short name acceptable in test context
		m := maps.NewDefaultOrderedMap(baseMap, func(k testKey) (string, error) {
			return "default-" + k.value, nil
		})

		// Add explicit entries
		err := m.Add(testKey{value: "first"}, "val1")
		require.NoError(t, err)

		// Trigger default generation
		_, _, err = m.Get(testKey{value: "second"})
		require.NoError(t, err)

		// Add another explicit entry
		err = m.Add(testKey{value: "third"}, "val3")
		require.NoError(t, err)

		expectedKeys := []string{"first", "second", "third"}
		idx := 0

		for _, entry := range m.Seq() {
			assert.Equal(t, expectedKeys[idx], entry.Key.value)

			idx++
		}
	})
}

func TestDefaultOrderedMapUnion(t *testing.T) {
	t.Parallel()

	t.Run("combines two maps preserving order", func(t *testing.T) {
		t.Parallel()

		baseMap1 := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)
		defaultMap1 := maps.NewDefaultOrderedMap(baseMap1, func(k testKey) (int, error) {
			return 99, nil
		})

		baseMap2 := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)

		err := defaultMap1.Add(testKey{value: "a"}, 1)
		require.NoError(t, err)
		err = defaultMap1.Add(testKey{value: "b"}, 2)
		require.NoError(t, err)

		err = baseMap2.Add(testKey{value: "c"}, 3)
		require.NoError(t, err)
		err = baseMap2.Add(testKey{value: "b"}, 20)
		require.NoError(t, err)

		result, err := defaultMap1.Union(baseMap2)
		require.NoError(t, err)
		assert.Equal(t, 3, result.Size())

		val, found, err := result.Get(testKey{value: "a"})
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, 1, val)

		val, found, err = result.Get(testKey{value: "b"})
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, 20, val) // Value from other map
	})

	t.Run("preserves default function", func(t *testing.T) {
		t.Parallel()

		baseMap1 := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)
		m1 := maps.NewDefaultOrderedMap(baseMap1, func(k testKey) (int, error) {
			return 99, nil
		})

		baseMap2 := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)

		result, err := m1.Union(baseMap2)
		require.NoError(t, err)

		// Test that default function works on result
		val, found, err := result.Get(testKey{value: "missing"})
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, 99, val)
	})
}

func TestDefaultOrderedMapIntersection(t *testing.T) {
	t.Parallel()

	t.Run("returns only common keys preserving order", func(t *testing.T) {
		t.Parallel()

		baseMap1 := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)
		defaultMap1 := maps.NewDefaultOrderedMap(baseMap1, func(k testKey) (int, error) {
			return 99, nil
		})

		baseMap2 := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)

		err := defaultMap1.Add(testKey{value: "a"}, 1)
		require.NoError(t, err)
		err = defaultMap1.Add(testKey{value: "b"}, 2)
		require.NoError(t, err)
		err = defaultMap1.Add(testKey{value: "c"}, 3)
		require.NoError(t, err)

		err = baseMap2.Add(testKey{value: "b"}, 20)
		require.NoError(t, err)
		err = baseMap2.Add(testKey{value: "c"}, 30)
		require.NoError(t, err)
		err = baseMap2.Add(testKey{value: "d"}, 40)
		require.NoError(t, err)

		result, err := defaultMap1.Intersection(baseMap2)
		require.NoError(t, err)
		assert.Equal(t, 2, result.Size())

		val, found, err := result.Get(testKey{value: "b"})
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, 2, val) // Value from this map, not other

		val, found, err = result.Get(testKey{value: "c"})
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, 3, val)

		// Check order preserved from m1
		expectedKeys := []string{"b", "c"}
		idx := 0

		for _, entry := range result.Seq() {
			assert.Equal(t, expectedKeys[idx], entry.Key.value)

			idx++
		}
	})
}

func TestDefaultOrderedMapClone(t *testing.T) {
	t.Parallel()

	t.Run("creates independent copy", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)
		//nolint:varnamelen // Short name acceptable in test context
		m := maps.NewDefaultOrderedMap(baseMap, func(k testKey) (int, error) {
			return 99, nil
		})

		err := m.Add(testKey{value: "a"}, 1)
		require.NoError(t, err)
		err = m.Add(testKey{value: "b"}, 2)
		require.NoError(t, err)

		clone := m.Clone()
		assert.Equal(t, 2, clone.Size())

		// Modify original
		err = m.Add(testKey{value: "c"}, 3)
		require.NoError(t, err)

		// Clone should be unchanged
		assert.Equal(t, 3, m.Size())
		assert.Equal(t, 2, clone.Size())
	})

	t.Run("preserves insertion order", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)
		m := maps.NewDefaultOrderedMap(baseMap, func(k testKey) (int, error) {
			return 99, nil
		})

		keys := []testKey{
			{value: "first"},
			{value: "second"},
			{value: "third"},
		}

		for i, key := range keys {
			err := m.Add(key, i)
			require.NoError(t, err)
		}

		clone := m.Clone()

		idx := 0
		for _, entry := range clone.Seq() {
			assert.Equal(t, keys[idx], entry.Key)

			idx++
		}
	})

	t.Run("preserves default function", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewOrderedHashMap[testKey, int](hashing.Sha256)
		m := maps.NewDefaultOrderedMap(baseMap, func(k testKey) (int, error) {
			return 99, nil
		})

		clone := m.Clone()

		val, found, err := clone.Get(testKey{value: "missing"})
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, 99, val)
	})
}

func TestDefaultOrderedMapHashFunction(t *testing.T) {
	t.Parallel()

	t.Run("returns underlying hash function", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewOrderedHashMap[testKey, string](hashing.Sha256)
		m := maps.NewDefaultOrderedMap(baseMap, func(k testKey) (string, error) {
			return defaultValueOrdered, nil
		})

		hashFunc := m.HashFunction()
		assert.NotNil(t, hashFunc)
	})
}
