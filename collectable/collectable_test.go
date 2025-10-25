package collectable

import (
	"testing"

	"github.com/amp-labs/amp-common/hashing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromComparable_Int(t *testing.T) {
	t.Parallel()

	value := 42
	collectable := FromComparable(value)

	// Test that it implements Collectable
	require.NotNil(t, collectable)

	// Test Equals
	other := FromComparable(42)

	assert.True(t, collectable.Equals(42))
	assert.False(t, collectable.Equals(43))

	// Test UpdateHash doesn't error
	hash, err := hashing.Sha256(collectable)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	// Test that same value produces same hash
	hash2, err := hashing.Sha256(other)
	require.NoError(t, err)
	assert.Equal(t, hash, hash2)

	// Test that different value produces different hash
	different := FromComparable(43)
	hash3, err := hashing.Sha256(different)
	require.NoError(t, err)
	assert.NotEqual(t, hash, hash3)
}

func TestFromComparable_IntTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value any
	}{
		{"int8", int8(42)},
		{"int16", int16(42)},
		{"int32", int32(42)},
		{"int64", int64(42)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			switch typedValue := tt.value.(type) { //nolint:gocritic
			case int8:
				collectable := FromComparable(typedValue)
				assert.True(t, collectable.Equals(typedValue))
				assert.False(t, collectable.Equals(int8(43)))

				hash, err := hashing.Sha256(collectable)
				require.NoError(t, err)
				assert.NotEmpty(t, hash)
			case int16:
				collectable := FromComparable(typedValue)
				assert.True(t, collectable.Equals(typedValue))
				assert.False(t, collectable.Equals(int16(43)))

				hash, err := hashing.Sha256(collectable)
				require.NoError(t, err)
				assert.NotEmpty(t, hash)
			case int32:
				collectable := FromComparable(typedValue)
				assert.True(t, collectable.Equals(typedValue))
				assert.False(t, collectable.Equals(int32(43)))

				hash, err := hashing.Sha256(collectable)
				require.NoError(t, err)
				assert.NotEmpty(t, hash)
			case int64:
				collectable := FromComparable(typedValue)
				assert.True(t, collectable.Equals(typedValue))
				assert.False(t, collectable.Equals(int64(43)))

				hash, err := hashing.Sha256(collectable)
				require.NoError(t, err)
				assert.NotEmpty(t, hash)
			}
		})
	}
}

func TestFromComparable_UintTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value any
	}{
		{"uint", uint(42)},
		{"uint8", uint8(42)},
		{"uint16", uint16(42)},
		{"uint32", uint32(42)},
		{"uint64", uint64(42)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			switch typedValue := tt.value.(type) { //nolint:gocritic
			case uint:
				collectable := FromComparable(typedValue)
				assert.True(t, collectable.Equals(typedValue))
				assert.False(t, collectable.Equals(uint(43)))

				hash, err := hashing.Sha256(collectable)
				require.NoError(t, err)
				assert.NotEmpty(t, hash)
			case uint8:
				collectable := FromComparable(typedValue)
				assert.True(t, collectable.Equals(typedValue))
				assert.False(t, collectable.Equals(uint8(43)))

				hash, err := hashing.Sha256(collectable)
				require.NoError(t, err)
				assert.NotEmpty(t, hash)
			case uint16:
				collectable := FromComparable(typedValue)
				assert.True(t, collectable.Equals(typedValue))
				assert.False(t, collectable.Equals(uint16(43)))

				hash, err := hashing.Sha256(collectable)
				require.NoError(t, err)
				assert.NotEmpty(t, hash)
			case uint32:
				collectable := FromComparable(typedValue)
				assert.True(t, collectable.Equals(typedValue))
				assert.False(t, collectable.Equals(uint32(43)))

				hash, err := hashing.Sha256(collectable)
				require.NoError(t, err)
				assert.NotEmpty(t, hash)
			case uint64:
				collectable := FromComparable(typedValue)
				assert.True(t, collectable.Equals(typedValue))
				assert.False(t, collectable.Equals(uint64(43)))

				hash, err := hashing.Sha256(collectable)
				require.NoError(t, err)
				assert.NotEmpty(t, hash)
			}
		})
	}
}

