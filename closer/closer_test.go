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
