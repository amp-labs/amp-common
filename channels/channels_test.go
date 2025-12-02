package channels

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreate_UnbufferedChannel(t *testing.T) {
	t.Parallel()

	input, output, lenFunc := Create[int](0)

	// Verify initial length is 0
	assert.Equal(t, 0, lenFunc())

	// Send and receive in separate goroutines (required for unbuffered channels)
	done := make(chan bool)

	go func() {
		input <- 42
		close(input)
	}()

	go func() {
		val := <-output
		assert.Equal(t, 42, val)
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for unbuffered channel communication")
	}
}

func TestCreate_BufferedChannel(t *testing.T) {
	t.Parallel()

	input, output, lenFunc := Create[string](3)

	// Verify initial length is 0
	assert.Equal(t, 0, lenFunc())

	// Send values without blocking (buffered channel)
	input <- "first"

	assert.Equal(t, 1, lenFunc())

	input <- "second"

	assert.Equal(t, 2, lenFunc())

	input <- "third"

	assert.Equal(t, 3, lenFunc())

	// Receive values
	assert.Equal(t, "first", <-output)
	assert.Equal(t, 2, lenFunc())

	assert.Equal(t, "second", <-output)
	assert.Equal(t, 1, lenFunc())

	assert.Equal(t, "third", <-output)
	assert.Equal(t, 0, lenFunc())
}

func TestCreate_InfiniteChannel(t *testing.T) {
	t.Parallel()

	input, output, _ := Create[int](-1)

	// Send many values without blocking (infinite buffering)
	for i := range 100 {
		input <- i
	}

	// Receive all values in order
	for i := range 100 {
		assert.Equal(t, i, <-output)
	}
}

func TestCreate_BufferedChannelCapacity(t *testing.T) {
	t.Parallel()

	input, output, lenFunc := Create[int](5)

	// Fill the buffer
	for i := range 5 {
		input <- i
	}

	assert.Equal(t, 5, lenFunc())

	// Drain the buffer
	for i := range 5 {
		val := <-output
		assert.Equal(t, i, val)
	}

	assert.Equal(t, 0, lenFunc())
}

func TestCreate_CloseChannel(t *testing.T) {
	t.Parallel()

	input, output, _ := Create[int](2)

	input <- 1
	input <- 2
	close(input)

	// Should receive all values before channel closes
	assert.Equal(t, 1, <-output)
	assert.Equal(t, 2, <-output)

	// Channel should be closed
	val, ok := <-output
	assert.False(t, ok)
	assert.Equal(t, 0, val)
}

func TestCreate_DifferentTypes(t *testing.T) {
	t.Parallel()

	t.Run("int", func(t *testing.T) {
		t.Parallel()

		input, output, _ := Create[int](1)
		input <- 42

		assert.Equal(t, 42, <-output)
	})

	t.Run("string", func(t *testing.T) {
		t.Parallel()

		input, output, _ := Create[string](1)
		input <- "hello"

		assert.Equal(t, "hello", <-output)
	})

	t.Run("struct", func(t *testing.T) {
		t.Parallel()

		type testStruct struct {
			ID   int
			Name string
		}

		input, output, _ := Create[testStruct](1)
		expected := testStruct{ID: 1, Name: "test"}
		input <- expected
		assert.Equal(t, expected, <-output)
	})
}

func TestCreate_ZeroSizeVsNegativeSize(t *testing.T) {
	t.Parallel()

	t.Run("zero creates unbuffered", func(t *testing.T) {
		t.Parallel()

		input, output, lenFunc := Create[int](0)

		// Unbuffered channels don't allow sending without receiver
		done := make(chan bool)

		go func() {
			input <- 1
			done <- true
		}()

		val := <-output
		assert.Equal(t, 1, val)
		assert.Equal(t, 0, lenFunc())

		<-done
	})

	t.Run("negative creates infinite", func(t *testing.T) {
		t.Parallel()

		input, _, _ := Create[int](-5)

		// Infinite channels allow sending many values without blocking
		// If this completes without blocking, infinite channel is working
		for i := range 1000 {
			input <- i
		}
	})
}

func TestInfiniteChan_BasicSendReceive(t *testing.T) {
	t.Parallel()

	input, output, _ := InfiniteChan[int]()

	input <- 1
	input <- 2
	input <- 3

	assert.Equal(t, 1, <-output)
	assert.Equal(t, 2, <-output)
	assert.Equal(t, 3, <-output)
}

