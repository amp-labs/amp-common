package common

import (
	"errors"
	"io"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	errCloseFailed     = errors.New("close failed")
	errConcurrentClose = errors.New("concurrent close error")
	errMultiple1       = errors.New("error 1")
	errMultiple2       = errors.New("error 2")
	errMultiple3       = errors.New("error 3")
	errCustomClose     = errors.New("custom close error")
	errCleanup1        = errors.New("cleanup error 1")
	errCleanup2        = errors.New("cleanup error 2")
	errCleanup3        = errors.New("cleanup error 3")
	errTransient       = errors.New("transient error")
)

// mockCloser is a test implementation of io.Closer.
type mockCloser struct {
	closeCount int
	closeError error
	mu         sync.Mutex
}

func (m *mockCloser) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.closeCount++

	return m.closeError
}

func (m *mockCloser) getCloseCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.closeCount
}

// Tests for CustomCloser

func TestCustomCloser_NilFunction(t *testing.T) {
	t.Parallel()

	result := CustomCloser(nil)
	assert.Nil(t, result, "CustomCloser should return nil for nil function")
}

func TestCustomCloser_BasicClose(t *testing.T) {
	t.Parallel()

	closeCalled := false
	closeFn := func() error {
		closeCalled = true

		return nil
	}

	closer := CustomCloser(closeFn)
	require.NotNil(t, closer)

	err := closer.Close()
	require.NoError(t, err)
	assert.True(t, closeCalled, "Close function should have been called")
}

func TestCustomCloser_ErrorPropagation(t *testing.T) {
	t.Parallel()

	closeFn := func() error {
		return errCustomClose
	}

	closer := CustomCloser(closeFn)
	require.NotNil(t, closer)

	err := closer.Close()
	assert.Equal(t, errCustomClose, err, "Error from close function should be propagated")
}

func TestCustomCloser_MultipleCloses(t *testing.T) {
	t.Parallel()

	closeCount := 0
	closeFn := func() error {
		closeCount++

		return nil
	}

	closer := CustomCloser(closeFn)
	require.NotNil(t, closer)

	// customCloser is NOT idempotent by itself (that's CloseOnce's job)
	err1 := closer.Close()
	err2 := closer.Close()
	err3 := closer.Close()

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NoError(t, err3)
	assert.Equal(t, 3, closeCount, "customCloser allows multiple closes")
}

func TestCustomCloser_WithCloseOnce(t *testing.T) {
	t.Parallel()

	closeCount := 0
	closeFn := func() error {
		closeCount++

		return nil
	}

	closer := CloseOnce(CustomCloser(closeFn))
	require.NotNil(t, closer)

	// Close multiple times
	err1 := closer.Close()
	err2 := closer.Close()
	err3 := closer.Close()

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NoError(t, err3)
	assert.Equal(t, 1, closeCount, "CloseOnce wrapper should ensure single close")
}

func TestCustomCloser_WithHandlePanic(t *testing.T) {
	t.Parallel()

	closeFn := func() error {
		panic("cleanup panic")
	}

	closer := HandlePanic(CustomCloser(closeFn))
	require.NotNil(t, closer)

	err := closer.Close()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cleanup panic", "Panic should be converted to error")
}

func TestCustomCloser_WithCloserCollector(t *testing.T) {
	t.Parallel()

	closedOrder := []int{}

	var mutex sync.Mutex

	makeCustomCloser := func(id int) io.Closer {
		return CustomCloser(func() error {
			mutex.Lock()
			closedOrder = append(closedOrder, id)
			mutex.Unlock()

			return nil
		})
	}

	collector := NewCloser()
	collector.Add(makeCustomCloser(1))
	collector.Add(makeCustomCloser(2))
	collector.Add(makeCustomCloser(3))

	err := collector.Close()
	require.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, closedOrder, "Custom closers should be closed in order")
}

func TestCustomCloser_MultipleErrors(t *testing.T) {
	t.Parallel()

	collector := NewCloser()
	collector.Add(CustomCloser(func() error { return errCleanup1 }))
	collector.Add(CustomCloser(func() error { return nil }))
	collector.Add(CustomCloser(func() error { return errCleanup2 }))
	collector.Add(CustomCloser(func() error { return errCleanup3 }))

	err := collector.Close()
	require.Error(t, err)

	// All errors should be present in the joined error
	require.ErrorIs(t, err, errCleanup1)
	require.ErrorIs(t, err, errCleanup2)
	require.ErrorIs(t, err, errCleanup3)
}

func TestCustomCloser_ConcurrentCloses(t *testing.T) {
	t.Parallel()

	closeCount := 0

	var mutex sync.Mutex

	closeFn := func() error {
		mutex.Lock()
		closeCount++
		mutex.Unlock()

		return nil
	}

	closer := CustomCloser(closeFn)
	require.NotNil(t, closer)

	const goroutines = 50

	var waitGroup sync.WaitGroup

	waitGroup.Add(goroutines)

	// Launch multiple goroutines trying to close simultaneously
	for range goroutines {
		go func() {
			defer waitGroup.Done()

			_ = closer.Close()
		}()
	}

	waitGroup.Wait()

	// Without CloseOnce, all goroutines will call the function
	assert.Equal(t, goroutines, closeCount, "All goroutines should call close function")
}

func TestCustomCloser_ConcurrentClosesWithCloseOnce(t *testing.T) {
	t.Parallel()

	closeCount := 0

	var mutex sync.Mutex

	closeFn := func() error {
		mutex.Lock()
		closeCount++
		mutex.Unlock()

		return nil
	}

	closer := CloseOnce(CustomCloser(closeFn))
	require.NotNil(t, closer)

	const goroutines = 100

	var waitGroup sync.WaitGroup

	waitGroup.Add(goroutines)

	// Launch multiple goroutines trying to close simultaneously
	for range goroutines {
		go func() {
			defer waitGroup.Done()

			_ = closer.Close()
		}()
	}

	waitGroup.Wait()

	// With CloseOnce, only one goroutine should actually call the function
	assert.Equal(t, 1, closeCount, "CloseOnce should ensure single close")
}

