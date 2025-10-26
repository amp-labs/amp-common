// Package future provides callback invocation utilities for Future callbacks.
package future

import (
	"context"
	"runtime/debug"

	"github.com/amp-labs/amp-common/logger"
	"github.com/amp-labs/amp-common/utils"
)

// invokeCallback invokes a callback in a separate goroutine with panic recovery and logging.
//
// This is the internal helper used by OnSuccess, OnError, and OnResult to safely invoke
// user-provided callbacks. It handles all the complexity of asynchronous callback execution:
//
// Safety guarantees:
//   - Nil callbacks are safely ignored without error
//   - Panics in callbacks are recovered and logged, preventing crashes
//   - Stack traces are captured for debugging panic sources
//   - Execution happens in a goroutine to avoid blocking the caller
//
// Parameters:
//   - kind: The callback type ("OnSuccess", "OnError", "OnResult") for logging
//   - callback: The user-provided callback function to invoke
//   - value: The value to pass to the callback
//
// Design notes:
//   - The goroutine ensures callbacks don't block promise fulfillment
//   - Panic recovery uses utils.GetPanicRecoveryError for consistent error formatting
//   - Logging uses the amp-common logger for observability
//   - The kind parameter helps identify which callback type panicked
//
// This function is intentionally unexported - callers should use OnSuccess/OnError/OnResult.
func invokeCallback[T any](kind string, callback func(T), value T) {
	if callback == nil {
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				if err := utils.GetPanicRecoveryError(r, debug.Stack()); err != nil {
					logger.Get().Error("panic encountered in future."+kind+" callback", "error", err)
				}
			}
		}()

		callback(value)
	}()
}

// invokeCallbackContext invokes a context-aware callback in a separate goroutine with panic recovery.
//
// This is the context-aware version of invokeCallback, used by OnSuccessContext, OnErrorContext,
// and OnResultContext to safely invoke user-provided callbacks that need a context parameter.
//
// Safety guarantees:
//   - Nil callbacks are safely ignored without error
//   - Nil contexts are replaced with context.Background() to prevent panics
//   - Creates a child context that is canceled when the callback completes (prevents leaks)
//   - Panics in callbacks are recovered and logged, preventing crashes
//   - Stack traces are captured for debugging panic sources
//   - Execution happens in a goroutine to avoid blocking the caller
//
// Parameters:
//   - ctx: The context to pass to the callback (nil is replaced with Background)
//   - kind: The callback type ("OnSuccessContext", "OnErrorContext", "OnResultContext") for logging
//   - callback: The user-provided callback function to invoke
//   - value: The value to pass to the callback
//
// Design notes:
//   - The child context (cctx) ensures the callback has a cancellable context
//   - The cancel is deferred to execute even if callback panics (cleanup guarantee)
//   - The goroutine ensures callbacks don't block promise fulfillment
//   - Panic recovery uses utils.GetPanicRecoveryError for consistent error formatting
//   - Logging uses the amp-common logger with context for observability
//
// This function is intentionally unexported - callers should use OnSuccessContext/OnErrorContext/OnResultContext.
func invokeCallbackContext[T any](ctx context.Context, kind string, callback func(context.Context, T), value T) {
	if callback == nil {
		return
	}

	go func() {
		if ctx == nil {
			ctx = context.Background()
		}

		cctx, cancel := context.WithCancel(ctx)
		defer cancel()

		defer func() {
			if r := recover(); r != nil {
				if err := utils.GetPanicRecoveryError(r, debug.Stack()); err != nil {
					logger.Get(cctx).Error("panic encountered in future."+kind+" callback", "error", err)
				}
			}
		}()

		callback(cctx, value)
	}()
}
