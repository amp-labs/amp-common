// Package logger provides structured logging utilities built on Go's slog package with OpenTelemetry integration.
package logger

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/amp-labs/amp-common/envutil"
	"github.com/amp-labs/amp-common/lazy"
	"github.com/amp-labs/amp-common/shutdown"
	"github.com/amp-labs/amp-common/tests"
	"github.com/neilotoole/slogt"
)

// subsystem stores the default subsystem name for the application.
// This identifies which service or component is generating logs (e.g., "api", "temporal", "messenger").
//
// The subsystem value can be overridden on a per-context basis using WithSubsystem().
// When no context override is present, GetSubsystem() returns this default value.
//
// Thread-safety: Uses atomic.Value for lock-free concurrent reads and writes.
// This allows the subsystem to be safely read from multiple goroutines while
// ConfigureLoggingWithOptions may be updating it.
var subsystem atomic.Value //nolint:gochecknoglobals

// configMutex protects concurrent calls to ConfigureLoggingWithOptions.
//
// ConfigureLoggingWithOptions modifies global state including:
//   - The default slog logger (via slog.SetDefault)
//   - The legacy log package's default logger (via log.Default)
//   - The default subsystem value (via atomic.Value.Store)
//
// Without this mutex, concurrent configuration calls could cause race conditions
// where loggers are partially configured or configuration changes interleave incorrectly.
// The mutex ensures that each configuration operation completes atomically from
// the perspective of other goroutines.
//
// Note: This mutex only protects configuration changes. Normal logging operations
// do not require this mutex and can proceed concurrently.
var configMutex sync.Mutex //nolint:gochecknoglobals

// contextKey is an unexported type used for storing values in context.Context.
//
// Using a custom unexported type instead of strings prevents key collisions with
// other packages. Even if another package uses the same string value (like "customer_id"),
// they will have different underlying types and won't conflict in the context map.
//
// This follows the best practice outlined in the context package documentation.
// All context keys in this package (mute, sensitive, customer_id, subsystem, etc.)
// use this type to ensure they are unique and cannot be accidentally accessed by
// external code.
type contextKey string

// Fatal logs an error message and exits the application.
func Fatal(msg string, args ...any) {
	slog.Error(msg, args...)

	shutdown.Shutdown()

	time.Sleep(time.Second)

	os.Exit(1)
}

// Debug logs a debug-level message using the logger retrieved from the context.
// Debug messages are typically used for detailed diagnostic information during development.
func Debug(ctx context.Context, msg string, args ...any) {
	Get(ctx).DebugContext(ctx, msg, args...)
}

// Info logs an info-level message using the logger retrieved from the context.
// Info messages are used for general informational messages about normal application flow.
func Info(ctx context.Context, msg string, args ...any) {
	Get(ctx).InfoContext(ctx, msg, args...)
}

// Warn logs a warning-level message using the logger retrieved from the context.
// Warning messages indicate potential issues that don't prevent the application from functioning.
func Warn(ctx context.Context, msg string, args ...any) {
	Get(ctx).WarnContext(ctx, msg, args...)
}

// Error logs an error-level message using the logger retrieved from the context.
// Error messages indicate failures or problems that need attention but don't cause application exit.
func Error(ctx context.Context, msg string, args ...any) {
	Get(ctx).ErrorContext(ctx, msg, args...)
}

// Options is used to configure logging behavior and output format.
// These options control both the modern slog logger and the legacy log package
// that may be used by third-party dependencies.
type Options struct {
	// Subsystem identifies the component or service generating the logs.
	// This is included in all log messages to help with filtering and routing.
	// For example: "api", "temporal", "messenger".
	Subsystem string

	// JSON determines the output format. When true, logs are formatted as JSON
	// (using slog.JSONHandler), suitable for structured log aggregation systems.
	// When false, logs use human-readable text format (slog.TextHandler).
	JSON bool

	// MinLevel is the minimum log level for the slog logger.
	// Log messages below this level will be discarded.
	// Common values: slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError.
	MinLevel slog.Level

	// LegacyLevel is the minimum log level for the legacy log package.
	// This is used when third-party packages call the standard library's log package.
	// Those log entries are redirected to slog at this level.
	LegacyLevel slog.Level

	// Output is the destination for log output. If nil, defaults to os.Stdout.
	// Can be set to os.Stderr, a file, or any io.Writer implementation.
	Output io.Writer
}

