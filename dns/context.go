package dns

import (
	"context"

	"github.com/amp-labs/amp-common/contexts"
)

// contextKey is the private key type used for values this package stores on a
// context, avoiding collisions with keys from other packages.
type contextKey string

// LogLevel controls how verbosely the resolution path logs. Logging is opt-in
// per request via [WithLogLevel] because resolution happens on a hot path and
// most callers want it silent.
type LogLevel int

const (
	// LogLevelNone disables all logging (the default).
	LogLevelNone LogLevel = iota
	// LogLevelErrorOnly logs only error events.
	LogLevelErrorOnly
	// LogLevelVerbose logs debug, info, and error events.
	LogLevelVerbose
)

// WithLogLevel returns a child context that causes DNS resolution performed
// with it to log at the given level. Without it, resolution is silent.
func WithLogLevel(ctx context.Context, level LogLevel) context.Context {
	return contexts.WithValue[contextKey, LogLevel](ctx, "logLevel", level)
}
