package dns

import (
	"context"

	"github.com/amp-labs/amp-common/contexts"
	"github.com/amp-labs/amp-common/logger"
)

// getLogLevel reads the [LogLevel] set on ctx by [WithLogLevel], defaulting to
// [LogLevelNone] when none is present.
func getLogLevel(ctx context.Context) LogLevel {
	value, ok := contexts.GetValue[contextKey, LogLevel](ctx, "logLevel")
	if ok {
		return value
	}

	return LogLevelNone
}

// logDebug logs at debug level only when the context requests verbose logging.
func logDebug(ctx context.Context, msg string, args ...any) {
	if getLogLevel(ctx) != LogLevelVerbose {
		return
	}

	logger.Debug(ctx, msg, args...)
}

// logInfo logs at info level only when the context requests verbose logging.
func logInfo(ctx context.Context, msg string, args ...any) {
	if getLogLevel(ctx) != LogLevelVerbose {
		return
	}

	logger.Info(ctx, msg, args...)
}

// logError logs at error level when the context requests either error-only or
// verbose logging.
func logError(ctx context.Context, msg string, args ...any) {
	lvl := getLogLevel(ctx)

	if lvl != LogLevelErrorOnly && lvl != LogLevelVerbose {
		return
	}

	logger.Error(ctx, msg, args...)
}
