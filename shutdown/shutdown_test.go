package shutdown

import (
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBeforeShutdown(t *testing.T) {
	// Reset global state
	hooks = nil
	channel = nil

	var called atomic.Int32

	BeforeShutdown(func() {
		called.Add(1)
	})

	BeforeShutdown(func() {
		called.Add(10)
	})

	// Verify hooks were registered
	mut.Lock()
	assert.Len(t, hooks, 2)
	mut.Unlock()

	// Trigger cleanup manually
	cleanup()

	// Verify all hooks were called
	assert.Equal(t, int32(11), called.Load())

	// Verify hooks were cleared
	mut.Lock()
	assert.Nil(t, hooks)
	mut.Unlock()
}

func TestSetupHandler(t *testing.T) {
	// Reset global state
	hooks = nil
	channel = nil

	ctx := SetupHandler()

	// Verify context is not canceled initially
	select {
	case <-ctx.Done():
		t.Fatal("context should not be canceled initially")
	default:
	}

	// Verify signal channel was created
	require.NotNil(t, channel)

	var hookCalled atomic.Bool
	BeforeShutdown(func() {
		hookCalled.Store(true)
	})

	// Send signal
	channel <- syscall.SIGTERM

	// Wait for context to be canceled
	select {
	case <-ctx.Done():
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("context was not canceled after signal")
	}

	// Verify hook was called
	assert.True(t, hookCalled.Load())

	// Verify channel was cleaned up
	assert.Nil(t, channel)
}

func TestSetupHandlerSIGINT(t *testing.T) {
	// Reset global state
	hooks = nil
	channel = nil

	ctx := SetupHandler()

	var hookCalled atomic.Bool
	BeforeShutdown(func() {
		hookCalled.Store(true)
	})

	// Send SIGINT instead of SIGTERM
	channel <- syscall.SIGINT

	// Wait for context to be canceled
	select {
	case <-ctx.Done():
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("context was not canceled after SIGINT")
	}

	// Verify hook was called
	assert.True(t, hookCalled.Load())
}

func TestShutdown(t *testing.T) {
	// Reset global state
	hooks = nil
	channel = nil

	ctx := SetupHandler()

	var hookCalled atomic.Bool
	BeforeShutdown(func() {
		hookCalled.Store(true)
	})

	// Trigger shutdown programmatically
	Shutdown()

	// Wait for context to be canceled
	select {
	case <-ctx.Done():
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("context was not canceled after Shutdown()")
	}

	// Verify hook was called
	assert.True(t, hookCalled.Load())

	// Verify channel was cleaned up
	assert.Nil(t, channel)
}

func TestShutdownWithoutSetup(t *testing.T) {
	// Reset global state
	hooks = nil
	channel = nil

	// Calling Shutdown without SetupHandler should not panic
	assert.NotPanics(t, func() {
		Shutdown()
	})
}

func TestMultipleHooksExecutionOrder(t *testing.T) {
	// Reset global state
	hooks = nil
	channel = nil

	var mu atomic.Value

	BeforeShutdown(func() {
		current := []int{1}
		if existing := mu.Load(); existing != nil {
			current = append(existing.([]int), 1)
		}
		mu.Store(current)
	})

	BeforeShutdown(func() {
		current := []int{2}
		if existing := mu.Load(); existing != nil {
			current = append(existing.([]int), 2)
		}
		mu.Store(current)
	})

	BeforeShutdown(func() {
		current := []int{3}
		if existing := mu.Load(); existing != nil {
			current = append(existing.([]int), 3)
		}
		mu.Store(current)
	})

	cleanup()

	result := mu.Load().([]int)
	assert.Equal(t, []int{1, 2, 3}, result)
}

func TestContextCanceledAfterHooks(t *testing.T) {
	// Reset global state
	hooks = nil
	channel = nil

	ctx := SetupHandler()

	var contextWasCanceled atomic.Bool
	BeforeShutdown(func() {
		// Check if context is still active during hook execution
		select {
		case <-ctx.Done():
			contextWasCanceled.Store(true)
		default:
			contextWasCanceled.Store(false)
		}
	})

	Shutdown()

	// Wait for shutdown to complete
	<-ctx.Done()

	// Context should NOT have been canceled during hook execution
	assert.False(t, contextWasCanceled.Load(), "context should be canceled after hooks, not during")
}

func TestConcurrentBeforeShutdown(t *testing.T) {
	// Reset global state
	hooks = nil
	channel = nil

	const numGoroutines = 100
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			BeforeShutdown(func() {})
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	mut.Lock()
	assert.Len(t, hooks, numGoroutines)
	mut.Unlock()
}
