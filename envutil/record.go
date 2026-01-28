package envutil

// This file implements a recording mechanism that tracks all environment variable
// read operations for debugging, auditing, and testing purposes.

import (
	"sync"
	"time"

	"go.uber.org/atomic"
)

// Source indicates where an environment variable value originated from.
// This helps distinguish between different sources of configuration values.
type Source string

const (
	// None indicates the environment variable was not set in any source.
	None Source = "none"

	// Environment indicates the value came from the operating system environment.
	Environment Source = "environment"

	// Context indicates the value came from a context.Context value.
	Context Source = "context"

	// File indicates that the value came from an external file.
	File Source = "file"
)

var (
	// recording tracks whether environment variable reads should be recorded.
	// Uses atomic.Bool for thread-safe access without locks.
	recording *atomic.Bool

	// wantStacks tracks whether stack traces should be captured with each event.
	// Stack traces are useful for debugging but add overhead.
	wantStacks *atomic.Bool

	// hasObservers tracks whether any observers are registered.
	// Used to optimize observer notification by avoiding overhead when no observers exist.
	// Automatically set to true when observers are registered and false when all are removed.
	hasObservers *atomic.Bool

	// dedupKeys tracks whether to deduplicate keys in recording.
	// When enabled, only the first read of each key is recorded.
	dedupKeys *atomic.Bool

	// seenKeys tracks which environment variable keys have been recorded.
	// Only used when dedupKeys is enabled. Protected by seenKeysMutex.
	seenKeys map[string]struct{}

	// seenKeysMutex protects concurrent access to the seenKeys map.
	seenKeysMutex sync.RWMutex

	// eventMutex protects concurrent access to the events slice.
	eventMutex sync.Mutex

	// events stores all recorded environment variable read events.
	// Protected by eventMutex to ensure thread-safe append operations.
	events []ValueReadEvent

	// observerMutex protects concurrent access to the observers slice.
	// Uses RWMutex to allow concurrent notification while serializing registration.
	observerMutex sync.RWMutex

	// observers stores all registered observer entries.
	// Protected by observerMutex to ensure thread-safe access.
	observers []observerEntry

	// nextObserverID generates unique IDs for observer registration.
	// Allows safe unregistration by ID rather than function pointer comparison.
	nextObserverID atomic.Int64
)

func init() {
	// Initialize atomic booleans with recording disabled by default.
	recording = atomic.NewBool(false)
	wantStacks = atomic.NewBool(false)
	hasObservers = atomic.NewBool(false)
	dedupKeys = atomic.NewBool(false)
	seenKeys = make(map[string]struct{})
}

// Observer is a callback function that gets invoked immediately when an
// environment variable is read (if recording is enabled).
// The function receives a ValueReadEvent containing details about the read operation.
//
// Observers are called synchronously during the environment variable read,
// so they should execute quickly to avoid blocking the caller.
type Observer func(ValueReadEvent)

// observerEntry wraps an Observer function with a unique ID for safe unregistration.
// The ID allows observers to be removed by identity rather than by function pointer comparison,
// which is not reliable in Go (function equality is not defined).
type observerEntry struct {
	// id is the unique identifier for this observer, used for unregistration.
	id int64
	// fn is the observer callback function to invoke on environment variable reads.
	fn Observer
}

// ValueReadEvent represents a single environment variable read operation.
// It captures the key, value, source, timestamp, and optionally the call stack.
type ValueReadEvent struct {
	// Time is when the environment variable was read.
	Time time.Time `json:"time"`

	// Key is the environment variable name (e.g., "PORT", "DATABASE_URL").
	Key string `json:"key"`

	// Value is the raw string value of the environment variable.
	// Omitted from JSON if empty.
	Value string `json:"value,omitempty"`

	// IsSet indicates whether the environment variable was actually set.
	// False means the variable was not found and a default may have been used.
	IsSet bool `json:"is_set"`

	// Source indicates where the value came from (environment, context, or none).
	Source Source `json:"source"`

	// Stack contains the call stack trace showing where the read occurred.
	// Only populated if stack traces are enabled via EnableRecording.
	// Omitted from JSON if empty.
	Stack []byte `json:"stack,omitempty"`
}

// EnableStackTraces controls whether stack traces are captured for each environment variable read.
// Stack traces are useful for debugging to see where reads occur, but they add performance overhead.
//
// This setting only affects reads when recording is enabled via EnableRecording.
// Can be toggled independently at runtime without affecting the recording state.
func EnableStackTraces(enable bool) {
	wantStacks.Store(enable)
}

// EnableDedupKeys controls whether duplicate keys are filtered from recording.
// When enabled, only the first read of each environment variable key is recorded.
// Subsequent reads of the same key are silently ignored.
//
// This is useful for reducing noise in audit logs when the same environment
// variables are read multiple times during application startup.
//
// This setting only affects recording - observers are still notified for all reads.
func EnableDedupKeys(enable bool) {
	dedupKeys.Store(enable)

	if enable {
		// Clear the seen keys map when enabling dedup to start fresh
		seenKeysMutex.Lock()

		seenKeys = make(map[string]struct{})

		seenKeysMutex.Unlock()
	}
}

