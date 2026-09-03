package feature

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/amp-labs/amp-common/logger"
)

// DefaultEnvPrefix is the prefix EnvSource and ParseEnviron use when given an
// empty one.
const DefaultEnvPrefix = "FEATURE_"

// EnvSource reads overrides from environment variables once, when Run starts,
// and then waits for ctx. Variables are matched by prefix; the rest of the
// name is lowercased and underscores become dashes, so
//
//	FEATURE_NEW_BILLING_ENGINE=on
//
// targets the flag named "new-billing-engine". Values use the syntax accepted
// by ParseRollout. Empty values are ignored and unparsable values are logged.
// An empty prefix means DefaultEnvPrefix.
//
// The environment cannot change while a process runs, so combine this with
// FileSource through Multi when live changes are needed as well.
func EnvSource(prefix string) Source {
	return SourceFunc(func(ctx context.Context, apply ApplyFunc) error {
		overrides, err := ParseEnviron(prefix, os.Environ())
		if err != nil {
			logger.Get(ctx).Warn("some feature flag environment variables were ignored", "error", err)
		}

		err = apply(overrides)
		if err != nil {
			logger.Get(ctx).Warn("some feature flag overrides from the environment were rejected", "error", err)
		}

		<-ctx.Done()

		return nil
	})
}

// ParseEnviron extracts flag overrides from environment entries of the form
// KEY=VALUE, as returned by os.Environ. See EnvSource for the naming rules.
// Entries that fail to parse are reported in the returned error, which wraps
// ErrInvalidRollout; the remaining entries are still returned.
func ParseEnviron(prefix string, environ []string) (map[Name]Rollout, error) {
	if prefix == "" {
		prefix = DefaultEnvPrefix
	}

	out := make(map[Name]Rollout)

	var errs []error

	for _, entry := range environ {
		key, value, found := strings.Cut(entry, "=")
		if !found || value == "" {
			continue
		}

		rest, matched := strings.CutPrefix(key, prefix)
		if !matched || rest == "" {
			continue
		}

		rollout, err := ParseRollout(value)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", key, err))

			continue
		}

		out[Name(strings.ReplaceAll(strings.ToLower(rest), "_", "-"))] = rollout
	}

	return out, errors.Join(errs...)
}

// EnvKey returns the environment variable that EnvSource maps to the given
// flag: the prefix followed by the upper-cased name with dashes turned into
// underscores. An empty prefix means DefaultEnvPrefix.
func EnvKey(prefix string, name Name) string {
	if prefix == "" {
		prefix = DefaultEnvPrefix
	}

	return prefix + strings.ReplaceAll(strings.ToUpper(string(name)), "-", "_")
}
