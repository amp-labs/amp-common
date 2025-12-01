// Package stage provides utilities for detecting and working with deployment environments.
// It determines the current running stage (local, test, dev, staging, prod) based on
// the RUNNING_ENV environment variable and test flag detection.
package stage

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/amp-labs/amp-common/contexts"
	"github.com/amp-labs/amp-common/envutil"
	"github.com/amp-labs/amp-common/lazy"
	"github.com/amp-labs/amp-common/logger"
)

// Stage represents a deployment environment.
type Stage string

// ErrUnrecognizedStage is returned when the RUNNING_ENV contains an invalid stage value.
var ErrUnrecognizedStage = errors.New("unrecognized stage")

const (
	// Unknown indicates the stage could not be determined.
	Unknown Stage = "unknown"
	// Local indicates the code is running on a developer's local machine.
	Local Stage = "local"
	// Test indicates the code is running in unit tests (e.g., GitHub Actions).
	Test Stage = "test"
	// Dev indicates the code is running in the development environment.
	Dev Stage = "dev"
	// Staging indicates the code is running in the staging environment.
	Staging Stage = "staging"
	// Prod indicates the code is running in the production environment.
	Prod Stage = "prod"
)

// contextKey is a typed key for storing stage information in context.
type contextKey string

// stageContextKey is the key used to store stage overrides in context.
const stageContextKey contextKey = "stage"

// WithStage returns a new context with the specified stage value.
// This is primarily useful for unit testing to override the detected stage.
func WithStage(ctx context.Context, stage Stage) context.Context {
	return contexts.WithValue[contextKey, Stage](ctx, stageContextKey, stage)
}

// Current returns the current running environment.
// The stage is determined once on first call and cached.
// The value can be overridden using a context, which is useful for unit testing.
func Current(ctx context.Context) Stage {
	stage, found := contexts.GetValue[contextKey, Stage](ctx, stageContextKey)
	if found {
		return stage
	}

	return runningStage.Get(ctx)
}

// IsLocal returns true if the current stage is Local.
func IsLocal(ctx context.Context) bool {
	return Current(ctx) == Local
}

// IsDev returns true if the current stage is Dev.
func IsDev(ctx context.Context) bool {
	return Current(ctx) == Dev
}

// IsStaging returns true if the current stage is Staging.
func IsStaging(ctx context.Context) bool {
	return Current(ctx) == Staging
}

// IsProd returns true if the current stage is Prod.
func IsProd(ctx context.Context) bool {
	return Current(ctx) == Prod
}

// IsTest returns true if the current stage is Test.
func IsTest(ctx context.Context) bool {
	return Current(ctx) == Test
}

// IsUnknown returns true if the current stage is Unknown.
func IsUnknown(ctx context.Context) bool {
	return Current(ctx) == Unknown
}

// runningStage lazily determines and caches the current stage.
var runningStage = lazy.NewCtx[Stage](func(ctx context.Context) Stage {
	value := getRunningStage(ctx)

	if value != Unknown {
		logger.Get().Info("Configured stage", "stage", value)
	}

	return value
})

// getRunningStage determines the current stage by reading the RUNNING_ENV environment variable.
// If the environment variable is not set or invalid, it falls back to Test (when in tests) or Unknown.
func getRunningStage(ctx context.Context) Stage {
	reader := envutil.String(ctx, "RUNNING_ENV")

	env := envutil.Map[string, Stage](reader, func(s string) (Stage, error) {
		switch Stage(s) {
		case Local, Test, Dev, Staging, Prod:
			return Stage(s), nil
		case Unknown:
			fallthrough
		default:
			logger.Get().Warn("unknown stage", "value", s)

			return "", fmt.Errorf("%w: %s", ErrUnrecognizedStage, s)
		}
	})

	// When running unit tests, the environment variable is not set.
	// Detect if we're in a test environment by checking if test.v flag exists.
	if flag.Lookup("test.v") != nil {
		return env.ValueOrElse(Test)
	}

	return env.ValueOrElse(Unknown)
}
