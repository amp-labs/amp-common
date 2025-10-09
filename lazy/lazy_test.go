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

func TestLazySet(t *testing.T) {
	t.Parallel()

	t.Run("Set before Get", func(t *testing.T) {
		t.Parallel()

		callCount := 0
		val := New[string](func() string {
			callCount++

			return "from-callback"
		})

		// Set the value before ever calling Get
		val.Set("from-set")

		assert.True(t, val.Initialized(), "val should be initialized after Set")
		assert.Equal(t, 0, callCount, "callback should never be called")

		// Get should return the set value, not invoke callback
		result := val.Get()
		assert.Equal(t, "from-set", result)
		assert.Equal(t, 0, callCount, "callback should never be called")
	})

	t.Run("Set after Get", func(t *testing.T) {
		t.Parallel()

		callCount := 0
		val := New[int](func() int {
			callCount++

			return 42
		})

		// Get the initial value
		result := val.Get()
		assert.Equal(t, 42, result)
		assert.Equal(t, 1, callCount)

		// Set a new value
		val.Set(100)

		// Get should now return the new value
		result = val.Get()
		assert.Equal(t, 100, result)
		assert.Equal(t, 1, callCount, "callback should not be called again")
	})

	t.Run("Set zero value", func(t *testing.T) {
		t.Parallel()

		callCount := 0
		val := New[int](func() int {
			callCount++

			return 42
		})

		// Set to zero value
		val.Set(0)

		assert.True(t, val.Initialized(), "val should be initialized")
		result := val.Get()
		assert.Equal(t, 0, result)
		assert.Equal(t, 0, callCount, "callback should never be called")
	})
}

func TestLazySetErr(t *testing.T) {
	t.Parallel()

	t.Run("Set before Get", func(t *testing.T) {
		t.Parallel()

		callCount := 0
		val := NewErr[string](func() (string, error) {
			callCount++

			return "from-callback", nil
		})

		// Set the value before ever calling Get
		val.Set("from-set")

		assert.True(t, val.Initialized(), "val should be initialized after Set")
		assert.Equal(t, 0, callCount, "callback should never be called")

		// Get should return the set value, not invoke callback
		result, err := val.Get()
		require.NoError(t, err)
		assert.Equal(t, "from-set", result)
		assert.Equal(t, 0, callCount, "callback should never be called")
	})

	t.Run("Set after successful Get", func(t *testing.T) {
		t.Parallel()

		callCount := 0
		val := NewErr[int](func() (int, error) {
			callCount++

			return 42, nil
		})

		// Get the initial value
		result, err := val.Get()
		require.NoError(t, err)
		assert.Equal(t, 42, result)
		assert.Equal(t, 1, callCount)

		// Set a new value
		val.Set(100)

		// Get should now return the new value
		result, err = val.Get()
		require.NoError(t, err)
		assert.Equal(t, 100, result)
		assert.Equal(t, 1, callCount, "callback should not be called again")
	})

	t.Run("Set after error", func(t *testing.T) {
		t.Parallel()

		callCount := 0
		val := NewErr[string](func() (string, error) {
			callCount++

			return "", assert.AnError
		})

		// First Get returns error
		result, err := val.Get()
		require.ErrorIs(t, err, assert.AnError)
		assert.Equal(t, "", result)
		assert.Equal(t, 1, callCount)
		assert.False(t, val.Initialized(), "val should not be initialized after error")

		// Set a value to recover from error
		val.Set("recovered")

		assert.True(t, val.Initialized(), "val should be initialized after Set")

		// Get should now return the set value
		result, err = val.Get()
		require.NoError(t, err)
		assert.Equal(t, "recovered", result)
		assert.Equal(t, 1, callCount, "callback should not be called again")
	})
}

