package maps_test

import (
	"testing"

	"github.com/amp-labs/amp-common/hashing"
	"github.com/amp-labs/amp-common/maps"
	"github.com/amp-labs/amp-common/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromSet(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when input set is nil", func(t *testing.T) {
		t.Parallel()

		var nilSet set.Set[testKey]
		result := maps.FromSet(nilSet, func(k testKey) string { return k.value })
		assert.Nil(t, result)
	})

	t.Run("converts empty set to empty map", func(t *testing.T) {
		t.Parallel()

		s := set.NewSet[testKey](hashing.Sha256)
		result := maps.FromSet(s, func(k testKey) string { return k.value })

		require.NotNil(t, result)
		assert.Equal(t, 0, result.Size())
	})

	t.Run("converts set with single element", func(t *testing.T) {
		t.Parallel()

		s := set.NewSet[testKey](hashing.Sha256)
		key := testKey{value: "hello"}
		err := s.Add(key)
		require.NoError(t, err)

		result := maps.FromSet(s, func(k testKey) int { return len(k.value) })

		require.NotNil(t, result)
		assert.Equal(t, 1, result.Size())
		value, found, err := result.Get(key)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, 5, value)
	})

	//nolint:dupl // Similar test pattern for ordered and unordered sets is expected
	t.Run("converts set with multiple elements", func(t *testing.T) {
		t.Parallel()

		s := set.NewSet[testKey](hashing.Sha256)
		keys := []testKey{
			{value: "apple"},
			{value: "banana"},
			{value: "cherry"},
		}

		for _, k := range keys {
			err := s.Add(k)
			require.NoError(t, err)
		}

		result := maps.FromSet(s, func(k testKey) int { return len(k.value) })

		require.NotNil(t, result)
		assert.Equal(t, 3, result.Size())

		// Verify all keys are present with correct values
		value1, found, err := result.Get(keys[0])
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, 5, value1) // "apple" has 5 characters

		value2, found, err := result.Get(keys[1])
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, 6, value2) // "banana" has 6 characters

		value3, found, err := result.Get(keys[2])
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, 6, value3) // "cherry" has 6 characters
	})

	t.Run("uses same hash function as input set", func(t *testing.T) {
		t.Parallel()

		s := set.NewSet[testKey](hashing.Sha256)
		key := testKey{value: "test"}
		err := s.Add(key)
		require.NoError(t, err)

		result := maps.FromSet(s, func(k testKey) string { return k.value })

		require.NotNil(t, result)
		// Both should use SHA256
		assert.NotNil(t, result.HashFunction())
	})

	t.Run("getValue function is called for each key", func(t *testing.T) {
		t.Parallel()

		s := set.NewSet[testKey](hashing.Sha256)
		keys := []testKey{
			{value: "one"},
			{value: "two"},
			{value: "three"},
		}

		for _, k := range keys {
			err := s.Add(k)
			require.NoError(t, err)
		}

		callCount := 0
		result := maps.FromSet(s, func(k testKey) string {
			callCount++

			return k.value + "_transformed"
		})

		require.NotNil(t, result)
		assert.Equal(t, 3, callCount)
		assert.Equal(t, 3, result.Size())
	})

	t.Run("getValue function can produce any value type", func(t *testing.T) {
		t.Parallel()

		testSet := set.NewSet[testKey](hashing.Sha256)
		keys := []testKey{
			{value: "key1"},
			{value: "key2"},
		}

		for _, k := range keys {
			err := testSet.Add(k)
			require.NoError(t, err)
		}

		// Test with struct values
		type customValue struct {
			name   string
			length int
		}

		result := maps.FromSet(testSet, func(k testKey) customValue {
			return customValue{
				name:   k.value,
				length: len(k.value),
			}
		})

		require.NotNil(t, result)
		assert.Equal(t, 2, result.Size())

		value, found, err := result.Get(keys[0])
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "key1", value.name)
		assert.Equal(t, 4, value.length)
	})

	t.Run("pre-allocates map with set size", func(t *testing.T) {
		t.Parallel()

		s := set.NewSet[testKey](hashing.Sha256)
		// Add many elements
		for i := range 100 {
			key := testKey{value: string(rune('a' + i))}
			err := s.Add(key)
			require.NoError(t, err)
		}

		result := maps.FromSet(s, func(k testKey) int { return 1 })

		require.NotNil(t, result)
		assert.Equal(t, 100, result.Size())
	})
}