func TestInfiniteChan_SendWithoutBlocking(t *testing.T) {
	t.Parallel()

	input, output, _ := InfiniteChan[int]()

	// Send many values without blocking
	for i := range 1000 {
		input <- i
	}

	// Receive all values in order
	for i := range 1000 {
		assert.Equal(t, i, <-output)
	}
}

func TestInfiniteChan_CloseInput(t *testing.T) {
	t.Parallel()

	input, output, _ := InfiniteChan[int]()

	input <- 1
	input <- 2
	close(input)

	assert.Equal(t, 1, <-output)
	assert.Equal(t, 2, <-output)

	// Output channel should be closed after all values are consumed
	val, ok := <-output
	assert.False(t, ok)
	assert.Equal(t, 0, val)
}

func TestInfiniteChan_CloseWithQueuedValues(t *testing.T) {
	t.Parallel()

	input, output, _ := InfiniteChan[string]()

	// Send values and close immediately
	input <- "first"
	input <- "second"
	input <- "third"
	close(input)

	// Should still receive all values
	assert.Equal(t, "first", <-output)
	assert.Equal(t, "second", <-output)
	assert.Equal(t, "third", <-output)

	// Then channel should be closed
	_, ok := <-output
	assert.False(t, ok)
}

func TestInfiniteChan_EmptyClose(t *testing.T) {
	t.Parallel()

	input, output, _ := InfiniteChan[int]()

	close(input)

	// Output channel should close immediately when empty
	_, ok := <-output
	assert.False(t, ok)
}

func TestInfiniteChan_SlowConsumer(t *testing.T) {
	t.Parallel()

	input, output, _ := InfiniteChan[int]()

	// Fast producer
	go func() {
		for i := range 10 {
			input <- i
		}

		close(input)
	}()

	// Slow consumer
	time.Sleep(50 * time.Millisecond)

	received := make([]int, 0, 10)
	for val := range output {
		received = append(received, val)
	}

	expected := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	assert.Equal(t, expected, received)
}

func TestInfiniteChan_FastConsumer(t *testing.T) {
	t.Parallel()

	input, output, _ := InfiniteChan[int]()

	received := make(chan int, 10)

	// Fast consumer
	go func() {
		for val := range output {
			received <- val
		}

		close(received)
	}()

	// Slow producer
	for i := range 5 {
		input <- i

		time.Sleep(10 * time.Millisecond)
	}

	close(input)

	result := make([]int, 0, 5)
	for val := range received {
		result = append(result, val)
	}

	expected := []int{0, 1, 2, 3, 4}
	assert.Equal(t, expected, result)
}

func TestInfiniteChan_DifferentTypes(t *testing.T) {
	t.Parallel()

	t.Run("string", func(t *testing.T) {
		t.Parallel()

		input, output, _ := InfiniteChan[string]()
		input <- "hello"
		input <- "world"
		close(input)

		assert.Equal(t, "hello", <-output)
		assert.Equal(t, "world", <-output)
		_, ok := <-output
		assert.False(t, ok)
	})

	t.Run("struct", func(t *testing.T) {
		t.Parallel()

		type testStruct struct {
			ID   int
			Name string
		}

		input, output, _ := InfiniteChan[testStruct]()
		input <- testStruct{ID: 1, Name: "first"}
		input <- testStruct{ID: 2, Name: "second"}
		close(input)

		assert.Equal(t, testStruct{ID: 1, Name: "first"}, <-output)
		assert.Equal(t, testStruct{ID: 2, Name: "second"}, <-output)
		_, ok := <-output
		assert.False(t, ok)
	})
}

func TestInfiniteChan_OrderPreservation(t *testing.T) {
	t.Parallel()

	input, output, _ := InfiniteChan[int]()

	// Send values in specific order
	expected := []int{5, 3, 9, 1, 7, 2, 8, 4, 6}
	for _, val := range expected {
		input <- val
	}

	close(input)

	// Verify order is preserved
	received := make([]int, 0, len(expected))
	for val := range output {
		received = append(received, val)
	}

	assert.Equal(t, expected, received)
}

func TestInfiniteChan_ConcurrentSenders(t *testing.T) {
	t.Parallel()

	input, output, _ := InfiniteChan[int]()

	// Note: InfiniteChan is NOT safe for concurrent senders on the input channel
	// This test verifies that values are still received correctly when sent sequentially
	done := make(chan bool)

	go func() {
		for i := range 5 {
			input <- i
		}

		close(input)
	}()

	go func() {
		count := 0
		for range output {
			count++
		}

		assert.Equal(t, 5, count)
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for values")
	}
}