func TestCustomCloser_ComplexCleanup(t *testing.T) {
	t.Parallel()

	// Simulate a complex cleanup scenario with multiple resources
	var resources []string

	var mutex sync.Mutex

	cleanup1 := CustomCloser(func() error {
		mutex.Lock()
		resources = append(resources, "database disconnected")
		mutex.Unlock()

		return nil
	})

	cleanup2 := CustomCloser(func() error {
		mutex.Lock()
		resources = append(resources, "cache cleared")
		mutex.Unlock()

		return nil
	})

	cleanup3 := CustomCloser(func() error {
		mutex.Lock()
		resources = append(resources, "files closed")
		mutex.Unlock()

		return nil
	})

	collector := NewCloser(cleanup1, cleanup2, cleanup3)

	err := collector.Close()
	require.NoError(t, err)
	assert.Equal(t, []string{
		"database disconnected",
		"cache cleared",
		"files closed",
	}, resources)
}

func TestCustomCloser_WithDeferPattern(t *testing.T) {
	t.Parallel()

	cleanupCalled := false

	// Simulate typical defer pattern
	runWithCleanup := func() error {
		closer := CustomCloser(func() error {
			cleanupCalled = true

			return nil
		})
		defer closer.Close()

		// Do some work...
		return nil
	}

	err := runWithCleanup()
	require.NoError(t, err)
	assert.True(t, cleanupCalled, "Cleanup should be called via defer")
}

func TestCustomCloser_TypeAssertion(t *testing.T) {
	t.Parallel()

	closeFn := func() error { return nil }
	closer := CustomCloser(closeFn)

	require.NotNil(t, closer)

	// Verify that the returned value is of the correct type
	_, ok := closer.(*customCloser)
	assert.True(t, ok, "CustomCloser should return a *customCloser")
}

func TestCustomCloser_AllWrappersComposed(t *testing.T) {
	t.Parallel()

	closeCount := 0

	var mutex sync.Mutex

	closeFn := func() error {
		mutex.Lock()
		closeCount++
		mutex.Unlock()

		// Simulate a panic on second call (if not protected by CloseOnce)
		if closeCount > 1 {
			panic("should not be called more than once")
		}

		return nil
	}

	// Compose all wrappers: HandlePanic + CloseOnce + CustomCloser
	closer := HandlePanic(CloseOnce(CustomCloser(closeFn)))

	// Close from multiple goroutines
	var waitGroup sync.WaitGroup

	waitGroup.Add(10)

	for range 10 {
		go func() {
			defer waitGroup.Done()

			_ = closer.Close()
		}()
	}

	waitGroup.Wait()

	// Should only be called once due to CloseOnce
	mutex.Lock()
	count := closeCount
	mutex.Unlock()

	assert.Equal(t, 1, count, "Function should only be called once")
}

func TestCustomCloser_ErrorRetry(t *testing.T) {
	t.Parallel()

	closeCount := 0
	closeFn := func() error {
		closeCount++
		if closeCount < 3 {
			return errTransient
		}

		return nil
	}

	closer := CustomCloser(closeFn)
	require.NotNil(t, closer)

	// First two attempts fail
	err1 := closer.Close()
	require.Error(t, err1)
	assert.Equal(t, 1, closeCount)

	err2 := closer.Close()
	require.Error(t, err2)
	assert.Equal(t, 2, closeCount)

	// Third attempt succeeds
	err3 := closer.Close()
	require.NoError(t, err3)
	assert.Equal(t, 3, closeCount)
}

func TestCustomCloser_MixedWithStandardClosers(t *testing.T) {
	t.Parallel()

	mock := &mockCloser{}
	customCloseCalled := false

	collector := NewCloser()
	collector.Add(mock)
	collector.Add(CustomCloser(func() error {
		customCloseCalled = true

		return nil
	}))

	err := collector.Close()
	require.NoError(t, err)
	assert.Equal(t, 1, mock.getCloseCount(), "Standard closer should be closed")
	assert.True(t, customCloseCalled, "Custom closer should be closed")
}

// Tests for Closer

func TestNewCloser_Empty(t *testing.T) {
	t.Parallel()

	closer := NewCloser()
	require.NotNil(t, closer)
	assert.Empty(t, closer.closers)

	// Should not error on empty closer
	err := closer.Close()
	require.NoError(t, err)
}

func TestNewCloser_WithInitialClosers(t *testing.T) {
	t.Parallel()

	mock1 := &mockCloser{}
	mock2 := &mockCloser{}
	mock3 := &mockCloser{}

	closer := NewCloser(mock1, mock2, mock3)
	require.NotNil(t, closer)
	assert.Len(t, closer.closers, 3)

	err := closer.Close()
	require.NoError(t, err)
	assert.Equal(t, 1, mock1.getCloseCount())
	assert.Equal(t, 1, mock2.getCloseCount())
	assert.Equal(t, 1, mock3.getCloseCount())
}

func TestCloser_Add(t *testing.T) {
	t.Parallel()

	closer := NewCloser()
	mock1 := &mockCloser{}
	mock2 := &mockCloser{}

	closer.Add(mock1)
	assert.Len(t, closer.closers, 1)

	closer.Add(mock2)
	assert.Len(t, closer.closers, 2)

	err := closer.Close()
	require.NoError(t, err)
	assert.Equal(t, 1, mock1.getCloseCount())
	assert.Equal(t, 1, mock2.getCloseCount())
}

func TestCloser_AddNil(t *testing.T) {
	t.Parallel()

	closer := NewCloser()
	mock := &mockCloser{}

	closer.Add(nil)
	closer.Add(mock)
	closer.Add(nil)

	err := closer.Close()
	require.NoError(t, err)
	assert.Equal(t, 1, mock.getCloseCount(), "Only non-nil closer should be closed")
}

func TestCloser_CloseOrder(t *testing.T) {
	t.Parallel()

	var closedOrder []int

	var mu sync.Mutex

	makeOrderedCloser := func(id int) io.Closer {
		return &mockOrderedCloser{
			onClose: func() {
				mu.Lock()
				closedOrder = append(closedOrder, id)
				mu.Unlock()
			},
		}
	}

	closer := NewCloser()
	closer.Add(makeOrderedCloser(1))
	closer.Add(makeOrderedCloser(2))
	closer.Add(makeOrderedCloser(3))

	err := closer.Close()
	require.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, closedOrder, "Closers should be closed in the order they were added")
}

func TestCloser_SingleError(t *testing.T) {
	t.Parallel()

	mock1 := &mockCloser{}
	mock2 := &mockCloser{closeError: errCloseFailed}
	mock3 := &mockCloser{}

	closer := NewCloser(mock1, mock2, mock3)

	err := closer.Close()
	require.Error(t, err)
	require.ErrorIs(t, err, errCloseFailed)

	// All closers should have been attempted
	assert.Equal(t, 1, mock1.getCloseCount())
	assert.Equal(t, 1, mock2.getCloseCount())
	assert.Equal(t, 1, mock3.getCloseCount())
}

