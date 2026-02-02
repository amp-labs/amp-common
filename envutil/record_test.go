package envutil

import (
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestObserver(t *testing.T) {
	// Clean up after test
	defer func() {
		EnableRecording(false)
		EnableStackTraces(false)

		_ = CollectRecordingEvents(true)
	}()

	// Track observer calls
	var callCount atomic.Int32

	var lastEvent ValueReadEvent

	var eventMu sync.Mutex

	// Register observer
	unregister := RegisterObserver(func(event ValueReadEvent) {
		eventMu.Lock()
		defer eventMu.Unlock()

		callCount.Add(1)

		lastEvent = event
	})
	defer unregister()

	// Enable recording
	EnableRecording(true)

	// Set a test environment variable
	testKey := "TEST_OBSERVER_VAR"
	testValue := "test_value"
	t.Setenv(testKey, testValue)

	// Read the environment variable
	value, err := String(t.Context(), testKey).Value()
	require.NoError(t, err)
	assert.Equal(t, testValue, value)

	// Give observer a moment to be called (should be immediate but be safe)
	time.Sleep(10 * time.Millisecond)

	// Verify observer was called
	assert.Equal(t, int32(1), callCount.Load(), "observer should be called once")

	eventMu.Lock()
	assert.Equal(t, testKey, lastEvent.Key)
	assert.Equal(t, testValue, lastEvent.Value)
	assert.True(t, lastEvent.IsSet)
	assert.Equal(t, Environment, lastEvent.Source)
	eventMu.Unlock()
}

func TestObserverUnregister(t *testing.T) {
	// Clean up after test
	defer func() {
		EnableRecording(false)
		EnableStackTraces(false)

		_ = CollectRecordingEvents(true)
	}()

	// Track observer calls
	var callCount atomic.Int32

	// Register observer
	unregister := RegisterObserver(func(event ValueReadEvent) {
		callCount.Add(1)
	})

	// Enable recording
	EnableRecording(true)

	// Set a test environment variable
	testKey := "TEST_UNREGISTER_VAR"
	t.Setenv(testKey, "value1")

	// First read - observer should be called
	_, _ = String(t.Context(), testKey).Value()

	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, int32(1), callCount.Load(), "observer should be called once")

	// Unregister observer
	unregister()

	// Second read - observer should NOT be called
	_, _ = String(t.Context(), testKey).Value()

	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, int32(1), callCount.Load(), "observer should still only have been called once")
}

func TestMultipleObservers(t *testing.T) {
	// Clean up after test
	defer func() {
		EnableRecording(false)
		EnableStackTraces(false)

		_ = CollectRecordingEvents(true)
	}()

	// Track calls from multiple observers
	var count1, count2, count3 atomic.Int32

	// Register multiple observers
	unregister1 := RegisterObserver(func(event ValueReadEvent) {
		count1.Add(1)
	})
	defer unregister1()

	unregister2 := RegisterObserver(func(event ValueReadEvent) {
		count2.Add(1)
	})
	defer unregister2()

	unregister3 := RegisterObserver(func(event ValueReadEvent) {
		count3.Add(1)
	})
	defer unregister3()

	// Enable recording
	EnableRecording(true)

	// Set and read environment variable
	testKey := "TEST_MULTI_OBSERVER_VAR"
	t.Setenv(testKey, "test")

	_, _ = String(t.Context(), testKey).Value()

	time.Sleep(10 * time.Millisecond)

	// All observers should be called
	assert.Equal(t, int32(1), count1.Load())
	assert.Equal(t, int32(1), count2.Load())
	assert.Equal(t, int32(1), count3.Load())
}

func TestObserverWithRecordingDisabled(t *testing.T) {
	// Clean up after test
	defer func() {
		EnableRecording(false)
		EnableStackTraces(false)

		_ = CollectRecordingEvents(true)
	}()

	// Track observer calls
	var callCount atomic.Int32

	// Register observer
	unregister := RegisterObserver(func(event ValueReadEvent) {
		callCount.Add(1)
	})
	defer unregister()

	// Recording is disabled by default
	assert.False(t, IsRecording())

	// Set and read environment variable
	testKey := "TEST_DISABLED_VAR"
	t.Setenv(testKey, "test")

	_, _ = String(t.Context(), testKey).Value()

	time.Sleep(10 * time.Millisecond)

	// Observer SHOULD still be called even when recording is disabled
	// because observers are independent of the recording flag
	assert.Equal(t, int32(1), callCount.Load(), "observer should be called even when recording is disabled")

	// But no events should be recorded in the events slice
	events := CollectRecordingEvents(false)
	assert.Empty(t, events, "no events should be recorded when recording is disabled")
}

