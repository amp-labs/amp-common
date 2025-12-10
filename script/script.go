// Package script provides utilities for running scripts with standardized logging,
// signal handling, and exit code management.
package script

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"sync"

	"github.com/amp-labs/amp-common/logger"
)

// Option is a function that configures a Script.
type Option func(script *Script)

// Exit returns an error that will cause the script to exit with the given code.
// Use this to exit with a specific code without logging an error.
func Exit(code int) error {
	return &exitError{
		code: code,
	}
}

// ExitWithError returns an error that will cause the script to exit with code 1
// and log the provided error.
func ExitWithError(err error) error {
	return &exitError{
		err:  err,
		code: 1,
	}
}

// ExitWithErrorMessage returns an error that will cause the script to exit with code 1
// and log a formatted error message.
func ExitWithErrorMessage(msg string, args ...any) error {
	return &exitError{
		err:  fmt.Errorf(msg, args...), //nolint:err113
		code: 1,
	}
}

// exitError is an error type that carries an exit code for script termination.
type exitError struct {
	err  error
	code int
}

func (e *exitError) Error() string {
	msg := "exit " + strconv.FormatInt(int64(e.code), 10)

	if e.err != nil {
		return msg + ": " + e.err.Error()
	}

	return msg
}

// LegacyLogLevel sets the legacy log level for the script's logger.
func LegacyLogLevel(lvl slog.Level) Option {
	return func(script *Script) {
		script.loggerOpts = append(script.loggerOpts, func(options *logger.Options) {
			options.LegacyLevel = lvl
		})
	}
}

// LogLevel sets the minimum log level for the script's logger.
func LogLevel(lvl slog.Level) Option {
	return func(script *Script) {
		script.loggerOpts = append(script.loggerOpts, func(options *logger.Options) {
			options.MinLevel = lvl
		})
	}
}

// LogOutput sets the output writer for the script's logger.
func LogOutput(writer io.Writer) Option {
	return func(script *Script) {
		script.loggerOpts = append(script.loggerOpts, func(options *logger.Options) {
			options.Output = writer
		})
	}
}

// EnableFlagParse controls whether flag.Parse() is called before running the script.
// Defaults to true.
func EnableFlagParse(enabled bool) Option {
	return func(script *Script) {
		script.flagParseEnable = enabled
	}
}

// Script represents a runnable script with configured logging and signal handling.
type Script struct {
	name            string
	flagParseEnable bool
	loggerOpts      []logger.Option
}

// New creates a new Script with the given name and options.
// By default, flag parsing is enabled.
func New(scriptName string, opts ...Option) *Script {
	script := &Script{
		name:            scriptName,
		flagParseEnable: true,
	}

	for _, opt := range opts {
		opt(script)
	}

	return script
}

// Run executes the script with the provided function, handling signal interrupts
// and exit codes. The context passed to f will be canceled on SIGINT.
// This function calls os.Exit and does not return.
func (r *Script) Run(f func(ctx context.Context) error) {
	os.Exit(run(r.name, f, r.flagParseEnable, r.loggerOpts...))
}

// run is the internal implementation that executes the script callback and returns
// an exit code. It configures logging, handles signals, and processes exitErrors.
func run(
	scriptName string,
	callback func(ctx context.Context) error,
	flagParseEnable bool,
	opts ...logger.Option,
) int {
	if flagParseEnable {
		flag.Parse()
	}

	// Catch Ctrl+C and handle it gracefully by shutting down the context
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)

	// Configure the logger
	_ = logger.ConfigureLogging(ctx, scriptName, opts...)

	// We want to cancel the context, but in the case of abort it's possible
	// to call it more than once. This ensures that we only call it once.
	stopOnce := sync.Once{}
	cancel := func() {
		stopOnce.Do(stop)
	}

	defer cancel()

	log := logger.Get(ctx)

	if callback == nil {
		log.Error("callback is nil")

		return 1
	}

	err := callback(ctx)
	if err != nil {
		var exitErr *exitError

		if errors.As(err, &exitErr) {
			if exitErr.code != 0 {
				log.Error("error running script", "error", err)
			}

			return exitErr.code
		} else {
			log.Error("error running script", "error", err)

			return 1
		}
	}

	return 0
}
