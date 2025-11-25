package hashing

import (
	"crypto/sha256"
	"errors"
	"hash"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSha256(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    Hashable
		expected string
	}{
		{
			name:     "empty string",
			input:    HashableString(""),
			expected: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:     "simple string",
			input:    HashableString("hello"),
			expected: "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
		},
		{
			name:     "string with spaces",
			input:    HashableString("hello world"),
			expected: "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
		},
		{
			name:     "empty bytes",
			input:    HashableBytes([]byte{}),
			expected: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:     "simple bytes",
			input:    HashableBytes([]byte("hello")),
			expected: "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := Sha256(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMd5(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    Hashable
		expected string
	}{
		{
			name:     "empty string",
			input:    HashableString(""),
			expected: "d41d8cd98f00b204e9800998ecf8427e",
		},
		{
			name:     "simple string",
			input:    HashableString("hello"),
			expected: "5d41402abc4b2a76b9719d911017c592",
		},
		{
			name:     "string with spaces",
			input:    HashableString("hello world"),
			expected: "5eb63bbbe01eeed093cb22bb8f5acdc3",
		},
		{
			name:     "empty bytes",
			input:    HashableBytes([]byte{}),
			expected: "d41d8cd98f00b204e9800998ecf8427e",
		},
		{
			name:     "simple bytes",
			input:    HashableBytes([]byte("hello")),
			expected: "5d41402abc4b2a76b9719d911017c592",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := Md5(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSha1(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    Hashable
		expected string
	}{
		{
			name:     "empty string",
			input:    HashableString(""),
			expected: "da39a3ee5e6b4b0d3255bfef95601890afd80709",
		},
		{
			name:     "simple string",
			input:    HashableString("hello"),
			expected: "aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d",
		},
		{
			name:     "string with spaces",
			input:    HashableString("hello world"),
			expected: "2aae6c35c94fcfb415dbe95f408b9ce91ee846ed",
		},
		{
			name:     "empty bytes",
			input:    HashableBytes([]byte{}),
			expected: "da39a3ee5e6b4b0d3255bfef95601890afd80709",
		},
		{
			name:     "simple bytes",
			input:    HashableBytes([]byte("hello")),
			expected: "aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := Sha1(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSha512(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    Hashable
		expected string
	}{
		{
			name:  "empty string",
			input: HashableString(""),
			expected: "cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce" +
				"47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e",
		},
		{
			name:  "simple string",
			input: HashableString("hello"),
			expected: "9b71d224bd62f3785d96d46ad3ea3d73319bfbc2890caadae2dff72519673ca7" +
				"2323c3d99ba5c11d7c7acc6e14b8c5da0c4663475c2e5c3adef46f73bcdec043",
		},
		{
			name:  "string with spaces",
			input: HashableString("hello world"),
			expected: "309ecc489c12d6eb4cc40f50c902f2b4d0ed77ee511a7c7a9bcd3ca86d4cd86f" +
				"989dd35bc5ff499670da34255b45b0cfd830e81f605dcf7dc5542e93ae9cd76f",
		},
		{
			name:  "empty bytes",
			input: HashableBytes([]byte{}),
			expected: "cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce" +
				"47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e",
		},
		{
			name:  "simple bytes",
			input: HashableBytes([]byte("hello")),
			expected: "9b71d224bd62f3785d96d46ad3ea3d73319bfbc2890caadae2dff72519673ca7" +
				"2323c3d99ba5c11d7c7acc6e14b8c5da0c4663475c2e5c3adef46f73bcdec043",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := Sha512(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// mockHashable is a test implementation of Hashable that can return errors.
type mockHashable struct {
	err error
}

func (m mockHashable) UpdateHash(h hash.Hash) error {
	if m.err != nil {
		return m.err
	}

	_, err := h.Write([]byte("test"))

	return err
}

var errHashTest = errors.New("hash error")

func TestHashFunctions_Error(t *testing.T) {
	t.Parallel()

	mock := mockHashable{err: errHashTest}

	t.Run("Sha256 error", func(t *testing.T) {
		t.Parallel()

		result, err := Sha256(mock)
		require.Error(t, err)
		assert.Equal(t, errHashTest, err)
		assert.Empty(t, result)
	})

	t.Run("Md5 error", func(t *testing.T) {
		t.Parallel()

		result, err := Md5(mock)
		require.Error(t, err)
		assert.Equal(t, errHashTest, err)
		assert.Empty(t, result)
	})

	t.Run("Sha1 error", func(t *testing.T) {
		t.Parallel()

		result, err := Sha1(mock)
		require.Error(t, err)
		assert.Equal(t, errHashTest, err)
		assert.Empty(t, result)
	})

	t.Run("Sha512 error", func(t *testing.T) {
		t.Parallel()

		result, err := Sha512(mock)
		require.Error(t, err)
		assert.Equal(t, errHashTest, err)
		assert.Empty(t, result)
	})
}

func TestHashableString_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    HashableString
		expected string
	}{
		{
			name:     "empty string",
			input:    HashableString(""),
			expected: "",
		},
		{
			name:     "simple string",
			input:    HashableString("hello"),
			expected: "hello",
		},
		{
			name:     "string with special characters",
			input:    HashableString("hello!@#$%^&*()"),
			expected: "hello!@#$%^&*()",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.input.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHashableString_Equals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		a        HashableString
		b        HashableString
		expected bool
	}{
		{
			name:     "equal strings",
			a:        HashableString("hello"),
			b:        HashableString("hello"),
			expected: true,
		},
		{
			name:     "different strings",
			a:        HashableString("hello"),
			b:        HashableString("world"),
			expected: false,
		},
		{
			name:     "empty strings",
			a:        HashableString(""),
			b:        HashableString(""),
			expected: true,
		},
		{
			name:     "empty vs non-empty",
			a:        HashableString(""),
			b:        HashableString("hello"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.a.Equals(tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHashableBytes_Equals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		a        HashableBytes
		b        HashableBytes
		expected bool
	}{
		{
			name:     "equal bytes",
			a:        HashableBytes([]byte("hello")),
			b:        HashableBytes([]byte("hello")),
			expected: true,
		},
		{
			name:     "different bytes",
			a:        HashableBytes([]byte("hello")),
			b:        HashableBytes([]byte("world")),
			expected: false,
		},
		{
			name:     "empty bytes",
			a:        HashableBytes([]byte{}),
			b:        HashableBytes([]byte{}),
			expected: true,
		},
		{
			name:     "nil vs empty",
			a:        HashableBytes(nil),
			b:        HashableBytes([]byte{}),
			expected: true,
		},
		{
			name:     "empty vs non-empty",
			a:        HashableBytes([]byte{}),
			b:        HashableBytes([]byte("hello")),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.a.Equals(tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHashableString_UpdateHash(t *testing.T) {
	t.Parallel()

	s := HashableString("hello")
	h := &mockHash{}

	err := s.UpdateHash(h)
	require.NoError(t, err)
	assert.Equal(t, []byte("hello"), h.data)
}

func TestHashableBytes_UpdateHash(t *testing.T) {
	t.Parallel()

	b := HashableBytes([]byte("hello"))
	h := &mockHash{}

	err := b.UpdateHash(h)
	require.NoError(t, err)
	assert.Equal(t, []byte("hello"), h.data)
}

// mockHash is a test implementation of hash.Hash for testing UpdateHash methods.
type mockHash struct {
	data []byte
}

func (m *mockHash) Write(p []byte) (n int, err error) {
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

func TestHashFunc(t *testing.T) {
	t.Parallel()

	// Test that HashFunc type works correctly
	var hashFunc HashFunc = Sha256

	result, err := hashFunc(HashableString("test"))
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	// Verify it produces the expected hash
	expected, err := Sha256(HashableString("test"))
	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestConsistency(t *testing.T) {
	t.Parallel()

	// Test that the same input produces the same output
	input := HashableString("consistency test")

	hash1, err1 := Sha256(input)
	require.NoError(t, err1)

	hash2, err2 := Sha256(input)
	require.NoError(t, err2)

	assert.Equal(t, hash1, hash2)
}

func TestDifferentInputsProduceDifferentHashes(t *testing.T) {
	t.Parallel()

	input1 := HashableString("hello")
	input2 := HashableString("world")

	hash1, err1 := Sha256(input1)
	require.NoError(t, err1)

	hash2, err2 := Sha256(input2)
	require.NoError(t, err2)

	assert.NotEqual(t, hash1, hash2)
}

func TestHashBase64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    Hashable
		expected string
	}{
		{
			name:     "empty string",
			input:    HashableString(""),
			expected: "47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU=",
		},
		{
			name:     "simple string",
			input:    HashableString("hello"),
			expected: "LPJNul+wow4m6DsqxbninhsWHlwfp0JecwQzYpOLmCQ=",
		},
		{
			name:     "string with spaces",
			input:    HashableString("hello world"),
			expected: "uU0nuZNNPgilLlLX2n2r+sSE7+N6U4DukIj3rOLvzek=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := HashBase64(tt.input, sha256.New())
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHashableInt_UpdateHash(t *testing.T) {
	t.Parallel()

	i := HashableInt(42)
	h := &mockHash{}

	err := i.UpdateHash(h)
	require.NoError(t, err)
	assert.Len(t, h.data, 8)
}

func TestHashableInt_Equals(t *testing.T) { //nolint:dupl
	t.Parallel()

	tests := []struct {
		name     string
		a        HashableInt
		b        HashableInt
		expected bool
	}{
		{
			name:     "equal values",
			a:        HashableInt(42),
			b:        HashableInt(42),
			expected: true,
		},
		{
			name:     "different values",
			a:        HashableInt(42),
			b:        HashableInt(43),
			expected: false,
		},
		{
			name:     "zero values",
			a:        HashableInt(0),
			b:        HashableInt(0),
			expected: true,
		},
		{
			name:     "negative values",
			a:        HashableInt(-1),
			b:        HashableInt(-1),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.a.Equals(tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHashableInt8_UpdateHash(t *testing.T) {
	t.Parallel()

	i := HashableInt8(42)
	h := &mockHash{}

	err := i.UpdateHash(h)
	require.NoError(t, err)
	assert.Len(t, h.data, 1)
}

func TestHashableInt8_Equals(t *testing.T) { //nolint:dupl
	t.Parallel()

	tests := []struct {
		name     string
		a        HashableInt8
		b        HashableInt8
		expected bool
	}{
		{
			name:     "equal values",
			a:        HashableInt8(42),
			b:        HashableInt8(42),
			expected: true,
		},
		{
			name:     "different values",
			a:        HashableInt8(42),
			b:        HashableInt8(43),
			expected: false,
		},
		{
			name:     "max value",
			a:        HashableInt8(127),
			b:        HashableInt8(127),
			expected: true,
		},
		{
			name:     "min value",
			a:        HashableInt8(-128),
			b:        HashableInt8(-128),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.a.Equals(tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHashableInt16_UpdateHash(t *testing.T) {
	t.Parallel()

	i := HashableInt16(42)
	h := &mockHash{}

	err := i.UpdateHash(h)
	require.NoError(t, err)
	assert.Len(t, h.data, 2)
}

func TestHashableInt16_Equals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		a        HashableInt16
		b        HashableInt16
		expected bool
	}{
		{
			name:     "equal values",
			a:        HashableInt16(1000),
			b:        HashableInt16(1000),
			expected: true,
		},
		{
			name:     "different values",
			a:        HashableInt16(1000),
			b:        HashableInt16(2000),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.a.Equals(tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHashableInt32_UpdateHash(t *testing.T) {
	t.Parallel()

	i := HashableInt32(42)
	h := &mockHash{}

	err := i.UpdateHash(h)
	require.NoError(t, err)
	assert.Len(t, h.data, 4)
}

func TestHashableInt32_Equals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		a        HashableInt32
		b        HashableInt32
		expected bool
	}{
		{
			name:     "equal values",
			a:        HashableInt32(100000),
			b:        HashableInt32(100000),
			expected: true,
		},
		{
			name:     "different values",
			a:        HashableInt32(100000),
			b:        HashableInt32(200000),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.a.Equals(tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHashableInt64_UpdateHash(t *testing.T) {
	t.Parallel()

	i := HashableInt64(42)
	h := &mockHash{}

	err := i.UpdateHash(h)
	require.NoError(t, err)
	assert.Len(t, h.data, 8)
}

func TestHashableInt64_Equals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		a        HashableInt64
		b        HashableInt64
		expected bool
	}{
		{
			name:     "equal values",
			a:        HashableInt64(1000000000),
			b:        HashableInt64(1000000000),
			expected: true,
		},
		{
			name:     "different values",
			a:        HashableInt64(1000000000),
			b:        HashableInt64(2000000000),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.a.Equals(tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHashableUint_UpdateHash(t *testing.T) {
	t.Parallel()

	u := HashableUint(42)
	h := &mockHash{}

	err := u.UpdateHash(h)
	require.NoError(t, err)
	assert.Len(t, h.data, 8)
}

func TestHashableUint_Equals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		a        HashableUint
		b        HashableUint
		expected bool
	}{
		{
			name:     "equal values",
			a:        HashableUint(42),
			b:        HashableUint(42),
			expected: true,
		},
		{
			name:     "different values",
			a:        HashableUint(42),
			b:        HashableUint(43),
			expected: false,
		},
		{
			name:     "zero values",
			a:        HashableUint(0),
			b:        HashableUint(0),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.a.Equals(tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHashableUint8_UpdateHash(t *testing.T) {
	t.Parallel()

	u := HashableUint8(42)
	h := &mockHash{}

	err := u.UpdateHash(h)
	require.NoError(t, err)
	assert.Len(t, h.data, 1)
}

func TestHashableUint8_Equals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		a        HashableUint8
		b        HashableUint8
		expected bool
	}{
		{
			name:     "equal values",
			a:        HashableUint8(42),
			b:        HashableUint8(42),
			expected: true,
		},
		{
			name:     "different values",
			a:        HashableUint8(42),
			b:        HashableUint8(43),
			expected: false,
		},
		{
			name:     "max value",
			a:        HashableUint8(255),
			b:        HashableUint8(255),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.a.Equals(tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHashableUint16_UpdateHash(t *testing.T) {
	t.Parallel()

	u := HashableUint16(42)
	h := &mockHash{}

	err := u.UpdateHash(h)
	require.NoError(t, err)
	assert.Len(t, h.data, 2)
}

func TestHashableUint16_Equals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		a        HashableUint16
		b        HashableUint16
		expected bool
	}{
		{
			name:     "equal values",
			a:        HashableUint16(1000),
			b:        HashableUint16(1000),
			expected: true,
		},
		{
			name:     "different values",
			a:        HashableUint16(1000),
			b:        HashableUint16(2000),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.a.Equals(tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHashableUint32_UpdateHash(t *testing.T) {
	t.Parallel()

	u := HashableUint32(42)
	h := &mockHash{}

	err := u.UpdateHash(h)
	require.NoError(t, err)
	assert.Len(t, h.data, 4)
}

func TestHashableUint32_Equals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		a        HashableUint32
		b        HashableUint32
		expected bool
	}{
		{
			name:     "equal values",
			a:        HashableUint32(100000),
			b:        HashableUint32(100000),
			expected: true,
		},
		{
			name:     "different values",
			a:        HashableUint32(100000),
			b:        HashableUint32(200000),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.a.Equals(tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHashableUint64_UpdateHash(t *testing.T) {
	t.Parallel()

	u := HashableUint64(42)
	h := &mockHash{}

	err := u.UpdateHash(h)
	require.NoError(t, err)
	assert.Len(t, h.data, 8)
}

func TestHashableUint64_Equals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		a        HashableUint64
		b        HashableUint64
		expected bool
	}{
		{
			name:     "equal values",
			a:        HashableUint64(1000000000),
			b:        HashableUint64(1000000000),
			expected: true,
		},
		{
			name:     "different values",
			a:        HashableUint64(1000000000),
			b:        HashableUint64(2000000000),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.a.Equals(tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHashableFloat32_UpdateHash(t *testing.T) {
	t.Parallel()

	f := HashableFloat32(3.14)
	h := &mockHash{}

	err := f.UpdateHash(h)
	require.NoError(t, err)
	assert.Len(t, h.data, 4)
}

func TestHashableFloat32_Equals(t *testing.T) { //nolint:dupl
	t.Parallel()

	tests := []struct {
		name     string
		a        HashableFloat32
		b        HashableFloat32
		expected bool
	}{
		{
			name:     "equal values",
			a:        HashableFloat32(3.14),
			b:        HashableFloat32(3.14),
			expected: true,
		},
		{
			name:     "different values",
			a:        HashableFloat32(3.14),
			b:        HashableFloat32(2.71),
			expected: false,
		},
		{
			name:     "zero values",
			a:        HashableFloat32(0.0),
			b:        HashableFloat32(0.0),
			expected: true,
		},
		{
			name:     "negative values",
			a:        HashableFloat32(-1.5),
			b:        HashableFloat32(-1.5),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.a.Equals(tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHashableFloat64_UpdateHash(t *testing.T) {
	t.Parallel()

	f := HashableFloat64(3.14159265359)
	h := &mockHash{}

	err := f.UpdateHash(h)
	require.NoError(t, err)
	assert.Len(t, h.data, 8)
}

func TestHashableFloat64_Equals(t *testing.T) { //nolint:dupl
	t.Parallel()

	tests := []struct {
		name     string
		a        HashableFloat64
		b        HashableFloat64
		expected bool
	}{
		{
			name:     "equal values",
			a:        HashableFloat64(3.14159265359),
			b:        HashableFloat64(3.14159265359),
			expected: true,
		},
		{
			name:     "different values",
			a:        HashableFloat64(3.14159265359),
			b:        HashableFloat64(2.71828182846),
			expected: false,
		},
		{
			name:     "zero values",
			a:        HashableFloat64(0.0),
			b:        HashableFloat64(0.0),
			expected: true,
		},
		{
			name:     "negative values",
			a:        HashableFloat64(-1.5),
			b:        HashableFloat64(-1.5),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.a.Equals(tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNumericTypes_HashConsistency(t *testing.T) {
	t.Parallel()

	// Test that same numeric values produce same hashes
	tests := []struct {
		name  string
		input Hashable
	}{
		{"HashableInt", HashableInt(42)},
		{"HashableInt8", HashableInt8(42)},
		{"HashableInt16", HashableInt16(42)},
		{"HashableInt32", HashableInt32(42)},
		{"HashableInt64", HashableInt64(42)},
		{"HashableUint", HashableUint(42)},
		{"HashableUint8", HashableUint8(42)},
		{"HashableUint16", HashableUint16(42)},
		{"HashableUint32", HashableUint32(42)},
		{"HashableUint64", HashableUint64(42)},
		{"HashableFloat32", HashableFloat32(3.14)},
		{"HashableFloat64", HashableFloat64(3.14)},
		{"HashableBool true", HashableBool(true)},
		{"HashableBool false", HashableBool(false)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			hash1, err1 := Sha256(tt.input)
			require.NoError(t, err1)

			hash2, err2 := Sha256(tt.input)
			require.NoError(t, err2)

			assert.Equal(t, hash1, hash2, "same input should produce same hash")
		})
	}
}

func TestHashableBool_UpdateHash(t *testing.T) {
	t.Parallel()

	t.Run("true and false produce different hashes", func(t *testing.T) {
		t.Parallel()

		hashTrue, err := Sha256(HashableBool(true))
		require.NoError(t, err)

		hashFalse, err := Sha256(HashableBool(false))
		require.NoError(t, err)

		assert.NotEqual(t, hashTrue, hashFalse, "true and false should have different hashes")
	})

	t.Run("same value produces same hash", func(t *testing.T) {
		t.Parallel()

		hash1, err := Sha256(HashableBool(true))
		require.NoError(t, err)

		hash2, err := Sha256(HashableBool(true))
		require.NoError(t, err)

		assert.Equal(t, hash1, hash2, "same bool value should produce same hash")
	})
}

func TestHashableBool_Equals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		a        HashableBool
		b        HashableBool
		expected bool
	}{
		{
			name:     "both true",
			a:        HashableBool(true),
			b:        HashableBool(true),
			expected: true,
		},
		{
			name:     "both false",
			a:        HashableBool(false),
			b:        HashableBool(false),
			expected: true,
		},
		{
			name:     "true and false",
			a:        HashableBool(true),
			b:        HashableBool(false),
			expected: false,
		},
		{
			name:     "false and true",
			a:        HashableBool(false),
			b:        HashableBool(true),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.a.Equals(tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestXxHash32(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input Hashable
	}{
		{
			name:  "empty string",
			input: HashableString(""),
		},
		{
			name:  "simple string",
			input: HashableString("hello"),
		},
		{
			name:  "string with spaces",
			input: HashableString("hello world"),
		},
		{
			name:  "bytes",
			input: HashableBytes([]byte("test")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := XxHash32(tt.input)
			require.NoError(t, err)
			assert.NotEmpty(t, result)
			assert.Len(t, result, 8, "xxHash32 should produce 8 hex characters (4 bytes)")
		})
	}
}

func TestXxHash64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input Hashable
	}{
		{
			name:  "empty string",
			input: HashableString(""),
		},
		{
			name:  "simple string",
			input: HashableString("hello"),
		},
		{
			name:  "string with spaces",
			input: HashableString("hello world"),
		},
		{
			name:  "bytes",
			input: HashableBytes([]byte("test")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := XxHash64(tt.input)
			require.NoError(t, err)
			assert.NotEmpty(t, result)
			assert.Len(t, result, 16, "xxHash64 should produce 16 hex characters (8 bytes)")
		})
	}
}

func TestXxh3(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input Hashable
	}{
		{
			name:  "empty string",
			input: HashableString(""),
		},
		{
			name:  "simple string",
			input: HashableString("hello"),
		},
		{
			name:  "string with spaces",
			input: HashableString("hello world"),
		},
		{
			name:  "bytes",
			input: HashableBytes([]byte("test")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := Xxh3(tt.input)
			require.NoError(t, err)
			assert.NotEmpty(t, result)
			assert.Len(t, result, 16, "xxh3 should produce 16 hex characters (8 bytes)")
		})
	}
}

func TestXxHashConsistency(t *testing.T) {
	t.Parallel()

	input := HashableString("consistency test")

	t.Run("XxHash32 consistency", func(t *testing.T) {
		t.Parallel()

		hash1, err1 := XxHash32(input)
		require.NoError(t, err1)

		hash2, err2 := XxHash32(input)
		require.NoError(t, err2)

		assert.Equal(t, hash1, hash2)
	})

	t.Run("XxHash64 consistency", func(t *testing.T) {
		t.Parallel()

		hash1, err1 := XxHash64(input)
		require.NoError(t, err1)

		hash2, err2 := XxHash64(input)
		require.NoError(t, err2)

		assert.Equal(t, hash1, hash2)
	})

	t.Run("Xxh3 consistency", func(t *testing.T) {
		t.Parallel()

		hash1, err1 := Xxh3(input)
		require.NoError(t, err1)

		hash2, err2 := Xxh3(input)
		require.NoError(t, err2)

		assert.Equal(t, hash1, hash2)
	})
}

func TestXxHashDifferentInputs(t *testing.T) {
	t.Parallel()

	input1 := HashableString("hello")
	input2 := HashableString("world")

	t.Run("XxHash32 different inputs", func(t *testing.T) {
		t.Parallel()

		hash1, err1 := XxHash32(input1)
		require.NoError(t, err1)

		hash2, err2 := XxHash32(input2)
		require.NoError(t, err2)

		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("XxHash64 different inputs", func(t *testing.T) {
		t.Parallel()

		hash1, err1 := XxHash64(input1)
		require.NoError(t, err1)

		hash2, err2 := XxHash64(input2)
		require.NoError(t, err2)

		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("Xxh3 different inputs", func(t *testing.T) {
		t.Parallel()

		hash1, err1 := Xxh3(input1)
		require.NoError(t, err1)

		hash2, err2 := Xxh3(input2)
		require.NoError(t, err2)

		assert.NotEqual(t, hash1, hash2)
	})
}

func TestXxHashError(t *testing.T) {
	t.Parallel()

	mock := mockHashable{err: errHashTest}

	t.Run("XxHash32 error", func(t *testing.T) {
		t.Parallel()

		result, err := XxHash32(mock)
		require.Error(t, err)
		assert.Equal(t, errHashTest, err)
		assert.Empty(t, result)
	})

	t.Run("XxHash64 error", func(t *testing.T) {
		t.Parallel()

		result, err := XxHash64(mock)
		require.Error(t, err)
		assert.Equal(t, errHashTest, err)
		assert.Empty(t, result)
	})

	t.Run("Xxh3 error", func(t *testing.T) {
		t.Parallel()

		result, err := Xxh3(mock)
		require.Error(t, err)
		assert.Equal(t, errHashTest, err)
		assert.Empty(t, result)
	})
}

func TestHashBase64_Error(t *testing.T) {
	t.Parallel()

	mock := mockHashable{err: errHashTest}

	result, err := HashBase64(mock, sha256.New())
	require.Error(t, err)
	assert.Equal(t, errHashTest, err)
	assert.Empty(t, result)
}

func TestHashableFloat32_NaN(t *testing.T) {
	t.Parallel()

	t.Run("NaN produces different hashes", func(t *testing.T) {
		t.Parallel()

		// Create two NaN values
		nan1 := HashableFloat32(float32(math.NaN()))
		nan2 := HashableFloat32(float32(math.NaN()))

		// Hash them multiple times
		hash1, err1 := Sha256(nan1)
		require.NoError(t, err1)

		hash2, err2 := Sha256(nan2)
		require.NoError(t, err2)

		// NaN values should produce different random hashes
		assert.NotEqual(t, hash1, hash2, "NaN values should produce different hashes")
	})

	t.Run("NaN Equals returns false", func(t *testing.T) {
		t.Parallel()

		nan1 := HashableFloat32(float32(math.NaN()))
		nan2 := HashableFloat32(float32(math.NaN()))
		normalValue := HashableFloat32(3.14)

		// NaN != NaN
		assert.False(t, nan1.Equals(nan2), "NaN should not equal NaN")

		// NaN != normal value
		assert.False(t, nan1.Equals(normalValue), "NaN should not equal normal value")
		assert.False(t, normalValue.Equals(nan1), "normal value should not equal NaN")
	})
}

func TestHashableFloat64_NaN(t *testing.T) {
	t.Parallel()

	t.Run("NaN produces different hashes", func(t *testing.T) {
		t.Parallel()

		// Create two NaN values
		nan1 := HashableFloat64(math.NaN())
		nan2 := HashableFloat64(math.NaN())

		// Hash them multiple times
		hash1, err1 := Sha256(nan1)
		require.NoError(t, err1)

		hash2, err2 := Sha256(nan2)
		require.NoError(t, err2)

		// NaN values should produce different random hashes
		assert.NotEqual(t, hash1, hash2, "NaN values should produce different hashes")
	})

	t.Run("NaN Equals returns false", func(t *testing.T) {
		t.Parallel()

		nan1 := HashableFloat64(math.NaN())
		nan2 := HashableFloat64(math.NaN())
		normalValue := HashableFloat64(3.14159265359)

		// NaN != NaN
		assert.False(t, nan1.Equals(nan2), "NaN should not equal NaN")

		// NaN != normal value
		assert.False(t, nan1.Equals(normalValue), "NaN should not equal normal value")
		assert.False(t, normalValue.Equals(nan1), "normal value should not equal NaN")
	})
}

func TestHashHex(t *testing.T) {
	t.Parallel()

	t.Run("produces hex output", func(t *testing.T) {
		t.Parallel()

		input := HashableString("test")
		result, err := HashHex(input, sha256.New())
		require.NoError(t, err)
		assert.NotEmpty(t, result)

		// Verify it's valid hex (should only contain 0-9, a-f)
		assert.Regexp(t, "^[0-9a-f]+$", result)
	})

	t.Run("error handling", func(t *testing.T) {
		t.Parallel()

		mock := mockHashable{err: errHashTest}
		result, err := HashHex(mock, sha256.New())
		require.Error(t, err)
		assert.Equal(t, errHashTest, err)
		assert.Empty(t, result)
	})
}

func TestHashableUUID_UpdateHash(t *testing.T) {
	t.Parallel()

	u := HashableUUID{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef}
	h := &mockHash{}

	err := u.UpdateHash(h)
	require.NoError(t, err)
	assert.Len(t, h.data, 16)
	assert.Equal(t, u[:], h.data)
}

func TestHashableUUID_Equals(t *testing.T) {
	t.Parallel()

	uuid1 := HashableUUID{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef}
	uuid2 := HashableUUID{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef}
	uuid3 := HashableUUID{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	zeroUUID := HashableUUID{}

	tests := []struct {
		name     string
		a        HashableUUID
		b        HashableUUID
		expected bool
	}{
		{
			name:     "equal UUIDs",
			a:        uuid1,
			b:        uuid2,
			expected: true,
		},
		{
			name:     "different UUIDs",
			a:        uuid1,
			b:        uuid3,
			expected: false,
		},
		{
			name:     "zero UUIDs",
			a:        zeroUUID,
			b:        HashableUUID{},
			expected: true,
		},
		{
			name:     "zero vs non-zero",
			a:        zeroUUID,
			b:        uuid1,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.a.Equals(tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHashableUUID_HashConsistency(t *testing.T) {
	t.Parallel()

	uuid := HashableUUID{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef}

	hash1, err1 := Sha256(uuid)
	require.NoError(t, err1)

	hash2, err2 := Sha256(uuid)
	require.NoError(t, err2)

	assert.Equal(t, hash1, hash2, "same UUID should produce same hash")
}

func TestHashableUUID_DifferentInputs(t *testing.T) {
	t.Parallel()

	uuid1 := HashableUUID{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef}
	uuid2 := HashableUUID{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}

	hash1, err1 := Sha256(uuid1)
	require.NoError(t, err1)

	hash2, err2 := Sha256(uuid2)
	require.NoError(t, err2)

	assert.NotEqual(t, hash1, hash2, "different UUIDs should produce different hashes")
}
