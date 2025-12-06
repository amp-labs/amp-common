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

func TestDefaultMap_GetOrElse(t *testing.T) {
	t.Parallel()

	t.Run("returns value for existing key", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, int](hashing.Sha256)
		m := maps.NewDefaultMap(baseMap, func(k testKey) (int, error) {
			return 88, nil
		})

		key := testKey{value: "test"}
		_ = m.Add(key, 42) //nolint:errcheck

		value, err := m.GetOrElse(key, 99)
		require.NoError(t, err)
		assert.Equal(t, 42, value)
	})
}

func TestDefaultMap_Keys(t *testing.T) {
	t.Parallel()

	t.Run("returns all keys", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, int](hashing.Sha256)
		m := maps.NewDefaultMap(baseMap, func(k testKey) (int, error) {
			return 0, nil
		})

		_ = m.Add(testKey{value: "a"}, 1) //nolint:errcheck
		_ = m.Add(testKey{value: "b"}, 2) //nolint:errcheck

		keySet := m.Keys()
		assert.Equal(t, 2, keySet.Size())
	})
}

func TestDefaultMap_ForEach(t *testing.T) {
	t.Parallel()

	t.Run("calls function for each entry", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, int](hashing.Sha256)
		m := maps.NewDefaultMap(baseMap, func(k testKey) (int, error) {
			return 0, nil
		})

		_ = m.Add(testKey{value: "a"}, 1) //nolint:errcheck
		_ = m.Add(testKey{value: "b"}, 2) //nolint:errcheck

		sum := 0

		m.ForEach(func(key testKey, value int) {
			sum += value
		})

		assert.Equal(t, 3, sum)
	})
}

func TestDefaultMap_ForAll(t *testing.T) {
	t.Parallel()

	t.Run("returns true when predicate holds", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, int](hashing.Sha256)
		m := maps.NewDefaultMap(baseMap, func(k testKey) (int, error) {
			return 0, nil
		})

		_ = m.Add(testKey{value: "a"}, 2) //nolint:errcheck
		_ = m.Add(testKey{value: "b"}, 4) //nolint:errcheck

		result := m.ForAll(func(key testKey, value int) bool {
			return value%2 == 0
		})

		assert.True(t, result)
	})
}

func TestDefaultMap_Filter(t *testing.T) {
	t.Parallel()

	t.Run("filters entries and preserves default function", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, int](hashing.Sha256)
		m := maps.NewDefaultMap(baseMap, func(k testKey) (int, error) {
			return 99, nil
		})

		_ = m.Add(testKey{value: "a"}, 1) //nolint:errcheck
		_ = m.Add(testKey{value: "b"}, 2) //nolint:errcheck
		_ = m.Add(testKey{value: "c"}, 3) //nolint:errcheck

		result := m.Filter(func(key testKey, value int) bool {
			return value%2 == 0
		})

		assert.Equal(t, 1, result.Size())

		// Verify default function is preserved
		val, found, err := result.Get(testKey{value: "missing"})
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, 99, val)
	})
}

func TestDefaultMap_FilterNot(t *testing.T) {
	t.Parallel()

	t.Run("filters entries and preserves default function", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, int](hashing.Sha256)
		m := maps.NewDefaultMap(baseMap, func(k testKey) (int, error) {
			return 99, nil
		})

		_ = m.Add(testKey{value: "a"}, 1) //nolint:errcheck
		_ = m.Add(testKey{value: "b"}, 2) //nolint:errcheck

		result := m.FilterNot(func(key testKey, value int) bool {
			return value%2 == 0
		})

		assert.Equal(t, 1, result.Size())

		// Verify default function is preserved
		val, found, err := result.Get(testKey{value: "missing"})
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, 99, val)
	})
}

func TestDefaultMap_Map(t *testing.T) {
	t.Parallel()

	t.Run("transforms entries and preserves default function", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, int](hashing.Sha256)
		m := maps.NewDefaultMap(baseMap, func(k testKey) (int, error) {
			return 99, nil
		})

		_ = m.Add(testKey{value: "a"}, 1) //nolint:errcheck

		result := m.Map(func(key testKey, value int) (testKey, int) {
			return key, value * 10
		})

		val, found, err := result.Get(testKey{value: "a"})
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, 10, val)

		// Verify default function is preserved
		val, found, err = result.Get(testKey{value: "missing"})
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, 99, val)
	})
}

