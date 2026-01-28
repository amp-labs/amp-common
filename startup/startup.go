// Package startup provides utilities for application initialization and environment configuration.
//
// The primary use case is loading environment variables from files during application startup,
// which is useful for local development, container initialization, and testing scenarios.
package startup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/amp-labs/amp-common/envtypes"
	"github.com/amp-labs/amp-common/envutil"
	"github.com/amp-labs/amp-common/future"
	"github.com/amp-labs/amp-common/logger"
	"github.com/amp-labs/amp-common/should"
	"github.com/amp-labs/amp-common/shutdown"
	"github.com/amp-labs/amp-common/xform"
)

const (
	// auditLogFlushInterval is the interval at which audit log events are flushed to disk.
	auditLogFlushInterval = 5 * time.Second
	// auditLogFilePerms is the file permission for audit log files.
	auditLogFilePerms = 0600
)

var (
	// ErrEnvDebugIsDirectory is returned when ENV_DEBUG points to a directory instead of a file.
	ErrEnvDebugIsDirectory = errors.New("ENV_DEBUG path is a directory")
)

// Option is a functional option for configuring environment loading behavior.
type Option func(*options)

// options holds the configuration for environment variable loading.
type options struct {
	// allowOverride determines whether environment variables loaded from files
	// can override existing environment variables. When false (default), existing
	// environment variables take precedence over file-based values.
	allowOverride bool
	// enableRecording determines whether envutil should record all environment
	// variable reads for debugging and auditing purposes.
	enableRecording bool
	// enableStackTraces enables capturing stack traces for environment variable
	// reads when recording is enabled, useful for tracking down where variables
	// are being accessed.
	enableStackTraces bool
	// dedupKeys determines whether to deduplicate keys in audit logs.
	// When true, only the first read of each environment variable key is logged.
	dedupKeys bool
	// observers is a list of observers that will be notified of environment
	// variable reads. Used for custom monitoring and auditing.
	observers []envutil.Observer
	// auditLogFile specifies the path to write audit logs of environment
	// variable reads. Setting this automatically enables recording.
	auditLogFile string
}

// WithAllowOverride configures whether loaded environment variables can override
// existing environment variables in the process.
//
// When allowOverride is true, variables from files will replace existing environment
// variables. When false (default), existing variables are preserved and file-based
// values are ignored for variables that already exist.
//
// Example:
//
//	// Allow files to override existing environment variables
//	err := ConfigureEnvironment(WithAllowOverride(true))
func WithAllowOverride(allowOverride bool) Option {
	return func(o *options) {
		o.allowOverride = allowOverride
	}
}

// WithEnableRecording configures whether envutil should record all environment
// variable reads during startup.
//
// When enabled, all calls to envutil functions (like envutil.String, envutil.Int, etc.)
// will be tracked. This is useful for debugging, auditing which environment variables
// your application depends on, and generating documentation.
//
// Recording has minimal performance overhead but should typically be disabled in
// production unless you need audit trails.
//
// Example:
//
//	// Enable recording to track which environment variables are read
//	err := ConfigureEnvironment(WithEnableRecording(true))
func WithEnableRecording(enableRecording bool) Option {
	return func(o *options) {
		o.enableRecording = enableRecording
	}
}

// WithEnableStackTraces configures whether to capture stack traces for each
// environment variable read when recording is enabled.
//
// Stack traces help identify exactly where in your code each environment variable
// is being read from, which is invaluable for debugging complex applications where
// the same variable might be accessed from multiple locations.
//
// Note: This option only takes effect when recording is also enabled (via
// WithEnableRecording or WithAuditLogFile). Stack traces add more overhead than
// basic recording, so use judiciously.
//
// Example:
//
//	// Enable both recording and stack traces for detailed debugging
//	err := ConfigureEnvironment(
//	    WithEnableRecording(true),
//	    WithEnableStackTraces(true),
//	)
func WithEnableStackTraces(enableStackTraces bool) Option {
	return func(o *options) {
		o.enableStackTraces = enableStackTraces
	}
}

// WithDedupKeys configures whether to deduplicate keys in audit logs.
//
// When enabled, only the first read of each environment variable key is recorded
// in the audit log. Subsequent reads of the same key are muted. This is useful for
// reducing noise in audit logs when the same environment variables are read multiple
// times during application startup.
//
// Note: This option only affects recording to audit logs. Observers are still
// notified for all reads regardless of this setting.
//
// Example:
//
//	// Enable recording with deduplication to reduce log noise
//	err := ConfigureEnvironment(
//	    WithAuditLogFile("/var/log/app/env-audit.log"),
//	    WithDedupKeys(true),
//	)
func WithDedupKeys(dedupKeys bool) Option {
	return func(o *options) {
		o.dedupKeys = dedupKeys
	}
}