func TestObserverStackTraces(t *testing.T) {
	// Clean up after test
	defer func() {
		EnableRecording(false)
		EnableStackTraces(false)

		_ = CollectRecordingEvents(true)
	}()

	// Track observer calls
	var lastEvent ValueReadEvent

	var eventMu sync.Mutex

	// Register observer
	unregister := RegisterObserver(func(event ValueReadEvent) {
		eventMu.Lock()
		defer eventMu.Unlock()

		lastEvent = event
	})
	defer unregister()

	// Enable recording with stack traces
	EnableRecording(true)
	EnableStackTraces(true)

	// Set and read environment variable
	testKey := "TEST_STACK_VAR"
	t.Setenv(testKey, "test")

	_, _ = String(t.Context(), testKey).Value()

	time.Sleep(10 * time.Millisecond)

	// Verify stack trace was captured
	eventMu.Lock()
	assert.NotEmpty(t, lastEvent.Stack, "stack trace should be captured when enabled")
	eventMu.Unlock()
}

//nolint:tparallel,paralleltest // Cannot use t.Parallel() due to global recording/observer state
func TestObserverConcurrency(t *testing.T) {
	// Clean up after test
	defer func() {
		EnableRecording(false)
		EnableStackTraces(false)

		_ = CollectRecordingEvents(true)
	}()

	// Track observer calls
	var callCount atomic.Int32

	// Register observer
	unregister := RegisterObserver(func(event ValueReadEvent) {
		callCount.Add(1)
	})
	defer unregister()

	// Enable recording
	EnableRecording(true)

	// Set test environment variables
	for i := range 10 {
		key := "TEST_CONCURRENT_VAR_" + string(rune('0'+i))
		t.Setenv(key, "test")
	}

	// Read environment variables concurrently
	var wg sync.WaitGroup
	for i := range 10 {
		wg.Add(1)

		go func(idx int) {
			defer wg.Done()

			key := "TEST_CONCURRENT_VAR_" + string(rune('0'+idx))
			_, _ = String(t.Context(), key).Value()
		}(i)
	}

	wg.Wait()
	time.Sleep(50 * time.Millisecond)

	// All reads should have triggered the observer
	assert.Equal(t, int32(10), callCount.Load())
}

//nolint:tparallel,paralleltest // Cannot use t.Parallel() due to global recording/observer state
func TestHasObserversFlag(t *testing.T) {
	// Initially, no observers should be registered
	assert.False(t, hasObservers.Load(), "hasObservers should be false initially")

	// Register first observer
	unregister1 := RegisterObserver(func(event ValueReadEvent) {})

	assert.True(t, hasObservers.Load(), "hasObservers should be true after registering first observer")

	// Register second observer
	unregister2 := RegisterObserver(func(event ValueReadEvent) {})

	assert.True(t, hasObservers.Load(), "hasObservers should still be true with multiple observers")

	// Unregister first observer
	unregister1()
	assert.True(t, hasObservers.Load(), "hasObservers should still be true with one observer remaining")

	// Unregister second observer
	unregister2()
	assert.False(t, hasObservers.Load(), "hasObservers should be false after all observers are removed")

	// Register and immediately unregister
	unregister3 := RegisterObserver(func(event ValueReadEvent) {})

	assert.True(t, hasObservers.Load(), "hasObservers should be true after re-registering")
	unregister3()
	assert.False(t, hasObservers.Load(), "hasObservers should be false after unregistering")
}