// CreateLoggerHandler creates and configures a slog.Handler based on the provided options.
// This is the core function that determines the log output format (JSON vs text) and
// filtering level.
//
// The handler can be used to create multiple logger instances or to configure the legacy
// log package. It supports both JSON and text output formats:
//   - JSON format is ideal for production environments with log aggregation systems
//   - Text format is more readable for local development and debugging
//
// The returned handler respects the MinLevel setting - log messages below this level
// will be filtered out and never reach the output destination.
//
// Parameters:
//   - opts: Configuration options including output format, destination, and minimum level
//
// Returns:
//   - A configured slog.Handler (either JSONHandler or TextHandler)
//
// Example:
//
//	handler := CreateLoggerHandler(Options{
//	    JSON: true,
//	    MinLevel: slog.LevelInfo,
//	    Output: os.Stdout,
//	})
//	logger := slog.New(handler)
func CreateLoggerHandler(opts Options) slog.Handler {
	var handler slog.Handler

	if opts.Output == nil {
		opts.Output = os.Stdout
	}

	if opts.JSON {
		// Configure logging for JSON output
		handler = slog.NewJSONHandler(opts.Output, &slog.HandlerOptions{
			Level: opts.MinLevel,
		})
	} else {
		// Configure logging for text output
		handler = slog.NewTextHandler(opts.Output, &slog.HandlerOptions{
			Level: opts.MinLevel,
		})
	}

	// Create a logger
	return &slogErrorLogger{
		inner: handler,
	}
}

// ConfigureLoggingWithOptions configures logging for the application.
// It returns the default logger.
// This function is thread-safe but modifies global state, so concurrent calls
// will be serialized.
func ConfigureLoggingWithOptions(opts Options) *slog.Logger {
	// Protect against concurrent configuration changes
	configMutex.Lock()
	defer configMutex.Unlock()

	handler := CreateLoggerHandler(opts)

	// Create a logger
	logger := slog.New(handler)

	// Set the default logger
	slog.SetDefault(logger)

	// Set up the legacy logger (we won't be using this directly, but 3rd party packages might)
	def := log.Default()
	*def = *slog.NewLogLogger(handler, opts.LegacyLevel)

	// Set the default name of the subsystem (only customers might care about
	// this, it's for informational purposes only)
	subsystem.Store(opts.Subsystem)

	return logger
}

// Option is a functional option for configuring logging via ConfigureLogging.
type Option func(*Options)

// ErrInvalidLogOutput is returned when an invalid log output destination is specified.
var ErrInvalidLogOutput = errors.New("invalid log output")

// ConfigureLogging configures logging for the application.
// It returns the default logger.
func ConfigureLogging(ctx context.Context, app string, opts ...Option) *slog.Logger {
	// Default log format is text
	logJSON := envutil.Bool(ctx, "LOG_JSON", envutil.Default(false)).ValueOrFatal()

	// Default log level is info
	minLevel := envutil.SlogLevel(ctx, "LOG_LEVEL", envutil.Default(slog.LevelInfo)).ValueOrFatal()

	// If any packages use the old log package, we'll need to configure that
	// as well (redirected in to slog). Since the old log package doesn't
	// support levels, we have to tell it what level to use.
	legacyLevel := envutil.SlogLevel(ctx, "LEGACY_LOG_LEVEL", envutil.Default(slog.LevelInfo)).ValueOrFatal()

	output := envutil.Map(envutil.String(ctx, "LOG_OUTPUT"), func(outName string) (*os.File, error) {
		switch outName {
		case "stdout":
			return os.Stdout, nil
		case "stderr":
			return os.Stderr, nil
		default:
			return nil, fmt.Errorf("%w: %q", ErrInvalidLogOutput, outName)
		}
	}).WithDefault(os.Stdout).ValueOrFatal()

	options := Options{
		Subsystem:   app,
		JSON:        logJSON,
		MinLevel:    minLevel,
		LegacyLevel: legacyLevel,
		Output:      output,
	}

	for _, o := range opts {
		o(&options)
	}

	// Do the actual configuration
	return ConfigureLoggingWithOptions(options)
}

