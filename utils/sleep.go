package utils

import (
	"context"
	"time"
)

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