// Test recording with real environment variables.
func TestRecordingWithRealEnvVars(t *testing.T) {
	// Clean up after test
	defer func() {
		EnableRecording(false)
		EnableStackTraces(false)

		_ = CollectRecordingEvents(true)
	}()

	// Enable recording
	EnableRecording(true)

	// Set real environment variable
	testKey := "TEST_REAL_ENV_VAR"
	testValue := "real_value"
	t.Setenv(testKey, testValue)

	// Read the environment variable
	value, err := String(t.Context(), testKey).Value()
	require.NoError(t, err)
	assert.Equal(t, testValue, value)

	// Check recorded events
	events := CollectRecordingEvents(false)
	require.Len(t, events, 1, "should have recorded one event")

	event := events[0]
	assert.Equal(t, testKey, event.Key)
	assert.Equal(t, testValue, event.Value)
	assert.True(t, event.IsSet)
	assert.Equal(t, Environment, event.Source, "source should be Environment for real env vars")
	assert.NotZero(t, event.Time)
}

// Test recording with context overrides.
//
//nolint:tparallel,paralleltest // Cannot use t.Parallel() due to global recording/observer state
func TestRecordingWithContextOverride(t *testing.T) {
	// Clean up after test
	defer func() {
		EnableRecording(false)
		EnableStackTraces(false)

		_ = CollectRecordingEvents(true)
	}()

	// Enable recording
	EnableRecording(true)

	// Create context with override
	testKey := "TEST_CONTEXT_OVERRIDE_VAR"
	testValue := "context_value"
	ctx := WithEnvOverride(t.Context(), testKey, testValue)

	// Read the environment variable from context
	value, err := String(ctx, testKey).Value()
	require.NoError(t, err)
	assert.Equal(t, testValue, value)

	// Check recorded events
	events := CollectRecordingEvents(false)
	require.Len(t, events, 1, "should have recorded one event")

	event := events[0]
	assert.Equal(t, testKey, event.Key)
	assert.Equal(t, testValue, event.Value)
	assert.True(t, event.IsSet)
	assert.Equal(t, Context, event.Source, "source should be Context for context overrides")
	assert.NotZero(t, event.Time)
}

// Test recording when environment variable is not set.
//
//nolint:tparallel,paralleltest // Cannot use t.Parallel() due to global recording/observer state
func TestRecordingWithUnsetVar(t *testing.T) {
	// Clean up after test
	defer func() {
		EnableRecording(false)
		EnableStackTraces(false)

		_ = CollectRecordingEvents(true)
	}()

	// Enable recording
	EnableRecording(true)

	// Read a non-existent environment variable
	testKey := "TEST_NONEXISTENT_VAR_12345"
	_ = os.Unsetenv(testKey) // Ensure it's not set

	_, err := String(t.Context(), testKey).Value()
	require.Error(t, err, "should error when var is not set")

	// Check recorded events
	events := CollectRecordingEvents(false)
	require.Len(t, events, 1, "should have recorded one event")

	event := events[0]
	assert.Equal(t, testKey, event.Key)
	assert.Empty(t, event.Value)
	assert.False(t, event.IsSet)
	assert.Equal(t, None, event.Source, "source should be None for unset vars")
	assert.NotZero(t, event.Time)
}

// Test observer with real environment variables.
func TestObserverWithRealEnvVars(t *testing.T) {
	// Clean up after test
	defer func() {
		EnableRecording(false)
		EnableStackTraces(false)

		_ = CollectRecordingEvents(true)
	}()

	// Track observer calls
	var lastEvent ValueReadEvent

	var eventMu sync.Mutex

	unregister := RegisterObserver(func(event ValueReadEvent) {
		eventMu.Lock()
		defer eventMu.Unlock()

		lastEvent = event
	})
	defer unregister()

	// Enable recording
	EnableRecording(true)

	// Set real environment variable
	testKey := "TEST_OBSERVER_REAL_VAR"
	testValue := "observer_real_value"
	t.Setenv(testKey, testValue)

	// Read the environment variable
	value, err := String(t.Context(), testKey).Value()
	require.NoError(t, err)
	assert.Equal(t, testValue, value)

	time.Sleep(10 * time.Millisecond)

	// Verify observer was called with correct source
	eventMu.Lock()
	assert.Equal(t, testKey, lastEvent.Key)
	assert.Equal(t, testValue, lastEvent.Value)
	assert.True(t, lastEvent.IsSet)
	assert.Equal(t, Environment, lastEvent.Source)
	eventMu.Unlock()
}