// WithMuted adds a muted flag to the context. When muted is true, all logging
// operations on this context will be suppressed (no log output will be produced).
// This is useful for silencing logs in specific code paths, such as health checks
// or other high-frequency operations that would otherwise create excessive log noise.
func WithMuted(ctx context.Context, muted bool) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	return context.WithValue(ctx, contextKey("mute"), muted)
}

// SetMuted configures the muted flag using a callback setter function.
// This is used with lazy value overrides to suppress logging without directly
// manipulating a context. The set function is typically provided by lazy override
// mechanisms to store the value for later retrieval.
//
// Parameters:
//   - muted: Whether to suppress all log output
//   - set: Callback function that stores the key-value pair. If nil, the function returns early.
func SetMuted(muted bool, set func(any, any)) {
	if set == nil {
		return
	}

	set(contextKey("mute"), muted)
}

// isMuted checks if the context has the muted flag set to true.
// Returns false if the context is nil or if the mute flag is not set.
// This is used internally by getBaseLogger to determine whether to return
// a nullLogger that suppresses all output.
func isMuted(ctx context.Context) bool {
	if ctx == nil {
		return false
	}

	val := ctx.Value(contextKey("mute"))
	if val == nil {
		return false
	}

	muted, ok := val.(bool)

	return ok && muted
}

// WithSensitive adds a sensitive flag to the context. If the sensitive flag is set to true,
// the logger will not log the customer ID. This is useful for logging sensitive information
// that should not be exposed to customers.
func WithSensitive(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	return context.WithValue(ctx, contextKey("sensitive"), true)
}

// SetSensitive configures the sensitive flag using a callback setter function.
// This is used with lazy value overrides to mark logs as sensitive without directly
// manipulating a context. The set function is typically provided by lazy override
// mechanisms to store the value for later retrieval.
//
// Parameters:
//   - set: Callback function that stores the key-value pair. If nil, the function returns early.
func SetSensitive(set func(any, any)) {
	if set == nil {
		return
	}

	set(contextKey("sensitive"), true)
}

// WithCustomerId adds a customer ID to the context.
func WithCustomerId(ctx context.Context, customerId string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	return context.WithValue(ctx, contextKey("customer_id"), customerId)
}

// SetCustomerId configures the customer ID using a callback setter function.
// This is used with lazy value overrides to set the customer ID without directly
// manipulating a context. The set function is typically provided by lazy override
// mechanisms to store the value for later retrieval.
//
// Parameters:
//   - customerId: The customer identifier to include in logs
//   - set: Callback function that stores the key-value pair. If nil, the function returns early.
func SetCustomerId(customerId string, set func(any, any)) {
	if set == nil {
		return
	}

	set(contextKey("customer_id"), customerId)
}

// GetCustomerId returns the customer ID from the context. If the customer ID is not provided,
// an empty string will be returned, along with a false value. Otherwise, the customer ID
// will be returned along with a true value.
func GetCustomerId(ctx context.Context) (string, bool) { //nolint:contextcheck
	if ctx == nil {
		ctx = context.Background()
	}

	cid := ctx.Value(contextKey("customer_id"))
	if cid != nil {
		val, ok := cid.(string)
		if ok {
			return val, true
		}
	}

	return "", false
}

// WithSubsystem adds a subsystem to the context. If the subsystem is not provided, the default subsystem
// will be used. The default subsystem is set by the ConfigureLogging function.
func WithSubsystem(ctx context.Context, subsystem string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	return context.WithValue(ctx, contextKey("subsystem"), subsystem)
}

// SetSubsystem configures the subsystem name using a callback setter function.
// This is used with lazy value overrides to set the subsystem without directly
// manipulating a context. The set function is typically provided by lazy override
// mechanisms to store the value for later retrieval.
//
// Parameters:
//   - subsystem: The subsystem name to include in logs
//   - set: Callback function that stores the key-value pair. If nil, the function returns early.
func SetSubsystem(subsystem string, set func(any, any)) {
	if set == nil {
		return
	}

	set(contextKey("subsystem"), subsystem)
}