func TestCloser_MultipleErrors(t *testing.T) {
	t.Parallel()

	mock1 := &mockCloser{closeError: errMultiple1}
	mock2 := &mockCloser{}
	mock3 := &mockCloser{closeError: errMultiple2}
	mock4 := &mockCloser{closeError: errMultiple3}

	closer := NewCloser(mock1, mock2, mock3, mock4)

	err := closer.Close()
	require.Error(t, err)

	// Should contain all three errors using errors.Join
	require.ErrorIs(t, err, errMultiple1)
	require.ErrorIs(t, err, errMultiple2)
	require.ErrorIs(t, err, errMultiple3)

	// All closers should have been attempted
	assert.Equal(t, 1, mock1.getCloseCount())
	assert.Equal(t, 1, mock2.getCloseCount())
	assert.Equal(t, 1, mock3.getCloseCount())
	assert.Equal(t, 1, mock4.getCloseCount())
}

func TestCloser_MultipleClose(t *testing.T) {
	t.Parallel()

	mock := &mockCloser{}
	closer := NewCloser(mock)

	err1 := closer.Close()
	require.NoError(t, err1)
	assert.Equal(t, 1, mock.getCloseCount())

	// Second close should close again (Closer itself is not idempotent, unlike CloseOnce)
	err2 := closer.Close()
	require.NoError(t, err2)
	assert.Equal(t, 2, mock.getCloseCount(), "Closer allows multiple closes unlike CloseOnce")
}

func TestCloser_WithNilClosers(t *testing.T) {
	t.Parallel()

	mock := &mockCloser{}
	closer := NewCloser(nil, mock, nil, nil)

	err := closer.Close()
	require.NoError(t, err)
	assert.Equal(t, 1, mock.getCloseCount(), "Only non-nil closer should be closed")
}

func TestCloser_EmptyAfterConstruction(t *testing.T) {
	t.Parallel()

	closer := NewCloser(nil, nil, nil)
	assert.Len(t, closer.closers, 3, "Nil closers should still be in the list")

	err := closer.Close()
	assert.NoError(t, err, "Closing nil closers should not error")
}

// mockOrderedCloser is a test helper that calls a function when closed.
type mockOrderedCloser struct {
	onClose func()
}

func (m *mockOrderedCloser) Close() error {
	if m.onClose != nil {
		m.onClose()
	}

	return nil
}

// Tests for CloseOnce

func TestCloseOnce_NilCloser(t *testing.T) {
	t.Parallel()

	result := CloseOnce(nil)
	assert.Nil(t, result, "CloseOnce should return nil for nil input")
}

func TestCloseOnce_SingleClose(t *testing.T) {
	t.Parallel()

	mock := &mockCloser{}
	closer := CloseOnce(mock)

	require.NotNil(t, closer)

	err := closer.Close()
	require.NoError(t, err)
	assert.Equal(t, 1, mock.getCloseCount(), "Close should be called exactly once")
}

func TestCloseOnce_MultipleCloses(t *testing.T) {
	t.Parallel()

	mock := &mockCloser{}
	closer := CloseOnce(mock)

	// Close multiple times
	err1 := closer.Close()
	err2 := closer.Close()
	err3 := closer.Close()

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NoError(t, err3)
	assert.Equal(t, 1, mock.getCloseCount(), "Close should be called exactly once despite multiple calls")
}

func TestCloseOnce_ErrorPropagation(t *testing.T) {
	t.Parallel()

	mock := &mockCloser{closeError: errCloseFailed}
	closer := CloseOnce(mock)

	err := closer.Close()
	assert.Equal(t, errCloseFailed, err, "Error from underlying closer should be propagated")
	assert.Equal(t, 1, mock.getCloseCount())

	// Second close will attempt to close again since closed flag is not set on error.
	err2 := closer.Close()
	assert.Equal(t, errCloseFailed, err2, "Subsequent closes will retry when first close errored")
	assert.Equal(t, 2, mock.getCloseCount(), "Close will be attempted again after error")
}

func TestCloseOnce_Idempotent(t *testing.T) {
	t.Parallel()

	mock := &mockCloser{}
	closer := CloseOnce(mock)

	// Calling CloseOnce on an already wrapped closer should return the same wrapper
	closer2 := CloseOnce(closer)

	assert.Equal(t, closer, closer2, "CloseOnce should be idempotent")

	err := closer2.Close()
	require.NoError(t, err)
	assert.Equal(t, 1, mock.getCloseCount())
}

func TestCloseOnce_ConcurrentCloses(t *testing.T) {
	t.Parallel()

	mock := &mockCloser{}
	closer := CloseOnce(mock)

	const goroutines = 100

	var waitGroup sync.WaitGroup

	waitGroup.Add(goroutines)

	// Launch multiple goroutines trying to close simultaneously
	for range goroutines {
		go func() {
			defer waitGroup.Done()

			_ = closer.Close()
		}()
	}

	waitGroup.Wait()

	assert.Equal(t, 1, mock.getCloseCount(), "Close should be called exactly once even with concurrent calls")
}

func TestCloseOnce_ConcurrentClosesWithError(t *testing.T) {
	t.Parallel()

	mock := &mockCloser{closeError: errConcurrentClose}
	closer := CloseOnce(mock)

	const goroutines = 100

	var waitGroup sync.WaitGroup

	waitGroup.Add(goroutines)

	errorCount := 0

	var mutex sync.Mutex

	// Launch multiple goroutines trying to close simultaneously
	for range goroutines {
		go func() {
			defer waitGroup.Done()

			if err := closer.Close(); err != nil {
				mutex.Lock()
				errorCount++
				mutex.Unlock()
			}
		}()
	}

	waitGroup.Wait()

	// Since close always fails and never sets closed=true, all goroutines will call Close()
	assert.Equal(t, goroutines, mock.getCloseCount(), "Close will be attempted by all goroutines when it always errors")
	assert.Equal(t, goroutines, errorCount, "All goroutines should receive the error")
}

func TestCloseOnce_WithNopCloser(t *testing.T) {
	t.Parallel()

	// Test with io.NopCloser which is a real io.Closer implementation
	nopCloser := io.NopCloser(nil)
	closer := CloseOnce(nopCloser)

	// Should not panic or error
	err := closer.Close()
	require.NoError(t, err)

	// Second close should also be fine
	err = closer.Close()
	assert.NoError(t, err)
}

