package channels

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInfiniteChan_BasicSendReceive(t *testing.T) {
	t.Parallel()

	input, output := InfiniteChan[int]()

	input <- 1
	input <- 2
	input <- 3

	assert.Equal(t, 1, <-output)
	assert.Equal(t, 2, <-output)
	assert.Equal(t, 3, <-output)
}

func TestInfiniteChan_SendWithoutBlocking(t *testing.T) {
	t.Parallel()

	input, output := InfiniteChan[int]()

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

	input, output := InfiniteChan[int]()

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

	input, output := InfiniteChan[string]()

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

	input, output := InfiniteChan[int]()

	close(input)

	// Output channel should close immediately when empty
	_, ok := <-output
	assert.False(t, ok)
}

func TestInfiniteChan_SlowConsumer(t *testing.T) {
	t.Parallel()

	input, output := InfiniteChan[int]()

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

	input, output := InfiniteChan[int]()

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

		input, output := InfiniteChan[string]()
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

		input, output := InfiniteChan[testStruct]()
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

	input, output := InfiniteChan[int]()

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

	input, output := InfiniteChan[int]()

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