// EnableRecording controls whether environment variable reads are recorded.
// When enabled, each read operation creates a ValueReadEvent that can be retrieved later.
//
// Parameters:
//   - enable: Whether to record environment variable reads.
//   - includeStacks: Whether to capture stack traces for each read (adds overhead).
//
// This is typically used in testing or debugging to track which environment
// variables are accessed and where in the code they're read from.
func EnableRecording(enable bool) {
	recording.Store(enable)
}

// IsRecording returns whether environment variable recording is currently enabled.
// Thread-safe - uses atomic load operation.
func IsRecording() bool {
	return recording.Load()
}

// CountRecordedEvents returns the number of recorded environment variable read events.
// Thread-safe - acquires mutex to safely read the events slice length.
func CountRecordedEvents() int {
	eventMutex.Lock()
	defer eventMutex.Unlock()

	return len(events)
}

// CollectRecordingEvents returns a copy of all recorded environment variable events.
// Optionally clears the internal event buffer after copying.
//
// Parameters:
//   - shouldClear: If true, the internal events buffer is cleared after copying.
//
// Returns a new slice containing copies of all recorded events.
// Thread-safe - acquires mutex during the copy operation.
//
// Example usage:
//
//	EnableRecording(true, false)
//	// ... code that reads environment variables ...
//	events := CollectRecordingEvents(true) // Get events and clear buffer
//	for _, event := range events {
//	    fmt.Printf("%s: %s from %s\n", event.Key, event.Value, event.Source)
//	}
func CollectRecordingEvents(shouldClear bool) []ValueReadEvent {
	eventMutex.Lock()
	defer eventMutex.Unlock()

	out := make([]ValueReadEvent, len(events))
	copy(out, events)

	if shouldClear {
		events = nil
	}

	return out
}

// RegisterObserver registers a callback function that will be invoked immediately
// whenever an environment variable is read (if recording is enabled).
//
// Observers are called synchronously during environment variable reads, so they
// should execute quickly to avoid blocking. For expensive operations, consider
// using a buffered channel or goroutine within your observer.
//
// Returns an unregister function that removes the observer when called.
// The unregister function is safe to call multiple times.
//
// Example usage:
//
//	unregister := RegisterObserver(func(event ValueReadEvent) {
//	    log.Printf("Read %s=%s from %s", event.Key, event.Value, event.Source)
//	})
//	defer unregister() // Clean up when done
//
//	EnableRecording(true, false)
//	// ... code that reads environment variables ...
//	// Observer is called immediately for each read
func RegisterObserver(obs Observer) func() {
	// Generate unique ID for this observer
	observerID := nextObserverID.Add(1)

	// Register the observer
	observerMutex.Lock()

	observers = append(observers, observerEntry{id: observerID, fn: obs})

	hasObservers.Store(true)

	observerMutex.Unlock()

	// Return unregister function
	unregistered := false

	return func() {
		if unregistered {
			return
		}

		unregistered = true

		observerMutex.Lock()
		defer observerMutex.Unlock()

		// Find and remove this observer by ID
		for i, entry := range observers {
			if entry.id == observerID {
				observers = append(observers[:i], observers[i+1:]...)

				break
			}
		}

		// Update hasObservers flag if no observers remain
		if len(observers) == 0 {
			hasObservers.Store(false)
		}
	}
}

// shouldRecordKey checks if a key should be recorded based on deduplication settings.
// Returns true if the key should be recorded, false if it should be skipped.
// When deduplication is enabled, this function tracks seen keys and only returns
// true for the first occurrence of each key.
func shouldRecordKey(key string) bool {
	if !dedupKeys.Load() {
		return true
	}

	seenKeysMutex.RLock()

	_, seen := seenKeys[key]

	seenKeysMutex.RUnlock()

	if seen {
		// Key already recorded, skip this event
		return false
	}

	// Mark key as seen
	seenKeysMutex.Lock()

	seenKeys[key] = struct{}{}

	seenKeysMutex.Unlock()

	return true
}

// notifyObservers calls all registered observers with the given event.
// This is an internal function called during environment variable reads.
// Uses read lock to allow concurrent notifications while observers are being called.
// Returns early if no observers are registered (performance optimization).
func notifyObservers(event ValueReadEvent) {
	// Fast path: return early if no observers are registered
	if !hasObservers.Load() {
		return
	}

	// Copy observers slice under read lock to minimize lock contention
	observerMutex.RLock()

	obs := make([]observerEntry, len(observers))
	copy(obs, observers)

	observerMutex.RUnlock()

	// Call each observer without holding the lock
	// This allows observers to register/unregister other observers if needed
	for _, entry := range obs {
		entry.fn(event)
	}
}