func TestLazyConcurrency(t *testing.T) {
	t.Parallel()

	t.Run("concurrent Get calls", func(t *testing.T) {
		t.Parallel()

		callCount := atomic.Int32{}
		val := New[int](func() int {
			callCount.Add(1)

			return 42
		})

		const goroutines = 100
		done := make(chan int, goroutines)

		// Launch many goroutines all calling Get at once
		for range goroutines {
			go func() {
				result := val.Get()
				done <- result
			}()
		}

		// Collect all results
		for range goroutines {
			result := <-done
			assert.Equal(t, 42, result)
		}

		// Callback should be called exactly once
		assert.Equal(t, int32(1), callCount.Load())
		assert.True(t, val.Initialized())
	})

	t.Run("concurrent Get and Set", func(t *testing.T) {
		t.Parallel()

		val := New[int](func() int {
			return 42
		})

		const goroutines = 50
		done := make(chan bool, goroutines*2)

		// Launch goroutines calling Get
		for range goroutines {
			go func() {
				val.Get()
				done <- true
			}()
		}

		// Launch goroutines calling Set
		for i := range goroutines {
			go func() {
				val.Set(i)
				done <- true
			}()
		}

		// Wait for all goroutines
		for range goroutines * 2 {
			<-done
		}

		// Should be initialized and not panic
		assert.True(t, val.Initialized())
		assert.NotPanics(t, func() {
			val.Get()
		})
	})
}

func TestLazyErrConcurrency(t *testing.T) {
	t.Parallel()

	t.Run("concurrent Get calls with success", func(t *testing.T) {
		t.Parallel()

		callCount := atomic.Int32{}
		val := NewErr[int](func() (int, error) {
			callCount.Add(1)

			return 42, nil
		})

		const goroutines = 100
		done := make(chan bool, goroutines)

		// Launch many goroutines all calling Get at once
		for range goroutines {
			go func() {
				result, err := val.Get()
				assert.NoError(t, err)
				assert.Equal(t, 42, result)
				done <- true
			}()
		}

		// Wait for all goroutines
		for range goroutines {
			<-done
		}

		// Callback should be called exactly once
		assert.Equal(t, int32(1), callCount.Load())
		assert.True(t, val.Initialized())
	})

	t.Run("concurrent Get calls with errors", func(t *testing.T) { //nolint:funlen
		t.Parallel()

		callCount := atomic.Int32{}
		val := NewErr[int](func() (int, error) {
			count := callCount.Add(1)
			// First 3 calls error, then succeed
			if count <= 3 {
				return 0, assert.AnError
			}

			return 42, nil
		})

		const goroutines = 100
		results := make(chan error, goroutines)

		// Launch many goroutines all calling Get at once
		for range goroutines {
			go func() {
				_, err := val.Get()
				results <- err
			}()
		}

		// Collect and analyze results
		errorCount := 0
		successCount := 0

		for range goroutines {
			if <-results != nil {
				errorCount++
			} else {
				successCount++
			}
		}

		// We should have gotten at least some errors and some successes
		// The exact count depends on timing, but callback should be called
		// multiple times since errors don't memoize
		assert.Greater(t, callCount.Load(), int32(1))
		assert.Positive(t, errorCount, "should have some errors")
		assert.Positive(t, successCount, "should have some successes")
	})

	t.Run("concurrent Get and Set", func(t *testing.T) {
		t.Parallel()

		val := NewErr[int](func() (int, error) {
			return 42, nil
		})

		const goroutines = 50
		done := make(chan bool, goroutines*2)

		// Launch goroutines calling Get
		for range goroutines {
			go func() {
				_, _ = val.Get()
				done <- true
			}()
		}

		// Launch goroutines calling Set
		for i := range goroutines {
			go func() {
				val.Set(i)
				done <- true
			}()
		}

		// Wait for all goroutines
		for range goroutines * 2 {
			<-done
		}

		// Should be initialized and not panic
		assert.True(t, val.Initialized())
		assert.NotPanics(t, func() {
			_, _ = val.Get()
		})
	})
}

