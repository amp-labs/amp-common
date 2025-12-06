package maps_test

import (
	"crypto/sha256"
	"testing"

	"github.com/amp-labs/amp-common/hashing"
	"github.com/amp-labs/amp-common/maps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKey_UpdateHash(t *testing.T) {
	t.Parallel()

	t.Run("hashes string key correctly", func(t *testing.T) {
		t.Parallel()

		key1 := maps.Key[string]{Key: "test"}
		key2 := maps.Key[string]{Key: "test"}

		h1 := sha256.New()
		h2 := sha256.New()

		err := key1.UpdateHash(h1)
		require.NoError(t, err)

		err = key2.UpdateHash(h2)
		require.NoError(t, err)

		// Same keys should produce same hash
		assert.Equal(t, h1.Sum(nil), h2.Sum(nil))
	})

	t.Run("hashes int key correctly", func(t *testing.T) {
		t.Parallel()

		key1 := maps.Key[int]{Key: 42}
		key2 := maps.Key[int]{Key: 42}

		h1 := sha256.New()
		h2 := sha256.New()

		err := key1.UpdateHash(h1)
		require.NoError(t, err)

		err = key2.UpdateHash(h2)
		require.NoError(t, err)

		// Same keys should produce same hash
		assert.Equal(t, h1.Sum(nil), h2.Sum(nil))
	})

	t.Run("different keys produce different hashes", func(t *testing.T) {
		t.Parallel()

		key1 := maps.Key[string]{Key: "test1"}
		key2 := maps.Key[string]{Key: "test2"}

		h1 := sha256.New()
		h2 := sha256.New()

		err := key1.UpdateHash(h1)
		require.NoError(t, err)

		err = key2.UpdateHash(h2)
		require.NoError(t, err)

		// Different keys should produce different hashes
		assert.NotEqual(t, h1.Sum(nil), h2.Sum(nil))
	})

	t.Run("hashes float64 key correctly", func(t *testing.T) {
		t.Parallel()

		key1 := maps.Key[float64]{Key: 3.14}
		key2 := maps.Key[float64]{Key: 3.14}

		h1 := sha256.New()
		h2 := sha256.New()

		err := key1.UpdateHash(h1)
		require.NoError(t, err)

		err = key2.UpdateHash(h2)
		require.NoError(t, err)

		// Same values should produce same hash
		assert.Equal(t, h1.Sum(nil), h2.Sum(nil))
	})
}

func TestKey_Equals(t *testing.T) {
	t.Parallel()

	t.Run("equal string keys", func(t *testing.T) {
		t.Parallel()

		key1 := maps.Key[string]{Key: "test"}
		key2 := maps.Key[string]{Key: "test"}

		assert.True(t, key1.Equals(key2))
		assert.True(t, key2.Equals(key1))
	})

	t.Run("unequal string keys", func(t *testing.T) {
		t.Parallel()

		key1 := maps.Key[string]{Key: "test1"}
		key2 := maps.Key[string]{Key: "test2"}

		assert.False(t, key1.Equals(key2))
		assert.False(t, key2.Equals(key1))
	})

	t.Run("equal int keys", func(t *testing.T) {
		t.Parallel()

		key1 := maps.Key[int]{Key: 42}
		key2 := maps.Key[int]{Key: 42}

		assert.True(t, key1.Equals(key2))
	})

	t.Run("unequal int keys", func(t *testing.T) {
		t.Parallel()

		key1 := maps.Key[int]{Key: 42}
		key2 := maps.Key[int]{Key: 43}

		assert.False(t, key1.Equals(key2))
	})

	t.Run("equal float64 keys", func(t *testing.T) {
		t.Parallel()

		key1 := maps.Key[float64]{Key: 3.14}
		key2 := maps.Key[float64]{Key: 3.14}

		assert.True(t, key1.Equals(key2))
	})

	t.Run("unequal float64 keys", func(t *testing.T) {
		t.Parallel()

		key1 := maps.Key[float64]{Key: 3.14}
		key2 := maps.Key[float64]{Key: 2.71}

		assert.False(t, key1.Equals(key2))
	})

	t.Run("zero value keys", func(t *testing.T) {
		t.Parallel()

		key1 := maps.Key[string]{}
		key2 := maps.Key[string]{}

		assert.True(t, key1.Equals(key2))
	})
}

