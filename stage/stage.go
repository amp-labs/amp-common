// Package stage provides utilities for detecting and working with deployment environments.
// It determines the current running stage (local, test, dev, staging, prod) based on
// the RUNNING_ENV environment variable and test flag detection.
package stage

import (
	"errors"
	"flag"
	"fmt"

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

// Current returns the current running environment.
// The stage is determined once on first call and cached.
func Current() Stage {
	return runningStage.Get()
}

// IsLocal returns true if the current stage is Local.
func IsLocal() bool {
	return Current() == Local
}

// IsDev returns true if the current stage is Dev.
func IsDev() bool {
	return Current() == Dev
}

// IsStaging returns true if the current stage is Staging.
func IsStaging() bool {
	return Current() == Staging
}

// IsProd returns true if the current stage is Prod.
func IsProd() bool {
	return Current() == Prod
}

// IsTest returns true if the current stage is Test.
func IsTest() bool {
	return Current() == Test
}

// IsUnknown returns true if the current stage is Unknown.
func IsUnknown() bool {
	return Current() == Unknown
}

// runningStage lazily determines and caches the current stage.
var runningStage = lazy.New[Stage](func() Stage {
	value := getRunningStage()

	if value != Unknown {
		logger.Get().Info("Configured stage", "stage", value)
	}

	return value
})

// getRunningStage determines the current stage by reading the RUNNING_ENV environment variable.
// If the environment variable is not set or invalid, it falls back to Test (when in tests) or Unknown.
func getRunningStage() Stage {
	reader := envutil.String("RUNNING_ENV")

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
