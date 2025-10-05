package assert_test

import (
	"testing"

	"github.com/amp-labs/amp-common/assert"
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
		require.Equal(t, "", result)
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