func TestDefaultMap_FlatMap(t *testing.T) {
	t.Parallel()

	t.Run("flattens maps and preserves default function", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, int](hashing.Sha256)
		m := maps.NewDefaultMap(baseMap, func(k testKey) (int, error) {
			return 99, nil
		})

		_ = m.Add(testKey{value: "a"}, 1) //nolint:errcheck

		result := m.FlatMap(func(key testKey, value int) maps.Map[testKey, int] {
			nested := maps.NewHashMap[testKey, int](hashing.Sha256)
			_ = nested.Add(testKey{value: key.value + "_1"}, value) //nolint:errcheck

			return nested
		})

		assert.Equal(t, 1, result.Size())

		// Verify default function is preserved
		val, found, err := result.Get(testKey{value: "missing"})
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, 99, val)
	})
}

func TestDefaultMap_Exists(t *testing.T) {
	t.Parallel()

	t.Run("returns true when entry matches", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, int](hashing.Sha256)
		m := maps.NewDefaultMap(baseMap, func(k testKey) (int, error) {
			return 0, nil
		})

		_ = m.Add(testKey{value: "a"}, 1) //nolint:errcheck
		_ = m.Add(testKey{value: "b"}, 2) //nolint:errcheck

		result := m.Exists(func(key testKey, value int) bool {
			return value == 2
		})

		assert.True(t, result)
	})
}

func TestDefaultMap_FindFirst(t *testing.T) {
	t.Parallel()

	t.Run("returns first matching entry", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, int](hashing.Sha256)
		m := maps.NewDefaultMap(baseMap, func(k testKey) (int, error) {
			return 0, nil
		})

		_ = m.Add(testKey{value: "a"}, 1) //nolint:errcheck
		_ = m.Add(testKey{value: "b"}, 5) //nolint:errcheck

		result := m.FindFirst(func(key testKey, value int) bool {
			return value > 1
		})

		assert.True(t, result.NonEmpty())
		pair := result.GetOrPanic()
		assert.Equal(t, 5, pair.Value)
	})

	t.Run("returns None when no match", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, int](hashing.Sha256)
		m := maps.NewDefaultMap(baseMap, func(k testKey) (int, error) {
			return 0, nil
		})

		_ = m.Add(testKey{value: "a"}, 1) //nolint:errcheck

		result := m.FindFirst(func(key testKey, value int) bool {
			return value > 10
		})

		assert.True(t, result.Empty())
	})
}

func TestNewDefaultZeroMap(t *testing.T) {
	t.Parallel()

	t.Run("creates map with zero value defaults for int", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, int](hashing.Sha256)
		m := maps.NewDefaultZeroMap(baseMap)

		require.NotNil(t, m)
		assert.Equal(t, 0, m.Size())
	})

	t.Run("creates map with zero value defaults for string", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, string](hashing.Sha256)
		m := maps.NewDefaultZeroMap(baseMap)

		require.NotNil(t, m)
		assert.Equal(t, 0, m.Size())
	})

	t.Run("creates map with zero value defaults for bool", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, bool](hashing.Sha256)
		m := maps.NewDefaultZeroMap(baseMap)

		require.NotNil(t, m)
		assert.Equal(t, 0, m.Size())
	})

	t.Run("creates map with zero value defaults for pointer", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, *string](hashing.Sha256)
		m := maps.NewDefaultZeroMap(baseMap)

		require.NotNil(t, m)
		assert.Equal(t, 0, m.Size())
	})

	t.Run("creates map with zero value defaults for slice", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, []string](hashing.Sha256)
		m := maps.NewDefaultZeroMap(baseMap)

		require.NotNil(t, m)
		assert.Equal(t, 0, m.Size())
	})

	t.Run("wrapping existing default map replaces default function", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, int](hashing.Sha256)
		m1 := maps.NewDefaultMap(baseMap, func(k testKey) (int, error) {
			return 99, nil
		})
		m2 := maps.NewDefaultZeroMap(m1)

		key := testKey{value: "missing"}
		val, found, err := m2.Get(key)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, 0, val) // Should be zero, not 99
	})
}

