// Package shutdown provides utilities for graceful shutdown coordination.
// It allows registering cleanup hooks and sets up signal handlers for SIGINT and SIGTERM.
//
// Typical usage:
//
//	ctx := shutdown.SetupHandler()
//	shutdown.BeforeShutdown(func() {
//	    // cleanup logic
//	})
//	// ... run application with ctx
package shutdown

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/amp-labs/amp-common/envutil"
)

// DrainTimeoutEnvVar is the environment variable that controls how long
// SetupDrainable will wait between Intake cancellation (SIGTERM) and an
// explicit DrainComplete call before forcing the drain to complete. The
// value is parsed by time.ParseDuration (e.g. "5m", "30s", "1h").
const DrainTimeoutEnvVar = "SHUTDOWN_DRAIN_TIMEOUT"

// DefaultDrainTimeout is used when DrainTimeoutEnvVar is unset or invalid.
const DefaultDrainTimeout = 5 * time.Minute

var (
	mut     sync.Mutex     //nolint:gochecknoglobals // Protects hooks slice
	hooks   []func()       //nolint:gochecknoglobals // Cleanup hooks to run before shutdown
	channel chan os.Signal //nolint:gochecknoglobals // Signal channel for shutdown coordination
)

// BeforeShutdown registers a function to be called before
// the shutdown process begins. The top-level context will
// still be alive at this point, so you can use it to clean
// up resources if needed.
func BeforeShutdown(h func()) {
	mut.Lock()
	defer mut.Unlock()

	hooks = append(hooks, h)
}

// Shutdown triggers the shutdown process. Usually the
// shutdown is kicked off by a signal handler, but this
// function can be used to trigger it programmatically.
func Shutdown() {
	if channel != nil {
		channel <- os.Interrupt
	}
}

// SetupHandler sets up a signal handler for SIGINT and SIGTERM
// and returns a context that will be canceled when the
// signal is received. You can use this context to clean up
// resources before the process exits.
//
// The returned context is canceled after all registered
// BeforeShutdown hooks have been executed.
func SetupHandler() context.Context {
	channel = make(chan os.Signal, 1)
	signal.Notify(channel, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		for c := range channel {
			slog.Warn("Received " + c.String() + ", shutting down...")
			close(channel)

			channel = nil

			cleanup()
			cancel()
		}
	}()

	return ctx
}

// DrainableContexts holds the two contexts that govern a drainable process
// lifecycle, plus the callback that transitions from "draining" to "done".
//
// Intake represents the scope during which the process is accepting new work.
// It is canceled on SIGINT/SIGTERM. Wire it into anything that pulls or
// receives new work (e.g. a Pub/Sub subscriber, an HTTP listener) so that
// cancellation translates into "stop accepting new work".
//
// Lifetime represents the scope during which the process is alive at all.
// It is canceled when DrainComplete is called. Wire it into supporting
// services that must outlive the drain — telemetry, database connections,
// metrics servers, in-flight handlers — so they remain available while
// queued work finishes.
//
// DrainComplete must be called by the caller once all in-flight work has
// finished. It runs registered BeforeShutdown hooks and then cancels
// Lifetime, allowing the rest of the process to tear down.
type DrainableContexts struct {
	Intake   context.Context //nolint:containedctx
	Lifetime context.Context //nolint:containedctx

	DrainComplete func()
}

// SetupDrainable installs SIGINT/SIGTERM handlers and returns a
// DrainableContexts whose Intake cancels on signal and whose Lifetime
// cancels only when the caller invokes DrainComplete.
//
// Unlike SetupHandler, BeforeShutdown hooks registered here run inside
// DrainComplete (i.e. after the drain), not on signal receipt — so they
// can rely on Lifetime still being alive while in-flight work finishes,
// and only fire once the process is genuinely shutting down.
//
// As a safety net, if DrainComplete has not been called within the
// configured drain timeout (see DrainTimeoutEnvVar) after Intake
// cancellation, DrainComplete is invoked automatically and a loud
// warning is logged. DrainComplete is idempotent, so an automatic
// trigger followed by a caller-initiated call (or vice versa) is safe.
func SetupDrainable() *DrainableContexts {
	channel = make(chan os.Signal, 1)
	signal.Notify(channel, syscall.SIGINT, syscall.SIGTERM)

	intakeCtx, cancelIntake := context.WithCancel(context.Background())
	lifetimeCtx, cancelLifetime := context.WithCancel(context.Background())

	var once sync.Once

	drainComplete := func() {
		once.Do(func() {
			cleanup()
			cancelLifetime()
		})
	}

	go func() {
		for c := range channel {
			slog.Warn("Received " + c.String() + ", shutting down...")
			close(channel)

			channel = nil

			cancelIntake()
		}
	}()

	timeout := envutil.Duration(context.Background(), DrainTimeoutEnvVar,
		envutil.Default(DefaultDrainTimeout)).ValueOrElse(DefaultDrainTimeout)

	go func() {
		<-intakeCtx.Done()

		select {
		case <-lifetimeCtx.Done():
			return
		case <-time.After(timeout):
			slog.Warn("!!! DRAIN TIMEOUT EXCEEDED — forcing shutdown; in-flight work may be abandoned !!!",
				"timeout", timeout,
				"envVar", DrainTimeoutEnvVar)
			drainComplete()
		}
	}()

	return &DrainableContexts{
		Intake:        intakeCtx,
		Lifetime:      lifetimeCtx,
		DrainComplete: drainComplete,
	}
}

// cleanup runs all registered hooks and clears the hooks slice.
// Must be called with channel closed to prevent concurrent modifications.
func cleanup() {
	mut.Lock()
	defer mut.Unlock()

	for _, h := range hooks {
		h()
	}

	hooks = nil
}