// Test observer with context overrides.
//
//nolint:tparallel,paralleltest // Cannot use t.Parallel() due to global recording/observer state
func TestObserverWithContextOverride(t *testing.T) {
	// Clean up after test
	defer func() {
		EnableRecording(false)
		EnableStackTraces(false)

		_ = CollectRecordingEvents(true)
	}()

	// Track observer calls
	var lastEvent ValueReadEvent

	var eventMu sync.Mutex

	unregister := RegisterObserver(func(event ValueReadEvent) {
		eventMu.Lock()
		defer eventMu.Unlock()

		lastEvent = event
	})
	defer unregister()

	// Enable recording
	EnableRecording(true)

	// Create context with override
	testKey := "TEST_OBSERVER_CONTEXT_VAR"
	testValue := "observer_context_value"
	ctx := WithEnvOverride(t.Context(), testKey, testValue)

	// Read the environment variable from context
	value, err := String(ctx, testKey).Value()
	require.NoError(t, err)
	assert.Equal(t, testValue, value)

	time.Sleep(10 * time.Millisecond)

	// Verify observer was called with correct source
	eventMu.Lock()
	assert.Equal(t, testKey, lastEvent.Key)
	assert.Equal(t, testValue, lastEvent.Value)
	assert.True(t, lastEvent.IsSet)
	assert.Equal(t, Context, lastEvent.Source)
	eventMu.Unlock()
}

// Test context override takes precedence over real environment variable.
func TestContextOverridePrecedence(t *testing.T) {
	// Clean up after test
	defer func() {
		EnableRecording(false)
		EnableStackTraces(false)

		_ = CollectRecordingEvents(true)
	}()

	// Enable recording
	EnableRecording(true)

	testKey := "TEST_PRECEDENCE_VAR"
	realValue := "real_value"
	contextValue := "context_value"

	// Set real environment variable
	t.Setenv(testKey, realValue)

	// Create context with override
	ctx := WithEnvOverride(t.Context(), testKey, contextValue)

	// Read from context - should get context value
	value, err := String(ctx, testKey).Value()
	require.NoError(t, err)
	assert.Equal(t, contextValue, value, "context override should take precedence")

	// Check recorded event
	events := CollectRecordingEvents(false)
	require.Len(t, events, 1)
	assert.Equal(t, Context, events[0].Source, "should record as Context source")
	assert.Equal(t, contextValue, events[0].Value)

	// Clear events
	_ = CollectRecordingEvents(true)

	// Read without context - should get real value
	value, err = String(t.Context(), testKey).Value()
	require.NoError(t, err)
	assert.Equal(t, realValue, value, "should get real env var without context")

	// Check recorded event
	events = CollectRecordingEvents(false)
	require.Len(t, events, 1)
	assert.Equal(t, Environment, events[0].Source, "should record as Environment source")
	assert.Equal(t, realValue, events[0].Value)
}

// Test recording with multiple context overrides.
//
//nolint:tparallel,paralleltest // Cannot use t.Parallel() due to global recording/observer state
func TestRecordingWithMultipleContextOverrides(t *testing.T) {
	// Clean up after test
	defer func() {
		EnableRecording(false)
		EnableStackTraces(false)

		_ = CollectRecordingEvents(true)
	}()

	// Enable recording
	EnableRecording(true)

	// Create context with multiple overrides
	overrides := map[string]string{
		"VAR1": "value1",
		"VAR2": "value2",
		"VAR3": "value3",
	}
	ctx := WithEnvOverrides(t.Context(), overrides)

	// Read all variables
	for key, expectedValue := range overrides {
		value, err := String(ctx, key).Value()
		require.NoError(t, err)
		assert.Equal(t, expectedValue, value)
	}

	// Check recorded events
	events := CollectRecordingEvents(false)
	require.Len(t, events, 3, "should have recorded three events")

	// Verify all events have Context source
	for _, event := range events {
		assert.Equal(t, Context, event.Source)
		assert.True(t, event.IsSet)
		assert.Equal(t, overrides[event.Key], event.Value)
	}
}

