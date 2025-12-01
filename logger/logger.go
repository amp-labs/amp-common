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
)

// Used for logging customer-specific messages (so the caller can know which part of the system is generating the log).
// Using atomic.Value to ensure thread-safe reads and writes.
var subsystem atomic.Value //nolint:gochecknoglobals

// configMutex protects concurrent calls to ConfigureLoggingWithOptions.
// This is necessary because the function modifies global state (slog.SetDefault and log.Default).
var configMutex sync.Mutex //nolint:gochecknoglobals

// It's considered good practice to use unexported custom types for context keys.
// This avoids collisions with other packages that might be using the same string
// values for their own keys.
type contextKey string

// Fatal logs an error message and exits the application.
func Fatal(msg string, args ...any) {
	slog.Error(msg, args...)

	shutdown.Shutdown()

	time.Sleep(time.Second)

	os.Exit(1)
}

// Options is used to configure logging.
type Options struct {
	Subsystem   string
	JSON        bool
	MinLevel    slog.Level
	LegacyLevel slog.Level
	Output      io.Writer
}

// ConfigureLoggingWithOptions configures logging for the application.
// It returns the default logger.
// This function is thread-safe but modifies global state, so concurrent calls
// will be serialized.
func ConfigureLoggingWithOptions(opts Options) *slog.Logger {
	// Protect against concurrent configuration changes
	configMutex.Lock()
	defer configMutex.Unlock()

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

// WithCustomerId adds a customer ID to the context.
func WithCustomerId(ctx context.Context, customerId string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	return context.WithValue(ctx, contextKey("customer_id"), customerId)
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

// WithSlackNotification adds a flag to the context to indicate that a Slack
// notification should be sent.
func WithSlackNotification(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	return context.WithValue(ctx, contextKey("slack"), true)
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

// hostname will hold, in a k8s deployment context, the pod name.
// For local development it will be the local machine name.
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

func (n *nullHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return false
}

func (n *nullHandler) Handle(_ context.Context, _ slog.Record) error {
	return nil
}

func (n *nullHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	return n
}

func (n *nullHandler) WithGroup(_ string) slog.Handler {
	return n
}

// nullLogger is a logger that discards all output. It is returned by getBaseLogger
// when the context has the muted flag set to true. This allows code to call logging
// methods without producing any output, useful for suppressing logs from health checks
// and other high-frequency operations.
var nullLogger = slog.New(&nullHandler{})

// getBaseLogger returns a logger with the subsystem and pod name already set.
func getBaseLogger(ctx context.Context) *slog.Logger {
	// If the logger is muted, we still return a logger,
	// but the logger is incapable of outputting anything.
	if isMuted(ctx) {
		return nullLogger
	}

	// Get the default logger
	logger := slog.Default()

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

// With returns a new context with the given values added.
// The values are added to the logger automatically.
func With(ctx context.Context, values ...any) context.Context {
	if len(values) == 0 && ctx != nil {
		// Corner case, don't bother creating a new context.
		return ctx
	}

	vals := append(getValues(ctx), values...)

	return context.WithValue(ctx, contextKey("loggerValues"), vals)
}

// getValues retrieves logger values from the context that were added via With.
// Returns nil if no values are present in the context.
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
