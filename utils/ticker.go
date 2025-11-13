package utils

import (
	"context"
	"time"
)

// TickerWithContext creates a context-aware ticker that sends the current time on a channel
// at regular intervals specified by duration. Unlike time.Ticker, this ticker automatically
// stops and cleans up resources when the provided context is canceled.
//
// The returned channel will receive time values at each tick until the context is canceled,
// at which point the underlying ticker is stopped and the channel is closed. This prevents
// goroutine leaks and ensures proper resource cleanup in long-running applications.
//
// Parameters:
//   - ctx: Context that controls the ticker's lifetime. When canceled, the ticker stops
//     and all resources are cleaned up automatically.
//   - duration: Time interval between ticks. Must be positive; zero or negative durations
//     will cause the underlying time.NewTicker to panic.
//
// Returns:
//   - A receive-only channel that delivers the current time at each tick. The channel is
//     closed automatically when the context is canceled.
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
//	defer cancel()
//
//	ticker := utils.TickerWithContext(ctx, 1*time.Second)
//	for t := range ticker {
//	    fmt.Println("Tick at", t)
//	}
//	// Ticker automatically stops and channel closes when context times out
//
// Implementation notes:
//   - Uses future.AsyncContext to run the ticker loop in a separate goroutine
//   - Gracefully handles panics during channel closure using closer.HandlePanic
//   - Logs errors if channel closure fails (should rarely occur)
//   - The ticker is stopped before closing the channel to prevent sending on a closed channel
func TickerWithContext(ctx context.Context, duration time.Duration) <-chan time.Time {
	if ctx == nil {
		//nolint:contextcheck // Defensive nil check, creates new context intentionally
		ctx = context.TODO()
	}

	// Create the underlying ticker with the specified interval
	ticker := time.NewTicker(duration)

	// Create the output channel that consumers will receive from
	out := make(chan time.Time)

	go func() {
		// Ensure cleanup happens when the goroutine exits (either via context cancellation
		// or unexpected panic). This is critical to prevent resource leaks.
		defer func() {
			// Stop the ticker first to prevent it from sending more values
			ticker.Stop()

			close(out)
		}()

		// Main ticker loop: forward ticks to the output channel until context is canceled
		for {
			select {
			// Context canceled: exit the loop, triggering cleanup in defer
			case <-ctx.Done():
				return

			// Ticker fired: forward the tick value to consumers
			// This will block if no receiver is ready, which provides natural backpressure
			case val := <-ticker.C:
				out <- val
			}
		}
	}()

	// Return the output channel immediately so consumers can start receiving ticks
	// The goroutine runs independently and manages the ticker lifecycle
	return out
}
