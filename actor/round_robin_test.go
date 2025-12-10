package actor

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoundRobinBasic(t *testing.T) {
	t.Parallel()

	// Track which actor processed each request
	var actor1Count, actor2Count, actor3Count atomic.Int32

	// Create three worker actors
	makeActor := func(counter *atomic.Int32) *Ref[int, int] {
		act := New[int, int](func(ref *Ref[int, int]) Processor[int, int] {
			return SimpleProcessor(func(req int) (int, error) {
				counter.Add(1)

				return req, nil
			})
		})

		return act.Run(t.Context(), "worker", 10)
	}

	actor1 := makeActor(&actor1Count)
	actor2 := makeActor(&actor2Count)
	actor3 := makeActor(&actor3Count)

	defer actor1.Stop()
	defer actor2.Stop()
	defer actor3.Stop()

	// Create round-robin actor
	rrActor, err := RoundRobin(actor1, actor2, actor3)
	require.NoError(t, err)

	rrRef := rrActor.Run(t.Context(), "round-robin", 10)
	defer rrRef.Stop()

	// Send 9 messages - should distribute evenly
	for i := range 9 {
		rrRef.SendCtx(t.Context(), i)
	}

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	// Each actor should have processed 3 messages
	assert.Equal(t, int32(3), actor1Count.Load())
	assert.Equal(t, int32(3), actor2Count.Load())
	assert.Equal(t, int32(3), actor3Count.Load())
}

func TestRoundRobinNoActors(t *testing.T) {
	t.Parallel()

	// Creating a round-robin with no actors should error
	_, err := RoundRobin[int, int]()
	require.ErrorIs(t, err, ErrNoActors)
}

func TestRoundRobinSingleActor(t *testing.T) {
	t.Parallel()

	var count atomic.Int32

	act := New[int, int](func(ref *Ref[int, int]) Processor[int, int] {
		return SimpleProcessor(func(req int) (int, error) {
			count.Add(1)

			return req * 2, nil
		})
	})

	worker := act.Run(t.Context(), "solo-worker", 10)
	defer worker.Stop()

	// Create round-robin with single actor
	rrActor, err := RoundRobin(worker)
	require.NoError(t, err)

	rrRef := rrActor.Run(t.Context(), "rr-solo", 10)
	defer rrRef.Stop()

	// Send messages
	for i := range 5 {
		rrRef.SendCtx(t.Context(), i)
	}

	time.Sleep(100 * time.Millisecond)

	// Single actor should process all messages
	assert.Equal(t, int32(5), count.Load())
}

func TestRoundRobinDistribution(t *testing.T) {
	t.Parallel()

	const numActors = 5

	const numMessages = 100

	counters := make([]*atomic.Int32, numActors)
	actors := make([]*Ref[int, empty], numActors)

	for i := range numActors {
		counters[i] = &atomic.Int32{}
		counter := counters[i]

		act := New[int, empty](func(ref *Ref[int, empty]) Processor[int, empty] {
			return SimpleProcessor(func(req int) (empty, error) {
				counter.Add(1)
				time.Sleep(1 * time.Millisecond) // Simulate work

				return empty{}, nil
			})
		})

		actors[i] = act.Run(t.Context(), "worker", 20)
		defer actors[i].Stop()
	}

	// Create round-robin
	rrActor, err := RoundRobin(actors...)
	require.NoError(t, err)

	rrRef := rrActor.Run(t.Context(), "load-balancer", 50)
	defer rrRef.Stop()

	// Send many messages
	for i := range numMessages {
		rrRef.SendCtx(t.Context(), i)
	}

	// Wait for all processing
	time.Sleep(500 * time.Millisecond)

	// Check distribution is even
	expectedPerActor := numMessages / numActors

	for i, counter := range counters {
		count := counter.Load()
		// Should be exactly equal since we're distributing evenly
		assert.Equal(t, int32(expectedPerActor), count, "actor %d should have processed %d messages", i, expectedPerActor)
	}
}

