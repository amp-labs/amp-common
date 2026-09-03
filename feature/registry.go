package feature

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/amp-labs/amp-common/logger"
	"github.com/amp-labs/amp-common/region"
	"github.com/amp-labs/amp-common/stage"
)

// Change sources, used in logs and metrics.
const (
	sourceSet   = "set"
	sourceReset = "reset"
	sourceLoad  = "load"
)

// Registry holds flag definitions and their current rollouts. Most programs
// use the package-level Default registry through Define; create your own with
// NewRegistry for tests or for fully isolated flag sets.
type Registry struct {
	mu      sync.RWMutex
	flags   map[Name]*Flag
	pending map[Name]Rollout
	globals atomic.Pointer[Subject]
	hasher  Hasher
}

// Option configures a Registry.
type Option func(*Registry)

// NewRegistry creates an empty registry.
func NewRegistry(opts ...Option) *Registry {
	reg := &Registry{
		flags:   make(map[Name]*Flag),
		pending: make(map[Name]Rollout),
		hasher:  defaultHasher,
	}
	reg.globals.Store(&Subject{})

	for _, opt := range opts {
		opt(reg)
	}

	return reg
}

// defaultRegistry backs the package-level functions.
var defaultRegistry = NewRegistry()

// Default returns the package-level registry used by Define and the other
// package-level functions.
func Default() *Registry {
	return defaultRegistry
}

// WithGlobalAttribute sets a process-wide subject attribute, such as the Host
// identity. Global attributes are overridden by the subject in the context and
// by explicit subjects.
func WithGlobalAttribute(dim Dimension, value string) Option {
	return func(r *Registry) {
		r.SetGlobalAttribute(dim, value)
	}
}

// WithHasher replaces the hash function used for percentage bucketing.
// A nil hasher is ignored.
func WithHasher(hasher Hasher) Option {
	return func(r *Registry) {
		if hasher != nil {
			r.hasher = hasher
		}
	}
}

// Define registers a flag with its code default. It panics if the name is
// empty, the default is invalid, or the name is already defined, since all
// three are programming errors best caught at startup. If a Source already
// loaded an override for this name, it takes effect immediately.
func (r *Registry) Define(name Name, dfl Rollout, opts ...FlagOption) *Flag {
	if name == "" {
		panic("feature: flag name is required")
	}

	err := dfl.Validate()
	if err != nil {
		panic(fmt.Sprintf("feature: flag %q: %v", name, err))
	}

	def := &Flag{name: name, dfl: dfl.clone(), registry: r}
	for _, opt := range opts {
		opt(def)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.flags[name]; exists {
		panic(fmt.Sprintf("feature: flag %q is defined twice", name))
	}

	current := dfl
	if override, ok := r.pending[name]; ok {
		current = override

		delete(r.pending, name)
	}

	def.state.Store(compile(name, current, !current.Equal(dfl)))
	r.flags[name] = def

	return def
}

// Lookup returns the flag with the given name, if it has been defined.
func (r *Registry) Lookup(name Name) (*Flag, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	def, ok := r.flags[name]

	return def, ok
}

// Flags returns every defined flag, sorted by name.
func (r *Registry) Flags() []*Flag {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]*Flag, 0, len(r.flags))
	for _, def := range r.flags {
		out = append(out, def)
	}

	slices.SortFunc(out, func(a, b *Flag) int {
		return strings.Compare(string(a.name), string(b.name))
	})

	return out
}

// Infos returns Info for every defined flag, sorted by name.
func (r *Registry) Infos() []Info {
	flags := r.Flags()

	out := make([]Info, 0, len(flags))
	for _, def := range flags {
		out = append(out, def.Info())
	}

	return out
}

// Snapshot returns the rollout currently in effect for every defined flag.
func (r *Registry) Snapshot() map[Name]Rollout {
	flags := r.Flags()

	out := make(map[Name]Rollout, len(flags))
	for _, def := range flags {
		out[def.name] = def.Current()
	}

	return out
}

// Set replaces a flag's rollout at runtime. The error wraps ErrUndefined if
// the flag has not been defined and ErrInvalidRollout if the rollout is
// malformed.
func (r *Registry) Set(name Name, rollout Rollout) error {
	def, ok := r.Lookup(name)
	if !ok {
		return fmt.Errorf("%w: %q", ErrUndefined, name)
	}

	return def.Set(rollout)
}

// Reset restores a flag to its code default. The error wraps ErrUndefined if
// the flag has not been defined.
func (r *Registry) Reset(name Name) error {
	def, ok := r.Lookup(name)
	if !ok {
		return fmt.Errorf("%w: %q", ErrUndefined, name)
	}

	def.Reset()

	return nil
}

