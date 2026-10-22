package shutdown

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:paralleltest // Test manipulates global state and cannot run in parallel
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

//nolint:paralleltest // Test manipulates global state and cannot run in parallel
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

//nolint:paralleltest // Test manipulates global state and cannot run in parallel
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

//nolint:paralleltest // Test manipulates global state and cannot run in parallel
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

//nolint:paralleltest // Test manipulates global state and cannot run in parallel
func TestShutdownWithoutSetup(t *testing.T) {
	// Reset global state
	hooks = nil
	channel = nil

	// Calling Shutdown without SetupHandler should not panic
	assert.NotPanics(t, func() {
		Shutdown()
	})
}

//nolint:paralleltest // Test manipulates global state and cannot run in parallel
func TestMultipleHooksExecutionOrder(t *testing.T) {
	// Reset global state
	hooks = nil
	channel = nil

	var executionOrder atomic.Value

	BeforeShutdown(func() {
		current := []int{1}
		existing := executionOrder.Load()

		if existing != nil {
			existingSlice, ok := existing.([]int)
			if ok {
				existingSlice = append(existingSlice, 1)
				executionOrder.Store(existingSlice)

				return
			}
		}

		executionOrder.Store(current)
	})

	BeforeShutdown(func() {
		current := []int{2}
		existing := executionOrder.Load()

		if existing != nil {
			existingSlice, ok := existing.([]int)
			if ok {
				existingSlice = append(existingSlice, 2)
				executionOrder.Store(existingSlice)

				return
			}
		}

		executionOrder.Store(current)
	})

	BeforeShutdown(func() {
		current := []int{3}
		existing := executionOrder.Load()

		if existing != nil {
			existingSlice, ok := existing.([]int)
			if ok {
				existingSlice = append(existingSlice, 3)
				executionOrder.Store(existingSlice)

				return
			}
		}

		executionOrder.Store(current)
	})

	cleanup()

	result, ok := executionOrder.Load().([]int)
	require.True(t, ok)
	assert.Equal(t, []int{1, 2, 3}, result)
}

//nolint:paralleltest // Test manipulates global state and cannot run in parallel
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

//nolint:paralleltest // Test manipulates global state and cannot run in parallel
func TestConcurrentBeforeShutdown(t *testing.T) {
	// Reset global state
	hooks = nil
	channel = nil

	const numGoroutines = 100

	done := make(chan bool, numGoroutines)

	for i := range numGoroutines {
		go func() {
			_ = i

			BeforeShutdown(func() {})

			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for range numGoroutines {
		<-done
	}

	mut.Lock()
	assert.Len(t, hooks, numGoroutines)
	mut.Unlock()
}

// signalChildEnv makes the test binary act as the child process of TestRepeatedSignalsDuringShutdown
// and selects which handler it sets up.
const signalChildEnv = "SHUTDOWN_SIGNAL_TEST_CHILD"

// TestRepeatedSignalsDuringShutdown sends real signals, which only the os/signal loop delivers, so it
// runs each scenario in a child process: a signal that crashed the child would otherwise take the
// whole test binary down with it.
//
//nolint:paralleltest // Child processes run serially to keep their signals and output apart
func TestRepeatedSignalsDuringShutdown(t *testing.T) {
	if mode := os.Getenv(signalChildEnv); mode != "" {
		repeatSignalsInChild(t, mode)

		return
	}

	for _, mode := range []string{"handler", "drainable"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
			defer cancel()

			//nolint:gosec // Re-runs this test binary, filtered to this test
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestRepeatedSignalsDuringShutdown$", "-test.v")

			cmd.Env = append(os.Environ(), signalChildEnv+"="+mode)

			out, err := cmd.CombinedOutput()
			require.NoError(t, err, "child process failed:\n%s", out)

			assert.Contains(t, string(out), "shutdown completed")
			assert.Equal(t, 2, strings.Count(string(out), "while already shutting down; ignoring"),
				"both repeated signals should be received and ignored:\n%s", out)
		})
	}
}

// repeatSignalsInChild sets up the handler named by mode, signals its own process with SIGTERM,
// then sends SIGTERM and SIGINT again mid-shutdown, waiting for each to be logged as ignored.
func repeatSignalsInChild(t *testing.T, mode string) {
	t.Helper()

	ignored := make(chan struct{}, 8) //nolint:mnd
	slog.SetDefault(slog.New(slog.NewTextHandler(&logWatcher{
		out:   os.Stdout,
		match: []byte("while already shutting down; ignoring"),
		seen:  ignored,
	}, nil)))

	var (
		shuttingDown <-chan struct{}
		finish       func()
	)

	switch mode {
	case "handler":
		ctx := SetupHandler()
		shuttingDown = ctx.Done()
		finish = func() {}
	case "drainable":
		ctxs := SetupDrainable()
		shuttingDown = ctxs.Intake.Done()
		finish = func() {
			require.NoError(t, ctxs.Lifetime.Err(), "repeated signals must not end the drain")
			ctxs.DrainComplete()
		}
	default:
		t.Fatalf("unknown mode %q", mode)
	}

	self, err := os.FindProcess(os.Getpid())
	require.NoError(t, err)

	require.NoError(t, self.Signal(syscall.SIGTERM))
	waitFor(t, shuttingDown, "shutdown to start")

	for _, sig := range []os.Signal{syscall.SIGTERM, syscall.SIGINT} {
		require.NoError(t, self.Signal(sig))
		waitFor(t, ignored, "repeated "+sig.String()+" to be ignored")
	}

	finish()
	fmt.Println("shutdown completed") //nolint:forbidigo // Read by the parent test
}

func waitFor[T any](t *testing.T, ch <-chan T, what string) {
	t.Helper()

	select {
	case <-ch:
	case <-time.After(5 * time.Second): //nolint:mnd
		t.Fatalf("timed out waiting for %s", what)
	}
}

// logWatcher passes log output through to out, and reports each write containing match on seen.
type logWatcher struct {
	out   io.Writer
	match []byte
	seen  chan<- struct{}
}

func (w *logWatcher) Write(p []byte) (int, error) {
	if bytes.Contains(p, w.match) {
		w.seen <- struct{}{}
	}

	return w.out.Write(p)
}
