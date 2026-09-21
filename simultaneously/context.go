package simultaneously

import (
	"context"

	"github.com/amp-labs/amp-common/contexts"
	"github.com/amp-labs/amp-common/utils"
)

// contextKey is a package-private key type for context values. Having a
// dedicated unexported type means our key can never collide with a key stored
// by another package, even if that package uses the same string.
type contextKey string

// executorContextKey is the sole key under which an ambient Executor is stored.
const executorContextKey contextKey = "executor"

// WithExecutor returns a copy of ctx carrying exec as the ambient executor.
//
// This lets a caller high up the stack choose the concurrency pool that the
// work below it will run on, without every intermediate function having to
// thread an Executor parameter through its signature. Every context-taking
// entry point in this package -- DoCtx and the whole MapXCtx/FlatMapXCtx
// family -- runs its work on exec instead of spinning up a throwaway executor
// when it receives the returned context, directly or via further derivation.
//
//	exec := simultaneously.NewDefaultExecutor(8)
//	defer exec.Close()
//
//	ctx = simultaneously.WithExecutor(ctx, exec)
//	// ...many layers later...
//	err := simultaneously.DoCtx(ctx, 0, jobs...) // runs on exec
//
// Three things to be aware of:
//
//  1. Ownership does not transfer. DoCtx never closes an executor it found on
//     the context, so whoever called WithExecutor remains responsible for
//     closing it. That is what makes the executor reusable across many DoCtx
//     calls, which is the point of putting it on the context.
//
//  2. The maxConcurrent argument to DoCtx is ignored once an executor is on the
//     context, because the executor already has a concurrency limit of its own
//     and that limit is shared by every caller using it. Set the limit you want
//     when you construct the executor.
//
//  3. Nested calls share the pool, and can deadlock. A job or transform started
//     here receives a context that still carries exec, so any DoCtx or MapXCtx
//     call made from inside one will queue onto the same executor that is
//     currently occupied by its caller. If the outer work holds every slot, the
//     inner call blocks waiting for a slot that cannot free up until it returns.
//     Only context cancellation breaks the cycle. If your work fans out further,
//     either size the executor above the peak depth of the nesting, or hand the
//     inner level its own executor with WithExecutor.
//
// Only the context-taking entry points consult the context. The Do, MapX and
// FlatMapX variants that take no context start from context.Background() and so
// never see an ambient executor, and the *WithExecutor variants use the executor
// they were handed.
func WithExecutor(ctx context.Context, exec Executor) context.Context {
	return contexts.WithValue[contextKey, Executor](ctx, executorContextKey, exec)
}

// GetExecutor returns the ambient executor attached to ctx by WithExecutor,
// reporting whether one was found.
//
// A false result means "no executor here, carry on" rather than an error: every
// caller is expected to fall back to creating its own. That is why a nil ctx is
// treated as simply having no executor instead of panicking, which keeps DoCtx
// usable from code paths that have not been fully context-plumbed yet.
func GetExecutor(ctx context.Context) (Executor, bool) {
	if ctx == nil {
		return nil, false
	}

	val, ok := contexts.GetValue[contextKey, Executor](ctx, executorContextKey)
	if !ok {
		return nil, false
	}

	if utils.IsNilish(val) {
		return nil, false
	}

	return val, true
}

// resolveExecutor picks the executor an entry point should run its work on, and
// returns the closer that matches that choice.
//
// This is the single place where the "ambient executor wins" rule lives, so that
// every Do/Map/FlatMap entry point applies it identically. Two cases:
//
//   - The context carries an executor. It is shared with other callers and
//     outlives this call, so maxConcurrent is ignored (the executor's own limit
//     governs) and the returned closer does nothing -- closing it here would pull
//     the pool out from under whoever attached it.
//
//   - It does not. We build one scoped to this call, sized to the smaller of
//     maxConcurrent and itemCount, and hand back its Close so the caller can shut
//     it down on the way out.
//
// Either way the caller defers the returned closer and reports its error, which
// keeps the two paths indistinguishable at the call site.
func resolveExecutor(ctx context.Context, maxConcurrent, itemCount int) (Executor, func() error) {
	if exec, ok := GetExecutor(ctx); ok {
		return exec, func() error { return nil }
	}

	exec := newDefaultExecutor(maxConcurrent, itemCount)

	return exec, exec.Close
}
