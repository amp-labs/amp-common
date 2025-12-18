package assert_test

import (
	"context"
	"testing"

	"github.com/amp-labs/amp-common/assert"
	"github.com/amp-labs/amp-common/contexts"
	commonerrors "github.com/amp-labs/amp-common/errors"
	"github.com/stretchr/testify/require"
)

func TestType_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    any
		expected any
	}{
		{
			name:     "string type assertion",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "int type assertion",
			input:    42,
			expected: 42,
		},
		{
			name:     "bool type assertion",
			input:    true,
			expected: true,
		},
		{
			name:     "float64 type assertion",
			input:    3.14,
			expected: 3.14,
		},
		{
			name:     "slice type assertion",
			input:    []int{1, 2, 3},
			expected: []int{1, 2, 3},
		},
		{
			name:     "map type assertion",
			input:    map[string]int{"a": 1},
			expected: map[string]int{"a": 1},
		},
		{
			name:     "struct type assertion",
			input:    struct{ Name string }{Name: "test"},
			expected: struct{ Name string }{Name: "test"},
		},
		{
			name:     "pointer type assertion",
			input:    new(int),
			expected: new(int),
		},
		{
			name:     "nil interface",
			input:    any(nil),
			expected: any(nil),
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			switch expected := testCase.expected.(type) {
			case string:
				result, err := assert.Type[string](testCase.input)
				require.NoError(t, err)
				require.Equal(t, expected, result)
			case int:
				result, err := assert.Type[int](testCase.input)
				require.NoError(t, err)
				require.Equal(t, expected, result)
			case bool:
				result, err := assert.Type[bool](testCase.input)
				require.NoError(t, err)
				require.Equal(t, expected, result)
			case float64:
				result, err := assert.Type[float64](testCase.input)
				require.NoError(t, err)
				require.InDelta(t, expected, result, 0.0001)
			case []int:
				result, err := assert.Type[[]int](testCase.input)
				require.NoError(t, err)
				require.Equal(t, expected, result)
			case map[string]int:
				result, err := assert.Type[map[string]int](testCase.input)
				require.NoError(t, err)
				require.Equal(t, expected, result)
			case struct{ Name string }:
				result, err := assert.Type[struct{ Name string }](testCase.input)
				require.NoError(t, err)
				require.Equal(t, expected, result)
			case *int:
				result, err := assert.Type[*int](testCase.input)
				require.NoError(t, err)
				require.NotNil(t, result)
			case any:
				result, err := assert.Type[any](testCase.input)
				require.NoError(t, err)
				require.Nil(t, result)
			}
		})
	}
}

func TestType_Failure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         any
		assertType    string
		expectedError string
	}{
		{
			name:          "string to int",
			input:         "hello",
			assertType:    "int",
			expectedError: "expected type int, but received string",
		},
		{
			name:          "int to string",
			input:         42,
			assertType:    "string",
			expectedError: "expected type string, but received int",
		},
		{
			name:          "bool to int",
			input:         true,
			assertType:    "int",
			expectedError: "expected type int, but received bool",
		},
		{
			name:          "float64 to int",
			input:         3.14,
			assertType:    "int",
			expectedError: "expected type int, but received float64",
		},
		{
			name:          "slice to map",
			input:         []int{1, 2, 3},
			assertType:    "map",
			expectedError: "expected type map[string]int, but received []int",
		},
		{
			name:          "nil to string",
			input:         nil,
			assertType:    "string",
			expectedError: "expected type string, but received <nil>",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var err error

			switch testCase.assertType {
			case "int":
				_, err = assert.Type[int](testCase.input)
			case "string":
				_, err = assert.Type[string](testCase.input)
			case "map":
				_, err = assert.Type[map[string]int](testCase.input)
			}

			require.Error(t, err)
			require.ErrorIs(t, err, commonerrors.ErrWrongType)
			require.Contains(t, err.Error(), testCase.expectedError)
		})
	}
}

func TestType_ZeroValue(t *testing.T) {
	t.Parallel()

	t.Run("returns zero value on failure", func(t *testing.T) {
		t.Parallel()

		result, err := assert.Type[int]("not an int")
		require.Error(t, err)
		require.Equal(t, 0, result)
	})

	t.Run("returns zero value for string on failure", func(t *testing.T) {
		t.Parallel()

		result, err := assert.Type[string](123)
		require.Error(t, err)
		require.Empty(t, result)
	})

	t.Run("returns nil pointer on failure", func(t *testing.T) {
		t.Parallel()

		result, err := assert.Type[*int]("not a pointer")
		require.Error(t, err)
		require.Nil(t, result)
	})
}