func TestCloseOnce_TypeAssertion(t *testing.T) {
	t.Parallel()

	mock := &mockCloser{}
	closer := CloseOnce(mock)

	// Verify that the returned value is of the correct type
	_, ok := closer.(*closeOnceImpl)
	assert.True(t, ok, "CloseOnce should return a *closeOnceImpl")

	// Verify that wrapping again returns the same instance
	closer2 := CloseOnce(closer)
	assert.Same(t, closer, closer2, "CloseOnce should detect and return existing wrapper")
}

func TestCloseOnce_WithNilInternalCloser(t *testing.T) {
	t.Parallel()

	// Test the edge case where closeOnceImpl has a nil closer
	closer := &closeOnceImpl{closer: nil}

	err := closer.Close()
	assert.NoError(t, err, "Closing with nil internal closer should not error")
}

// Tests for HandlePanic

func TestHandlePanic_NilCloser(t *testing.T) {
	t.Parallel()

	result := HandlePanic(nil)
	assert.Nil(t, result, "HandlePanic should return nil for nil input")
}

func TestHandlePanic_NormalClose(t *testing.T) {
	t.Parallel()

	mock := &mockCloser{}
	closer := HandlePanic(mock)

	require.NotNil(t, closer)

	err := closer.Close()
	require.NoError(t, err)
	assert.Equal(t, 1, mock.getCloseCount(), "Close should be called once")
}

func TestHandlePanic_ErrorWithoutPanic(t *testing.T) {
	t.Parallel()

	mock := &mockCloser{closeError: errCloseFailed}
	closer := HandlePanic(mock)

	err := closer.Close()
	require.ErrorIs(t, err, errCloseFailed, "Error from underlying closer should be propagated")
	assert.Equal(t, 1, mock.getCloseCount())
}

func TestHandlePanic_RecoverFromPanic(t *testing.T) {
	t.Parallel()

	panicCloser := &panicCloser{panicMessage: "something went wrong"}
	closer := HandlePanic(panicCloser)

	err := closer.Close()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "something went wrong", "Error should contain panic message")
	assert.Equal(t, 1, panicCloser.getCloseCount(), "Close should have been called")
}

func TestHandlePanic_RecoverFromPanicWithNonStringValue(t *testing.T) {
	t.Parallel()

	panicCloser := &panicCloser{panicMessage: 12345}
	closer := HandlePanic(panicCloser)

	err := closer.Close()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "12345", "Error should contain panic value")
	assert.Equal(t, 1, panicCloser.getCloseCount())
}

func TestHandlePanic_PanicOnly(t *testing.T) {
	t.Parallel()

	// Closer that panics (the panic happens before the error can be returned)
	panicCloser := &panicCloser{
		panicMessage: "panic occurred",
	}
	closer := HandlePanic(panicCloser)

	err := closer.Close()
	require.Error(t, err)

	// Should contain the panic message
	assert.Contains(t, err.Error(), "panic occurred", "Should contain the panic message")
	assert.Equal(t, 1, panicCloser.getCloseCount())
}

func TestHandlePanic_PanicInDefer(t *testing.T) {
	t.Parallel()

	// Closer that sets a return error but then panics in a defer
	// When a panic happens in a defer, it overrides the named return value
	// The outer function only sees the panic, not the error that was set
	panicInDeferCloser := &mockCloserWithDeferPanic{
		returnError:  errCloseFailed,
		panicMessage: "deferred panic",
	}
	closer := HandlePanic(panicInDeferCloser)

	err := closer.Close()
	require.Error(t, err)

	// Only the panic is captured, not the original error
	// This is because the panic in defer overrides the named return value
	assert.Contains(t, err.Error(), "deferred panic", "Should contain panic message")
}

func TestHandlePanic_CombinedWrappers(t *testing.T) {
	t.Parallel()

	// Test combining HandlePanic with CloseOnce
	mock := &mockCloser{}
	closer := HandlePanic(CloseOnce(mock))

	err1 := closer.Close()
	err2 := closer.Close()

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	// CloseOnce ensures only one actual close
	assert.Equal(t, 1, mock.getCloseCount(), "Should only close once due to CloseOnce wrapper")
}

func TestHandlePanic_Idempotent(t *testing.T) {
	t.Parallel()

	mock := &mockCloser{}
	closer := HandlePanic(mock)

	// Calling HandlePanic on an already wrapped closer should return the same wrapper
	closer2 := HandlePanic(closer)

	assert.Same(t, closer, closer2, "HandlePanic should be idempotent")

	err := closer2.Close()
	require.NoError(t, err)
	assert.Equal(t, 1, mock.getCloseCount())
}

func TestHandlePanic_MultipleCloses(t *testing.T) {
	t.Parallel()

	mock := &mockCloser{}
	closer := HandlePanic(mock)

	// HandlePanic itself is NOT idempotent for multiple closes (unlike CloseOnce)
	err1 := closer.Close()
	err2 := closer.Close()

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, 2, mock.getCloseCount(), "HandlePanic allows multiple closes")
}

func TestHandlePanic_ConcurrentCloses(t *testing.T) {
	t.Parallel()

	mock := &mockCloser{}
	closer := HandlePanic(mock)

	const goroutines = 50

	var waitGroup sync.WaitGroup

	waitGroup.Add(goroutines)

	// Launch multiple goroutines trying to close simultaneously
	for range goroutines {
		go func() {
			defer waitGroup.Done()

			_ = closer.Close()
		}()
	}

	waitGroup.Wait()

	// HandlePanic does not prevent concurrent closes (that's CloseOnce's job)
	assert.Equal(t, goroutines, mock.getCloseCount(), "All goroutines should close successfully")
}

func TestHandlePanic_WithNilInternalCloser(t *testing.T) {
	t.Parallel()

	// Test the edge case where panicHandlingImpl has a nil closer
	closer := &panicHandlingImpl{closer: nil}

	err := closer.Close()
	assert.NoError(t, err, "Closing with nil internal closer should not error")
}

func TestHandlePanic_StackTraceIncluded(t *testing.T) {
	t.Parallel()

	panicCloser := &panicCloser{panicMessage: "panic with stack trace"}
	closer := HandlePanic(panicCloser)

	err := closer.Close()
	require.Error(t, err)

	// The error should include stack trace information
	errString := err.Error()
	assert.Contains(t, errString, "panic with stack trace", "Should contain panic message")
	// Stack traces typically contain function names or file paths
	assert.Greater(t, len(errString), 100, "Error message should include stack trace details")
}