func TestFromOrderedSet(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when input set is nil", func(t *testing.T) {
		t.Parallel()

		var nilSet set.OrderedSet[testKey]
		result := maps.FromOrderedSet(nilSet, func(k testKey) string { return k.value })
		assert.Nil(t, result)
	})

	t.Run("converts empty ordered set to empty ordered map", func(t *testing.T) {
		t.Parallel()

		s := set.NewOrderedSet[testKey](hashing.Sha256)
		result := maps.FromOrderedSet(s, func(k testKey) string { return k.value })

		require.NotNil(t, result)
		assert.Equal(t, 0, result.Size())
	})

	t.Run("converts ordered set with single element", func(t *testing.T) {
		t.Parallel()

		s := set.NewOrderedSet[testKey](hashing.Sha256)
		key := testKey{value: "world"}
		err := s.Add(key)
		require.NoError(t, err)

		result := maps.FromOrderedSet(s, func(k testKey) int { return len(k.value) })

		require.NotNil(t, result)
		assert.Equal(t, 1, result.Size())
		value, found, err := result.Get(key)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, 5, value)
	})

	//nolint:dupl // Similar test pattern for ordered and unordered sets is expected
	t.Run("converts ordered set with multiple elements", func(t *testing.T) {
		t.Parallel()

		s := set.NewOrderedSet[testKey](hashing.Sha256)
		keys := []testKey{
			{value: "first"},
			{value: "second"},
			{value: "third"},
		}

		for _, k := range keys {
			err := s.Add(k)
			require.NoError(t, err)
		}

		result := maps.FromOrderedSet(s, func(k testKey) int { return len(k.value) })

		require.NotNil(t, result)
		assert.Equal(t, 3, result.Size())

		// Verify all keys are present with correct values
		value1, found, err := result.Get(keys[0])
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, 5, value1) // "first" has 5 characters

		value2, found, err := result.Get(keys[1])
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, 6, value2) // "second" has 6 characters

		value3, found, err := result.Get(keys[2])
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, 5, value3) // "third" has 5 characters
	})

	t.Run("preserves insertion order from set", func(t *testing.T) {
		t.Parallel()

		s := set.NewOrderedSet[testKey](hashing.Sha256)
		keys := []testKey{
			{value: "alpha"},
			{value: "beta"},
			{value: "gamma"},
			{value: "delta"},
		}

		for _, k := range keys {
			err := s.Add(k)
			require.NoError(t, err)
		}

		result := maps.FromOrderedSet(s, func(k testKey) string { return k.value + "_value" })

		require.NotNil(t, result)
		assert.Equal(t, 4, result.Size())

		// Verify insertion order is preserved by iterating
		i := 0
		for _, entry := range result.Seq() {
			assert.Equal(t, keys[i], entry.Key)
			assert.Equal(t, keys[i].value+"_value", entry.Value)

			i++
		}

		assert.Equal(t, 4, i) // Ensure we iterated over all elements
	})

	t.Run("uses same hash function as input set", func(t *testing.T) {
		t.Parallel()

		s := set.NewOrderedSet[testKey](hashing.Sha256)
		key := testKey{value: "test"}
		err := s.Add(key)
		require.NoError(t, err)

		result := maps.FromOrderedSet(s, func(k testKey) string { return k.value })

		require.NotNil(t, result)
		// Both should use SHA256
		assert.NotNil(t, result.HashFunction())
	})

	t.Run("getValue function is called for each key in order", func(t *testing.T) {
		t.Parallel()

		s := set.NewOrderedSet[testKey](hashing.Sha256)
		keys := []testKey{
			{value: "one"},
			{value: "two"},
			{value: "three"},
		}

		for _, k := range keys {
			err := s.Add(k)
			require.NoError(t, err)
		}

		var callOrder []string

		result := maps.FromOrderedSet(s, func(k testKey) string {
			callOrder = append(callOrder, k.value)

			return k.value + "_transformed"
		})

		require.NotNil(t, result)
		assert.Len(t, callOrder, 3)
		// Verify the call order matches insertion order
		assert.Equal(t, []string{"one", "two", "three"}, callOrder)
	})

	t.Run("getValue function can produce any value type", func(t *testing.T) {
		t.Parallel()

		s := set.NewOrderedSet[testKey](hashing.Sha256)
		keys := []testKey{
			{value: "item1"},
			{value: "item2"},
		}

		for _, k := range keys {
			err := s.Add(k)
			require.NoError(t, err)
		}

		// Test with slice values
		result := maps.FromOrderedSet(s, func(k testKey) []int {
			return []int{len(k.value), len(k.value) * 2}
		})

		require.NotNil(t, result)
		assert.Equal(t, 2, result.Size())

		value, found, err := result.Get(keys[0])
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, []int{5, 10}, value)
	})

	t.Run("maintains order with large set", func(t *testing.T) {
		t.Parallel()

		s := set.NewOrderedSet[testKey](hashing.Sha256)
		expectedOrder := make([]testKey, 50)

		// Add elements in specific order
		for i := range 50 {
			key := testKey{value: string(rune('A' + i))}
			expectedOrder[i] = key
			err := s.Add(key)
			require.NoError(t, err)
		}

		result := maps.FromOrderedSet(s, func(k testKey) int { return int(k.value[0]) })

		require.NotNil(t, result)
		assert.Equal(t, 50, result.Size())

		// Verify order is maintained
		i := 0
		for _, entry := range result.Seq() {
			assert.Equal(t, expectedOrder[i], entry.Key)

			i++
		}

		assert.Equal(t, 50, i)
	})
}