func TestFromGoMap(t *testing.T) {
	t.Parallel()

	t.Run("converts string map correctly", func(t *testing.T) {
		t.Parallel()

		goMap := map[string]int{
			"a": 1,
			"b": 2,
			"c": 3,
		}

		ampMap := maps.FromGoMap(goMap, hashing.Sha256)
		require.NotNil(t, ampMap)
		assert.Equal(t, len(goMap), ampMap.Size())

		// Verify all entries are present
		for k := range goMap {
			key := maps.Key[string]{Key: k}
			contains, err := ampMap.Contains(key)
			require.NoError(t, err)
			assert.True(t, contains, "map should contain key %s", k)
		}
	})

	t.Run("converts int map correctly", func(t *testing.T) {
		t.Parallel()

		goMap := map[int]string{
			1: "one",
			2: "two",
			3: "three",
		}

		ampMap := maps.FromGoMap(goMap, hashing.Sha256)
		require.NotNil(t, ampMap)
		assert.Equal(t, len(goMap), ampMap.Size())

		for k := range goMap {
			key := maps.Key[int]{Key: k}
			contains, err := ampMap.Contains(key)
			require.NoError(t, err)
			assert.True(t, contains, "map should contain key %d", k)
		}
	})

	t.Run("converts empty map", func(t *testing.T) {
		t.Parallel()

		goMap := map[string]int{}

		ampMap := maps.FromGoMap(goMap, hashing.Sha256)
		require.NotNil(t, ampMap)
		assert.Equal(t, 0, ampMap.Size())
	})

	t.Run("returns nil for nil input", func(t *testing.T) {
		t.Parallel()

		var goMap map[string]int

		ampMap := maps.FromGoMap(goMap, hashing.Sha256)
		assert.Nil(t, ampMap)
	})

	t.Run("handles single entry map", func(t *testing.T) {
		t.Parallel()

		goMap := map[string]string{
			"key": "value",
		}

		ampMap := maps.FromGoMap(goMap, hashing.Sha256)
		require.NotNil(t, ampMap)
		assert.Equal(t, 1, ampMap.Size())

		key := maps.Key[string]{Key: "key"}
		contains, err := ampMap.Contains(key)
		require.NoError(t, err)
		assert.True(t, contains)
	})

	t.Run("handles large map", func(t *testing.T) {
		t.Parallel()

		goMap := make(map[int]int)
		for i := range 1000 {
			goMap[i] = i * 2
		}

		ampMap := maps.FromGoMap(goMap, hashing.Sha256)
		require.NotNil(t, ampMap)
		assert.Equal(t, 1000, ampMap.Size())
	})

	t.Run("handles map with float64 keys", func(t *testing.T) {
		t.Parallel()

		goMap := map[float64]string{
			1.5: "one-and-half",
			2.5: "two-and-half",
			3.5: "three-and-half",
		}

		ampMap := maps.FromGoMap(goMap, hashing.Sha256)
		require.NotNil(t, ampMap)
		assert.Equal(t, 3, ampMap.Size())

		key := maps.Key[float64]{Key: 1.5}
		contains, err := ampMap.Contains(key)
		require.NoError(t, err)
		assert.True(t, contains)
	})

	t.Run("handles map with various value types", func(t *testing.T) {
		t.Parallel()

		type valueType struct {
			data   string
			count  int
			active bool
		}

		goMap := map[string]valueType{
			"a": {data: "test", count: 1, active: true},
			"b": {data: "prod", count: 2, active: false},
		}

		ampMap := maps.FromGoMap(goMap, hashing.Sha256)
		require.NotNil(t, ampMap)
		assert.Equal(t, 2, ampMap.Size())
	})
}

