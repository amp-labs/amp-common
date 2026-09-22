package simultaneously

import (
	"context"
	"sync"
)

// limitedExecutor caps how many of its own callbacks can be in flight at once
// while delegating the actual execution to another executor.
//
// It exists so that a call running on a shared ambient executor (see
// WithExecutor) still honors its own maxConcurrent. The shared executor bounds
// the process-wide total; this bounds the one call. Both limits apply, so the
// effective concurrency of the call is the smaller of the two.
//
// A slot here is taken before the callback is handed to the inner executor, so
// a callback waiting on this limit does not occupy a slot in the shared pool.
type limitedExecutor struct {
	inner Executor
	sem   chan struct{} // counting semaphore: a send acquires a slot, a receive releases it
}

// newLimitedExecutor wraps inner so that at most maxConcurrent callbacks
// submitted through the wrapper are in flight at once. maxConcurrent must be at
// least 1.
func newLimitedExecutor(inner Executor, maxConcurrent int) *limitedExecutor {
	return &limitedExecutor{
		inner: inner,
		sem:   make(chan struct{}, maxConcurrent),
	}
}

func (l *limitedExecutor) Go(fn func(context.Context) error, done func(error)) {
	l.GoContext(context.Background(), fn, done)
}

// GoContext waits for a local slot (or for ctx to end), then submits callback to the
// inner executor. The slot is released when the inner executor reports the
// result, just before done is called, so the next waiting callback can proceed.
func (l *limitedExecutor) GoContext(ctx context.Context, callback func(context.Context) error, done func(error)) {
	select {
	case <-ctx.Done():
		done(ctx.Err())

		return
	case l.sem <- struct{}{}:
	}

	var releaseOnce sync.Once

	release := func() {
		releaseOnce.Do(func() { <-l.sem })
	}

	// If the inner executor panics instead of reporting a result, give the slot
	// back before the panic continues, so the remaining callbacks of this call
	// are not starved. The collector recovers the panic and reports it.
	defer func() {
		if r := recover(); r != nil {
			release()

			panic(r)
		}
	}()

	l.inner.GoContext(ctx, callback, func(err error) {
		release()
		done(err)
	})
}

// Close is a no-op. The wrapper is scoped to a single call and does not own the
// inner executor, which belongs to whoever attached it to the context.
func (l *limitedExecutor) Close() error {
	return nil
}