func TestDefaultZeroMap_Get(t *testing.T) {
	t.Parallel()

	t.Run("returns existing value when key exists", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, int](hashing.Sha256)
		m := maps.NewDefaultZeroMap(baseMap)

		key := testKey{value: "test"}
		err := m.Add(key, 42)
		require.NoError(t, err)

		val, found, err := m.Get(key)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, 42, val)
	})

	t.Run("returns zero value for int when key is missing", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, int](hashing.Sha256)
		m := maps.NewDefaultZeroMap(baseMap)

		key := testKey{value: "missing"}
		val, found, err := m.Get(key)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, 0, val)
		assert.Equal(t, 1, m.Size()) // Key should be added
	})

	t.Run("returns zero value for string when key is missing", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, string](hashing.Sha256)
		m := maps.NewDefaultZeroMap(baseMap)

		key := testKey{value: "missing"}
		val, found, err := m.Get(key)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "", val)
		assert.Equal(t, 1, m.Size())
	})

	t.Run("returns zero value for bool when key is missing", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, bool](hashing.Sha256)
		m := maps.NewDefaultZeroMap(baseMap)

		key := testKey{value: "missing"}
		val, found, err := m.Get(key)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, false, val)
		assert.Equal(t, 1, m.Size())
	})

	t.Run("returns nil for pointer when key is missing", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, *int](hashing.Sha256)
		m := maps.NewDefaultZeroMap(baseMap)

		key := testKey{value: "missing"}
		val, found, err := m.Get(key)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Nil(t, val)
		assert.Equal(t, 1, m.Size())
	})

	t.Run("returns empty slice when key is missing", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, []string](hashing.Sha256)
		m := maps.NewDefaultZeroMap(baseMap)

		key := testKey{value: "missing"}
		val, found, err := m.Get(key)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Nil(t, val) // Empty slice is nil
		assert.Equal(t, 1, m.Size())
	})

	t.Run("returns empty map when key is missing", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, map[string]int](hashing.Sha256)
		m := maps.NewDefaultZeroMap(baseMap)

		key := testKey{value: "missing"}
		val, found, err := m.Get(key)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Nil(t, val) // Empty map is nil
		assert.Equal(t, 1, m.Size())
	})

	t.Run("returns zero struct when key is missing", func(t *testing.T) {
		t.Parallel()

		type testStruct struct {
			Field1 int
			Field2 string
		}

		baseMap := maps.NewHashMap[testKey, testStruct](hashing.Sha256)
		m := maps.NewDefaultZeroMap(baseMap)

		key := testKey{value: "missing"}
		val, found, err := m.Get(key)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, testStruct{Field1: 0, Field2: ""}, val)
		assert.Equal(t, 1, m.Size())
	})

	t.Run("adds default value only once for repeated Gets", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, int](hashing.Sha256)
		m := maps.NewDefaultZeroMap(baseMap)

		key := testKey{value: "missing"}
		val1, found1, err1 := m.Get(key)
		require.NoError(t, err1)
		assert.True(t, found1)
		assert.Equal(t, 0, val1)
		assert.Equal(t, 1, m.Size())

		// Second Get should not increase size
		val2, found2, err2 := m.Get(key)
		require.NoError(t, err2)
		assert.True(t, found2)
		assert.Equal(t, 0, val2)
		assert.Equal(t, 1, m.Size()) // Still 1
	})

	t.Run("never returns error for missing key", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, int](hashing.Sha256)
		m := maps.NewDefaultZeroMap(baseMap)

		// NewDefaultZeroMap should never return an error for missing keys
		for i := range 10 {
			key := testKey{value: fmt.Sprintf("key%d", i)}
			_, found, err := m.Get(key)
			require.NoError(t, err)
			assert.True(t, found)
		}

		assert.Equal(t, 10, m.Size())
	})
}

func TestDefaultZeroMap_Contains(t *testing.T) {
	t.Parallel()

	t.Run("returns true for existing key", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, int](hashing.Sha256)
		m := maps.NewDefaultZeroMap(baseMap)

		key := testKey{value: "test"}
		err := m.Add(key, 42)
		require.NoError(t, err)

		contains, err := m.Contains(key)
		require.NoError(t, err)
		assert.True(t, contains)
	})

	t.Run("generates default and returns true for missing key", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, int](hashing.Sha256)
		m := maps.NewDefaultZeroMap(baseMap)

		key := testKey{value: "missing"}
		contains, err := m.Contains(key)
		require.NoError(t, err)
		assert.True(t, contains)
		assert.Equal(t, 1, m.Size())

		// Verify value was actually added
		val, found, err := m.Get(key)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, 0, val)
	})

	t.Run("never returns false or error", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, string](hashing.Sha256)
		m := maps.NewDefaultZeroMap(baseMap)

		// NewDefaultZeroMap should always return true for Contains
		for i := range 10 {
			key := testKey{value: fmt.Sprintf("key%d", i)}
			contains, err := m.Contains(key)
			require.NoError(t, err)
			assert.True(t, contains)
		}

		assert.Equal(t, 10, m.Size())
	})
}

