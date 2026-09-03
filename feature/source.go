package feature

import (
	"context"
	"maps"
	"sync"
)

// Source pushes rollouts into a registry at runtime. Run blocks until ctx is
// done, calling apply with the complete desired set of overrides each time it
// changes. Flags absent from the set revert to their code defaults, so a
// Source is authoritative for the registry it feeds; use Multi to combine
// several. Run returns nil when ctx is done and an error only on failure.
type Source interface {
	Run(ctx context.Context, apply ApplyFunc) error
}

// ApplyFunc receives a complete set of overrides from a Source. Registry.Load
// is one.
type ApplyFunc func(overrides map[Name]Rollout) error

// SourceFunc adapts a function to the Source interface.
type SourceFunc func(ctx context.Context, apply ApplyFunc) error

// Run implements Source.
func (fn SourceFunc) Run(ctx context.Context, apply ApplyFunc) error {
	return fn(ctx, apply)
}

// Static returns a Source that applies the given overrides once and then waits
// for ctx to be done. If any override is invalid, Run returns that error.
// It is useful in tests and for composing fixed overrides with Multi.
func Static(overrides map[Name]Rollout) Source {
	return SourceFunc(func(ctx context.Context, apply ApplyFunc) error {
		err := apply(maps.Clone(overrides))
		if err != nil {
			return err
		}

		<-ctx.Done()

		return nil
	})
}

// Multi combines sources into one. Each runs concurrently; whenever any of
// them produces a new set, the latest sets from all sources are merged in
// argument order, later sources winning on conflicting names, and applied.
// Run returns nil when ctx is done. If any source fails, the others are
// stopped and that error is returned.
func Multi(sources ...Source) Source {
	return &multiSource{sources: sources}
}

type multiSource struct {
	sources []Source
}

// Run implements Source.
func (m *multiSource) Run(parent context.Context, apply ApplyFunc) error {
	ctx, cancel := context.WithCancelCause(parent)
	defer cancel(nil)

	var (
		mu      sync.Mutex
		latest  = make([]map[Name]Rollout, len(m.sources))
		runners sync.WaitGroup
	)

	for i, source := range m.sources {
		runners.Go(func() {
			err := source.Run(ctx, func(overrides map[Name]Rollout) error {
				// Holding the lock across apply keeps updates ordered.
				mu.Lock()
				defer mu.Unlock()

				latest[i] = overrides

				return apply(mergeOverrides(latest))
			})
			if err != nil {
				cancel(err)
			}
		})
	}

	runners.Wait()

	if parent.Err() != nil {
		return nil
	}

	return context.Cause(ctx)
}

func mergeOverrides(sets []map[Name]Rollout) map[Name]Rollout {
	out := make(map[Name]Rollout)
	for _, set := range sets {
		maps.Copy(out, set)
	}

	return out
}
