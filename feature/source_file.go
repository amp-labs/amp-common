package feature

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/amp-labs/amp-common/logger"
	"github.com/zeebo/xxh3"
)

// DefaultPollInterval is how often FileSource checks its file for changes.
const DefaultPollInterval = 10 * time.Second

// FileOption configures a FileSource.
type FileOption func(*fileSource)

// PollInterval sets how often the file is checked for changes. Values of zero
// or less are ignored.
func PollInterval(interval time.Duration) FileOption {
	return func(s *fileSource) {
		if interval > 0 {
			s.interval = interval
		}
	}
}

// FileSource watches a JSON file of the form
//
//	{"flag-name": <rollout>, ...}
//
// where each rollout is either the object form or a shorthand string such as
// "on" or "25% user" (see ParseRollout). The file is read when Run starts and
// again whenever its contents change, checked every PollInterval. A missing
// file means "no overrides", so every flag reverts to its default. A file that
// fails to parse is logged and ignored, leaving the previous state in effect.
func FileSource(path string, opts ...FileOption) Source {
	src := &fileSource{path: path, interval: DefaultPollInterval}
	for _, opt := range opts {
		opt(src)
	}

	return src
}

type fileSource struct {
	path     string
	interval time.Duration
}

// fingerprint identifies file contents so that unchanged files are not
// re-applied on every poll.
type fingerprint struct {
	missing bool
	hash    uint64
}

// Run implements Source.
func (s *fileSource) Run(ctx context.Context, apply ApplyFunc) error {
	last := s.reload(ctx, nil, apply)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			last = s.reload(ctx, last, apply)
		}
	}
}

// reload reads the file and applies it if its contents changed since last.
// It returns the fingerprint to compare against next time.
func (s *fileSource) reload(ctx context.Context, last *fingerprint, apply ApplyFunc) *fingerprint {
	data, err := os.ReadFile(s.path) //nolint:gosec // the path is operator configuration
	current := &fingerprint{}

	switch {
	case errors.Is(err, os.ErrNotExist):
		current.missing = true
	case err != nil:
		logger.Get(ctx).Error("failed to read feature flag file; keeping current flags",
			"path", s.path, "error", err)

		return last
	default:
		current.hash = xxh3.Hash(data)
	}

	if last != nil && *last == *current {
		return last
	}

	overrides := make(map[Name]Rollout)

	if current.missing {
		logger.Get(ctx).Info("feature flag file not found; flags use their defaults", "path", s.path)
	} else {
		err = json.Unmarshal(data, &overrides)
		if err != nil {
			logger.Get(ctx).Error("failed to parse feature flag file; keeping current flags",
				"path", s.path, "error", err)

			// Remember the bad contents so they are not re-parsed every poll.
			return current
		}
	}

	err = apply(overrides)
	if err != nil {
		logger.Get(ctx).Warn("some feature flag overrides were rejected", "path", s.path, "error", err)
	}

	return current
}
