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

const defaultValue = "default"

var errGenerationFailed = errors.New("generation failed")

//nolint:dupl // Test structure intentionally mirrors DefaultOrderedMap tests for consistency
func TestNewDefaultMap(t *testing.T) {
	t.Parallel()

	t.Run("creates default map with hash map storage", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, string](hashing.Sha256)
		m := maps.NewDefaultMap(baseMap, func(k testKey) (string, error) {
			return defaultValue, nil
		})

		require.NotNil(t, m)
		assert.Equal(t, 0, m.Size())
	})

	t.Run("wrapping existing default map replaces default function", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, string](hashing.Sha256)
		m1 := maps.NewDefaultMap(baseMap, func(k testKey) (string, error) {
			return "first", nil
		})
		m2 := maps.NewDefaultMap(m1, func(k testKey) (string, error) {
			return "second", nil
		})

		key := testKey{value: "missing"}
		val, found, err := m2.Get(key)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "second", val)
	})
}

//nolint:dupl // Test structure intentionally mirrors DefaultOrderedMapGet tests for consistency
func TestDefaultMapGet(t *testing.T) {
	t.Parallel()

	t.Run("returns existing value when key exists", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, string](hashing.Sha256)
		m := maps.NewDefaultMap(baseMap, func(k testKey) (string, error) {
			return defaultValue, nil
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

		baseMap := maps.NewHashMap[testKey, int](hashing.Sha256)
		m := maps.NewDefaultMap(baseMap, func(k testKey) (int, error) {
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

		baseMap := maps.NewHashMap[testKey, string](hashing.Sha256)
		m := maps.NewDefaultMap(baseMap, func(k testKey) (string, error) {
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

		baseMap := maps.NewHashMap[testKey, string](hashing.Sha256)
		m := maps.NewDefaultMap(baseMap, func(k testKey) (string, error) {
			return "", errGenerationFailed
		})

		key := testKey{value: "missing"}
		val, found, err := m.Get(key)
		require.Error(t, err)
		assert.False(t, found)
		assert.Equal(t, "", val)
		assert.ErrorIs(t, err, errGenerationFailed)
	})

	t.Run("default function receives correct key", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, string](hashing.Sha256)

		var receivedKey testKey

		m := maps.NewDefaultMap(baseMap, func(k testKey) (string, error) {
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

func TestDefaultMapAdd(t *testing.T) {
	t.Parallel()

	t.Run("adds new key-value pair", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, string](hashing.Sha256)
		m := maps.NewDefaultMap(baseMap, func(k testKey) (string, error) {
			return defaultValue, nil
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

	t.Run("updates existing key", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, int](hashing.Sha256)
		m := maps.NewDefaultMap(baseMap, func(k testKey) (int, error) {
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

	t.Run("bypasses default function", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, string](hashing.Sha256)
		called := false
		m := maps.NewDefaultMap(baseMap, func(k testKey) (string, error) {
			called = true

			return defaultValue, nil
		})

		key := testKey{value: "test"}
		err := m.Add(key, "direct")
		require.NoError(t, err)
		assert.False(t, called)
	})
}

func TestDefaultMapRemove(t *testing.T) {
	t.Parallel()

	t.Run("removes existing key", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, string](hashing.Sha256)
		m := maps.NewDefaultMap(baseMap, func(k testKey) (string, error) {
			return defaultValue, nil
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

		baseMap := maps.NewHashMap[testKey, string](hashing.Sha256)
		m := maps.NewDefaultMap(baseMap, func(k testKey) (string, error) {
			return defaultValue, nil
		})

		key := testKey{value: "missing"}
		err := m.Remove(key)
		require.NoError(t, err)
		assert.Equal(t, 0, m.Size())
	})
}

func TestDefaultMapClear(t *testing.T) {
	t.Parallel()

	t.Run("removes all entries", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, int](hashing.Sha256)
		m := maps.NewDefaultMap(baseMap, func(k testKey) (int, error) {
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

func TestDefaultMapContains(t *testing.T) {
	t.Parallel()

	t.Run("returns true for existing key", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, string](hashing.Sha256)
		m := maps.NewDefaultMap(baseMap, func(k testKey) (string, error) {
			return defaultValue, nil
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

		baseMap := maps.NewHashMap[testKey, string](hashing.Sha256)
		m := maps.NewDefaultMap(baseMap, func(k testKey) (string, error) {
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

		baseMap := maps.NewHashMap[testKey, string](hashing.Sha256)
		m := maps.NewDefaultMap(baseMap, func(k testKey) (string, error) {
			return "", maps.ErrNoDefaultValue
		})

		key := testKey{value: "missing"}
		contains, err := m.Contains(key)
		require.NoError(t, err)
		assert.False(t, contains)
		assert.Equal(t, 0, m.Size())
	})
}

func TestDefaultMapSize(t *testing.T) {
	t.Parallel()

	t.Run("returns correct size", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, int](hashing.Sha256)
		m := maps.NewDefaultMap(baseMap, func(k testKey) (int, error) {
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

func TestDefaultMapSeq(t *testing.T) {
	t.Parallel()

	t.Run("iterates over all entries", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, int](hashing.Sha256)
		m := maps.NewDefaultMap(baseMap, func(k testKey) (int, error) {
			return 0, nil
		})

		expected := make(map[string]int)

		for i := range 5 {
			key := testKey{value: fmt.Sprintf("key%d", i)}
			err := m.Add(key, i*10)
			require.NoError(t, err)

			expected[key.value] = i * 10
		}

		count := 0
		for key, val := range m.Seq() {
			count++

			assert.Equal(t, expected[key.value], val)
		}

		assert.Equal(t, 5, count)
	})
}

func TestDefaultMapUnion(t *testing.T) {
	t.Parallel()

	t.Run("combines two maps", func(t *testing.T) {
		t.Parallel()

		baseMap1 := maps.NewHashMap[testKey, int](hashing.Sha256)
		defaultMap1 := maps.NewDefaultMap(baseMap1, func(k testKey) (int, error) {
			return 99, nil
		})

		baseMap2 := maps.NewHashMap[testKey, int](hashing.Sha256)

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

		val, found, err = result.Get(testKey{value: "c"})
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, 3, val)
	})

	t.Run("preserves default function", func(t *testing.T) {
		t.Parallel()

		baseMap1 := maps.NewHashMap[testKey, int](hashing.Sha256)
		m1 := maps.NewDefaultMap(baseMap1, func(k testKey) (int, error) {
			return 99, nil
		})

		baseMap2 := maps.NewHashMap[testKey, int](hashing.Sha256)

		result, err := m1.Union(baseMap2)
		require.NoError(t, err)

		// Test that default function works on result
		val, found, err := result.Get(testKey{value: "missing"})
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, 99, val)
	})
}

func TestDefaultMapIntersection(t *testing.T) {
	t.Parallel()

	t.Run("returns only common keys", func(t *testing.T) {
		t.Parallel()

		baseMap1 := maps.NewHashMap[testKey, int](hashing.Sha256)
		defaultMap1 := maps.NewDefaultMap(baseMap1, func(k testKey) (int, error) {
			return 99, nil
		})

		baseMap2 := maps.NewHashMap[testKey, int](hashing.Sha256)

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
	})
}

func TestDefaultMapClone(t *testing.T) {
	t.Parallel()

	t.Run("creates independent copy", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, int](hashing.Sha256)
		defaultMap := maps.NewDefaultMap(baseMap, func(k testKey) (int, error) {
			return 99, nil
		})

		err := defaultMap.Add(testKey{value: "a"}, 1)
		require.NoError(t, err)
		err = defaultMap.Add(testKey{value: "b"}, 2)
		require.NoError(t, err)

		clone := defaultMap.Clone()
		assert.Equal(t, 2, clone.Size())

		// Modify original
		err = defaultMap.Add(testKey{value: "c"}, 3)
		require.NoError(t, err)

		// Clone should be unchanged
		assert.Equal(t, 3, defaultMap.Size())
		assert.Equal(t, 2, clone.Size())
	})

	t.Run("preserves default function", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, int](hashing.Sha256)
		m := maps.NewDefaultMap(baseMap, func(k testKey) (int, error) {
			return 99, nil
		})

		clone := m.Clone()

		val, found, err := clone.Get(testKey{value: "missing"})
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, 99, val)
	})
}

func TestDefaultMapHashFunction(t *testing.T) {
	t.Parallel()

	t.Run("returns underlying hash function", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, string](hashing.Sha256)
		m := maps.NewDefaultMap(baseMap, func(k testKey) (string, error) {
			return defaultValue, nil
		})

		hashFunc := m.HashFunction()
		assert.NotNil(t, hashFunc)
	})
}
