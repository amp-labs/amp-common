package actor

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/amp-labs/amp-common/try"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type empty struct{}

func TestActorPanic(t *testing.T) {
	t.Parallel()

	// Create a new actor with a panic handler
	act := New[empty, empty](func(ref *Ref[empty, empty]) Processor[empty, empty] {
		return NewProcessor[empty, empty](func(m Message[empty, empty]) {
			panic("test panic")
		})
	})

	ref := act.Run(t.Context(), "test", 1)

	_, err := ref.RequestCtx(t.Context(), empty{})

	require.Error(t, err)
	require.ErrorContains(t, err, "test panic")
}

func TestActorBasicRequestResponse(t *testing.T) {
	t.Parallel()

	// Create an actor that doubles integers
	act := New[int, int](func(ref *Ref[int, int]) Processor[int, int] {
		return SimpleProcessor(func(req int) (int, error) {
			return req * 2, nil
		})
	})

	ref := act.Run(t.Context(), "doubler", 10)
	defer ref.Stop()

	// Test request/response
	result, err := ref.RequestCtx(t.Context(), 5)
	require.NoError(t, err)
	assert.Equal(t, 10, result)

	result, err = ref.RequestCtx(t.Context(), 100)
	require.NoError(t, err)
	assert.Equal(t, 200, result)
}

func TestActorSendFireAndForget(t *testing.T) {
	t.Parallel()

	var counter atomic.Int32

	// Create an actor that increments a counter
	act := New[int, empty](func(ref *Ref[int, empty]) Processor[int, empty] {
		return SimpleProcessor(func(req int) (empty, error) {
			counter.Add(int32(req)) //nolint:gosec

			return empty{}, nil
		})
	})

	ref := act.Run(t.Context(), "counter", 10)
	defer ref.Stop()

	// Send messages without waiting for responses
	ref.SendCtx(t.Context(), 1)
	ref.SendCtx(t.Context(), 2)
	ref.SendCtx(t.Context(), 3)

	// Wait a bit for processing
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, int32(6), counter.Load())
}

func TestActorContextCancellation(t *testing.T) {
	t.Parallel()

	// Create an actor that sleeps
	act := New[int, int](func(ref *Ref[int, int]) Processor[int, int] {
		return SimpleProcessor(func(req int) (int, error) {
			time.Sleep(time.Duration(req) * time.Millisecond)

			return req, nil
		})
	})

	ctx, cancel := context.WithCancel(t.Context())

	ref := act.Run(ctx, "sleeper", 10)
	defer ref.Stop()

	// Start a request that will take a while
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	// This should be canceled
	reqCtx, reqCancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer reqCancel()

	_, err := ref.RequestCtx(reqCtx, 1000)

	// Should get a context error or dead actor error
	require.Error(t, err)
}

func TestActorDeadActorError(t *testing.T) {
	t.Parallel()

	act := New[int, int](func(ref *Ref[int, int]) Processor[int, int] {
		return SimpleProcessor(func(req int) (int, error) {
			return req, nil
		})
	})

	ref := act.Run(t.Context(), "temporary", 1)

	// Stop the actor
	ref.Stop()
	ref.Wait()

	// Try to send to stopped actor
	_, err := ref.Request(42)
	require.ErrorIs(t, err, ErrDeadActor)

	// Check alive status
	assert.False(t, ref.Alive())
}

var errEmptyString = errors.New("empty string")

func TestActorSimpleProcessor(t *testing.T) {
	t.Parallel()

	// Test SimpleProcessor with errors
	act := New[string, int](func(ref *Ref[string, int]) Processor[string, int] {
		return SimpleProcessor(func(req string) (int, error) {
			if req == "" {
				return 0, errEmptyString
			}

			return len(req), nil
		})
	})

	ref := act.Run(t.Context(), "strlen", 5)
	defer ref.Stop()

	// Successful case
	result, err := ref.RequestCtx(t.Context(), "hello")
	require.NoError(t, err)
	assert.Equal(t, 5, result)

	// Error case
	_, err = ref.RequestCtx(t.Context(), "")
	require.Error(t, err)
	require.ErrorContains(t, err, "empty string")
}

func TestActorBufferedVsUnbuffered(t *testing.T) {
	t.Parallel()

	t.Run("unbuffered", func(t *testing.T) {
		t.Parallel()

		act := New[int, int](func(ref *Ref[int, int]) Processor[int, int] {
			return SimpleProcessor(func(req int) (int, error) {
				return req, nil
			})
		})

		ref := act.Run(t.Context(), "unbuffered", 0)
		defer ref.Stop()

		result, err := ref.RequestCtx(t.Context(), 42)
		require.NoError(t, err)
		assert.Equal(t, 42, result)
	})

	t.Run("buffered", func(t *testing.T) {
		t.Parallel()

		act := New[int, int](func(ref *Ref[int, int]) Processor[int, int] {
			return SimpleProcessor(func(req int) (int, error) {
				return req, nil
			})
		})

		ref := act.Run(t.Context(), "buffered", 100)
		defer ref.Stop()

		result, err := ref.RequestCtx(t.Context(), 42)
		require.NoError(t, err)
		assert.Equal(t, 42, result)
	})
}