// Test observer called independently of recording flag.
func TestObserverIndependentOfRecordingFlag(t *testing.T) {
	// Clean up after test
	defer func() {
		EnableRecording(false)
		EnableStackTraces(false)

		_ = CollectRecordingEvents(true)
	}()

	// Track observer calls
	var callCount atomic.Int32

	unregister := RegisterObserver(func(event ValueReadEvent) {
		callCount.Add(1)
	})
	defer unregister()

	// Recording disabled - but observer should still be called
	EnableRecording(false)

	testKey := "TEST_OBSERVER_INDEPENDENT"
	t.Setenv(testKey, "value")

	// Read variable - observer SHOULD be called even with recording disabled
	_, _ = String(t.Context(), testKey).Value()

	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, int32(1), callCount.Load(), "observer should be called even when recording is disabled")

	// Verify no events were recorded
	events := CollectRecordingEvents(false)
	assert.Empty(t, events, "no events should be recorded when recording is disabled")

	// Enable recording
	EnableRecording(true)

	// Read variable again - observer should be called again AND event should be recorded
	_, _ = String(t.Context(), testKey).Value()

	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, int32(2), callCount.Load(), "observer should be called again")

	// Verify event was recorded this time
	events = CollectRecordingEvents(false)
	assert.Len(t, events, 1, "one event should be recorded when recording is enabled")
}

// Test both recording and observer together.
func TestRecordingAndObserverTogether(t *testing.T) {
	// Clean up after test
	defer func() {
		EnableRecording(false)
		EnableStackTraces(false)

		_ = CollectRecordingEvents(true)
	}()

	// Track observer calls
	var observedEvents []ValueReadEvent

	var eventMu sync.Mutex

	unregister := RegisterObserver(func(event ValueReadEvent) {
		eventMu.Lock()
		defer eventMu.Unlock()

		observedEvents = append(observedEvents, event)
	})
	defer unregister()

	// Enable recording
	EnableRecording(true)

	// Test 1: Real environment variable
	testKey1 := "TEST_BOTH_REAL"
	t.Setenv(testKey1, "real")
	_, _ = String(t.Context(), testKey1).Value()

	// Test 2: Context override
	testKey2 := "TEST_BOTH_CONTEXT"
	ctx := WithEnvOverride(t.Context(), testKey2, "context")
	_, _ = String(ctx, testKey2).Value()

	// Test 3: Unset variable
	testKey3 := "TEST_BOTH_UNSET"
	_ = os.Unsetenv(testKey3)
	_, _ = String(t.Context(), testKey3).Value()

	time.Sleep(20 * time.Millisecond)

	// Verify recorded events
	recordedEvents := CollectRecordingEvents(false)
	require.Len(t, recordedEvents, 3, "should have recorded three events")

	// Verify observed events
	eventMu.Lock()
	require.Len(t, observedEvents, 3, "should have observed three events")
	eventMu.Unlock()

	// Check that recorded and observed events match
	for i := range recordedEvents {
		eventMu.Lock()
		assert.Equal(t, recordedEvents[i].Key, observedEvents[i].Key)
		assert.Equal(t, recordedEvents[i].Value, observedEvents[i].Value)
		assert.Equal(t, recordedEvents[i].IsSet, observedEvents[i].IsSet)
		assert.Equal(t, recordedEvents[i].Source, observedEvents[i].Source)
		eventMu.Unlock()
	}

	// Verify sources
	assert.Equal(t, Environment, recordedEvents[0].Source)
	assert.Equal(t, Context, recordedEvents[1].Source)
	assert.Equal(t, None, recordedEvents[2].Source)
}

// Test recording with stack traces.
func TestRecordingWithStackTraces(t *testing.T) {
	// Clean up after test
	defer func() {
		EnableRecording(false)
		EnableStackTraces(false)

		_ = CollectRecordingEvents(true)
	}()

	// Enable recording with stack traces
	EnableRecording(true)
	EnableStackTraces(true)

	testKey := "TEST_STACK_TRACE"
	t.Setenv(testKey, "value")

	_, _ = String(t.Context(), testKey).Value()

	// Check recorded events have stack traces
	events := CollectRecordingEvents(false)
	require.Len(t, events, 1)
	assert.NotEmpty(t, events[0].Stack, "stack trace should be captured")
	assert.Contains(
		t, string(events[0].Stack), "TestRecordingWithStackTraces",
		"stack trace should include test function name",
	)
}