func TestType_InterfaceTypes(t *testing.T) {
	t.Parallel()

	t.Run("any type always succeeds", func(t *testing.T) {
		t.Parallel()

		result, err := assert.Type[any]("anything")
		require.NoError(t, err)
		require.Equal(t, "anything", result)
	})

	t.Run("error interface", func(t *testing.T) {
		t.Parallel()

		inputErr := commonerrors.ErrWrongType
		result, err := assert.Type[error](inputErr)
		require.NoError(t, err)
		require.Equal(t, inputErr, result)
	})
}

func TestType_PointerTypes(t *testing.T) {
	t.Parallel()

	t.Run("pointer to value type", func(t *testing.T) {
		t.Parallel()

		val := 42
		result, err := assert.Type[*int](&val)
		require.NoError(t, err)
		require.Equal(t, &val, result)
		require.Equal(t, 42, *result)
	})

	t.Run("value to pointer type fails", func(t *testing.T) {
		t.Parallel()

		val := 42
		result, err := assert.Type[*int](val)
		require.Error(t, err)
		require.ErrorIs(t, err, commonerrors.ErrWrongType)
		require.Nil(t, result)
	})
}

func TestType_CustomTypes(t *testing.T) {
	t.Parallel()

	type CustomString string

	t.Run("custom string type", func(t *testing.T) {
		t.Parallel()

		input := CustomString("test")
		result, err := assert.Type[CustomString](input)
		require.NoError(t, err)
		require.Equal(t, CustomString("test"), result)
	})

	t.Run("custom type to underlying type fails", func(t *testing.T) {
		t.Parallel()

		input := CustomString("test")
		_, err := assert.Type[string](input)
		require.Error(t, err)
		require.ErrorIs(t, err, commonerrors.ErrWrongType)
	})

	t.Run("underlying type to custom type fails", func(t *testing.T) {
		t.Parallel()

		input := "test"
		_, err := assert.Type[CustomString](input)
		require.Error(t, err)
		require.ErrorIs(t, err, commonerrors.ErrWrongType)
	})
}

func TestTrue(t *testing.T) {
	t.Parallel()

	t.Run("does not panic when value is true", func(t *testing.T) {
		t.Parallel()

		require.NotPanics(t, func() {
			assert.True(true)
		})
	})

	t.Run("panics with default message when value is false", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "assertion failed", func() {
			assert.True(false)
		})
	})

	t.Run("panics with custom message when value is false", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "custom error message", func() {
			assert.True(false, "custom error message")
		})
	})

	t.Run("panics with formatted message when value is false", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "expected 42 but got 0", func() {
			assert.True(false, "expected %d but got %d", 42, 0)
		})
	})

	t.Run("panics with args when first arg is not string", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "assertion failed: [42 test]", func() {
			assert.True(false, 42, "test")
		})
	})
}

func TestFalse(t *testing.T) {
	t.Parallel()

	t.Run("does not panic when value is false", func(t *testing.T) {
		t.Parallel()

		require.NotPanics(t, func() {
			assert.False(false)
		})
	})

	t.Run("panics with default message when value is true", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "assertion failed", func() {
			assert.False(true)
		})
	})

	t.Run("panics with custom message when value is true", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "expected false but got true", func() {
			assert.False(true, "expected false but got true")
		})
	})

	t.Run("panics with formatted message when value is true", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "expected false", func() {
			assert.False(true, "expected %s", "false")
		})
	})
}

func TestNil(t *testing.T) {
	t.Parallel()

	t.Run("does not panic when value is nil", func(t *testing.T) {
		t.Parallel()

		require.NotPanics(t, func() {
			assert.Nil(nil)
		})
	})

	t.Run("panics when typed nil pointer is not recognized as nil", func(t *testing.T) {
		t.Parallel()

		// This is a Go gotcha: typed nil pointers are not nil when passed as any
		// because the interface contains type information
		var ptr *int

		require.PanicsWithValue(t, "assertion failed", func() {
			assert.Nil(ptr)
		})
	})

	t.Run("panics with default message when value is not nil", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "assertion failed", func() {
			assert.Nil("not nil")
		})
	})

	t.Run("panics with custom message when value is not nil", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "expected nil value", func() {
			assert.Nil(42, "expected nil value")
		})
	})

	t.Run("panics when non-nil pointer", func(t *testing.T) {
		t.Parallel()

		val := 42

		require.PanicsWithValue(t, "pointer should be nil", func() {
			assert.Nil(&val, "pointer should be nil")
		})
	})
}

