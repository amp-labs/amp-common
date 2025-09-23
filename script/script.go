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

type Option func(script *Script)

func Exit(code int) error {
	return &exitError{
		code: code,
	}
}

func ExitWithError(err error) error {
	return &exitError{
		err:  err,
		code: 1,
	}
}

func ExitWithErrorMessage(msg string, args ...any) error {
	return &exitError{
		err:  fmt.Errorf(msg, args...), //nolint:err113
		code: 1,
	}
}

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

func LegacyLogLevel(lvl slog.Level) Option {
	return func(script *Script) {
		script.loggerOpts = append(script.loggerOpts, func(options *logger.Options) {
			options.LegacyLevel = lvl
		})
	}
}

func LogLevel(lvl slog.Level) Option {
	return func(script *Script) {
		script.loggerOpts = append(script.loggerOpts, func(options *logger.Options) {
			options.MinLevel = lvl
		})
	}
}

func LogOutput(writer io.Writer) Option {
	return func(script *Script) {
		script.loggerOpts = append(script.loggerOpts, func(options *logger.Options) {
			options.Output = writer
		})
	}
}

func EnableFlagParse(enabled bool) Option {
	return func(script *Script) {
		script.flagParseEnable = enabled
	}
}

type Script struct {
	name            string
	flagParseEnable bool
	loggerOpts      []logger.Option
}

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

func (r *Script) Run(f func(ctx context.Context) error) {
	os.Exit(run(r.name, f, r.flagParseEnable, r.loggerOpts...))
}

func run(
	scriptName string,
	callback func(ctx context.Context) error,
	flagParseEnable bool,
	opts ...logger.Option,
) int {
	if flagParseEnable {
		flag.Parse()
	}

	// Configure the logger
	_ = logger.ConfigureLogging(scriptName, opts...)

	// Catch Ctrl+C and handle it gracefully by shutting down the context
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)

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

	if err := callback(ctx); err != nil {
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
