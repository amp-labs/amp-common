package zero_test

import (
	"testing"

	"github.com/amp-labs/amp-common/zero"
	"github.com/stretchr/testify/assert"
)

type testStruct struct {
	Field1 string
	Field2 int
}

func TestValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "int returns 0",
			testFunc: func(t *testing.T) {
				t.Helper()

				result := zero.Value[int]()
				assert.Equal(t, 0, result)
			},
		},
		{
			name: "string returns empty string",
			testFunc: func(t *testing.T) {
				t.Helper()

				result := zero.Value[string]()
				assert.Empty(t, result)
			},
		},
		{
			name: "bool returns false",
			testFunc: func(t *testing.T) {
				t.Helper()

				result := zero.Value[bool]()
				assert.False(t, result)
			},
		},
		{
			name: "float64 returns 0.0",
			testFunc: func(t *testing.T) {
				t.Helper()

				result := zero.Value[float64]()
				assert.Zero(t, result)
			},
		},
		{
			name: "pointer returns nil",
			testFunc: func(t *testing.T) {
				t.Helper()

				result := zero.Value[*testStruct]()
				assert.Nil(t, result)
			},
		},
		{
			name: "struct returns zero-valued struct",
			testFunc: func(t *testing.T) {
				t.Helper()

				result := zero.Value[testStruct]()
				assert.Equal(t, testStruct{}, result)
				assert.Empty(t, result.Field1)
				assert.Equal(t, 0, result.Field2)
			},
		},
		{
			name: "slice returns nil slice",
			testFunc: func(t *testing.T) {
				t.Helper()

				result := zero.Value[[]string]()
				assert.Nil(t, result)
			},
		},
		{
			name: "map returns nil map",
			testFunc: func(t *testing.T) {
				t.Helper()

				result := zero.Value[map[string]int]()
				assert.Nil(t, result)
			},
		},
		{
			name: "channel returns nil channel",
			testFunc: func(t *testing.T) {
				t.Helper()

				result := zero.Value[chan int]()
				assert.Nil(t, result)
			},
		},
		{
			name: "interface returns nil",
			testFunc: func(t *testing.T) {
				t.Helper()

				result := zero.Value[error]()
				assert.NoError(t, result)
			},
		},
		{
			name: "uint returns 0",
			testFunc: func(t *testing.T) {
				t.Helper()

				result := zero.Value[uint]()
				assert.Equal(t, uint(0), result)
			},
		},
		{
			name: "int64 returns 0",
			testFunc: func(t *testing.T) {
				t.Helper()

				result := zero.Value[int64]()
				assert.Equal(t, int64(0), result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.testFunc(t)
		})
	}
}