func TestNotNil(t *testing.T) {
	t.Parallel()

	t.Run("does not panic when value is not nil", func(t *testing.T) {
		t.Parallel()

		require.NotPanics(t, func() {
			assert.NotNil("not nil")
		})
	})

	t.Run("does not panic when non-nil pointer", func(t *testing.T) {
		t.Parallel()

		val := 42

		require.NotPanics(t, func() {
			assert.NotNil(&val)
		})
	})

	t.Run("panics with default message when value is nil", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "assertion failed", func() {
			assert.NotNil(nil)
		})
	})

	t.Run("panics with custom message when value is nil", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "value must not be nil", func() {
			assert.NotNil(nil, "value must not be nil")
		})
	})

	t.Run("does not panic for typed nil pointer", func(t *testing.T) {
		t.Parallel()

		// This is a Go gotcha: typed nil pointers are not nil when passed as any
		// because the interface contains type information
		var ptr *int

		require.NotPanics(t, func() {
			assert.NotNil(ptr)
		})
	})
}

func TestContextHasValue(t *testing.T) {
	t.Parallel()

	type key string

	t.Run("does not panic when context has value", func(t *testing.T) {
		t.Parallel()

		ctx := contexts.WithValue(context.Background(), key("testKey"), "testValue")

		require.NotPanics(t, func() {
			assert.ContextHasValue[key, string](ctx, "testKey")
		})
	})

	t.Run("panics with default message when context does not have value", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		require.PanicsWithValue(t, "assertion failed", func() {
			assert.ContextHasValue[key, string](ctx, "missingKey")
		})
	})

	t.Run("panics with custom message when context does not have value", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		require.PanicsWithValue(t, "key not found in context", func() {
			assert.ContextHasValue[key, string](ctx, "missingKey", "key not found in context")
		})
	})

	t.Run("panics with formatted message when context does not have value", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		require.PanicsWithValue(t, "expected key myKey in context", func() {
			assert.ContextHasValue[key, string](ctx, "missingKey", "expected key %s in context", "myKey")
		})
	})
}

func TestContextDoesNotHaveValue(t *testing.T) {
	t.Parallel()

	type key string

	t.Run("does not panic when context does not have value", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		require.NotPanics(t, func() {
			assert.ContextDoesNotHaveValue[key, string](ctx, "missingKey")
		})
	})

	t.Run("panics with default message when context has value", func(t *testing.T) {
		t.Parallel()

		ctx := contexts.WithValue(context.Background(), key("testKey"), "testValue")

		require.PanicsWithValue(t, "assertion failed", func() {
			assert.ContextDoesNotHaveValue[key, string](ctx, "testKey")
		})
	})

	t.Run("panics with custom message when context has value", func(t *testing.T) {
		t.Parallel()

		ctx := contexts.WithValue(context.Background(), key("testKey"), "testValue")

		require.PanicsWithValue(t, "key should not be in context", func() {
			assert.ContextDoesNotHaveValue[key, string](ctx, "testKey", "key should not be in context")
		})
	})

	t.Run("panics with formatted message when context has value", func(t *testing.T) {
		t.Parallel()

		ctx := contexts.WithValue(context.Background(), key("testKey"), "testValue")

		require.PanicsWithValue(t, "unexpected key testKey found", func() {
			assert.ContextDoesNotHaveValue[key, string](ctx, "testKey", "unexpected key %s found", "testKey")
		})
	})
}

func TestContextIsAlive(t *testing.T) {
	t.Parallel()

	t.Run("does not panic when context is alive", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		require.NotPanics(t, func() {
			assert.ContextIsAlive(ctx)
		})
	})

	t.Run("panics with default message when context is cancelled", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		require.PanicsWithValue(t, "assertion failed", func() {
			assert.ContextIsAlive(ctx)
		})
	})

	t.Run("panics with custom message when context is cancelled", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		require.PanicsWithValue(t, "context should be alive", func() {
			assert.ContextIsAlive(ctx, "context should be alive")
		})
	})

	t.Run("panics with formatted message when context is cancelled", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		require.PanicsWithValue(t, "expected context to be alive but was cancelled", func() {
			assert.ContextIsAlive(ctx, "expected context to be %s but was %s", "alive", "cancelled")
		})
	})
}

func TestContextIsDead(t *testing.T) {
	t.Parallel()

	t.Run("does not panic when context is cancelled", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		require.NotPanics(t, func() {
			assert.ContextIsDead(ctx)
		})
	})

	t.Run("panics with default message when context is alive", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		require.PanicsWithValue(t, "assertion failed", func() {
			assert.ContextIsDead(ctx)
		})
	})

	t.Run("panics with custom message when context is alive", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		require.PanicsWithValue(t, "context should be cancelled", func() {
			assert.ContextIsDead(ctx, "context should be cancelled")
		})
	})

	t.Run("panics with formatted message when context is alive", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		require.PanicsWithValue(t, "expected context to be dead", func() {
			assert.ContextIsDead(ctx, "expected context to be %s", "dead")
		})
	})
}