func TestHandlePanic_TypeAssertion(t *testing.T) {
	t.Parallel()

	mock := &mockCloser{}
	closer := HandlePanic(mock)

	// Verify that the returned value is of the correct type
	_, ok := closer.(*panicHandlingImpl)
	assert.True(t, ok, "HandlePanic should return a *panicHandlingImpl")

	// Verify that wrapping again returns the same instance
	closer2 := HandlePanic(closer)
	assert.Same(t, closer, closer2, "HandlePanic should detect and return existing wrapper")
}

// panicCloser is a test implementation that panics when closed.
type panicCloser struct {
	closeCount   int
	panicMessage any
	closeError   error
	mu           sync.Mutex
}

func (p *panicCloser) Close() error {
	p.mu.Lock()
	p.closeCount++
	err := p.closeError
	p.mu.Unlock()

	// Return error first if present, then panic
	if p.panicMessage != nil {
		panic(p.panicMessage)
	}

	return err
}

func (p *panicCloser) getCloseCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.closeCount
}

// mockCloserWithDeferPanic is a test implementation that returns an error and also panics in a defer.
// This tests the scenario where the function sets a return error and then a deferred function panics.
type mockCloserWithDeferPanic struct {
	returnError  error
	panicMessage any
}

func (m *mockCloserWithDeferPanic) Close() (err error) {
	// Set the return error first
	err = m.returnError

	// Defer a panic - this will run and panic even though err was set
	defer func() {
		if m.panicMessage != nil {
			panic(m.panicMessage)
		}
	}()

	return err
}

// Tests for ChannelCloser

func TestChannelCloser_NilChannel(t *testing.T) {
	t.Parallel()

	var ch chan int

	closer := ChannelCloser(ch)
	assert.Nil(t, closer, "ChannelCloser should return nil for nil channel")
}

func TestChannelCloser_BasicClose(t *testing.T) {
	t.Parallel()

	ch := make(chan int, 1)
	closer := ChannelCloser(ch)

	require.NotNil(t, closer)

	// Channel should be open - send a value to verify
	ch <- 42
	val := <-ch
	assert.Equal(t, 42, val, "Channel should be open and working")

	// Close the channel
	err := closer.Close()
	require.NoError(t, err)

	// Channel should be closed now
	_, ok := <-ch
	assert.False(t, ok, "Channel should be closed")
}

func TestChannelCloser_DoubleClose(t *testing.T) {
	t.Parallel()

	ch := make(chan string)
	closer := ChannelCloser(ch)

	// First close should succeed
	err1 := closer.Close()
	require.NoError(t, err1)

	// Second close should panic (closing an already-closed channel)
	assert.Panics(t, func() {
		_ = closer.Close()
	}, "Double close should panic")
}

func TestChannelCloser_DoubleCloseWithHandlePanic(t *testing.T) {
	t.Parallel()

	ch := make(chan bool)
	closer := HandlePanic(ChannelCloser(ch))

	// First close should succeed
	err1 := closer.Close()
	require.NoError(t, err1)

	// Second close should return an error (panic converted to error by HandlePanic)
	err2 := closer.Close()
	require.Error(t, err2, "Second close should return error from HandlePanic")
	assert.Contains(t, err2.Error(), "close of closed channel")
}

func TestChannelCloser_WithBufferedChannel(t *testing.T) {
	t.Parallel()

	testCh := make(chan int, 10)
	closer := ChannelCloser(testCh)

	// Send some values
	testCh <- 1
	testCh <- 2
	testCh <- 3

	// Close the channel
	err := closer.Close()
	require.NoError(t, err)

	// Should still be able to receive buffered values
	val1 := <-testCh
	val2 := <-testCh
	val3 := <-testCh

	assert.Equal(t, 1, val1)
	assert.Equal(t, 2, val2)
	assert.Equal(t, 3, val3)

	// Next receive should indicate channel is closed
	_, ok := <-testCh
	assert.False(t, ok, "Channel should be closed after draining buffer")
}

func TestChannelCloser_DifferentTypes(t *testing.T) {
	t.Parallel()

	// Test with different channel types
	t.Run("int channel", func(t *testing.T) {
		t.Parallel()

		ch := make(chan int)
		closer := ChannelCloser(ch)
		err := closer.Close()
		require.NoError(t, err)

		_, ok := <-ch
		assert.False(t, ok)
	})

	t.Run("string channel", func(t *testing.T) {
		t.Parallel()

		ch := make(chan string)
		closer := ChannelCloser(ch)
		err := closer.Close()
		require.NoError(t, err)

		_, ok := <-ch
		assert.False(t, ok)
	})

	t.Run("struct channel", func(t *testing.T) {
		t.Parallel()

		type testStruct struct {
			ID   int
			Name string
		}

		ch := make(chan testStruct)
		closer := ChannelCloser(ch)
		err := closer.Close()
		require.NoError(t, err)

		_, ok := <-ch
		assert.False(t, ok)
	})

	t.Run("pointer channel", func(t *testing.T) {
		t.Parallel()

		ch := make(chan *mockCloser)
		closer := ChannelCloser(ch)
		err := closer.Close()
		require.NoError(t, err)

		_, ok := <-ch
		assert.False(t, ok)
	})

	t.Run("empty struct channel", func(t *testing.T) {
		t.Parallel()

		ch := make(chan struct{})
		closer := ChannelCloser(ch)
		err := closer.Close()
		require.NoError(t, err)

		_, ok := <-ch
		assert.False(t, ok)
	})
}

func TestChannelCloser_WithCloseOnce(t *testing.T) {
	t.Parallel()

	ch := make(chan int)
	closer := CloseOnce(ChannelCloser(ch))

	// Close multiple times
	err1 := closer.Close()
	err2 := closer.Close()
	err3 := closer.Close()

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NoError(t, err3)

	// Channel should be closed
	_, ok := <-ch
	assert.False(t, ok, "Channel should be closed")
}

func TestChannelCloser_ConcurrentClosesRaw(t *testing.T) {
	t.Parallel()

	testCh := make(chan int)
	closer := ChannelCloser(testCh)

	const goroutines = 10

	var waitGroup sync.WaitGroup

	waitGroup.Add(goroutines)

	panicCount := 0

	var mutex sync.Mutex

	// Launch multiple goroutines trying to close simultaneously
	// Some will panic when trying to close an already-closed channel
	for range goroutines {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					mutex.Lock()
					panicCount++
					mutex.Unlock()
				}

				waitGroup.Done()
			}()

			_ = closer.Close()
		}()
	}

	waitGroup.Wait()

	// At least one goroutine should have panicked
	assert.Positive(t, panicCount, "Some goroutines should have panicked from double close")

	// Channel should be closed
	_, ok := <-testCh
	assert.False(t, ok, "Channel should be closed")
}