// Load replaces every runtime override with the given set, which is what a
// Source calls on each change. Flags absent from the set revert to their code
// defaults. Names that are not (yet) defined are remembered and applied if
// they are defined later. Invalid rollouts are skipped, leaving those flags
// untouched, and reported in the returned error, which wraps ErrInvalidRollout.
func (r *Registry) Load(overrides map[Name]Rollout) error {
	valid := make(map[Name]Rollout, len(overrides))
	invalid := make(map[Name]struct{})

	var errs []error

	for name, rollout := range overrides {
		err := rollout.Validate()
		if err != nil {
			invalid[name] = struct{}{}

			errs = append(errs, fmt.Errorf("flag %q: %w", name, err))

			continue
		}

		valid[name] = rollout
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.pending = make(map[Name]Rollout)

	var unknown []Name

	for name, rollout := range valid {
		if _, defined := r.flags[name]; !defined {
			r.pending[name] = rollout

			unknown = append(unknown, name)
		}
	}

	for name, def := range r.flags {
		if _, skip := invalid[name]; skip {
			continue
		}

		rollout, ok := valid[name]
		if !ok {
			rollout = def.dfl
		}

		r.apply(def, rollout, sourceLoad)
	}

	if len(unknown) > 0 {
		slices.Sort(unknown)
		logger.Get().Info("loaded overrides for feature flags that are not defined; they apply if defined later",
			"flags", unknown)
	}

	return errors.Join(errs...)
}

// SetGlobalAttribute sets a process-wide subject attribute at runtime, such as
// the Host identity. An empty value removes the attribute.
func (r *Registry) SetGlobalAttribute(dim Dimension, value string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	next := r.globals.Load().With(dim, value)
	if value == "" {
		delete(next, dim)
	}

	r.globals.Store(&next)
}

// GlobalAttributes returns a copy of the process-wide subject attributes.
func (r *Registry) GlobalAttributes() Subject {
	return maps.Clone(*r.globals.Load())
}

// Enabled reports whether the named flag is on for the subject in ctx.
// Undefined flags are off. Prefer Flag.Enabled when the flag is known at
// compile time; this is for names that come from configuration.
func (r *Registry) Enabled(ctx context.Context, name Name) bool {
	return r.Evaluate(ctx, name, nil).Enabled
}

// Evaluate returns the full result for the named flag. Undefined flags yield
// ReasonUndefined.
func (r *Registry) Evaluate(ctx context.Context, name Name, subject Subject) Result {
	def, ok := r.Lookup(name)
	if !ok {
		undefinedTotal.Inc()
		warnUndefined(ctx, name)

		return Result{Flag: name, Reason: ReasonUndefined, Bucket: -1}
	}

	return def.Evaluate(ctx, subject)
}

// Watch feeds the registry from a Source until ctx is done. It blocks, so run
// it in a goroutine. It returns nil on normal shutdown and the Source's error
// otherwise.
func (r *Registry) Watch(ctx context.Context, source Source) error {
	return source.Run(ctx, r.Load)
}

// apply installs a rollout on a flag if it differs from the current one.
func (r *Registry) apply(def *Flag, rollout Rollout, source string) {
	previous := def.state.Load()
	if previous != nil && previous.rollout.Equal(rollout) {
		return
	}

	def.state.Store(compile(def.name, rollout, !rollout.Equal(def.dfl)))
	recordChange(def.name, source)

	previousText := "<none>"
	if previous != nil {
		previousText = previous.rollout.String()
	}

	logger.Get().Info("feature flag rollout changed",
		"flag", def.name, "source", source, "previous", previousText, "current", rollout.String())
}

// resolver builds the lookup used during one evaluation. Precedence, highest
// first: the explicit subject, the subject in ctx, global attributes, then the
// automatic values for Host (the hostname), Stage (stage.Current), and
// Region (region.Current).
func (r *Registry) resolver(ctx context.Context, explicit Subject) resolver {
	inContext := SubjectFrom(ctx)
	globals := *r.globals.Load()

	return func(dim Dimension) (string, bool) {
		if value, ok := explicit.Get(dim); ok {
			return value, true
		}

		if value, ok := inContext.Get(dim); ok {
			return value, true
		}

		if value, ok := globals.Get(dim); ok {
			return value, true
		}

		if dim == Host {
			identity := processIdentity()

			return identity, identity != ""
		}

		if dim == Stage {
			return string(stage.Current(ctx)), true
		}

		if dim == Region {
			return string(region.Current(ctx)), true
		}

		return "", false
	}
}

// processIdentity is the default Host value: the hostname, which is the pod
// name on Kubernetes. It is empty, and therefore absent, if it cannot be read.
var processIdentity = sync.OnceValue(func() string {
	host, err := os.Hostname()
	if err != nil {
		return ""
	}

	return host
})

// undefinedWarned dedupes the undefined-flag warning to once per name.
var undefinedWarned sync.Map

func warnUndefined(ctx context.Context, name Name) {
	if _, seen := undefinedWarned.LoadOrStore(name, struct{}{}); seen {
		return
	}

	logger.Get(ctx).Warn("evaluated a feature flag that was never defined; treating it as off", "flag", name)
}

// Define registers a flag in the Default registry. See Registry.Define.
func Define(name Name, dfl Rollout, opts ...FlagOption) *Flag {
	return defaultRegistry.Define(name, dfl, opts...)
}

// Lookup finds a flag in the Default registry. See Registry.Lookup.
func Lookup(name Name) (*Flag, bool) {
	return defaultRegistry.Lookup(name)
}

// Enabled evaluates a flag in the Default registry by name. See Registry.Enabled.
func Enabled(ctx context.Context, name Name) bool {
	return defaultRegistry.Enabled(ctx, name)
}

// Set changes a flag in the Default registry. See Registry.Set.
func Set(name Name, rollout Rollout) error {
	return defaultRegistry.Set(name, rollout)
}

// Reset restores a flag in the Default registry. See Registry.Reset.
func Reset(name Name) error {
	return defaultRegistry.Reset(name)
}

// Load replaces the overrides in the Default registry. See Registry.Load.
func Load(overrides map[Name]Rollout) error {
	return defaultRegistry.Load(overrides)
}

// Watch feeds the Default registry from a Source. See Registry.Watch.
func Watch(ctx context.Context, source Source) error {
	return defaultRegistry.Watch(ctx, source)
}