func TestActorConcurrentRequests(t *testing.T) {
	t.Parallel()

	var counter atomic.Int32

	act := New[int, int](func(ref *Ref[int, int]) Processor[int, int] {
		return SimpleProcessor(func(req int) (int, error) {
			counter.Add(1)
			time.Sleep(10 * time.Millisecond) // Simulate work

			return req * 2, nil
		})
	})

	ref := act.Run(t.Context(), "concurrent", 100)
	defer ref.Stop()

	// Send multiple concurrent requests
	const numRequests = 10
	results := make(chan int, numRequests)
	errs := make(chan error, numRequests)

	for i := range numRequests {
		go func() {
			result, err := ref.RequestCtx(t.Context(), i)
			if err != nil {
				errs <- err

				return
			}
			results <- result
		}()
	}

	// Collect results
	for range numRequests {
		select {
		case err := <-errs:
			t.Fatalf("unexpected error: %v", err)
		case result := <-results:
			// Results should be valid (even number)
			assert.Equal(t, 0, result%2)
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for results")
		}
	}

	// All requests should have been processed
	assert.Equal(t, int32(numRequests), counter.Load())
}

func TestActorPublish(t *testing.T) {
	t.Parallel()

	var counter atomic.Int32

	act := New[int, empty](func(ref *Ref[int, empty]) Processor[int, empty] {
		return NewProcessor(func(msg Message[int, empty]) {
			counter.Add(1)
		})
	})

	ref := act.Run(t.Context(), "publisher", 10)
	defer ref.Stop()

	// Publish messages
	ref.Publish(Message[int, empty]{Request: 1})
	ref.Publish(Message[int, empty]{Request: 2})
	ref.PublishCtx(t.Context(), Message[int, empty]{Request: 3})

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, int32(3), counter.Load())
}

func TestActorName(t *testing.T) {
	t.Parallel()

	act := New[int, int](func(ref *Ref[int, int]) Processor[int, int] {
		// Verify the ref has the correct name
		assert.Equal(t, "my-actor", ref.Name())

		return SimpleProcessor(func(req int) (int, error) {
			return req, nil
		})
	})

	ref := act.Run(t.Context(), "my-actor", 1)
	defer ref.Stop()

	assert.Equal(t, "my-actor", ref.Name())
}

var errProcessingFailed = errors.New("processing failed")

func TestActorProcessorError(t *testing.T) {
	t.Parallel()

	act := New[int, int](func(ref *Ref[int, int]) Processor[int, int] {
		return SimpleProcessor(func(req int) (int, error) {
			if req < 0 {
				return 0, errProcessingFailed
			}

			return req, nil
		})
	})

	ref := act.Run(t.Context(), "error-test", 5)
	defer ref.Stop()

	// Successful request
	result, err := ref.RequestCtx(t.Context(), 10)
	require.NoError(t, err)
	assert.Equal(t, 10, result)

	// Failed request
	_, err = ref.RequestCtx(t.Context(), -1)
	require.Error(t, err)
	require.ErrorIs(t, err, errProcessingFailed)
}

func TestActorStopMultipleTimes(t *testing.T) {
	t.Parallel()

	act := New[int, int](func(ref *Ref[int, int]) Processor[int, int] {
		return SimpleProcessor(func(req int) (int, error) {
			return req, nil
		})
	})

	ref := act.Run(t.Context(), "stop-test", 1)

	// Stop multiple times should be safe
	ref.Stop()
	ref.Stop()
	ref.Stop()

	ref.Wait()

	assert.False(t, ref.Alive())
}

func TestActorRequestWithoutContext(t *testing.T) {
	t.Parallel()

	act := New[string, string](func(ref *Ref[string, string]) Processor[string, string] {
		return SimpleProcessor(func(req string) (string, error) {
			return "hello " + req, nil
		})
	})

	ref := act.Run(t.Context(), "greeter", 5)
	defer ref.Stop()

	// Use Request (without context)
	result, err := ref.Request("world")
	require.NoError(t, err)
	assert.Equal(t, "hello world", result)
}

func TestActorSendWithoutContext(t *testing.T) {
	t.Parallel()

	var received atomic.Value

	received.Store("")

	act := New[string, empty](func(ref *Ref[string, empty]) Processor[string, empty] {
		return SimpleProcessor(func(req string) (empty, error) {
			received.Store(req)

			return empty{}, nil
		})
	})

	ref := act.Run(t.Context(), "send-test", 5)
	defer ref.Stop()

	// Use Send (without context)
	ref.Send("test message")

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	result, ok := received.Load().(string)
	require.True(t, ok)
	assert.Equal(t, "test message", result)
}

func TestActorCustomProcessor(t *testing.T) {
	t.Parallel()

	// Create a custom processor that handles response channel manually
	act := New[int, int](func(ref *Ref[int, int]) Processor[int, int] {
		return NewProcessor(func(msg Message[int, int]) {
			result := msg.Request * 3

			if msg.ResponseChan != nil {
				msg.ResponseChan <- try.Try[int]{Value: result}
				close(msg.ResponseChan)
			}
		})
	})

	ref := act.Run(t.Context(), "custom", 5)
	defer ref.Stop()

	result, err := ref.RequestCtx(t.Context(), 7)
	require.NoError(t, err)
	assert.Equal(t, 21, result)
}
