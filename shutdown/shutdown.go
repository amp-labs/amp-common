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
	mut     sync.Mutex     //nolint:gochecknoglobals
	hooks   []func()       //nolint:gochecknoglobals
	channel chan os.Signal //nolint:gochecknoglobals
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

// SetupHandler sets up a signal handler for SIGTERM
// and returns a context that will be canceled when the
// signal is received. You can use this context to clean up
// resources before the process exits.
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

func cleanup() {
	mut.Lock()
	defer mut.Unlock()

	for _, h := range hooks {
		h()
	}

	hooks = nil
}