// WithObservers registers custom observers that will be notified of every
// environment variable read operation.
//
// Observers implement the envutil.Observer interface and can be used for:
//   - Custom logging and monitoring
//   - Security auditing
//   - Real-time alerting on sensitive variable access
//   - Integration with external monitoring systems
//
// Observers are automatically unregistered during application shutdown.
// Multiple observers can be added, and they will all be notified in order.
//
// Example:
//
//	type MyObserver struct{}
//	func (o *MyObserver) OnRead(key, value string) {
//	    log.Printf("Env var read: %s", key)
//	}
//
//	observer := &MyObserver{}
//	err := ConfigureEnvironment(WithObservers(observer))
func WithObservers(observers ...envutil.Observer) Option {
	return func(o *options) {
		o.observers = append(o.observers, observers...)
	}
}

// WithAuditLogFile specifies a file path where environment variable read operations
// should be logged for auditing purposes.
//
// This option automatically enables recording (you don't need to also call
// WithEnableRecording). The audit log will contain details about which environment
// variables were read, their values, and when they were accessed.
//
// Audit logs are useful for:
//   - Security compliance and auditing
//   - Debugging configuration issues in production
//   - Documenting environment variable dependencies
//   - Tracking down where sensitive values are being accessed
//
// Example:
//
//	// Write audit logs to a specific file
//	err := ConfigureEnvironment(WithAuditLogFile("/var/log/app/env-audit.log"))
func WithAuditLogFile(auditLogFile string) Option {
	return func(o *options) {
		o.enableRecording = true
		o.auditLogFile = auditLogFile
	}
}

// ConfigureEnvironment loads environment variables from files specified in the ENV_FILE
// environment variable and sets them in the current process.
//
// The ENV_FILE variable should contain a semicolon-separated list of file paths.
// For example: ENV_FILE="/path/to/.env;/path/to/.env.local"
//
// By default, existing environment variables take precedence over file-based values.
// Use WithAllowOverride(true) to allow files to override existing variables.
//
// The function performs the following steps:
//  1. Parses the ENV_FILE environment variable (semicolon-separated file paths)
//  2. Loads all specified files using envutil.Loader
//  3. Sets environment variables from the loaded files (respecting override policy)
//
// If ENV_FILE is not set or empty, the function returns nil without error.
//
// Example usage:
//
//	// Load environment files, preserving existing variables
//	if err := ConfigureEnvironment(); err != nil {
//	    return fmt.Errorf("failed to configure environment: %w", err)
//	}
//
//	// Load environment files, allowing files to override existing variables
//	if err := ConfigureEnvironment(WithAllowOverride(true)); err != nil {
//	    return fmt.Errorf("failed to configure environment: %w", err)
//	}
func ConfigureEnvironment(opts ...Option) error {
	cfg := getOptions(opts)

	// We do this here so that ENV_FILE getting read gets captured
	envutil.EnableRecording(cfg.enableRecording)
	envutil.EnableStackTraces(cfg.enableStackTraces)

	// Parse the ENV_FILE environment variable into a list of file paths.
	// The transformation pipeline: read ENV_FILE -> trim whitespace -> split on semicolon -> sanitize list
	envFiles := envutil.Map(
		envutil.Map(
			envutil.Map(
				envutil.String(context.Background(), "ENV_FILE"),
				xform.TrimString),
			xform.SplitString(";")),
		sanitizeEnvFileList).
		ValueOrElse(nil)

	envDebug :=
		envutil.Map(
			envutil.FilePath(context.Background(), "ENV_DEBUG"),
			func(path envtypes.LocalPath) (envtypes.LocalPath, error) {
				if path.Info != nil && path.Info.IsDir() {
					return path, fmt.Errorf("%w: %s", ErrEnvDebugIsDirectory, path.Path)
				}

				return path, nil
			})

	envTraces := envutil.Bool(context.Background(),
		"ENV_TRACES", envutil.Default(false))

	envDebug.DoWithValue(func(path envtypes.LocalPath) {
		if cfg.auditLogFile == "" {
			opts = append(opts, WithAuditLogFile(path.Path))
		}

		opts = append(opts,
			WithEnableRecording(true),
			WithEnableStackTraces(envTraces.ValueOrElse(false)))
	})

	return ConfigureEnvironmentFromFiles(envFiles, opts...)
}