// Test CountRecordedEvents.
func TestCountRecordedEvents(t *testing.T) {
	// Clean up after test
	defer func() {
		EnableRecording(false)
		EnableStackTraces(false)

		_ = CollectRecordingEvents(true)
	}()

	// Enable recording
	EnableRecording(true)

	assert.Equal(t, 0, CountRecordedEvents(), "should start with 0 events")

	// Record some events
	testKey := "TEST_COUNT"
	t.Setenv(testKey, "value")

	for range 5 {
		_, _ = String(t.Context(), testKey).Value()
	}

	assert.Equal(t, 5, CountRecordedEvents(), "should have 5 recorded events")

	// Clear events
	_ = CollectRecordingEvents(true)

	assert.Equal(t, 0, CountRecordedEvents(), "should have 0 events after clearing")
}

// Test CollectRecordingEvents with and without clear.
func TestCollectRecordingEventsClear(t *testing.T) {
	// Clean up after test
	defer func() {
		EnableRecording(false)
		EnableStackTraces(false)

		_ = CollectRecordingEvents(true)
	}()

	// Enable recording
	EnableRecording(true)

	testKey := "TEST_COLLECT"
	t.Setenv(testKey, "value")

	// Record 3 events
	for range 3 {
		_, _ = String(t.Context(), testKey).Value()
	}

	// Collect without clearing
	events1 := CollectRecordingEvents(false)
	assert.Len(t, events1, 3)
	assert.Equal(t, 3, CountRecordedEvents(), "events should still be present")

	// Collect again without clearing
	events2 := CollectRecordingEvents(false)
	assert.Len(t, events2, 3)
	assert.Equal(t, 3, CountRecordedEvents(), "events should still be present")

	// Collect with clearing
	events3 := CollectRecordingEvents(true)
	assert.Len(t, events3, 3)
	assert.Equal(t, 0, CountRecordedEvents(), "events should be cleared")

	// Collect again - should be empty
	events4 := CollectRecordingEvents(false)
	assert.Empty(t, events4)
}

// Test recording when environment variable is explicitly set to empty string.
func TestRecordingWithEmptyStringValue(t *testing.T) {
	// Clean up after test
	defer func() {
		EnableRecording(false)
		EnableStackTraces(false)

		_ = CollectRecordingEvents(true)
	}()

	// Enable recording
	EnableRecording(true)

	// Set environment variable to empty string
	testKey := "TEST_EMPTY_STRING_VAR"
	t.Setenv(testKey, "")

	// Read the environment variable
	value, err := String(t.Context(), testKey).Value()
	require.NoError(t, err)
	assert.Empty(t, value)

	// Check recorded events
	events := CollectRecordingEvents(false)
	require.Len(t, events, 1)

	event := events[0]
	assert.Equal(t, testKey, event.Key)
	assert.Empty(t, event.Value, "value should be empty string")
	assert.True(t, event.IsSet, "IsSet should be true because var was explicitly set to empty")
	assert.Equal(t, Environment, event.Source)
}

// Test observer when environment variable is not set.
//
//nolint:tparallel,paralleltest // Cannot use t.Parallel() due to global recording/observer state
func TestObserverWithUnsetVar(t *testing.T) {
	// Clean up after test
	defer func() {
		EnableRecording(false)
		EnableStackTraces(false)

		_ = CollectRecordingEvents(true)
	}()

	// Track observer calls
	var lastEvent ValueReadEvent

	var eventMu sync.Mutex

	unregister := RegisterObserver(func(event ValueReadEvent) {
		eventMu.Lock()
		defer eventMu.Unlock()

		lastEvent = event
	})
	defer unregister()

	// Enable recording
	EnableRecording(true)

	// Read a non-existent environment variable
	testKey := "TEST_OBSERVER_UNSET_VAR_99999"
	_ = os.Unsetenv(testKey)

	_, err := String(t.Context(), testKey).Value()
	require.Error(t, err)

	time.Sleep(10 * time.Millisecond)

	// Verify observer was called with unset variable info
	eventMu.Lock()
	assert.Equal(t, testKey, lastEvent.Key)
	assert.Empty(t, lastEvent.Value)
	assert.False(t, lastEvent.IsSet)
	assert.Equal(t, None, lastEvent.Source)
	eventMu.Unlock()
}

