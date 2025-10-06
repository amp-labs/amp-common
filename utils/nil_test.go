package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsNilish(t *testing.T) {
	t.Parallel()

	t.Run("returns true for nil", func(t *testing.T) {
		t.Parallel()

		assert.True(t, IsNilish(nil))
	})

	t.Run("returns true for nil pointer", func(t *testing.T) {
		t.Parallel()

		var ptr *int

		assert.True(t, IsNilish(ptr))
	})

	t.Run("returns false for non-nil pointer", func(t *testing.T) {
		t.Parallel()

		val := 42
		ptr := &val
		assert.False(t, IsNilish(ptr))
	})

	t.Run("returns true for nil slice", func(t *testing.T) {
		t.Parallel()

		var slice []int

		assert.True(t, IsNilish(slice))
	})

	t.Run("returns false for empty slice", func(t *testing.T) {
		t.Parallel()

		slice := []int{}
		assert.False(t, IsNilish(slice))
	})

	t.Run("returns true for nil map", func(t *testing.T) {
		t.Parallel()

		var m map[string]int

		assert.True(t, IsNilish(m))
	})

	t.Run("returns false for empty map", func(t *testing.T) {
		t.Parallel()

		m := make(map[string]int)
		assert.False(t, IsNilish(m))
	})

	t.Run("returns true for nil channel", func(t *testing.T) {
		t.Parallel()

		var ch chan int

		assert.True(t, IsNilish(ch))
	})

	t.Run("returns false for non-nil channel", func(t *testing.T) {
		t.Parallel()

		ch := make(chan int)
		defer close(ch)
		assert.False(t, IsNilish(ch))
	})

	t.Run("returns true for nil function", func(t *testing.T) {
		t.Parallel()

		var fn func()

		assert.True(t, IsNilish(fn))
	})

	t.Run("returns false for non-nil function", func(t *testing.T) {
		t.Parallel()

		fn := func() {}
		assert.False(t, IsNilish(fn))
	})

	t.Run("returns true for nil interface", func(t *testing.T) {
		t.Parallel()

		var iface interface{}

		assert.True(t, IsNilish(iface))
	})

	t.Run("returns false for non-nil interface with nil value", func(t *testing.T) {
		t.Parallel()

		var ptr *int

		var iface interface{} = ptr

		assert.True(t, IsNilish(iface))
	})

	t.Run("returns false for primitive types", func(t *testing.T) {
		t.Parallel()

		assert.False(t, IsNilish(0))
		assert.False(t, IsNilish(""))
		assert.False(t, IsNilish(false))
	})

	t.Run("returns false for struct", func(t *testing.T) {
		t.Parallel()

		type testStruct struct{}

		assert.False(t, IsNilish(testStruct{}))
	})
}
