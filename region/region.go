// Package region provides utilities for detecting and working with the data
// region a process is deployed in (e.g. "us" or "eu"). It determines the
// current region based on the REGION environment variable.
//
// This package deliberately mirrors the structure of package stage: a lazily
// cached, process-wide value that can be overridden per-context (primarily for
// unit tests).
package region

import (
	"context"
	"errors"
	"fmt"

	"github.com/amp-labs/amp-common/contexts"
	"github.com/amp-labs/amp-common/envutil"
	"github.com/amp-labs/amp-common/lazy"
	"github.com/amp-labs/amp-common/logger"
)

// Region represents a deployment region. It is a plain string so that it
// serializes naturally in JSON, logs, and metric labels.
type Region string

// ErrUnrecognizedRegion is returned when the REGION environment variable
// contains a value that is not one of the known regions.
var ErrUnrecognizedRegion = errors.New("unrecognized region")

const (
	// Unknown indicates the region could not be determined. This is the
	// fallback when REGION is unset or invalid. Note that Unknown is never
	// accepted as a valid value of the REGION environment variable itself.
	Unknown Region = "unknown"
	// Us indicates the United States region.
	Us Region = "us" //nolint:varnamelen // Two-letter region codes are the conventional names.
	// Eu indicates the European Union region.
	Eu Region = "eu" //nolint:varnamelen // Two-letter region codes are the conventional names.
)

// contextKey is a typed key for storing region information in context.
// Using a private type prevents collisions with keys from other packages.
type contextKey string

// regionContextKey is the key used to store region overrides in context.
const regionContextKey contextKey = "region"

// WithRegion returns a new context with the specified region value.
// A region stored this way takes precedence over the environment-derived
// region in Current. This is primarily useful for unit testing, or for
// request-scoped code that must act on behalf of a specific region
// regardless of where the process is running.
func WithRegion(ctx context.Context, region Region) context.Context {
	return contexts.WithValue[contextKey, Region](ctx, regionContextKey, region)
}

// SetRegion configures the region using a callback setter function. This is
// the callback-based counterpart of WithRegion, following the same
// Set*/With* convention used throughout amp-common (see stage.SetStage,
// lazy.SetValueOverride, envutil.SetEnvOverrides).
//
// Because regionContextKey is unexported, this is the only way for external
// code to seed a custom context-like store (e.g. a context builder or lazy
// override mechanism) with a value that Current will later recognize.
//
// Parameters:
//   - region: The region to store (Us, Eu, or Unknown)
//   - set: Callback that stores the key-value pair. If nil, the function returns early.
func SetRegion(region Region, set func(any, any)) {
	if set == nil {
		return
	}

	set(regionContextKey, region)
}

// Current returns the region the process is running in.
//
// Resolution order:
//  1. A region placed in ctx via WithRegion (or SetRegion), if any. This is
//     returned as-is, even if it is Unknown or otherwise not Valid.
//  2. The process-wide region derived from the REGION environment variable.
//     This is determined once on first call and cached for the lifetime of
//     the process (see currentRegion).
//
// A nil ctx is tolerated and simply skips the context override.
func Current(ctx context.Context) Region {
	region, found := contexts.GetValue[contextKey, Region](ctx, regionContextKey)
	if found {
		return region
	}

	return currentRegion.Get(ctx)
}

// String returns the region as a plain string, implementing fmt.Stringer.
func (r Region) String() string {
	return string(r)
}

// Valid reports whether r is one of the known, deployable regions (Us or Eu).
// Unknown, the empty string, and any other value are not valid.
func (r Region) Valid() bool {
	return r == Eu || r == Us
}

// OrElse returns r if it is Valid, otherwise dfl. Note that dfl is returned
// unchecked, so callers should pass a valid region as the default.
//
// Typical usage is region.Current(ctx).OrElse(region.Us).
func (r Region) OrElse(dfl Region) Region {
	if r.Valid() {
		return r
	}

	return dfl
}

// currentRegion lazily determines and caches the process-wide region.
//
// The value is computed exactly once, using the context supplied to the first
// Current call that misses the context override. Any envutil context overrides
// present on that first context are therefore baked into the cached value;
// later callers with different contexts get the same result. This matches the
// behavior of stage.runningStage.
//
// The lazy value is intentionally unnamed (no WithName), so lazy-level
// overrides do not apply. Use WithRegion to override instead.
var currentRegion = lazy.NewCtx[Region](func(ctx context.Context) Region {
	value := getRegion(ctx)

	// Only announce a region that was actually configured. An Unknown region
	// is the expected state in local development and tests, so it stays quiet.
	if value != Unknown {
		logger.Get().Info("Configured region", "region", value)
	}

	return value
})

// getRegion determines the region by reading the REGION environment variable.
// If the variable is unset or holds an unrecognized value, it returns Unknown.
// Unlike stage.getRunningStage, there is no special-casing for unit tests:
// a test that does not set REGION (or use WithRegion) sees Unknown.
func getRegion(ctx context.Context) Region {
	reader := envutil.String(ctx, "REGION")

	env := envutil.Map[string, Region](reader, func(s string) (Region, error) {
		switch Region(s) {
		case Us, Eu:
			return Region(s), nil
		case Unknown:
			// Listed explicitly (rather than folded into default) so the
			// exhaustive linter sees every Region constant handled. Setting
			// REGION=unknown is treated as a misconfiguration, not a choice.
			fallthrough
		default:
			logger.Get(ctx).Warn("unknown region", "value", s)

			return "", fmt.Errorf("%w: %s", ErrUnrecognizedRegion, s)
		}
	})

	// ValueOrElse swallows both the missing and the invalid case. For an
	// invalid value it also emits its own warning, so a bad REGION logs twice.
	return env.ValueOrElse(Unknown)
}
