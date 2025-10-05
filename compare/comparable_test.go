package compare

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestString is a simple string wrapper that implements Comparable.
type TestString string

func (s TestString) Equals(other TestString) bool {
	return string(s) == string(other)
}

// TestNumber is a numeric type that implements Comparable.
type TestNumber int

func (n TestNumber) Equals(other TestNumber) bool {
	return int(n) == int(other)
}

// TestStruct is a struct that implements Comparable with custom equality logic.
type TestStruct struct {
	ID   int
	Name string
}

func (t TestStruct) Equals(other TestStruct) bool {
	return t.ID == other.ID && t.Name == other.Name
}

// CaseInsensitiveString demonstrates custom equality semantics.
type CaseInsensitiveString string

func (s CaseInsensitiveString) Equals(other CaseInsensitiveString) bool {
	return string(s) == string(other) // Simple equality for testing
}

func TestComparable_TestString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		a        TestString
		b        TestString
		expected bool
	}{
		{
			name:     "equal strings",
			a:        "hello",
			b:        "hello",
			expected: true,
		},
		{
			name:     "different strings",
			a:        "hello",
			b:        "world",
			expected: false,
		},
		{
			name:     "empty strings",
			a:        "",
			b:        "",
			expected: true,
		},
		{
			name:     "one empty string",
			a:        "hello",
			b:        "",
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

func TestComparable_TestNumber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		a        TestNumber
		b        TestNumber
		expected bool
	}{
		{
			name:     "equal numbers",
			a:        42,
			b:        42,
			expected: true,
		},
		{
			name:     "different numbers",
			a:        42,
			b:        24,
			expected: false,
		},
		{
			name:     "zero values",
			a:        0,
			b:        0,
			expected: true,
		},
		{
			name:     "negative numbers",
			a:        -5,
			b:        -5,
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

func TestComparable_TestStruct(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		a        TestStruct
		b        TestStruct
		expected bool
	}{
		{
			name:     "equal structs",
			a:        TestStruct{ID: 1, Name: "Alice"},
			b:        TestStruct{ID: 1, Name: "Alice"},
			expected: true,
		},
		{
			name:     "different IDs",
			a:        TestStruct{ID: 1, Name: "Alice"},
			b:        TestStruct{ID: 2, Name: "Alice"},
			expected: false,
		},
		{
			name:     "different names",
			a:        TestStruct{ID: 1, Name: "Alice"},
			b:        TestStruct{ID: 1, Name: "Bob"},
			expected: false,
		},
		{
			name:     "zero values",
			a:        TestStruct{},
			b:        TestStruct{},
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

func TestEquals_Function(t *testing.T) {
	t.Parallel()

	t.Run("with TestString", func(t *testing.T) {
		t.Parallel()

		a := TestString("hello")
		b := TestString("hello")
		c := TestString("world")

		assert.True(t, Equals(a, b))
		assert.False(t, Equals(a, c))
	})

	t.Run("with TestNumber", func(t *testing.T) {
		t.Parallel()

		a := TestNumber(42)
		b := TestNumber(42)
		c := TestNumber(24)

		assert.True(t, Equals(a, b))
		assert.False(t, Equals(a, c))
	})

	t.Run("with TestStruct", func(t *testing.T) {
		t.Parallel()

		a := TestStruct{ID: 1, Name: "Alice"}
		b := TestStruct{ID: 1, Name: "Alice"}
		c := TestStruct{ID: 2, Name: "Bob"}

		assert.True(t, Equals(a, b))
		assert.False(t, Equals(a, c))
	})
}