func TestIsZero(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "int zero value returns true",
			testFunc: func(t *testing.T) {
				t.Helper()

				result := zero.IsZero(0)
				assert.True(t, result)
			},
		},
		{
			name: "int non-zero value returns false",
			testFunc: func(t *testing.T) {
				t.Helper()

				result := zero.IsZero(42)
				assert.False(t, result)
			},
		},
		{
			name: "int negative value returns false",
			testFunc: func(t *testing.T) {
				t.Helper()

				result := zero.IsZero(-1)
				assert.False(t, result)
			},
		},
		{
			name: "string empty returns true",
			testFunc: func(t *testing.T) {
				t.Helper()

				result := zero.IsZero("")
				assert.True(t, result)
			},
		},
		{
			name: "string non-empty returns false",
			testFunc: func(t *testing.T) {
				t.Helper()

				result := zero.IsZero("hello")
				assert.False(t, result)
			},
		},
		{
			name: "bool false returns true",
			testFunc: func(t *testing.T) {
				t.Helper()

				result := zero.IsZero(false)
				assert.True(t, result)
			},
		},
		{
			name: "bool true returns false",
			testFunc: func(t *testing.T) {
				t.Helper()

				result := zero.IsZero(true)
				assert.False(t, result)
			},
		},
		{
			name: "float64 zero returns true",
			testFunc: func(t *testing.T) {
				t.Helper()

				result := zero.IsZero(0.0)
				assert.True(t, result)
			},
		},
		{
			name: "float64 non-zero returns false",
			testFunc: func(t *testing.T) {
				t.Helper()

				result := zero.IsZero(3.14)
				assert.False(t, result)
			},
		},
		{
			name: "pointer nil returns true",
			testFunc: func(t *testing.T) {
				t.Helper()

				var ptr *testStruct

				result := zero.IsZero(ptr)
				assert.True(t, result)
			},
		},
		{
			name: "pointer non-nil returns false",
			testFunc: func(t *testing.T) {
				t.Helper()

				ptr := &testStruct{Field1: "test"}
				result := zero.IsZero(ptr)
				assert.False(t, result)
			},
		},
		{
			name: "struct zero-valued returns true",
			testFunc: func(t *testing.T) {
				t.Helper()

				result := zero.IsZero(testStruct{})
				assert.True(t, result)
			},
		},
		{
			name: "struct with values returns false",
			testFunc: func(t *testing.T) {
				t.Helper()

				result := zero.IsZero(testStruct{Field1: "test", Field2: 42})
				assert.False(t, result)
			},
		},
		{
			name: "struct with partial values returns false",
			testFunc: func(t *testing.T) {
				t.Helper()

				result := zero.IsZero(testStruct{Field1: "test"})
				assert.False(t, result)
			},
		},
		{
			name: "slice nil returns true",
			testFunc: func(t *testing.T) {
				t.Helper()

				var slice []string

				result := zero.IsZero(slice)
				assert.True(t, result)
			},
		},
		{
			name: "slice empty returns false",
			testFunc: func(t *testing.T) {
				t.Helper()

				slice := []string{}
				result := zero.IsZero(slice)
				assert.False(t, result)
			},
		},
		{
			name: "slice with values returns false",
			testFunc: func(t *testing.T) {
				t.Helper()

				slice := []string{"a", "b"}
				result := zero.IsZero(slice)
				assert.False(t, result)
			},
		},
		{
			name: "map nil returns true",
			testFunc: func(t *testing.T) {
				t.Helper()

				var m map[string]int

				result := zero.IsZero(m)
				assert.True(t, result)
			},
		},
		{
			name: "map empty returns false",
			testFunc: func(t *testing.T) {
				t.Helper()

				m := map[string]int{}
				result := zero.IsZero(m)
				assert.False(t, result)
			},
		},
		{
			name: "map with values returns false",
			testFunc: func(t *testing.T) {
				t.Helper()

				m := map[string]int{"key": 42}
				result := zero.IsZero(m)
				assert.False(t, result)
			},
		},
		{
			name: "channel nil returns true",
			testFunc: func(t *testing.T) {
				t.Helper()

				var ch chan int

				result := zero.IsZero(ch)
				assert.True(t, result)
			},
		},
		{
			name: "channel initialized returns false",
			testFunc: func(t *testing.T) {
				t.Helper()

				ch := make(chan int)
				defer close(ch)

				result := zero.IsZero(ch)
				assert.False(t, result)
			},
		},
		{
			name: "interface nil returns true",
			testFunc: func(t *testing.T) {
				t.Helper()

				var err error

				result := zero.IsZero(err)
				assert.True(t, result)
			},
		},
		{
			name: "interface with value returns false",
			testFunc: func(t *testing.T) {
				t.Helper()

				err := assert.AnError
				result := zero.IsZero(err)
				assert.False(t, result)
			},
		},
		{
			name: "uint zero returns true",
			testFunc: func(t *testing.T) {
				t.Helper()

				result := zero.IsZero(uint(0))
				assert.True(t, result)
			},
		},
		{
			name: "uint non-zero returns false",
			testFunc: func(t *testing.T) {
				t.Helper()

				result := zero.IsZero(uint(42))
				assert.False(t, result)
			},
		},
		{
			name: "int64 zero returns true",
			testFunc: func(t *testing.T) {
				t.Helper()

				result := zero.IsZero(int64(0))
				assert.True(t, result)
			},
		},
		{
			name: "int64 non-zero returns false",
			testFunc: func(t *testing.T) {
				t.Helper()

				result := zero.IsZero(int64(42))
				assert.False(t, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.testFunc(t)
		})
	}
}