func TestEmptySlice(t *testing.T) {
	t.Parallel()

	t.Run("does not panic when slice is empty", func(t *testing.T) {
		t.Parallel()

		require.NotPanics(t, func() {
			assert.EmptySlice([]int{})
		})
	})

	t.Run("does not panic when slice is nil", func(t *testing.T) {
		t.Parallel()

		var slice []string

		require.NotPanics(t, func() {
			assert.EmptySlice(slice)
		})
	})

	t.Run("panics with default message when slice is not empty", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "assertion failed", func() {
			assert.EmptySlice([]int{1, 2, 3})
		})
	})

	t.Run("panics with custom message when slice is not empty", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "slice should be empty", func() {
			assert.EmptySlice([]string{"a", "b"}, "slice should be empty")
		})
	})

	t.Run("panics with formatted message when slice is not empty", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "expected empty slice but got 3 items", func() {
			assert.EmptySlice([]int{1, 2, 3}, "expected empty slice but got %d items", 3)
		})
	})

	t.Run("panics with args when first arg is not string", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "assertion failed: [42 test]", func() {
			assert.EmptySlice([]int{1}, 42, "test")
		})
	})
}

func TestNonEmptySlice(t *testing.T) {
	t.Parallel()

	t.Run("does not panic when slice is not empty", func(t *testing.T) {
		t.Parallel()

		require.NotPanics(t, func() {
			assert.NonEmptySlice([]int{1, 2, 3})
		})
	})

	t.Run("panics with default message when slice is empty", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "assertion failed", func() {
			assert.NonEmptySlice([]int{})
		})
	})

	t.Run("panics with default message when slice is nil", func(t *testing.T) {
		t.Parallel()

		var slice []string

		require.PanicsWithValue(t, "assertion failed", func() {
			assert.NonEmptySlice(slice)
		})
	})

	t.Run("panics with custom message when slice is empty", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "slice should not be empty", func() {
			assert.NonEmptySlice([]string{}, "slice should not be empty")
		})
	})

	t.Run("panics with formatted message when slice is empty", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "expected at least 1 item", func() {
			assert.NonEmptySlice([]int{}, "expected at least %d item", 1)
		})
	})

	t.Run("panics with args when first arg is not string", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "assertion failed: [99]", func() {
			assert.NonEmptySlice([]int{}, 99)
		})
	})
}

func TestEmptyMap(t *testing.T) {
	t.Parallel()

	t.Run("does not panic when map is empty", func(t *testing.T) {
		t.Parallel()

		require.NotPanics(t, func() {
			assert.EmptyMap(map[string]int{})
		})
	})

	t.Run("does not panic when map is nil", func(t *testing.T) {
		t.Parallel()

		var m map[string]string

		require.NotPanics(t, func() {
			assert.EmptyMap(m)
		})
	})

	t.Run("panics with default message when map is not empty", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "assertion failed", func() {
			assert.EmptyMap(map[string]int{"a": 1, "b": 2})
		})
	})

	t.Run("panics with custom message when map is not empty", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "map should be empty", func() {
			assert.EmptyMap(map[int]bool{1: true}, "map should be empty")
		})
	})

	t.Run("panics with formatted message when map is not empty", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "expected empty map but got 2 entries", func() {
			assert.EmptyMap(map[string]int{"a": 1, "b": 2}, "expected empty map but got %d entries", 2)
		})
	})

	t.Run("panics with args when first arg is not string", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "assertion failed: [123 error]", func() {
			assert.EmptyMap(map[string]int{"a": 1}, 123, "error")
		})
	})
}

func TestNonEmptyMap(t *testing.T) {
	t.Parallel()

	t.Run("does not panic when map is not empty", func(t *testing.T) {
		t.Parallel()

		require.NotPanics(t, func() {
			assert.NonEmptyMap(map[string]int{"a": 1})
		})
	})

	t.Run("panics with default message when map is empty", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "assertion failed", func() {
			assert.NonEmptyMap(map[string]int{})
		})
	})

	t.Run("panics with default message when map is nil", func(t *testing.T) {
		t.Parallel()

		var m map[int]string

		require.PanicsWithValue(t, "assertion failed", func() {
			assert.NonEmptyMap(m)
		})
	})

	t.Run("panics with custom message when map is empty", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "map should not be empty", func() {
			assert.NonEmptyMap(map[string]bool{}, "map should not be empty")
		})
	})

	t.Run("panics with formatted message when map is empty", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "expected at least 1 entry", func() {
			assert.NonEmptyMap(map[string]int{}, "expected at least %d entry", 1)
		})
	})

	t.Run("panics with args when first arg is not string", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, "assertion failed: [false]", func() {
			assert.NonEmptyMap(map[string]int{}, false)
		})
	})
}