func TestChannelCloser_ConcurrentClosesWithCloseOnce(t *testing.T) {
	t.Parallel()

	testCh := make(chan string)
	closer := CloseOnce(ChannelCloser(testCh))

	const goroutines = 100

	var waitGroup sync.WaitGroup

	waitGroup.Add(goroutines)

	// Launch multiple goroutines trying to close simultaneously
	for range goroutines {
		go func() {
			defer waitGroup.Done()

			_ = closer.Close()
		}()
	}

	waitGroup.Wait()

	// Channel should be closed
	_, ok := <-testCh
	assert.False(t, ok, "Channel should be closed")
}

func TestChannelCloser_WithCloserCollector(t *testing.T) {
	t.Parallel()

	ch1 := make(chan int)
	ch2 := make(chan string)
	ch3 := make(chan bool)

	collector := NewCloser()
	collector.Add(ChannelCloser(ch1))
	collector.Add(ChannelCloser(ch2))
	collector.Add(ChannelCloser(ch3))

	err := collector.Close()
	require.NoError(t, err)

	// All channels should be closed
	_, ok1 := <-ch1
	_, ok2 := <-ch2
	_, ok3 := <-ch3

	assert.False(t, ok1, "Channel 1 should be closed")
	assert.False(t, ok2, "Channel 2 should be closed")
	assert.False(t, ok3, "Channel 3 should be closed")
}

func TestChannelCloser_WithMixedClosers(t *testing.T) {
	t.Parallel()

	ch := make(chan int)
	mock := &mockCloser{}

	collector := NewCloser()
	collector.Add(ChannelCloser(ch))
	collector.Add(mock)

	err := collector.Close()
	require.NoError(t, err)

	// Channel should be closed
	_, ok := <-ch
	assert.False(t, ok, "Channel should be closed")

	// Mock closer should have been closed
	assert.Equal(t, 1, mock.getCloseCount(), "Mock closer should be closed once")
}

func TestChannelCloser_WithHandlePanic(t *testing.T) {
	t.Parallel()

	ch := make(chan int)
	closer := HandlePanic(ChannelCloser(ch))

	err := closer.Close()
	require.NoError(t, err)

	// Channel should be closed
	_, ok := <-ch
	assert.False(t, ok, "Channel should be closed")

	// Second close should return an error (panic converted to error by HandlePanic)
	err2 := closer.Close()
	require.Error(t, err2, "Second close should return error from HandlePanic")
	assert.Contains(t, err2.Error(), "close of closed channel")
}

func TestChannelCloser_AllWrappersComposed(t *testing.T) {
	t.Parallel()

	testCh := make(chan string, 5)

	// Compose all wrappers: HandlePanic + CloseOnce + ChannelCloser
	closer := HandlePanic(CloseOnce(ChannelCloser(testCh)))

	// Send some data
	testCh <- "hello"
	testCh <- "world"

	// Close from multiple goroutines
	var waitGroup sync.WaitGroup

	waitGroup.Add(10)

	for range 10 {
		go func() {
			defer waitGroup.Done()

			_ = closer.Close()
		}()
	}

	waitGroup.Wait()

	// Should be able to receive buffered values
	val1 := <-testCh
	val2 := <-testCh

	assert.Equal(t, "hello", val1)
	assert.Equal(t, "world", val2)

	// Channel should be closed
	_, ok := <-testCh
	assert.False(t, ok, "Channel should be closed")
}

func TestChannelCloser_TypeAssertion(t *testing.T) {
	t.Parallel()

	ch := make(chan int)
	closer := ChannelCloser(ch)

	// Verify that the returned value is of the correct type
	_, ok := closer.(*channelCloserImpl[int])
	assert.True(t, ok, "ChannelCloser should return a *channelCloserImpl")
}

func TestChannelCloser_WithNilInternalChannel(t *testing.T) {
	t.Parallel()

	// Test the edge case where channelCloserImpl has a nil channel
	closer := &channelCloserImpl[int]{ch: nil}

	err := closer.Close()
	assert.NoError(t, err, "Closing with nil internal channel should not error")
}

func TestChannelCloser_ClosedChannelReadBehavior(t *testing.T) {
	t.Parallel()

	testCh := make(chan int, 2)
	testCh <- 10
	testCh <- 20

	closer := ChannelCloser(testCh)
	err := closer.Close()
	require.NoError(t, err)

	// Reading from closed channel with buffer should return buffered values
	val1 := <-testCh
	val2 := <-testCh

	assert.Equal(t, 10, val1)
	assert.Equal(t, 20, val2)

	// Reading from closed empty channel should return zero value and false
	val3, ok := <-testCh
	assert.Equal(t, 0, val3, "Should get zero value from closed channel")
	assert.False(t, ok, "ok should be false for closed channel")
}

func TestChannelCloser_SendOnlyChannel(t *testing.T) {
	t.Parallel()

	// Test with send-only channel type
	testCh := make(chan int, 1)

	var sendCh chan<- int = testCh

	closer := ChannelCloser(sendCh)

	// Send a value
	sendCh <- 42

	// Close using the closer
	err := closer.Close()
	require.NoError(t, err)

	// Verify channel is closed by reading from original channel
	val, ok := <-testCh
	assert.Equal(t, 42, val)
	assert.True(t, ok)

	_, ok = <-testCh
	assert.False(t, ok, "Channel should be closed")
}

func TestChannelCloser_SendOnlyInFunction(t *testing.T) {
	t.Parallel()

	// Simulate a pattern where you pass send-only channel to a worker
	worker := func(ch chan<- string, closer io.Closer) {
		defer closer.Close()
		ch <- "hello"
		ch <- "world"
	}

	ch := make(chan string, 2)
	closer := ChannelCloser(ch)

	// Pass send-only channel and closer to worker
	worker(ch, closer)

	// Read values
	val1 := <-ch
	val2 := <-ch

	assert.Equal(t, "hello", val1)
	assert.Equal(t, "world", val2)

	// Verify closed
	_, ok := <-ch
	assert.False(t, ok, "Channel should be closed")
}

// Tests for CancelableCloser