func TestLazyZeroValue(t *testing.T) {
	t.Parallel()

	t.Run("zero value Of without New", func(t *testing.T) {
		t.Parallel()

		var val Of[string]

		// Should not be initialized
		assert.False(t, val.Initialized())

		// Get should return zero value and not panic
		result := val.Get()
		assert.Equal(t, "", result)

		// When create is nil, the initialized flag is NOT set by Get()
		// This is because only the if block inside once.Do sets it
		assert.False(t, val.Initialized())
	})

	t.Run("zero value OfErr without New", func(t *testing.T) {
		t.Parallel()

		var val OfErr[string]

		// create is nil, so Initialized() returns true (since create == nil)
		// This is the actual behavior based on the implementation
		assert.True(t, val.Initialized())

		// Get should return zero value and no error
		result, err := val.Get()
		require.NoError(t, err)
		assert.Equal(t, "", result)

		// Should still be initialized
		assert.True(t, val.Initialized())
	})

	t.Run("New with nil function", func(t *testing.T) {
		t.Parallel()

		val := &Of[int]{create: nil}

		// With nil create function, not initialized yet
		assert.False(t, val.Initialized())

		// With nil create function, should not panic
		result := val.Get()
		assert.Equal(t, 0, result)

		// After Get with nil create, initialized is still false
		// because the if block inside once.Do doesn't execute
		assert.False(t, val.Initialized())
	})

	t.Run("zero value Of[int] returns zero", func(t *testing.T) {
		t.Parallel()

		var val Of[int]

		result := val.Get()
		assert.Equal(t, 0, result)
	})

	t.Run("zero value Of[*string] returns nil", func(t *testing.T) {
		t.Parallel()

		var val Of[*string]

		result := val.Get()
		assert.Nil(t, result)
	})
}

func TestLazyStructTypes(t *testing.T) {
	t.Parallel()

	t.Run("lazy with custom struct", func(t *testing.T) {
		t.Parallel()

		type TestStruct struct {
			Name  string
			Count int
		}

		callCount := 0
		val := New[TestStruct](func() TestStruct {
			callCount++

			return TestStruct{Name: "test", Count: 42}
		})

		result := val.Get()
		assert.Equal(t, "test", result.Name)
		assert.Equal(t, 42, result.Count)
		assert.Equal(t, 1, callCount)

		// Second call should not invoke callback
		_ = val.Get()

		assert.Equal(t, 1, callCount)
	})

	t.Run("lazy with pointer type", func(t *testing.T) {
		t.Parallel()

		type TestStruct struct {
			Value int
		}

		callCount := 0
		val := New[*TestStruct](func() *TestStruct {
			callCount++

			return &TestStruct{Value: 100}
		})

		result := val.Get()
		require.NotNil(t, result)
		assert.Equal(t, 100, result.Value)
		assert.Equal(t, 1, callCount)

		// Verify it returns same pointer
		result2 := val.Get()
		assert.Same(t, result, result2)
		assert.Equal(t, 1, callCount)
	})
}

func TestLazyMultiplePanics(t *testing.T) {
	t.Parallel()

	shouldPanic := atomic.Bool{}
	shouldPanic.Store(true)

	callCount := 0
	val := New[string](func() string {
		callCount++

		if shouldPanic.Load() {
			panic("intentional panic")
		}

		return "success"
	})

	// First call should panic
	assert.Panics(t, func() {
		val.Get()
	})
	assert.Equal(t, 1, callCount)
	assert.False(t, val.Initialized())

	// Second call should also panic
	assert.Panics(t, func() {
		val.Get()
	})
	assert.Equal(t, 2, callCount)
	assert.False(t, val.Initialized())

	// Disable panic and succeed
	shouldPanic.Store(false)

	result := val.Get()
	assert.Equal(t, "success", result)
	assert.Equal(t, 3, callCount)
	assert.True(t, val.Initialized())

	// Fourth call should not invoke callback
	result = val.Get()
	assert.Equal(t, "success", result)
	assert.Equal(t, 3, callCount)
}

func TestLazyErrMultiplePanics(t *testing.T) {
	t.Parallel()

	shouldPanic := atomic.Bool{}
	shouldPanic.Store(true)

	callCount := 0
	val := NewErr[string](func() (string, error) {
		callCount++

		if shouldPanic.Load() {
			panic("intentional panic")
		}

		return "success", nil
	})

	// First call should panic
	assert.Panics(t, func() {
		_, _ = val.Get()
	})
	assert.Equal(t, 1, callCount)
	assert.False(t, val.Initialized())

	// Second call should also panic
	assert.Panics(t, func() {
		_, _ = val.Get()
	})
	assert.Equal(t, 2, callCount)
	assert.False(t, val.Initialized())

	// Disable panic and succeed
	shouldPanic.Store(false)

	result, err := val.Get()
	require.NoError(t, err)
	assert.Equal(t, "success", result)
	assert.Equal(t, 3, callCount)
	assert.True(t, val.Initialized())

	// Fourth call should not invoke callback
	result, err = val.Get()
	require.NoError(t, err)
	assert.Equal(t, "success", result)
	assert.Equal(t, 3, callCount)
}