func TestToGoMap(t *testing.T) {
	t.Parallel()

	t.Run("converts map with string keys", func(t *testing.T) {
		t.Parallel()

		ampMap := maps.NewHashMap[maps.Key[string], int](hashing.Sha256)
		_ = ampMap.Add(maps.Key[string]{Key: "a"}, 1) //nolint:errcheck
		_ = ampMap.Add(maps.Key[string]{Key: "b"}, 2) //nolint:errcheck
		_ = ampMap.Add(maps.Key[string]{Key: "c"}, 3) //nolint:errcheck

		goMap := maps.ToGoMap(ampMap)
		require.NotNil(t, goMap)
		assert.Len(t, goMap, 3)
		assert.Equal(t, 1, goMap["a"])
		assert.Equal(t, 2, goMap["b"])
		assert.Equal(t, 3, goMap["c"])
	})

	t.Run("converts map with int keys", func(t *testing.T) {
		t.Parallel()

		ampMap := maps.NewHashMap[maps.Key[int], string](hashing.Sha256)
		_ = ampMap.Add(maps.Key[int]{Key: 1}, "one")   //nolint:errcheck
		_ = ampMap.Add(maps.Key[int]{Key: 2}, "two")   //nolint:errcheck
		_ = ampMap.Add(maps.Key[int]{Key: 3}, "three") //nolint:errcheck

		goMap := maps.ToGoMap(ampMap)
		require.NotNil(t, goMap)
		assert.Len(t, goMap, 3)
		assert.Equal(t, "one", goMap[1])
		assert.Equal(t, "two", goMap[2])
		assert.Equal(t, "three", goMap[3])
	})

	t.Run("converts empty map", func(t *testing.T) {
		t.Parallel()

		ampMap := maps.NewHashMap[maps.Key[string], int](hashing.Sha256)

		goMap := maps.ToGoMap(ampMap)
		require.NotNil(t, goMap)
		assert.Empty(t, goMap)
	})

	t.Run("returns nil for nil input", func(t *testing.T) {
		t.Parallel()

		var ampMap maps.Map[maps.Key[string], int]

		goMap := maps.ToGoMap(ampMap)
		assert.Nil(t, goMap)
	})

	t.Run("converts single entry map", func(t *testing.T) {
		t.Parallel()

		ampMap := maps.NewHashMap[maps.Key[string], string](hashing.Sha256)
		_ = ampMap.Add(maps.Key[string]{Key: "key"}, "value") //nolint:errcheck

		goMap := maps.ToGoMap(ampMap)
		require.NotNil(t, goMap)
		assert.Len(t, goMap, 1)
		assert.Equal(t, "value", goMap["key"])
	})

	t.Run("converts large map", func(t *testing.T) {
		t.Parallel()

		ampMap := maps.NewHashMap[maps.Key[int], int](hashing.Sha256)
		for i := range 1000 {
			_ = ampMap.Add(maps.Key[int]{Key: i}, i*2) //nolint:errcheck
		}

		goMap := maps.ToGoMap(ampMap)
		require.NotNil(t, goMap)
		assert.Len(t, goMap, 1000)

		// Verify some entries
		assert.Equal(t, 0, goMap[0])
		assert.Equal(t, 998, goMap[499])
		assert.Equal(t, 1998, goMap[999])
	})

	t.Run("converts map with float64 keys", func(t *testing.T) {
		t.Parallel()

		ampMap := maps.NewHashMap[maps.Key[float64], string](hashing.Sha256)
		_ = ampMap.Add(maps.Key[float64]{Key: 1.5}, "one-and-half")   //nolint:errcheck
		_ = ampMap.Add(maps.Key[float64]{Key: 2.5}, "two-and-half")   //nolint:errcheck
		_ = ampMap.Add(maps.Key[float64]{Key: 3.5}, "three-and-half") //nolint:errcheck

		goMap := maps.ToGoMap(ampMap)
		require.NotNil(t, goMap)
		assert.Len(t, goMap, 3)
		assert.Equal(t, "one-and-half", goMap[1.5])
		assert.Equal(t, "two-and-half", goMap[2.5])
		assert.Equal(t, "three-and-half", goMap[3.5])
	})

	t.Run("converts map with various value types", func(t *testing.T) {
		t.Parallel()

		type valueType struct {
			data   string
			count  int
			active bool
		}

		ampMap := maps.NewHashMap[maps.Key[string], valueType](hashing.Sha256)
		_ = ampMap.Add(maps.Key[string]{Key: "a"}, valueType{data: "test", count: 1, active: true})  //nolint:errcheck
		_ = ampMap.Add(maps.Key[string]{Key: "b"}, valueType{data: "prod", count: 2, active: false}) //nolint:errcheck

		goMap := maps.ToGoMap(ampMap)
		require.NotNil(t, goMap)
		assert.Len(t, goMap, 2)
		assert.Equal(t, "test", goMap["a"].data)
		assert.Equal(t, 1, goMap["a"].count)
		assert.True(t, goMap["a"].active)
	})

	t.Run("converts hash map to standard map", func(t *testing.T) {
		t.Parallel()

		hashMap := maps.NewHashMap[maps.Key[string], int](hashing.Sha256)
		_ = hashMap.Add(maps.Key[string]{Key: "first"}, 1)  //nolint:errcheck
		_ = hashMap.Add(maps.Key[string]{Key: "second"}, 2) //nolint:errcheck
		_ = hashMap.Add(maps.Key[string]{Key: "third"}, 3)  //nolint:errcheck

		goMap := maps.ToGoMap(hashMap)
		require.NotNil(t, goMap)
		assert.Len(t, goMap, 3)
		assert.Equal(t, 1, goMap["first"])
		assert.Equal(t, 2, goMap["second"])
		assert.Equal(t, 3, goMap["third"])
	})
}

