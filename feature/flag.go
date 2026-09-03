package feature

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/amp-labs/amp-common/contexts"
)

// Flag is a feature flag defined in code. Obtain one with Define. A Flag is
// safe for concurrent use and cheap to evaluate: reads never take a lock.
type Flag struct {
	name        Name
	description string
	owner       string
	dfl         Rollout
	registry    *Registry
	state       atomic.Pointer[compiled]
}

// FlagOption configures a Flag at definition time.
type FlagOption func(*Flag)

// Info describes a flag for listings such as the HTTP Handler.
type Info struct {
	Name        Name    `json:"name"`
	Description string  `json:"description,omitempty"`
	Owner       string  `json:"owner,omitempty"`
	Default     Rollout `json:"default"`
	Current     Rollout `json:"current"`
	Overridden  bool    `json:"overridden"`
}

// Description attaches human-readable documentation to a flag.
func Description(text string) FlagOption {
	return func(f *Flag) {
		f.description = text
	}
}

// Owner records who owns the flag, a team or a person, for listings.
func Owner(text string) FlagOption {
	return func(f *Flag) {
		f.owner = text
	}
}

// Name returns the flag's name.
func (f *Flag) Name() Name {
	return f.name
}

// Description returns the documentation attached with the Description option.
func (f *Flag) Description() string {
	return f.description
}

// Owner returns the owner attached with the Owner option.
func (f *Flag) Owner() string {
	return f.owner
}

// Default returns the rollout defined in code.
func (f *Flag) Default() Rollout {
	return f.dfl.clone()
}

// Current returns the rollout in effect right now.
func (f *Flag) Current() Rollout {
	return f.state.Load().rollout.clone()
}

// Overridden reports whether the current rollout differs from the code default.
func (f *Flag) Overridden() bool {
	return f.state.Load().overridden
}

// Info returns a description of the flag and its state.
func (f *Flag) Info() Info {
	state := f.state.Load()

	return Info{
		Name:        f.name,
		Description: f.description,
		Owner:       f.owner,
		Default:     f.dfl.clone(),
		Current:     state.rollout.clone(),
		Overridden:  state.overridden,
	}
}

// Enabled reports whether the flag is on for the subject in ctx. The subject
// is combined with the registry's global attributes and the automatic Host,
// Stage, and Region dimensions. A nil ctx is allowed.
func (f *Flag) Enabled(ctx context.Context) bool {
	return f.Evaluate(ctx, nil).Enabled
}

// EnabledFor is Enabled with an explicit subject, whose attributes take
// precedence over the subject in ctx.
func (f *Flag) EnabledFor(ctx context.Context, subject Subject) bool {
	return f.Evaluate(ctx, subject).Enabled
}

// Evaluate returns the full result including the reason, for logging and
// debugging. The subject may be nil.
func (f *Flag) Evaluate(ctx context.Context, subject Subject) Result {
	ctx = contexts.EnsureContext(ctx)

	resolve := f.registry.resolver(ctx, subject) //nolint:contextcheck // EnsureContext may substitute Background
	result := f.state.Load().evaluate(f.name, f.registry.hasher, resolve)
	recordEvaluation(result)

	return result
}

// Set replaces the flag's rollout at runtime. The error wraps
// ErrInvalidRollout if the rollout is malformed. A later Registry.Load from a
// Source replaces whatever Set installed.
func (f *Flag) Set(rollout Rollout) error {
	err := rollout.Validate()
	if err != nil {
		return fmt.Errorf("flag %q: %w", f.name, err)
	}

	f.registry.apply(f, rollout, sourceSet)

	return nil
}

// Reset restores the flag to its code default.
func (f *Flag) Reset() {
	f.registry.apply(f, f.dfl, sourceReset)
}
