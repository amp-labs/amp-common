package hashing

import (
	"errors"
	"hash"
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