func TestCancelableCloser_NilCloser(t *testing.T) {
	t.Parallel()

	closer, cancel := CancelableCloser(nil)
	assert.Nil(t, closer, "CancelableCloser should return nil closer for nil input")
	assert.NotNil(t, cancel, "Cancel function should not be nil even for nil input")

	// Calling cancel should not panic
	cancel()
}

func TestCancelableCloser_NormalClose(t *testing.T) {
	t.Parallel()

	mock := &mockCloser{}
	closer, _ := CancelableCloser(mock)

	require.NotNil(t, closer)

	err := closer.Close()
	require.NoError(t, err)
	assert.Equal(t, 1, mock.getCloseCount(), "Close should be called once")
}

func TestCancelableCloser_CloseAfterCancel(t *testing.T) {
	t.Parallel()

	mock := &mockCloser{}
	closer, cancel := CancelableCloser(mock)

	require.NotNil(t, closer)

	// Cancel first, then close
	cancel()

	err := closer.Close()
	require.NoError(t, err)
	assert.Equal(t, 0, mock.getCloseCount(), "Close should not be called after cancel")
}

func TestCancelableCloser_CancelAfterClose(t *testing.T) {
	t.Parallel()

	mock := &mockCloser{}
	closer, cancel := CancelableCloser(mock)

	// Close first
	err := closer.Close()
	require.NoError(t, err)
	assert.Equal(t, 1, mock.getCloseCount())

	// Cancel after close should not affect anything
	cancel()

	// Second close should be a no-op due to cancel
	err = closer.Close()
	require.NoError(t, err)
	assert.Equal(t, 1, mock.getCloseCount(), "Should not close again after cancel")
}

func TestCancelableCloser_MultipleCloses(t *testing.T) {
	t.Parallel()

	mock := &mockCloser{}
	closer, _ := CancelableCloser(mock)

	// Close multiple times without cancel
	err1 := closer.Close()
	err2 := closer.Close()
	err3 := closer.Close()

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NoError(t, err3)
	assert.Equal(t, 3, mock.getCloseCount(), "CancelableCloser allows multiple closes without cancel")
}

func TestCancelableCloser_MultipleClosesThenCancel(t *testing.T) {
	t.Parallel()

	mock := &mockCloser{}
	closer, cancel := CancelableCloser(mock)

	// Close multiple times
	err1 := closer.Close()
	err2 := closer.Close()

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, 2, mock.getCloseCount())

	// Cancel, then try to close again
	cancel()

	err3 := closer.Close()
	require.NoError(t, err3)
	assert.Equal(t, 2, mock.getCloseCount(), "Should not close again after cancel")
}

func TestCancelableCloser_MultipleCancels(t *testing.T) {
	t.Parallel()

	mock := &mockCloser{}
	closer, cancel := CancelableCloser(mock)

	// Cancel multiple times
	cancel()
	cancel()
	cancel()

	// Close should be a no-op
	err := closer.Close()
	require.NoError(t, err)
	assert.Equal(t, 0, mock.getCloseCount(), "Should not close after multiple cancels")
}

func TestCancelableCloser_ErrorPropagation(t *testing.T) {
	t.Parallel()

	mock := &mockCloser{closeError: errCloseFailed}
	closer, _ := CancelableCloser(mock)

	err := closer.Close()
	assert.Equal(t, errCloseFailed, err, "Error from underlying closer should be propagated")
	assert.Equal(t, 1, mock.getCloseCount())
}

func TestCancelableCloser_ErrorIgnoredAfterCancel(t *testing.T) {
	t.Parallel()

	mock := &mockCloser{closeError: errCloseFailed}
	closer, cancel := CancelableCloser(mock)

	// Cancel first
	cancel()

	// Close should not call the underlying closer, so no error
	err := closer.Close()
	require.NoError(t, err)
	assert.Equal(t, 0, mock.getCloseCount(), "Should not attempt close after cancel")
}

func TestCancelableCloser_Idempotent(t *testing.T) {
	t.Parallel()

	mock := &mockCloser{}
	closer1, cancel1 := CancelableCloser(mock)

	// Calling CancelableCloser on an already wrapped closer should return the same wrapper
	closer2, cancel2 := CancelableCloser(closer1)

	assert.Same(t, closer1, closer2, "CancelableCloser should be idempotent")

	// Both cancel functions should work
	cancel1()

	err := closer2.Close()
	require.NoError(t, err)
	assert.Equal(t, 0, mock.getCloseCount(), "Should not close after cancel via first cancel function")

	// Try the second cancel function (should be a no-op since already canceled)
	cancel2()

	err = closer2.Close()
	require.NoError(t, err)
	assert.Equal(t, 0, mock.getCloseCount())
}

func TestCancelableCloser_ConcurrentCloses(t *testing.T) {
	t.Parallel()

	mock := &mockCloser{}
	closer, _ := CancelableCloser(mock)

	const goroutines = 50

	var waitGroup sync.WaitGroup

	waitGroup.Add(goroutines)

	// Launch multiple goroutines trying to close simultaneously
	for range goroutines {
		go func() {
			defer waitGroup.Done()

			_ = closer.Close()
		}()
	}

	waitGroup.Wait()

	assert.Equal(t, goroutines, mock.getCloseCount(), "All goroutines should close successfully")
}

func TestCancelableCloser_ConcurrentClosesAndCancel(t *testing.T) {
	t.Parallel()

	mock := &mockCloser{}
	closer, cancel := CancelableCloser(mock)

	const goroutines = 100

	var waitGroup sync.WaitGroup

	waitGroup.Add(goroutines + 1)

	// Launch multiple goroutines trying to close simultaneously
	for range goroutines {
		go func() {
			defer waitGroup.Done()

			_ = closer.Close()
		}()
	}

	// Also try to cancel concurrently
	go func() {
		defer waitGroup.Done()

		cancel()
	}()

	waitGroup.Wait()

	// Some closes happened before cancel, some after
	// The count should be less than goroutines due to cancel
	closeCount := mock.getCloseCount()
	assert.GreaterOrEqual(t, goroutines, closeCount, "Close count should not exceed number of goroutines")
}

func TestCancelableCloser_WithCustomCloser(t *testing.T) {
	t.Parallel()

	closeCalled := false
	customCloser := CustomCloser(func() error {
		closeCalled = true

		return nil
	})

	closer, cancel := CancelableCloser(customCloser)

	// Close should work
	err := closer.Close()
	require.NoError(t, err)
	assert.True(t, closeCalled, "Custom close function should be called")

	// Reset flag
	closeCalled = false

	// Cancel then close
	cancel()

	err = closer.Close()
	require.NoError(t, err)
	assert.False(t, closeCalled, "Custom close function should not be called after cancel")
}

