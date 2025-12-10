package pointer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const testString = "hello"

func TestTo(t *testing.T) {
	t.Parallel()

	t.Run("string", func(t *testing.T) {
		t.Parallel()

		str := testString
		ptr := To(str)

		assert.NotNil(t, ptr)
		assert.Equal(t, str, *ptr)

		// Ensure it's a different address
		assert.NotSame(t, &str, ptr)
	})

	t.Run("int", func(t *testing.T) {
		t.Parallel()

		num := 42
		ptr := To(num)

		assert.NotNil(t, ptr)
		assert.Equal(t, num, *ptr)
	})

	t.Run("bool", func(t *testing.T) {
		t.Parallel()

		b := true
		ptr := To(b)

		assert.NotNil(t, ptr)
		assert.Equal(t, b, *ptr)
	})

	t.Run("struct", func(t *testing.T) {
		t.Parallel()

		type testStruct struct {
			Name  string
			Value int
		}

		s := testStruct{Name: "test", Value: 123}
		ptr := To(s)

		assert.NotNil(t, ptr)
		assert.Equal(t, s, *ptr)
	})

	t.Run("literal", func(t *testing.T) {
		t.Parallel()

		// Test taking address of literal
		ptr := To("literal")

		assert.NotNil(t, ptr)
		assert.Equal(t, "literal", *ptr)
	})

	t.Run("zero value", func(t *testing.T) {
		t.Parallel()

		var zero int

		ptr := To(zero)

		assert.NotNil(t, ptr)
		assert.Equal(t, 0, *ptr)
	})
}

func TestValue(t *testing.T) {
	t.Parallel()

	t.Run("nil pointer", func(t *testing.T) {
		t.Parallel()

		var ptr *string

		val, ok := Value(ptr)

		assert.False(t, ok)
		assert.Empty(t, val) // zero value for string
	})

	t.Run("non-nil string pointer", func(t *testing.T) {
		t.Parallel()

		str := testString
		ptr := &str

		val, ok := Value(ptr)

		assert.True(t, ok)
		assert.Equal(t, "hello", val)
	})

	t.Run("non-nil int pointer", func(t *testing.T) {
		t.Parallel()

		num := 42
		ptr := &num

		val, ok := Value(ptr)

		assert.True(t, ok)
		assert.Equal(t, 42, val)
	})

	t.Run("nil int pointer returns zero value", func(t *testing.T) {
		t.Parallel()

		var ptr *int

		val, ok := Value(ptr)

		assert.False(t, ok)
		assert.Equal(t, 0, val)
	})

	t.Run("non-nil bool pointer", func(t *testing.T) {
		t.Parallel()

		b := true
		ptr := &b

		val, ok := Value(ptr)

		assert.True(t, ok)
		assert.True(t, val)
	})

	t.Run("nil struct pointer", func(t *testing.T) {
		t.Parallel()

		type testStruct struct {
			Name  string
			Value int
		}

		var ptr *testStruct

		val, ok := Value(ptr)

		assert.False(t, ok)
		assert.Equal(t, testStruct{}, val) // zero value for struct
	})

	t.Run("non-nil struct pointer", func(t *testing.T) {
		t.Parallel()

		type testStruct struct {
			Name  string
			Value int
		}

		s := testStruct{Name: "test", Value: 123}
		ptr := &s

		val, ok := Value(ptr)

		assert.True(t, ok)
		assert.Equal(t, s, val)
	})
}

func TestToAndValue(t *testing.T) {
	t.Parallel()

	t.Run("round trip", func(t *testing.T) {
		t.Parallel()

		original := "test value"
		ptr := To(original)
		retrieved, ok := Value(ptr)

		assert.True(t, ok)
		assert.Equal(t, original, retrieved)
	})

	t.Run("round trip with struct", func(t *testing.T) {
		t.Parallel()

		type testStruct struct {
			Name  string
			Value int
		}

		original := testStruct{Name: "foo", Value: 999}
		ptr := To(original)
		retrieved, ok := Value(ptr)

		assert.True(t, ok)
		assert.Equal(t, original, retrieved)
	})
}

func TestPointerWithSlice(t *testing.T) {
	t.Parallel()

	t.Run("slice pointer", func(t *testing.T) {
		t.Parallel()

		slice := []int{1, 2, 3}
		ptr := To(slice)

		assert.NotNil(t, ptr)
		assert.Equal(t, slice, *ptr)
	})

	t.Run("nil slice value", func(t *testing.T) {
		t.Parallel()

		var ptr *[]int

		val, ok := Value(ptr)

		assert.False(t, ok)
		assert.Nil(t, val) // zero value for slice is nil
	})
}

func TestPointerWithMap(t *testing.T) {
	t.Parallel()

	t.Run("map pointer", func(t *testing.T) {
		t.Parallel()

		m := map[string]int{"a": 1, "b": 2}
		ptr := To(m)

		assert.NotNil(t, ptr)
		assert.Equal(t, m, *ptr)
	})

	t.Run("nil map value", func(t *testing.T) {
		t.Parallel()

		var ptr *map[string]int

		val, ok := Value(ptr)

		assert.False(t, ok)
		assert.Nil(t, val) // zero value for map is nil
	})
}