func TestCloseChannelIgnorePanic_NormalClose(t *testing.T) {
	t.Parallel()

	ch := make(chan int, 1)
	ch <- 42

	// Should close without panic
	CloseChannelIgnorePanic(ch)

	// Verify channel is closed
	val := <-ch
	assert.Equal(t, 42, val)

	val, ok := <-ch
	assert.False(t, ok)
	assert.Equal(t, 0, val)
}

func TestCloseChannelIgnorePanic_NilChannel(t *testing.T) {
	t.Parallel()

	var ch chan int

	// Should not panic with nil channel
	CloseChannelIgnorePanic(ch)
}

func TestCloseChannelIgnorePanic_AlreadyClosed(t *testing.T) {
	t.Parallel()

	ch := make(chan int)
	close(ch)

	// Should not panic when closing an already-closed channel
	CloseChannelIgnorePanic(ch)
	CloseChannelIgnorePanic(ch) // Close again
}

func TestSendCatchPanic_SuccessfulSend(t *testing.T) {
	t.Parallel()

	ch := make(chan int, 1)

	err := SendCatchPanic(ch, 42)

	require.NoError(t, err)
	assert.Equal(t, 42, <-ch)
}

func TestSendCatchPanic_NilChannel(t *testing.T) {
	t.Parallel()

	var ch chan int

	err := SendCatchPanic(ch, 42)

	assert.NoError(t, err)
}

func TestSendCatchPanic_ClosedChannel(t *testing.T) {
	t.Parallel()

	ch := make(chan int)
	close(ch)

	err := SendCatchPanic(ch, 42)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "panic")
}

func TestSendCatchPanic_UnbufferedChannel(t *testing.T) {
	t.Parallel()

	ch := make(chan int)

	done := make(chan error)
	go func() {
		done <- SendCatchPanic(ch, 42)
	}()

	// Receive the value
	val := <-ch
	assert.Equal(t, 42, val)

	// Verify no error
	err := <-done
	assert.NoError(t, err)
}

func TestSendContextCatchPanic_SuccessfulSend(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	ch := make(chan int, 1)

	err := SendContextCatchPanic(ctx, ch, 42)

	require.NoError(t, err)
	assert.Equal(t, 42, <-ch)
}

func TestSendContextCatchPanic_NilContext(t *testing.T) {
	t.Parallel()

	ch := make(chan int, 1)

	err := SendContextCatchPanic(nil, ch, 42) //nolint:usetesting,staticcheck // Testing nil context fallback behavior

	require.NoError(t, err)
	assert.Equal(t, 42, <-ch)
}

func TestSendContextCatchPanic_NilChannel(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	var ch chan int

	err := SendContextCatchPanic(ctx, ch, 42)

	assert.NoError(t, err)
}

func TestSendContextCatchPanic_CanceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel() // Cancel immediately

	ch := make(chan int)

	err := SendContextCatchPanic(ctx, ch, 42)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestSendContextCatchPanic_ContextCanceledDuringSend(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	ch := make(chan int) // Unbuffered, will block

	// Start send in goroutine (will block since no receiver)
	done := make(chan error)
	go func() {
		done <- SendContextCatchPanic(ctx, ch, 42)
	}()

	// Wait for context to timeout
	select {
	case err := <-done:
		require.Error(t, err)
		require.ErrorIs(t, err, context.DeadlineExceeded)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for context cancellation")
	}
}

func TestSendContextCatchPanic_ClosedChannel(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	ch := make(chan int)
	close(ch)

	err := SendContextCatchPanic(ctx, ch, 42)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "panic")
}

func TestSendContextCatchPanic_UnbufferedChannelWithReceiver(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	ch := make(chan int)

	done := make(chan error)
	received := make(chan int)

	// Start sender
	go func() {
		done <- SendContextCatchPanic(ctx, ch, 42)
	}()

	// Start receiver
	go func() {
		received <- <-ch
	}()

	// Verify both operations complete successfully
	select {
	case val := <-received:
		assert.Equal(t, 42, val)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for receive")
	}

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for send")
	}
}

func TestSendContextCatchPanic_NilContextWithClosedChannel(t *testing.T) {
	t.Parallel()

	ch := make(chan int)
	close(ch)

	// Nil context should fall back to SendCatchPanic behavior
	err := SendContextCatchPanic(nil, ch, 42) //nolint:usetesting,staticcheck // Testing nil context fallback behavior

	require.Error(t, err)
	assert.Contains(t, err.Error(), "panic")
}