func TestFromComparable_FloatTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value any
	}{
		{"float32", float32(42.5)},
		{"float64", float64(42.5)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			switch typedValue := tt.value.(type) { //nolint:gocritic
			case float32:
				collectable := FromComparable(typedValue)
				assert.True(t, collectable.Equals(typedValue))
				assert.False(t, collectable.Equals(float32(43.5)))

				hash, err := hashing.Sha256(collectable)
				require.NoError(t, err)
				assert.NotEmpty(t, hash)
			case float64:
				collectable := FromComparable(typedValue)
				assert.True(t, collectable.Equals(typedValue))
				assert.False(t, collectable.Equals(float64(43.5)))

				hash, err := hashing.Sha256(collectable)
				require.NoError(t, err)
				assert.NotEmpty(t, hash)
			}
		})
	}
}

func TestFromComparable_String(t *testing.T) {
	t.Parallel()

	value := "hello"
	collectable := FromComparable(value)

	// Test Equals
	assert.True(t, collectable.Equals("hello"))
	assert.False(t, collectable.Equals("world"))

	// Test UpdateHash
	hash, err := hashing.Sha256(collectable)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	// Test that same value produces same hash
	other := FromComparable("hello")
	hash2, err := hashing.Sha256(other)
	require.NoError(t, err)
	assert.Equal(t, hash, hash2)

	// Test that different value produces different hash
	different := FromComparable("world")
	hash3, err := hashing.Sha256(different)
	require.NoError(t, err)
	assert.NotEqual(t, hash, hash3)
}

func TestFromComparable_Bool(t *testing.T) {
	t.Parallel()

	trueValue := true
	trueCollectable := FromComparable(trueValue)

	// Test Equals
	assert.True(t, trueCollectable.Equals(true))
	assert.False(t, trueCollectable.Equals(false))

	// Test UpdateHash
	trueHash, err := hashing.Sha256(trueCollectable)
	require.NoError(t, err)
	assert.NotEmpty(t, trueHash)

	// Test false value
	falseValue := false
	falseCollectable := FromComparable(falseValue)

	assert.True(t, falseCollectable.Equals(false))
	assert.False(t, falseCollectable.Equals(true))

	falseHash, err := hashing.Sha256(falseCollectable)
	require.NoError(t, err)
	assert.NotEmpty(t, falseHash)

	// Test that true and false produce different hashes
	assert.NotEqual(t, trueHash, falseHash)

	// Test that same value produces same hash
	anotherTrue := FromComparable(true)
	anotherTrueHash, err := hashing.Sha256(anotherTrue)
	require.NoError(t, err)
	assert.Equal(t, trueHash, anotherTrueHash)
}

func TestFromComparable_HashConsistency(t *testing.T) {
	t.Parallel()

	// Test that FromComparable produces the same hash as direct Hashable types
	intValue := 42
	collectable := FromComparable(intValue)

	collectableHash, err := hashing.Sha256(collectable)
	require.NoError(t, err)

	directHash, err := hashing.Sha256(hashing.HashableInt(intValue))
	require.NoError(t, err)

	assert.Equal(t, directHash, collectableHash, "FromComparable should produce same hash as direct HashableInt")

	// Test bool consistency
	boolValue := true
	boolCollectable := FromComparable(boolValue)

	boolCollectableHash, err := hashing.Sha256(boolCollectable)
	require.NoError(t, err)

	boolDirectHash, err := hashing.Sha256(hashing.HashableBool(boolValue))
	require.NoError(t, err)

	assert.Equal(t, boolDirectHash, boolCollectableHash, "FromComparable should produce same hash as direct HashableBool")
}

func TestFromComparable_StringHashConsistency(t *testing.T) {
	t.Parallel()

	stringValue := "test"
	collectable := FromComparable(stringValue)

	collectableHash, err := hashing.Sha256(collectable)
	require.NoError(t, err)

	directHash, err := hashing.Sha256(hashing.HashableString(stringValue))
	require.NoError(t, err)

	assert.Equal(t, directHash, collectableHash, "FromComparable should produce same hash as direct HashableString")
}

func TestComparableWrapper_ImplementsCollectable(t *testing.T) {
	t.Parallel()

	// Compile-time check that comparableWrapper implements Collectable
	var _ Collectable[int] = &comparableWrapper[int]{value: 42}

	var _ Collectable[string] = &comparableWrapper[string]{value: "test"}

	var _ Collectable[float64] = &comparableWrapper[float64]{value: 3.14}
}