func TestValueOrDefault(t *testing.T) {
	t.Parallel()

	t.Run("nil pointer returns default", func(t *testing.T) {
		t.Parallel()

		var ptr *string

		val := ValueOrDefault(ptr, "default")

		assert.Equal(t, "default", val)
	})

	t.Run("non-nil pointer returns value", func(t *testing.T) {
		t.Parallel()

		str := testString
		ptr := &str

		val := ValueOrDefault(ptr, "default")

		assert.Equal(t, testString, val)
	})

	t.Run("nil int pointer returns default", func(t *testing.T) {
		t.Parallel()

		var ptr *int

		val := ValueOrDefault(ptr, 42)

		assert.Equal(t, 42, val)
	})

	t.Run("non-nil int pointer returns value", func(t *testing.T) {
		t.Parallel()

		num := 100
		ptr := &num

		val := ValueOrDefault(ptr, 42)

		assert.Equal(t, 100, val)
	})

	t.Run("nil bool pointer returns default", func(t *testing.T) {
		t.Parallel()

		var ptr *bool

		val := ValueOrDefault(ptr, true)

		assert.True(t, val)
	})

	t.Run("non-nil struct pointer returns value", func(t *testing.T) {
		t.Parallel()

		type testStruct struct {
			Name  string
			Value int
		}

		s := testStruct{Name: "test", Value: 123}
		ptr := &s
		def := testStruct{Name: "default", Value: 0}

		val := ValueOrDefault(ptr, def)

		assert.Equal(t, s, val)
	})

	t.Run("nil struct pointer returns default", func(t *testing.T) {
		t.Parallel()

		type testStruct struct {
			Name  string
			Value int
		}

		var ptr *testStruct

		def := testStruct{Name: "default", Value: 999}
		val := ValueOrDefault(ptr, def)

		assert.Equal(t, def, val)
	})
}

func TestValueOrZero(t *testing.T) {
	t.Parallel()

	t.Run("nil string pointer returns empty string", func(t *testing.T) {
		t.Parallel()

		var ptr *string

		val := ValueOrZero(ptr)

		assert.Empty(t, val)
	})

	t.Run("non-nil string pointer returns value", func(t *testing.T) {
		t.Parallel()

		str := testString
		ptr := &str

		val := ValueOrZero(ptr)

		assert.Equal(t, testString, val)
	})

	t.Run("nil int pointer returns zero", func(t *testing.T) {
		t.Parallel()

		var ptr *int

		val := ValueOrZero(ptr)

		assert.Equal(t, 0, val)
	})

	t.Run("non-nil int pointer returns value", func(t *testing.T) {
		t.Parallel()

		num := 42
		ptr := &num

		val := ValueOrZero(ptr)

		assert.Equal(t, 42, val)
	})

	t.Run("nil bool pointer returns false", func(t *testing.T) {
		t.Parallel()

		var ptr *bool

		val := ValueOrZero(ptr)

		assert.False(t, val)
	})

	t.Run("non-nil bool pointer returns value", func(t *testing.T) {
		t.Parallel()

		b := true
		ptr := &b

		val := ValueOrZero(ptr)

		assert.True(t, val)
	})

	t.Run("nil struct pointer returns zero value", func(t *testing.T) {
		t.Parallel()

		type testStruct struct {
			Name  string
			Value int
		}

		var ptr *testStruct

		val := ValueOrZero(ptr)

		assert.Equal(t, testStruct{}, val)
	})

	t.Run("non-nil struct pointer returns value", func(t *testing.T) {
		t.Parallel()

		type testStruct struct {
			Name  string
			Value int
		}

		s := testStruct{Name: "test", Value: 123}
		ptr := &s

		val := ValueOrZero(ptr)

		assert.Equal(t, s, val)
	})

	t.Run("nil slice pointer returns nil", func(t *testing.T) {
		t.Parallel()

		var ptr *[]int

		val := ValueOrZero(ptr)

		assert.Nil(t, val)
	})

	t.Run("non-nil slice pointer returns value", func(t *testing.T) {
		t.Parallel()

		slice := []int{1, 2, 3}
		ptr := &slice

		val := ValueOrZero(ptr)

		assert.Equal(t, slice, val)
	})
}

func TestValueOrPanic(t *testing.T) {
	t.Parallel()

	t.Run("nil pointer panics", func(t *testing.T) {
		t.Parallel()

		var ptr *string

		assert.Panics(t, func() {
			ValueOrPanic(ptr)
		})
	})

	t.Run("non-nil string pointer returns value", func(t *testing.T) {
		t.Parallel()

		str := testString
		ptr := &str

		val := ValueOrPanic(ptr)

		assert.Equal(t, testString, val)
	})

	t.Run("nil int pointer panics", func(t *testing.T) {
		t.Parallel()

		var ptr *int

		assert.Panics(t, func() {
			ValueOrPanic(ptr)
		})
	})

	t.Run("non-nil int pointer returns value", func(t *testing.T) {
		t.Parallel()

		num := 42
		ptr := &num

		val := ValueOrPanic(ptr)

		assert.Equal(t, 42, val)
	})

	t.Run("nil bool pointer panics", func(t *testing.T) {
		t.Parallel()

		var ptr *bool

		assert.Panics(t, func() {
			ValueOrPanic(ptr)
		})
	})

	t.Run("non-nil struct pointer returns value", func(t *testing.T) {
		t.Parallel()

		type testStruct struct {
			Name  string
			Value int
		}

		s := testStruct{Name: "test", Value: 123}
		ptr := &s

		val := ValueOrPanic(ptr)

		assert.Equal(t, s, val)
	})

	t.Run("nil struct pointer panics", func(t *testing.T) {
		t.Parallel()

		type testStruct struct {
			Name  string
			Value int
		}

		var ptr *testStruct

		assert.Panics(t, func() {
			ValueOrPanic(ptr)
		})
	})

	t.Run("panic message contains expected text", func(t *testing.T) {
		t.Parallel()

		var ptr *string

		assert.PanicsWithValue(t, "pointer is nil", func() {
			ValueOrPanic(ptr)
		})
	})
}