func TestDefaultZeroMap_UseCases(t *testing.T) {
	t.Parallel()

	t.Run("counter map use case", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, int](hashing.Sha256)
		counters := maps.NewDefaultZeroMap(baseMap)

		// Increment counters
		key1 := testKey{value: "page1"}
		key2 := testKey{value: "page2"}

		count1, _, _ := counters.Get(key1)
		_ = counters.Add(key1, count1+1) //nolint:errcheck

		count1, _, _ = counters.Get(key1)
		_ = counters.Add(key1, count1+1) //nolint:errcheck

		count2, _, _ := counters.Get(key2)
		_ = counters.Add(key2, count2+1) //nolint:errcheck

		val1, _, _ := counters.Get(key1)
		val2, _, _ := counters.Get(key2)

		assert.Equal(t, 2, val1)
		assert.Equal(t, 1, val2)
	})

	t.Run("boolean flag map use case", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, bool](hashing.Sha256)
		flags := maps.NewDefaultZeroMap(baseMap)

		// Check flags
		key1 := testKey{value: "feature1"}
		key2 := testKey{value: "feature2"}

		enabled1, _, _ := flags.Get(key1)
		assert.False(t, enabled1)

		// Enable feature2
		_ = flags.Add(key2, true) //nolint:errcheck

		enabled2, _, _ := flags.Get(key2)
		assert.True(t, enabled2)
	})

	t.Run("list accumulator use case", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, []string](hashing.Sha256)
		lists := maps.NewDefaultZeroMap(baseMap)

		key := testKey{value: "items"}

		// Get default empty list
		items, _, _ := lists.Get(key)
		assert.Nil(t, items)

		// Add items
		items = append(items, "item1", "item2")
		_ = lists.Add(key, items) //nolint:errcheck

		retrieved, _, _ := lists.Get(key)
		assert.Equal(t, []string{"item1", "item2"}, retrieved)
	})
}

func TestDefaultZeroMap_Clone(t *testing.T) {
	t.Parallel()

	t.Run("creates independent copy", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, int](hashing.Sha256)
		m := maps.NewDefaultZeroMap(baseMap)

		err := m.Add(testKey{value: "a"}, 1)
		require.NoError(t, err)

		clone := m.Clone()
		assert.Equal(t, 1, clone.Size())

		// Modify original
		err = m.Add(testKey{value: "b"}, 2)
		require.NoError(t, err)

		// Clone should be unchanged
		assert.Equal(t, 2, m.Size())
		assert.Equal(t, 1, clone.Size())
	})

	t.Run("preserves zero value default behavior", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, int](hashing.Sha256)
		m := maps.NewDefaultZeroMap(baseMap)

		clone := m.Clone()

		val, found, err := clone.Get(testKey{value: "missing"})
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, 0, val)
	})
}

func TestDefaultZeroMap_Union(t *testing.T) {
	t.Parallel()

	t.Run("combines two maps and preserves zero value defaults", func(t *testing.T) {
		t.Parallel()

		baseMap1 := maps.NewHashMap[testKey, int](hashing.Sha256)
		m1 := maps.NewDefaultZeroMap(baseMap1)

		baseMap2 := maps.NewHashMap[testKey, int](hashing.Sha256)

		err := m1.Add(testKey{value: "a"}, 1)
		require.NoError(t, err)

		err = baseMap2.Add(testKey{value: "b"}, 2)
		require.NoError(t, err)

		result, err := m1.Union(baseMap2)
		require.NoError(t, err)
		assert.Equal(t, 2, result.Size())

		// Verify zero value default works on result
		val, found, err := result.Get(testKey{value: "missing"})
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, 0, val)
	})
}

func TestDefaultZeroMap_Filter(t *testing.T) {
	t.Parallel()

	t.Run("filters entries and preserves zero value defaults", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, int](hashing.Sha256)
		m := maps.NewDefaultZeroMap(baseMap)

		_ = m.Add(testKey{value: "a"}, 1) //nolint:errcheck
		_ = m.Add(testKey{value: "b"}, 2) //nolint:errcheck
		_ = m.Add(testKey{value: "c"}, 3) //nolint:errcheck

		result := m.Filter(func(key testKey, value int) bool {
			return value%2 == 0
		})

		assert.Equal(t, 1, result.Size())

		// Verify zero value default is preserved
		val, found, err := result.Get(testKey{value: "missing"})
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, 0, val)
	})
}

func TestDefaultZeroMap_GetOrElse(t *testing.T) {
	t.Parallel()

	t.Run("returns existing value when key exists", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, int](hashing.Sha256)
		m := maps.NewDefaultZeroMap(baseMap)

		key := testKey{value: "test"}
		_ = m.Add(key, 42) //nolint:errcheck

		value, err := m.GetOrElse(key, 99)
		require.NoError(t, err)
		assert.Equal(t, 42, value)
	})

	t.Run("returns zero value not GetOrElse parameter for missing key", func(t *testing.T) {
		t.Parallel()

		baseMap := maps.NewHashMap[testKey, int](hashing.Sha256)
		m := maps.NewDefaultZeroMap(baseMap)

		key := testKey{value: "missing"}
		value, err := m.GetOrElse(key, 99)
		require.NoError(t, err)
		assert.Equal(t, 0, value) // Should be zero, not 99

		// Verify key was added with zero value
		assert.Equal(t, 1, m.Size())
		val, _, _ := m.Get(key)
		assert.Equal(t, 0, val)
	})
}
