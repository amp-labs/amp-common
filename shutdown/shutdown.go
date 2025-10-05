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
)

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