func TestCancelableCloser_TransactionPattern(t *testing.T) {
	t.Parallel()

	// Test 1: Successful transaction (cancel = commit, no rollback)
	t.Run("success case", func(t *testing.T) {
		t.Parallel()

		// Use local variables to avoid races between subtests
		rolledBack := false
		committed := false

		rollbackFn := func() error { //nolint:unparam
			rolledBack = true

			return nil
		}

		commitFn := func() error { //nolint:unparam
			committed = true

			return nil
		}

		closer, cancel := CancelableCloser(CustomCloser(rollbackFn))
		defer closer.Close() // Will rollback unless canceled

		// Do work...
		err := commitFn()
		require.NoError(t, err)

		// Success, cancel the rollback
		cancel()

		assert.True(t, committed, "Should have committed")
		assert.False(t, rolledBack, "Should not have rolled back")
	})

	// Test 2: Failed transaction (no cancel = rollback)
	t.Run("failure case", func(t *testing.T) {
		t.Parallel()

		// Use local variables to avoid races between subtests
		rolledBack := false
		committed := false

		rollbackFn := func() error { //nolint:unparam
			rolledBack = true

			return nil
		}

		closer, _ := CancelableCloser(CustomCloser(rollbackFn))

		// Simulate failure - don't call cancel
		// Close manually to trigger rollback before assertions
		err := closer.Close()
		require.NoError(t, err)

		assert.False(t, committed, "Should not have committed")
		assert.True(t, rolledBack, "Should have rolled back")
	})
}

func TestCancelableCloser_TemporaryFilePattern(t *testing.T) {
	t.Parallel()

	// Test 1: Keep file (cancel = keep)
	t.Run("keep file", func(t *testing.T) {
		t.Parallel()

		// Use local variables to avoid races between subtests
		fileDeleted := false

		deleteFn := func() error { //nolint:unparam
			fileDeleted = true

			return nil
		}

		closer, cancel := CancelableCloser(CustomCloser(deleteFn))
		defer closer.Close() // Will delete unless canceled

		// Success, keep the file
		cancel()

		assert.False(t, fileDeleted, "File should not be deleted when canceled")
	})

	// Test 2: Delete file (no cancel = delete)
	t.Run("delete file", func(t *testing.T) {
		t.Parallel()

		// Use local variables to avoid races between subtests
		fileDeleted := false

		deleteFn := func() error { //nolint:unparam
			fileDeleted = true

			return nil
		}

		closer, _ := CancelableCloser(CustomCloser(deleteFn))

		// Don't cancel, close manually to trigger deletion before assertion
		err := closer.Close()
		require.NoError(t, err)

		assert.True(t, fileDeleted, "File should be deleted when not canceled")
	})
}

func TestCancelableCloser_WithCloserCollector(t *testing.T) {
	t.Parallel()

	mock1 := &mockCloser{}
	mock2 := &mockCloser{}
	mock3 := &mockCloser{}

	closer1, cancel1 := CancelableCloser(mock1)
	closer2, cancel2 := CancelableCloser(mock2)
	closer3, _ := CancelableCloser(mock3)

	collector := NewCloser(closer1, closer2, closer3)

	// Cancel first two
	cancel1()
	cancel2()

	// Close all
	err := collector.Close()
	require.NoError(t, err)

	// Only the third should have closed
	assert.Equal(t, 0, mock1.getCloseCount(), "First should not close due to cancel")
	assert.Equal(t, 0, mock2.getCloseCount(), "Second should not close due to cancel")
	assert.Equal(t, 1, mock3.getCloseCount(), "Third should close normally")
}

func TestCancelableCloser_WithAllWrappers(t *testing.T) {
	t.Parallel()

	closeCount := 0

	var mutex sync.Mutex

	closeFn := func() error {
		mutex.Lock()
		closeCount++
		mutex.Unlock()

		return nil
	}

	// Compose: HandlePanic + CloseOnce + CancelableCloser + CustomCloser
	innerCloser := CustomCloser(closeFn)
	cancelable, cancel := CancelableCloser(innerCloser)
	withOnce := CloseOnce(cancelable)
	withPanic := HandlePanic(withOnce)

	// Close from multiple goroutines
	var waitGroup sync.WaitGroup

	waitGroup.Add(10)

	for range 10 {
		go func() {
			defer waitGroup.Done()

			_ = withPanic.Close()
		}()
	}

	waitGroup.Wait()

	// Should only be called once due to CloseOnce
	mutex.Lock()
	count := closeCount
	mutex.Unlock()

	assert.Equal(t, 1, count, "Function should only be called once")

	// Now cancel and try again
	closeCount = 0

	cancel()

	err := withPanic.Close()
	require.NoError(t, err)

	mutex.Lock()
	count = closeCount
	mutex.Unlock()

	assert.Equal(t, 0, count, "Function should not be called after cancel")
}

func TestCancelableCloser_TypeAssertion(t *testing.T) {
	t.Parallel()

	mock := &mockCloser{}
	closer, _ := CancelableCloser(mock)

	// Verify that the returned value is of the correct type
	_, ok := closer.(*cancelableCloser)
	assert.True(t, ok, "CancelableCloser should return a *cancelableCloser")
}

func TestCancelableCloser_WithNilInternalCloser(t *testing.T) {
	t.Parallel()

	// Test the edge case where cancelableCloser has a nil closer
	closer := &cancelableCloser{
		shouldClose: nil, // Will panic if not handled correctly
		closer:      nil,
	}

	// Should handle nil closer gracefully
	err := closer.Close()
	assert.NoError(t, err, "Closing with nil internal closer should not error")
}

func TestCancelableCloser_RaceConditionCancelAndClose(t *testing.T) {
	t.Parallel()

	mock := &mockCloser{}
	closer, cancel := CancelableCloser(mock)

	const goroutines = 100

	var waitGroup sync.WaitGroup

	waitGroup.Add(goroutines)

	// Half try to close, half try to cancel
	for i := range goroutines {
		if i%2 == 0 {
			go func() {
				defer waitGroup.Done()

				_ = closer.Close()
			}()
		} else {
			go func() {
				defer waitGroup.Done()

				cancel()
			}()
		}
	}

	waitGroup.Wait()

	// Test should complete without data races
	// The close count will vary depending on timing
	closeCount := mock.getCloseCount()
	assert.GreaterOrEqual(t, goroutines, closeCount, "Close count should be reasonable")
}