// WithSlackNotification adds a flag to the context to indicate that a Slack
// notification should be sent.
func WithSlackNotification(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	return context.WithValue(ctx, contextKey("slack"), true)
}

// SetSlackNotification configures the Slack notification flag using a callback setter function.
// This is used with lazy value overrides to enable Slack notifications without directly
// manipulating a context. The set function is typically provided by lazy override
// mechanisms to store the value for later retrieval.
//
// Parameters:
//   - set: Callback function that stores the key-value pair. If nil, the function returns early.
func SetSlackNotification(set func(any, any)) {
	if set == nil {
		return
	}

	set(contextKey("slack"), true)
}

// WithSlackChannel adds a Slack channel to the context. The webhook
// will use this to decide where to send the Slack notification.
// If this method is used, it is unnecessary to also use WithSlackNotification.
func WithSlackChannel(ctx context.Context, channel string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	return context.WithValue(
		context.WithValue(ctx, contextKey("slack"), true),
		contextKey("slack_channel"), channel)
}

// SetSlackChannel configures the Slack channel using a callback setter function.
// This is used with lazy value overrides to set the Slack channel without directly
// manipulating a context. The set function is typically provided by lazy override
// mechanisms to store the value for later retrieval.
//
// Parameters:
//   - channel: The Slack channel for notifications
//   - set: Callback function that stores the key-value pair. If nil, the function returns early.
func SetSlackChannel(channel string, set func(any, any)) {
	if set == nil {
		return
	}

	set(contextKey("slack_channel"), channel)
}

// GetSlackNotification returns true if a Slack notification should be sent.
func GetSlackNotification(ctx context.Context) bool { //nolint:contextcheck
	if ctx == nil {
		ctx = context.Background()
	}

	// Check for a subsystem override.
	slack := ctx.Value(contextKey("slack"))
	if slack != nil {
		val, ok := slack.(bool)
		if ok {
			return val
		} else {
			return false
		}
	} else {
		return false
	}
}

// GetSlackChannel returns the Slack channel from the context.
// If the Slack channel is not provided, an empty string will be returned,
// along with a false value. Otherwise, the Slack channel will be returned
// along with a true value.
func GetSlackChannel(ctx context.Context) (string, bool) { //nolint:contextcheck
	if ctx == nil {
		ctx = context.Background()
	}

	slackChannel := ctx.Value(contextKey("slack_channel"))
	if slackChannel != nil {
		val, ok := slackChannel.(string)
		if ok {
			return val, true
		} else {
			return "", false
		}
	} else {
		return "", false
	}
}

// logProjectContextKey is the key that is used to identify logs that need to be routed to the builder.
func logProjectContextKey() contextKey {
	return contextKey("log_project")
}

// WithRoutingToBuilder embeds the project ID into the logging context.
// This is defined explicitly as it will be used for log routing within
// GCP, enabling logs to be routed to the customer's project.
// Note: the routing logic of which Ampersand project should be synced with
// which GCP project live inside GCP, in the Log Sinks.
func WithRoutingToBuilder(ctx context.Context, projectId string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	return context.WithValue(ctx, logProjectContextKey(), projectId)
}

// SetRoutingToBuilder configures the project ID for log routing using a callback setter function.
// This is used with lazy value overrides to set the project ID without directly
// manipulating a context. The set function is typically provided by lazy override
// mechanisms to store the value for later retrieval.
//
// Parameters:
//   - projectId: The project ID for log routing to the builder
//   - set: Callback function that stores the key-value pair. If nil, the function returns early.
func SetRoutingToBuilder(projectId string, set func(any, any)) {
	if set == nil {
		return
	}

	set(logProjectContextKey(), projectId)
}

// GetRoutingToBuilder returns the project ID from the context. If the
// project ID is not provided, an empty string will be returned, along with
// a false value. Otherwise, the project ID will be returned along with
// a true value.
func GetRoutingToBuilder(ctx context.Context) (string, bool) { //nolint:contextcheck
	if ctx == nil {
		ctx = context.Background()
	}

	destinationProject := ctx.Value(logProjectContextKey())
	if destinationProject != nil {
		val, ok := destinationProject.(string)
		if ok {
			return val, true
		}
	}

	return "", false
}