// ConfigureEnvironmentFromFiles loads environment variables from the specified list of files
// and sets them in the current process.
//
// This function is similar to ConfigureEnvironment but accepts an explicit list of file paths
// instead of reading from the ENV_FILE environment variable. This is useful for testing or
// when you want programmatic control over which files to load.
//
// By default, existing environment variables take precedence over file-based values.
// Use WithAllowOverride(true) to allow files to override existing variables.
//
// The function performs the following steps:
//  1. Loads all specified files using envutil.Loader
//  2. Sets environment variables from the loaded files (respecting override policy)
//
// If the envFiles list is empty, the function returns nil without error.
//
// Example usage:
//
//	// Load specific environment file paths
//	filePaths := []string{"/path/to/.env", "/path/to/.env.local"}
//	if err := ConfigureEnvironmentFromFiles(filePaths); err != nil {
//	    return fmt.Errorf("failed to configure environment: %w", err)
//	}
//
//	// Load with override enabled
//	if err := ConfigureEnvironmentFromFiles(filePaths, WithAllowOverride(true)); err != nil {
//	    return fmt.Errorf("failed to configure environment: %w", err)
//	}
func ConfigureEnvironmentFromFiles(envFiles []string, opts ...Option) error {
	cfg := setupEnvutil(opts)

	if len(envFiles) == 0 {
		// Nothing to do - no files specified
		return nil
	}

	// Create a new environment variable loader
	loader := envutil.NewLoader()

	// Load all specified files into the loader
	for _, file := range envFiles {
		_, err := loader.LoadFile(file)
		if err != nil {
			return fmt.Errorf("loading environment variables from file %q: %w", file, err)
		}
	}

	// Set environment variables from the loaded files, respecting the override policy
	for k, v := range loader.AsMap() {
		oldValue, exists := os.LookupEnv(k)
		if exists && (!cfg.allowOverride || oldValue == v) {
			// Skip setting the variable if it already exists and either:
			// 1. Override is not allowed (respect existing environment), OR
			// 2. The value is identical to what we would set (no point in redundant operation)
			continue
		}

		// Set the environment variable (either it doesn't exist, or override is allowed and value differs)
		err := os.Setenv(k, v)
		if err != nil {
			return fmt.Errorf("setting environment variable %q=%q: %w", k, v, err)
		}
	}

	err := setupAuditFile(cfg)
	if err != nil {
		return err
	}

	return nil
}

func setupAuditFile(opts *options) error {
	if opts.auditLogFile == "" || !opts.enableRecording {
		return nil
	}

	output, err := os.OpenFile(opts.auditLogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, auditLogFilePerms)
	if err != nil {
		return fmt.Errorf("opening audit log file %q: %w", opts.auditLogFile, err)
	}

	defer should.Close(output)

	ctx, cancel := context.WithCancel(context.Background())
	shutdown.BeforeShutdown(cancel)

	future.AsyncContext(ctx, func(ctx context.Context) {
		ticker := time.NewTicker(auditLogFlushInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				events := envutil.CollectRecordingEvents(true)

				for _, event := range events {
					bts, err := json.Marshal(event)
					if err != nil {
						logger.Error(ctx, "Error marshalling envutil recording event", "error", err)

						continue
					}

					_, err = output.Write(bts)
					if err != nil {
						logger.Error(ctx, "Error writing envutil recording event", "error", err)

						continue
					}

					_, err = output.WriteString("\n")
					if err != nil {
						logger.Error(ctx, "Error writing envutil recording event", "error", err)

						continue
					}
				}

				_ = output.Sync()
			}
		}
	})

	return nil
}

// setupEnvutil configures the envutil package based on the provided options and
// registers cleanup handlers.
//
// This internal helper function:
//  1. Applies the provided functional options to get configuration
//  2. Enables recording and stack traces in the envutil package if configured
//  3. Registers any custom observers with envutil
//  4. Schedules cleanup functions to unregister observers during shutdown
//
// Returns the parsed configuration for use by the caller.
func setupEnvutil(opts []Option) *options {
	cfg := getOptions(opts)

	// Enable recording and stack traces in envutil based on configuration
	envutil.EnableRecording(cfg.enableRecording)
	envutil.EnableStackTraces(cfg.enableStackTraces)
	envutil.EnableDedupKeys(cfg.dedupKeys)

	var cleanupFuncs []func()

	// Register all observers and collect their cleanup functions
	if len(cfg.observers) > 0 {
		for _, observer := range cfg.observers {
			cancel := envutil.RegisterObserver(observer)

			cleanupFuncs = append(cleanupFuncs, cancel)
		}
	}

	// Schedule observer cleanup to run before application shutdown
	if len(cleanupFuncs) > 0 {
		shutdown.BeforeShutdown(func() {
			for _, cancel := range cleanupFuncs {
				cancel()
			}
		})
	}

	return cfg
}

// sanitizeEnvFileList removes empty and whitespace-only entries from a list of file paths.
//
// This function is used to clean up the file path list after splitting ENV_FILE on semicolons,
// removing any empty strings that may result from trailing semicolons, double semicolons,
// or whitespace-only entries.
//
// Returns nil if the input is empty or if all entries are empty/whitespace after trimming.
func sanitizeEnvFileList(in []string) ([]string, error) {
	if len(in) == 0 {
		return nil, nil
	}

	out := make([]string, 0, len(in))

	for _, s := range in {
		s = strings.TrimSpace(s)

		if len(s) == 0 {
			continue
		}

		out = append(out, s)
	}

	if len(out) == 0 {
		return nil, nil
	}

	return out, nil
}

// getOptions applies the provided functional options and returns the resulting configuration.
//
// This helper function creates a default options struct and applies each provided Option
// function to it, building up the final configuration. Nil options are safely ignored.
func getOptions(opts []Option) *options {
	cfg := &options{}

	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}

	return cfg
}
