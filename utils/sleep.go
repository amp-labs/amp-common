package utils

import (
	"context"
	"time"
)

// SleepCtx sleeps for the specified duration or until the context is canceled.
// Returns nil if the sleep completes successfully.
// Returns ctx.Err() if the context is canceled before the duration elapses.
// Returns immediately without error if dur <= 0.
func SleepCtx(ctx context.Context, dur time.Duration) error {
	if dur <= 0 {
		return nil
	}

	timer := time.NewTimer(dur)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
