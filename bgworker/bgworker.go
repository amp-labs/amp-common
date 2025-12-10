// Package bgworker provides background worker management with graceful lifecycle control.
package bgworker

import (
	"context"
	"log/slog"

	"github.com/alitto/pond/v2"
	"github.com/amp-labs/amp-common/envutil"
	"github.com/amp-labs/amp-common/lazy"
	"github.com/amp-labs/amp-common/shutdown"
)

const defaultWorkerCount = 10

// workerPool is a lazy-initialized background worker pool.
var workerPool = lazy.NewCtx[pond.Pool](func(ctx context.Context) pond.Pool {
	count := envutil.Int[int](ctx, "BACKGROUND_WORKER_COUNT",
		envutil.Default[int](defaultWorkerCount)).ValueOrElse(defaultWorkerCount)

	slog.Debug("Initializing background worker pool", "count", count)

	pool := pond.NewPool(count)

	shutdown.BeforeShutdown(func() {
		slog.Debug("Stopping background worker pool")
		pool.StopAndWait()
		slog.Debug("Background worker pool stopped")
	})

	return pool
})

// Submit submits a function to the background worker pool.
// It returns a Task that can be used to wait for the function to complete.
func Submit(ctx context.Context, f func()) pond.Task { //nolint:ireturn
	return workerPool.Get(ctx).Submit(f)
}

// Go submits a function to the background worker pool. It returns immediately.
// It returns an error if the pool is stopped.
func Go(ctx context.Context, f func()) error {
	return workerPool.Get(ctx).Go(f)
}
