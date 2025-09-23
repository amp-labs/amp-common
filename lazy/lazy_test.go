package lazy

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLazy(t *testing.T) {
	t.Parallel()

	count := 0
	stringToTest := "foo"
	strPtr := atomic.Pointer[string]{}
	strPtr.Store(&stringToTest)

	val := New[string](func() string {
		defer func() {
			// Increment the counter, but only if we don't panic.
			if err := recover(); err != nil {
				panic(err)
			}

			count++
		}()

		return *strPtr.Load() // might panic if strPtr is nil
	})

	// Never called
	assert.Equal(t, 0, count)
	assert.Falsef(t, val.Initialized(), "val should not be initialized")

	// Called once, but it should panic. Panics don't memoize.
	strPtr.Store(nil)

	assert.Panics(t, func() {
		val.Get()
	})

	// The lazy value should still be uninitialized after the panic.
	strPtr.Store(&stringToTest)

	assert.Equal(t, 0, count)
	assert.Falsef(t, val.Initialized(), "val should not be initialized")

	// The callback will get called once
	assert.Equal(t, "foo", val.Get())
	assert.Equal(t, 1, count)
	assert.Truef(t, val.Initialized(), "val should be initialized")

	// Called Get twice - should not invoke the callback again.
	assert.Equal(t, "foo", val.Get())
	assert.Equal(t, 1, count)
	assert.Truef(t, val.Initialized(), "val should be initialized")

	// Now do something which breaks the callback. Since the value is already
	// initialized, it shouldn't be called again (and thus won't panic this time).
	strPtr.Store(nil)

	assert.NotPanics(t, func() {
		assert.Equal(t, "foo", val.Get())
	})

	assert.Equal(t, 1, count)
	assert.Truef(t, val.Initialized(), "val should be initialized")
}

//nolint:funlen
func TestLazyErr(t *testing.T) {
	t.Parallel()

	count := 0
	stringToTest := "foo"
	strPtr := atomic.Pointer[string]{}
	strPtr.Store(&stringToTest)

	val := NewErr[string](func() (string, error) {
		defer func() {
			// Increment the counter, but only if we don't panic.
			if err := recover(); err != nil {
				panic(err)
			}

			count++
		}()

		toReturn := *strPtr.Load() // might panic if strPtr is nil

		if count < 3 {
			return toReturn, assert.AnError
		}

		return toReturn, nil
	})

	// Never called
	assert.Equal(t, 0, count)
	assert.Falsef(t, val.Initialized(), "val should not be initialized")

	// Called once, but it should panic. Panics don't memoize.
	strPtr.Store(nil)

	assert.Panics(t, func() {
		_, _ = val.Get()
	})

	// The lazy value should still be uninitialized after the panic.
	strPtr.Store(&stringToTest)

	assert.Equal(t, 0, count)
	assert.Falsef(t, val.Initialized(), "val should not be initialized")

	// The first callback will error out
	str, err := val.Get()
	assert.Equal(t, "", str)
	require.ErrorIs(t, err, assert.AnError)
	assert.Equal(t, 1, count)
	assert.Falsef(t, val.Initialized(), "val should not be initialized")

	// The second callback will error out
	str, err = val.Get()
	assert.Equal(t, "", str)
	require.ErrorIs(t, err, assert.AnError)
	assert.Equal(t, 2, count)
	assert.Falsef(t, val.Initialized(), "val should not be initialized")

	// The third callback will error out
	str, err = val.Get()
	assert.Equal(t, "", str)
	require.ErrorIs(t, err, assert.AnError)
	assert.Equal(t, 3, count)
	assert.Falsef(t, val.Initialized(), "val should not be initialized")

	// The callback will get called once
	str, err = val.Get()
	require.NoErrorf(t, err, "err should be nil")
	assert.Equal(t, "foo", str)
	assert.Equal(t, 4, count)
	assert.Truef(t, val.Initialized(), "val should be initialized")

	// Called Get twice - should not invoke the callback again.
	str, err = val.Get()
	require.NoErrorf(t, err, "err should be nil")
	assert.Equal(t, "foo", str)
	assert.Equal(t, 4, count)
	assert.Truef(t, val.Initialized(), "val should be initialized")

	// Now do something which breaks the callback. Since the value is already
	// initialized, it shouldn't be called again (and thus won't panic this time).
	strPtr.Store(nil)

	assert.NotPanics(t, func() {
		s, err := val.Get()
		require.NoErrorf(t, err, "err should be nil")
		assert.Equal(t, "foo", s)
	})

	assert.Equal(t, 4, count)
	assert.Truef(t, val.Initialized(), "val should be initialized")
}