// GetSubsystem returns the subsystem from the context. If the
// subsystem is not provided, the default subsystem will be used.
func GetSubsystem(ctx context.Context) string { //nolint:contextcheck
	if ctx == nil {
		ctx = context.Background()
	}

	// Check for a subsystem override.
	sub := ctx.Value(contextKey("subsystem"))
	if sub != nil {
		val, ok := sub.(string)
		if ok {
			return val
		}
	}

	// Return the default subsystem value (thread-safe read)
	if defaultSub := subsystem.Load(); defaultSub != nil {
		if val, ok := defaultSub.(string); ok {
			return val
		}
	}

	return ""
}

// WithRequestId adds a Kong request ID to the context.
func WithRequestId(ctx context.Context, requestId string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	return context.WithValue(ctx, contextKey("request_id"), requestId)
}

// SetRequestId configures the request ID using a callback setter function.
// This is used with lazy value overrides to set the Kong request ID without directly
// manipulating a context. The set function is typically provided by lazy override
// mechanisms to store the value for later retrieval.
//
// Parameters:
//   - requestId: The Kong request ID to include in logs
//   - set: Callback function that stores the key-value pair. If nil, the function returns early.
func SetRequestId(requestId string, set func(any, any)) {
	if set == nil {
		return
	}

	set(contextKey("request_id"), requestId)
}

// GetRequestId returns the request ID from the context. If the request ID is not provided,
// an empty string will be returned, along with a false value. Otherwise, the request ID
// will be returned along with a true value.
func GetRequestId(ctx context.Context) (string, bool) { //nolint:contextcheck
	if ctx == nil {
		ctx = context.Background()
	}

	reqId := ctx.Value(contextKey("request_id"))
	if reqId == nil {
		return "", false
	}

	val, ok := reqId.(string)
	if !ok {
		return "", false
	}

	return val, true
}

// hostname holds the system's hostname, which serves different purposes depending on
// the deployment environment:
//   - In Kubernetes: Contains the pod name (e.g., "api-deployment-7d9f8b5c4-xh2k9")
//   - In local development: Contains the machine's hostname
//
// This value is included in all log messages via the "pod" attribute, which helps with:
//   - Correlating logs from specific pods in distributed systems
//   - Debugging issues that only occur on specific instances
//   - Understanding log volume distribution across replicas
//
// The value is computed lazily on first access and cached for the lifetime of the process.
// If os.Hostname() fails, it returns "unknown" to ensure logging can continue.
//
// Thread-safety: Uses lazy.New for safe concurrent initialization.
// nolint:gochecknoglobals
var hostname = lazy.New[string](func() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}

	return h
})

// GetPodName returns the pod name (or hostname if not running in k8s).
func GetPodName() string {
	return hostname.Get()
}

// getRealContext extracts the first non-nil context from a variadic list.
// If no context is provided or all are nil, it returns context.Background().
//
// This is a helper function used by Get() to handle the optional context parameter.
// It enables Get() to be called either as Get() or Get(ctx), making the API more
// ergonomic while still supporting context-aware logging.
//
// The function performs a simple first-non-nil search:
//   - If any non-nil context is found, it's returned immediately
//   - If all contexts are nil or the list is empty, context.Background() is returned
//
// In practice, this function expects 0 or 1 context arguments. If multiple contexts
// are provided, only the first non-nil one is used.
func getRealContext(ctx ...context.Context) context.Context {
	var realCtx context.Context

	// Honestly we only care if there's zero or one contexts.
	// If there's more than one, we'll just use the first one.
	for _, c := range ctx {
		if c != nil {
			realCtx = c //nolint:fatcontext

			break
		}
	}

	if realCtx == nil {
		// No context provided, so we'll just use a sane default
		realCtx = context.Background()
	}

	return realCtx
}