func TestRoundRobinConcurrent(t *testing.T) {
	t.Parallel()

	var total atomic.Int32

	// Create multiple worker actors
	workers := make([]*Ref[int, int], 4)

	for i := range 4 {
		act := New[int, int](func(ref *Ref[int, int]) Processor[int, int] {
			return SimpleProcessor(func(req int) (int, error) {
				total.Add(int32(req)) //nolint:gosec
				time.Sleep(5 * time.Millisecond)

				return req, nil
			})
		})

		workers[i] = act.Run(t.Context(), "worker", 20)
		defer workers[i].Stop()
	}

	// Create round-robin
	rrActor, err := RoundRobin(workers...)
	require.NoError(t, err)

	rrRef := rrActor.Run(t.Context(), "concurrent-rr", 50)
	defer rrRef.Stop()

	// Send messages concurrently
	const numGoroutines = 10

	const msgsPerGoroutine = 10

	done := make(chan bool, numGoroutines)

	for g := range numGoroutines {
		go func(base int) {
			for i := range msgsPerGoroutine {
				rrRef.SendCtx(t.Context(), base+i)
			}

			done <- true
		}(g * msgsPerGoroutine)
	}

	// Wait for all goroutines to finish sending
	for range numGoroutines {
		<-done
	}

	// Wait for processing
	time.Sleep(1 * time.Second)

	// Verify all messages were processed (sum of 0..99)
	expectedSum := (numGoroutines * msgsPerGoroutine * (numGoroutines*msgsPerGoroutine - 1)) / 2
	assert.Equal(t, int32(expectedSum), total.Load())
}

func TestRoundRobinCounterWraparound(t *testing.T) {
	t.Parallel()

	// Create a small number of actors
	var count1, count2 atomic.Int32

	act1 := New[int, empty](func(ref *Ref[int, empty]) Processor[int, empty] {
		return SimpleProcessor(func(req int) (empty, error) {
			count1.Add(1)

			return empty{}, nil
		})
	})

	act2 := New[int, empty](func(ref *Ref[int, empty]) Processor[int, empty] {
		return SimpleProcessor(func(req int) (empty, error) {
			count2.Add(1)

			return empty{}, nil
		})
	})

	worker1 := act1.Run(t.Context(), "w1", 100)
	worker2 := act2.Run(t.Context(), "w2", 100)

	defer worker1.Stop()
	defer worker2.Stop()

	rrActor, err := RoundRobin(worker1, worker2)
	require.NoError(t, err)

	rrRef := rrActor.Run(t.Context(), "wraparound-test", 100)
	defer rrRef.Stop()

	// Send many messages to test counter wraparound behavior
	const numMessages = 1000
	for i := range numMessages {
		rrRef.SendCtx(t.Context(), i)
	}

	time.Sleep(500 * time.Millisecond)

	// Should still distribute evenly
	assert.Equal(t, int32(numMessages/2), count1.Load())
	assert.Equal(t, int32(numMessages/2), count2.Load())
}

func TestRoundRobinWithFailingActor(t *testing.T) {
	t.Parallel()

	var successCount atomic.Int32

	// One actor that works
	goodAct := New[int, empty](func(ref *Ref[int, empty]) Processor[int, empty] {
		return SimpleProcessor(func(req int) (empty, error) {
			successCount.Add(1)

			return empty{}, nil
		})
	})

	// One actor that panics
	badAct := New[int, empty](func(ref *Ref[int, empty]) Processor[int, empty] {
		return NewProcessor(func(msg Message[int, empty]) {
			panic("intentional panic")
		})
	})

	good := goodAct.Run(t.Context(), "good", 10)
	bad := badAct.Run(t.Context(), "bad", 10)

	defer good.Stop()
	defer bad.Stop()

	rrActor, err := RoundRobin(good, bad)
	require.NoError(t, err)

	rrRef := rrActor.Run(t.Context(), "mixed-rr", 20)
	defer rrRef.Stop()

	// Send messages - half will go to good actor, half to bad
	for i := range 10 {
		rrRef.SendCtx(t.Context(), i)
	}

	time.Sleep(200 * time.Millisecond)

	// Good actor should have processed 5 messages
	// Bad actor will panic but round-robin should continue
	assert.Equal(t, int32(5), successCount.Load())
}
