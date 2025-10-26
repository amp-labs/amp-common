package future

import (
	"context"

	"github.com/amp-labs/amp-common/logger"
)

// Async runs the given function asynchronously in a goroutine without blocking.
// This is a fire-and-forget operation - the caller does not wait for completion
// or receive a result. Any panics that occur during execution are recovered and
// logged as errors using the default logger.
//
// Use this when you want to perform work in the background without needing to
// track its completion or handle results. For context-aware operations, use AsyncContext.
func Async(f func()) {
	fut := Go[struct{}](func() (struct{}, error) {
		f()

		return struct{}{}, nil
	})

	// Register error handler to log any panics or errors
	fut.OnError(func(err error) {
		logger.Get().Error("future.Async", "error", err)
	})
}

// AsyncWithError runs the given function asynchronously in a goroutine without blocking,
// allowing the function to return an error. This is a fire-and-forget operation - the caller
// does not wait for completion or receive a result. Any errors returned by the function or
// panics that occur during execution are recovered and logged as errors using the default logger.
//
// Use this when you want to perform background work that may fail and you want errors logged,
// but you don't need to handle the result. For context-aware operations, use AsyncContextWithError.
func AsyncWithError(f func() error) {
	fut := Go[struct{}](func() (struct{}, error) {
		err := f()

		return struct{}{}, err
	})

	// Register error handler to log any panics or errors
	fut.OnError(func(err error) {
		logger.Get().Error("future.Async", "error", err)
	})
}

// AsyncContext runs the given function asynchronously in a goroutine without blocking,
// with support for context cancellation. This is a fire-and-forget operation - the caller
// does not wait for completion or receive a result. Any panics that occur during execution
// are recovered and logged as errors using the default logger.
//
// The provided context can be used to cancel the async operation or propagate deadlines
// and values. If the context is canceled, the function may terminate early depending on
// whether it respects context cancellation.
//
// Use this when you want to perform background work that should respect cancellation signals.
// For simple async operations without context, use Async.
func AsyncContext(ctx context.Context, f func(ctx context.Context)) {
	fut := GoContext[struct{}](ctx, func(ctx context.Context) (struct{}, error) {
		f(ctx)

		return struct{}{}, nil
	})

	// Register error handler to log any panics or errors
	fut.OnError(func(err error) {
		logger.Get(ctx).Error("future.AsyncContext", "error", err)
	})
}

// AsyncContextWithError runs the given function asynchronously in a goroutine without blocking,
// with support for context cancellation and allowing the function to return an error. This is a
// fire-and-forget operation - the caller does not wait for completion or receive a result. Any
// errors returned by the function or panics that occur during execution are recovered and logged
// as errors using the context-aware logger.
//
// The provided context can be used to cancel the async operation or propagate deadlines
// and values. If the context is canceled, the function may terminate early depending on
// whether it respects context cancellation.
//
// Use this when you want to perform background work that should respect cancellation signals
// and may fail. For simple async operations without context, use AsyncWithError.
func AsyncContextWithError(ctx context.Context, f func(ctx context.Context) error) {
	fut := GoContext[struct{}](ctx, func(ctx context.Context) (struct{}, error) {
		err := f(ctx)

		return struct{}{}, err
	})

	// Register error handler to log any panics or errors
	fut.OnError(func(err error) {
		logger.Get(ctx).Error("future.AsyncContext", "error", err)
	})
}