// IsSensitiveMessage returns true if the message is marked as sensitive.
// Sensitive messages won't be routed to customers.
func IsSensitiveMessage(ctx context.Context) bool {
	// Check for a sensitive flag.
	isSensitive := false
	sensitive := ctx.Value(contextKey("sensitive"))

	if sensitive != nil {
		val, ok := sensitive.(bool)
		if ok {
			isSensitive = val
		}
	}

	return isSensitive
}

// nullHandler is a slog.Handler implementation that discards all log output.
// It is used to implement the muted logging feature. All methods are no-ops:
// - Enabled always returns false (no log levels are enabled)
// - Handle does nothing with log records
// - WithAttrs and WithGroup return the same handler (no-op transformations).
type nullHandler struct{}

// Enabled always returns false to indicate that no log levels are enabled.
// This causes the slog infrastructure to skip creating log records entirely,
// providing efficient muting with zero overhead.
func (n *nullHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return false
}

// Handle is a no-op that discards any log records passed to it.
// In practice, this is rarely called because Enabled returns false,
// but it must be implemented to satisfy the slog.Handler interface.
func (n *nullHandler) Handle(_ context.Context, _ slog.Record) error {
	return nil
}

// WithAttrs returns the same handler without modification.
// Since all output is discarded, there's no need to track attributes.
func (n *nullHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	return n
}

// WithGroup returns the same handler without modification.
// Since all output is discarded, there's no need to track attribute groups.
func (n *nullHandler) WithGroup(_ string) slog.Handler {
	return n
}

// nullLogger is a pre-configured logger that discards all output.
//
// This logger is returned by getBaseLogger when the context has the muted flag set
// (via WithMuted(ctx, true)). It allows code to call logging methods normally without
// producing any output, which is useful for:
//   - Health check endpoints that would otherwise flood logs
//   - High-frequency background operations
//   - Test scenarios where log output should be suppressed
//
// The nullLogger uses nullHandler, which returns false from Enabled() for all log
// levels. This causes the slog package to skip the work of constructing log records
// entirely, making muted logging very efficient with near-zero overhead.
var nullLogger = slog.New(&nullHandler{})

// getBaseLogger returns a logger with standard contextual attributes pre-configured.
//
// This is an internal helper function that constructs the base logger used by Get().
// It handles several important responsibilities:
//
//  1. Muted logging: If the context has isMuted(ctx) == true, returns nullLogger
//     which discards all output. This is used to suppress logs from health checks
//     and other high-frequency operations.
//
//  2. Test integration: When running in test mode (tests.GetTestInfo returns data),
//     creates a test-aware logger using slogt that properly integrates with Go's
//     testing package. Test loggers include test name and ID for easier debugging.
//
// 3. Standard attributes: Adds common attributes to all log messages:
//   - subsystem: The service/component name (from GetSubsystem)
//   - pod: The hostname/pod name (useful in Kubernetes deployments)
//   - request-id: The Kong request ID if present (for request tracing)
//
// 4. Custom values: Includes any additional key-value pairs added via With()
//
// The returned logger is the foundation for all logging in the application, ensuring
// consistent metadata across all log messages. Callers (particularly Get()) will
// further augment this logger with customer-specific or route-specific attributes.
func getBaseLogger(ctx context.Context) *slog.Logger {
	// If the logger is muted, we still return a logger,
	// but the logger is incapable of outputting anything.
	if isMuted(ctx) {
		return nullLogger
	}

	// Get the default logger
	logger := slog.Default()

	// Special test logic
	testInfo, found := tests.GetTestInfo(ctx)
	if found {
		if testInfo.Test != nil {
			logger = slogt.New(testInfo.Test, slogt.JSON(), slogt.Factory(func(w io.Writer) slog.Handler {
				return CreateLoggerHandler(Options{
					JSON:        true,
					MinLevel:    slog.LevelDebug,
					LegacyLevel: slog.LevelDebug,
					Output:      w,
				})
			}))
		}

		if len(testInfo.Name) > 0 {
			logger = logger.With("test-name", testInfo.Name)
		}

		if len(testInfo.Id) > 0 {
			logger = logger.With("test-id", testInfo.Id)
		}
	}

	// Add the subsystem name, and the pod name.
	logger = logger.With(
		"subsystem", GetSubsystem(ctx),
		"pod", hostname.Get())

	requestId, found := GetRequestId(ctx)
	if found {
		logger = logger.With("request-id", requestId)
	}

	// Check for key-values to add to the logger.
	vals := getValues(ctx)
	if vals != nil {
		logger = logger.With(vals...)
	}

	return logger
}

