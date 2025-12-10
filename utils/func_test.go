package utils //nolint:revive // utils is an appropriate package name for utility functions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetFunctionName(t *testing.T) {
	t.Parallel()

	t.Run("returns function name for named function", func(t *testing.T) {
		t.Parallel()

		name := GetFunctionName(TestGetFunctionName)
		assert.Contains(t, name, "TestGetFunctionName")
	})

	t.Run("returns <nil> for nil function", func(t *testing.T) {
		t.Parallel()

		name := GetFunctionName(nil)
		assert.Equal(t, "<nil>", name)
	})

	t.Run("returns <nil> for nil function pointer", func(t *testing.T) {
		t.Parallel()

		var f func()

		name := GetFunctionName(f)
		assert.Equal(t, "<nil>", name)
	})

	t.Run("returns function name for anonymous function", func(t *testing.T) {
		t.Parallel()

		fn := func() {}
		name := GetFunctionName(fn)
		assert.Contains(t, name, "func")
	})

	t.Run("returns <not a function> for non-function", func(t *testing.T) {
		t.Parallel()

		name := GetFunctionName("not a function")
		assert.Equal(t, "<not a function>", name)
	})
}