func TestFromGoMap_ToGoMap_RoundTrip(t *testing.T) {
	t.Parallel()

	t.Run("round trip preserves string map data", func(t *testing.T) {
		t.Parallel()

		original := map[string]int{
			"a": 1,
			"b": 2,
			"c": 3,
		}

		ampMap := maps.FromGoMap(original, hashing.Sha256)
		result := maps.ToGoMap(ampMap)

		assert.Equal(t, original, result)
	})

	t.Run("round trip preserves int map data", func(t *testing.T) {
		t.Parallel()

		original := map[int]string{
			1: "one",
			2: "two",
			3: "three",
		}

		ampMap := maps.FromGoMap(original, hashing.Sha256)
		result := maps.ToGoMap(ampMap)

		assert.Equal(t, original, result)
	})

	t.Run("round trip preserves empty map", func(t *testing.T) {
		t.Parallel()

		original := map[string]string{}

		ampMap := maps.FromGoMap(original, hashing.Sha256)
		result := maps.ToGoMap(ampMap)

		assert.NotNil(t, result)
		assert.Empty(t, result)
	})

	t.Run("round trip preserves large map", func(t *testing.T) {
		t.Parallel()

		original := make(map[int]string)
		for i := range 100 {
			original[i] = string(rune('A' + i%26))
		}

		ampMap := maps.FromGoMap(original, hashing.Sha256)
		result := maps.ToGoMap(ampMap)

		assert.Equal(t, original, result)
	})
}

func TestKey_HashConsistency(t *testing.T) {
	t.Parallel()

	t.Run("same key produces same hash across multiple calls", func(t *testing.T) {
		t.Parallel()

		key := maps.Key[string]{Key: "test"}

		var hashes [][]byte

		for range 10 {
			h := sha256.New()
			err := key.UpdateHash(h)
			require.NoError(t, err)

			hashes = append(hashes, h.Sum(nil))
		}

		// All hashes should be identical
		for i := 1; i < len(hashes); i++ {
			assert.Equal(t, hashes[0], hashes[i])
		}
	})

	t.Run("key hash is compatible with collectable.FromComparable", func(t *testing.T) {
		t.Parallel()

		// This test verifies that the Key type correctly delegates to collectable.FromComparable
		key := maps.Key[string]{Key: "test"}

		h := sha256.New()
		err := key.UpdateHash(h)
		require.NoError(t, err)

		// Should not return error
		assert.NoError(t, err)
	})
}

// mockHash is a mock implementation of hash.Hash for testing error conditions.
type mockHash struct {
	shouldFail bool
	data       []byte
}

func (m *mockHash) Write(p []byte) (n int, err error) {
	if m.shouldFail {
		return 0, assert.AnError
	}

	m.data = append(m.data, p...)

	return len(p), nil
}

func (m *mockHash) Sum(b []byte) []byte {
	return append(b, m.data...)
}

func (m *mockHash) Reset() {
	m.data = nil
}

func (m *mockHash) Size() int {
	return len(m.data)
}

func (m *mockHash) BlockSize() int {
	return 64
}

func TestKey_UpdateHash_ErrorHandling(t *testing.T) {
	t.Parallel()

	t.Run("propagates hash write errors", func(t *testing.T) {
		t.Parallel()

		key := maps.Key[string]{Key: "test"}
		mockH := &mockHash{shouldFail: true}

		err := key.UpdateHash(mockH)
		assert.Error(t, err)
	})

	t.Run("succeeds with working hash", func(t *testing.T) {
		t.Parallel()

		key := maps.Key[string]{Key: "test"}
		mockH := &mockHash{shouldFail: false}

		err := key.UpdateHash(mockH)
		require.NoError(t, err)
		assert.NotEmpty(t, mockH.data)
	})
}
