package dns

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetLogLevel_DefaultsToNone(t *testing.T) {
	t.Parallel()

	assert.Equal(t, LogLevelNone, getLogLevel(context.Background()))
}

func TestWithLogLevel_RoundTrips(t *testing.T) {
	t.Parallel()

	for _, lvl := range []LogLevel{LogLevelNone, LogLevelErrorOnly, LogLevelVerbose} {
		ctx := WithLogLevel(context.Background(), lvl)
		assert.Equal(t, lvl, getLogLevel(ctx))
	}
}

func TestLogHelpers_DoNotPanicAtAnyLevel(t *testing.T) {
	t.Parallel()

	// The log helpers gate on the context level; exercise every level to make
	// sure the gating logic itself is sound regardless of whether output occurs.
	for _, lvl := range []LogLevel{LogLevelNone, LogLevelErrorOnly, LogLevelVerbose} {
		ctx := WithLogLevel(context.Background(), lvl)

		assert.NotPanics(t, func() {
			logDebug(ctx, "debug", "k", "v")
			logInfo(ctx, "info", "k", "v")
			logError(ctx, "error", "k", "v")
		})
	}
}