// Get returns a logger. If a context is provided, it will check for a customer ID in the context,
// and if found, will log with that customer ID. Otherwise, it will log without a customer ID.
// Use the WithCustomerId function to embed a customer ID in the context.
//
//nolint:contextcheck
func Get(ctx ...context.Context) *slog.Logger {
	realCtx := getRealContext(ctx...)
	logger := getBaseLogger(realCtx)

	if GetSlackNotification(realCtx) {
		// This will trigger a special log route which will publish
		// the entire log message to Slack.
		// Logs -> Pub/Sub -> Cloud Function -> Slack
		logger = logger.With("slack", "1") // slack=1 just means slack notifications are enabled
	}

	if channel, ok := GetSlackChannel(realCtx); ok {
		// If slack notifications are enabled, this will tell
		// the Slack cloud function which channel to send the message to.
		// If this is not set, the message will be sent to the default channel,
		// which is chosen at the discretion of the cloud function.
		logger = logger.With("slack_channel", channel)
	}

	if IsSensitiveMessage(realCtx) {
		// We don't want to expose customers to sensitive info, so we'll redact
		// it (from them) by omitting the customer id entirely. In other words,
		// there's no chance this will accidentally be routed to a customer.
		return logger
	}

	// If we have a customer ID, add it to the log context.
	custId, ok := GetCustomerId(realCtx)
	if ok {
		logger = logger.With("customer_id", custId)
	}

	// If we have a project ID for routing to the builder, add it to the log context.
	logProject, ok := GetRoutingToBuilder(realCtx)
	if ok {
		// This will trigger a special log route which will route the log
		logger = logger.With("log_project", logProject)
	}

	return logger
}

// With returns a new context with additional key-value pairs that will be included
// in all log messages created from that context.
//
// This function enables structured logging by allowing you to attach metadata to a
// context that will automatically appear in all subsequent log messages. The values
// are stored in the context and retrieved by getBaseLogger(), which adds them to
// the logger via With().
//
// Parameters:
//   - ctx: The context to augment. If nil, a background context is used.
//   - values: Key-value pairs in the same format expected by slog (alternating keys and values).
//     Keys should be strings, values can be any type.
//
// Usage pattern:
//
//	ctx = logger.With(ctx, "operation", "user_sync", "batch_size", 100)
//	logger.Get(ctx).Info("Starting operation")  // Will include operation=user_sync, batch_size=100
//
// The values are cumulative - if the input context already has values from a previous
// With() call, the new values are appended to the existing ones. This allows building
// up context as execution flows through different layers of the application.
//
// Note: This is separate from the context.With* functions in this package (WithCustomerId,
// WithSubsystem, etc.) which handle specific well-known attributes. Use With() for
// arbitrary structured data that varies by use case.
func With(ctx context.Context, values ...any) context.Context {
	if len(values) == 0 && ctx != nil {
		// Corner case, don't bother creating a new context.
		return ctx
	}

	vals := append(getValues(ctx), values...)

	return context.WithValue(ctx, contextKey("loggerValues"), vals)
}

// getValues retrieves structured logging key-value pairs from the context.
//
// This is an internal helper that extracts values previously stored via the With() function.
// The returned slice contains alternating keys and values in the format expected by slog.
//
// The function is called by getBaseLogger() to augment the logger with context-specific
// attributes. This enables structured logging where metadata flows naturally with the
// context through the application.
//
// Returns:
//   - A slice of key-value pairs if values were stored via With()
//   - nil if no values are present in the context
//
// Thread-safety: This function is safe to call concurrently as it only reads from
// the context, which is immutable.
func getValues(ctx context.Context) []any { //nolint:contextcheck
	if ctx == nil {
		ctx = context.Background()
	}

	// Check for a value override.
	vals := ctx.Value(contextKey("loggerValues"))
	if vals != nil {
		val, ok := vals.([]any)
		if ok {
			return val
		}

		return nil
	}

	return nil
}