// Test observer with empty string context override.
//
//nolint:tparallel,paralleltest // Cannot use t.Parallel() due to global recording/observer state
func TestObserverWithEmptyStringContextOverride(t *testing.T) {
	// Clean up after test
	defer func() {
		EnableRecording(false)
		EnableStackTraces(false)

		_ = CollectRecordingEvents(true)
	}()

	// Track observer calls
	var lastEvent ValueReadEvent

	var eventMu sync.Mutex

	unregister := RegisterObserver(func(event ValueReadEvent) {
		eventMu.Lock()
		defer eventMu.Unlock()

		lastEvent = event
	})
	defer unregister()

	// Enable recording
	EnableRecording(true)

	// Create context with empty string override
	testKey := "TEST_OBSERVER_EMPTY_CONTEXT"
	ctx := WithEnvOverride(t.Context(), testKey, "")

	// Read the environment variable from context
	value, err := String(ctx, testKey).Value()
	require.NoError(t, err)
	assert.Empty(t, value)

	time.Sleep(10 * time.Millisecond)

	// Verify observer was called with empty value but IsSet=true
	eventMu.Lock()
	assert.Equal(t, testKey, lastEvent.Key)
	assert.Empty(t, lastEvent.Value)
	assert.True(t, lastEvent.IsSet, "should be set even with empty value")
	assert.Equal(t, Context, lastEvent.Source)
	eventMu.Unlock()
}

// Test recording multiple unset variables.
//
//nolint:tparallel,paralleltest // Cannot use t.Parallel() due to global recording/observer state
func TestRecordingMultipleUnsetVariables(t *testing.T) {
	// Clean up after test
	defer func() {
		EnableRecording(false)
		EnableStackTraces(false)

		_ = CollectRecordingEvents(true)
	}()

	// Enable recording
	EnableRecording(true)

	// Try to read multiple non-existent variables
	unsetKeys := []string{
		"UNSET_VAR_1",
		"UNSET_VAR_2",
		"UNSET_VAR_3",
	}

	for _, key := range unsetKeys {
		_ = os.Unsetenv(key)
		_, _ = String(t.Context(), key).Value()
	}

	// Check recorded events
	events := CollectRecordingEvents(false)
	require.Len(t, events, 3)

	// All should have None source and IsSet=false
	for i, event := range events {
		assert.Equal(t, unsetKeys[i], event.Key)
		assert.Empty(t, event.Value)
		assert.False(t, event.IsSet)
		assert.Equal(t, None, event.Source)
	}
}

// Test mixed set and unset variables.
//
//nolint:tparallel,paralleltest // Cannot use t.Parallel() due to global recording/observer state
func TestRecordingMixedSetAndUnset(t *testing.T) {
	// Clean up after test
	defer func() {
		EnableRecording(false)
		EnableStackTraces(false)

		_ = CollectRecordingEvents(true)
	}()

	// Enable recording
	EnableRecording(true)

	// Set some variables
	t.Setenv("SET_VAR_1", "value1")
	t.Setenv("SET_VAR_2", "")

	_ = os.Unsetenv("UNSET_VAR_1")

	ctx := WithEnvOverride(t.Context(), "CONTEXT_VAR", "contextValue")

	// Read all variables
	_, _ = String(t.Context(), "SET_VAR_1").Value()
	_, _ = String(t.Context(), "SET_VAR_2").Value()
	_, _ = String(t.Context(), "UNSET_VAR_1").Value()
	_, _ = String(ctx, "CONTEXT_VAR").Value()

	// Check recorded events
	events := CollectRecordingEvents(false)
	require.Len(t, events, 4)

	// Verify first event - set with value
	assert.Equal(t, "SET_VAR_1", events[0].Key)
	assert.Equal(t, "value1", events[0].Value)
	assert.True(t, events[0].IsSet)
	assert.Equal(t, Environment, events[0].Source)

	// Verify second event - set but empty
	assert.Equal(t, "SET_VAR_2", events[1].Key)
	assert.Empty(t, events[1].Value)
	assert.True(t, events[1].IsSet)
	assert.Equal(t, Environment, events[1].Source)

	// Verify third event - not set
	assert.Equal(t, "UNSET_VAR_1", events[2].Key)
	assert.Empty(t, events[2].Value)
	assert.False(t, events[2].IsSet)
	assert.Equal(t, None, events[2].Source)

	// Verify fourth event - context override
	assert.Equal(t, "CONTEXT_VAR", events[3].Key)
	assert.Equal(t, "contextValue", events[3].Value)
	assert.True(t, events[3].IsSet)
	assert.Equal(t, Context, events[3].Source)
}
