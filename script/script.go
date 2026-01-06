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

	"github.com/amp-labs/amp-common/envutil"
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

// simpleLoader wraps a static string value in a provider function.
func simpleLoader(value string) func() (string, bool) {
	return func() (string, bool) {
		return value, len(value) > 0
	}
}

// WithEnvFile configures the script to load environment variables from the specified file.
// The file is loaded before the script callback executes and variables are set in both
// the OS environment and the context.
func WithEnvFile(envFile string) Option {
	return func(script *Script) {
		script.envFiles = append(script.envFiles, simpleLoader(envFile))
	}
}

// WithEnvFileProvider configures the script to load environment variables from a file
// whose path is determined by calling the provider function at runtime.
// This allows for dynamic file paths based on runtime conditions.
func WithEnvFileProvider(provider func() (string, bool)) Option {
	return func(script *Script) {
		script.envFiles = append(script.envFiles, provider)
	}
}

// WithEnvFiles configures the script to load environment variables from multiple files.
// Files are loaded in the order provided, with later files overriding earlier ones.
func WithEnvFiles(envFiles ...string) Option {
	return func(script *Script) {
		loaders := make([]func() (string, bool), 0, len(envFiles))
		for _, envFile := range envFiles {
			loaders = append(loaders, simpleLoader(envFile))
		}

		script.envFiles = append(script.envFiles, loaders...)
	}
}

// WithEnvFilesProvider configures the script to load environment variables from multiple files
// whose paths are determined by calling the provider functions at runtime.
// This allows for dynamic file paths based on runtime conditions.
func WithEnvFilesProvider(envFiles ...func() (string, bool)) Option {
	return func(script *Script) {
		script.envFiles = append(script.envFiles, envFiles...)
	}
}

// WithSetEnv configures the script to set a specific environment variable programmatically.
// This can be used instead of or in addition to loading from files.
// Variables set this way will override those loaded from env files.
// Can be called multiple times to set multiple variables.
func WithSetEnv(key, value string) Option {
	return func(script *Script) {
		script.setEnv = append(script.setEnv, func() (string, string) {
			return key, value
		})
	}
}

// WithSetEnvProvider configures the script to set an environment variable
// whose key and value are determined by calling the provider function at runtime.
// This allows for dynamic environment variable values based on runtime conditions.
// Variables set this way will override those loaded from env files.
// Can be called multiple times to set multiple variables.
func WithSetEnvProvider(provider func() (string, string)) Option {
	return func(script *Script) {
		script.setEnv = append(script.setEnv, provider)
	}
}

// Script represents a runnable script with configured logging and signal handling.
type Script struct {
	name            string
	flagParseEnable bool
	loggerOpts      []logger.Option
	envFiles        []func() (string, bool)
	setEnv          []func() (string, string)
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
	os.Exit(run(r.name, f, r.flagParseEnable, r.envFiles, r.setEnv, r.loggerOpts...))
}

// run is the internal implementation that executes the script callback and returns
// an exit code. It configures logging, handles signals, and processes exitErrors.
func run(
	scriptName string,
	callback func(ctx context.Context) error,
	flagParseEnable bool,
	envFiles []func() (string, bool),
	setEnv []func() (string, string),
	opts ...logger.Option,
) int {
	if flagParseEnable {
		flag.Parse()
	}

	// Catch Ctrl+C and handle it gracefully by shutting down the context
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)

	if len(envFiles) > 0 || len(setEnv) > 0 { //nolint:nestif
		loader := envutil.NewLoader()
		loader.LoadEnv()

		for _, envFile := range envFiles {
			filePath, valid := envFile()

			if filePath == "" || !valid {
				continue
			}

			_, err := loader.LoadFile(filePath)
			if err != nil {
				logger.Get(ctx).Error("unable to load env file",
					"file", filePath,
					"error", err)

				return 1
			}
		}

		if len(setEnv) > 0 {
			for _, fetch := range setEnv {
				key, val := fetch()

				loader.Set(key, val)
			}
		}

		m := loader.AsMap()

		if len(m) > 0 {
			ctx = envutil.WithEnvOverrides(ctx, m)

			for k, v := range m {
				err := os.Setenv(k, v)
				if err != nil {
					logger.Get(ctx).Warn("unable to set env var",
						"key", k,
						"value", v,
						"error", err)

					return 1
				}
			}
		}
	}

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
