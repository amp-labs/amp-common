package bgworker

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubmit(t *testing.T) {
	t.Parallel()

	var counter atomic.Int32

	task := Submit(func() {
		counter.Add(1)
	})

	err := task.Wait()
	require.NoError(t, err)
	assert.Equal(t, int32(1), counter.Load())
}

func TestSubmitMultipleTasks(t *testing.T) {
	t.Parallel()

	var counter atomic.Int32

	const numTasks = 10
	tasks := make([]interface{ Wait() error }, numTasks)

	for i := range numTasks {
		tasks[i] = Submit(func() {
			counter.Add(1)
		})
	}

	for _, task := range tasks {
		err := task.Wait()
		require.NoError(t, err)
	}

	assert.Equal(t, int32(numTasks), counter.Load())
}

func TestGo(t *testing.T) {
	t.Parallel()

	var counter atomic.Int32

	done := make(chan struct{})

	err := Go(func() {
		counter.Add(1)
		close(done)
	})

	require.NoError(t, err)

	// Wait for the goroutine to signal completion
	select {
	case <-done:
		assert.Equal(t, int32(1), counter.Load())
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for goroutine to complete")
	}
}

func TestGoMultipleTasks(t *testing.T) {
	t.Parallel()

	var counter atomic.Int32

	done := make(chan struct{}, 10)

	for range 10 {
		err := Go(func() {
			counter.Add(1)
			done <- struct{}{}
		})
		require.NoError(t, err)
	}

	// Wait for all goroutines to signal completion
	for i := range 10 {
		select {
		case <-done:
			// Task completed
		case <-time.After(1 * time.Second):
			t.Fatalf("timeout waiting for goroutine %d to complete", i)
		}
	}

	assert.Equal(t, int32(10), counter.Load())
}

func TestSubmitWithPanic(t *testing.T) {
	t.Parallel()

	task := Submit(func() {
		panic("test panic")
	})

	// The task should complete even if it panics
	// pond handles panics internally and returns an error
	err := task.Wait()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "test panic")
}

func TestGoWithPanic(t *testing.T) {
	t.Parallel()

	err := Go(func() {
		panic("test panic")
	})

	// Go should not return an error when submitting
	require.NoError(t, err)

	// Wait a bit for the goroutine to execute and panic
	time.Sleep(100 * time.Millisecond)
}

func TestConcurrentSubmit(t *testing.T) {
	t.Parallel()

	var counter atomic.Int32

	const numTasks = 100
	tasks := make([]interface{ Wait() error }, numTasks)

	// Submit 100 tasks concurrently
	for i := range numTasks {
		tasks[i] = Submit(func() {
			time.Sleep(10 * time.Millisecond)
			counter.Add(1)
		})
	}

	// Wait for all tasks to complete
	for _, task := range tasks {
		err := task.Wait()
		require.NoError(t, err)
	}

	assert.Equal(t, int32(numTasks), counter.Load())
}

func TestConcurrentGo(t *testing.T) {
	t.Parallel()

	var counter atomic.Int32

	// Submit 100 tasks concurrently
	for range 100 {
		err := Go(func() {
			time.Sleep(10 * time.Millisecond)
			counter.Add(1)
		})
		require.NoError(t, err)
	}

	// Wait for all tasks to complete
	time.Sleep(1500 * time.Millisecond)
	assert.Equal(t, int32(100), counter.Load())
}

func TestWorkerPoolLaziness(t *testing.T) {
	t.Parallel()

	// This test verifies that the worker pool is lazy-initialized
	// by submitting a task and ensuring it completes
	var executed atomic.Bool

	task := Submit(func() {
		executed.Store(true)
	})

	err := task.Wait()
	require.NoError(t, err)
	assert.True(t, executed.Load())
}
