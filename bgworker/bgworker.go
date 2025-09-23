package bgworker

import (
	"log/slog"

	"github.com/alitto/pond/v2"
	"github.com/amp-labs/amp-common/envutil"
	"github.com/amp-labs/amp-common/lazy"
	"github.com/amp-labs/amp-common/shutdown"
)

const defaultWorkerCount = 10

// workerPool is a lazy-initialized background worker pool.
var workerPool = lazy.New[pond.Pool](func() pond.Pool {
	count := envutil.Int[int]("BACKGROUND_WORKER_COUNT",
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
func Submit(f func()) pond.Task { //nolint:ireturn
	return workerPool.Get().Submit(f)
}

// Go submits a function to the background worker pool. It returns immediately.
// It returns an error if the pool is stopped.
func Go(f func()) error {
	return workerPool.Get().Go(f)
}
