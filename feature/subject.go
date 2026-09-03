package feature

import (
	"context"
	"maps"

	"github.com/amp-labs/amp-common/contexts"
)

// Name is a flag's stable identifier. It is also part of the hash input for
// percentage rollouts, so renaming a flag reshuffles its buckets.
type Name string

// Dimension names a population that a rollout can slice: users, tenants,
// hosts, and so on. Any string is a valid Dimension. Programs define the
// dimensions that exist in their own domain, typically as constants:
//
//	const (
//		User   feature.Dimension = "user"
//		Tenant feature.Dimension = "tenant"
//	)
//
// Only the dimensions that every deployment has are built in: Host, Stage,
// and Region.
type Dimension string

const (
	// Host identifies the machine or container the process runs on, so a
	// percentage of Host is a percentage of the fleet. If no value is provided,
	// it defaults to the hostname, which is the pod name on Kubernetes.
	Host Dimension = "host"

	// Stage is the deployment stage (local, test, dev, staging, prod) as
	// defined by the stage package. If no value is provided, it is filled in
	// from stage.Current(ctx).
	Stage Dimension = "stage"

	// Region is the data region the process is deployed in (us, eu) as defined
	// by the region package. If no value is provided, it is filled in from
	// region.Current(ctx), which is "unknown" when REGION is not configured.
	Region Dimension = "region"
)

// Subject describes who or what is asking for a flag: a value for each
// dimension that is known. Empty values are treated as absent, so an anonymous
// request never collapses into a single bucket.
type Subject map[Dimension]string

// With returns a copy of the subject with the given attribute set.
// The receiver is not modified; a nil receiver is allowed.
func (s Subject) With(dim Dimension, value string) Subject {
	out := make(Subject, len(s)+1)
	maps.Copy(out, s)
	out[dim] = value

	return out
}

// Merge returns a copy of the subject with all attributes from other applied
// on top. Attributes in other win. The receiver is not modified.
func (s Subject) Merge(other Subject) Subject {
	if len(other) == 0 {
		return s
	}

	out := make(Subject, len(s)+len(other))
	maps.Copy(out, s)
	maps.Copy(out, other)

	return out
}

// Get returns the value for the dimension and whether it is present.
// Empty strings count as absent.
func (s Subject) Get(dim Dimension) (string, bool) {
	value, ok := s[dim]
	if !ok || value == "" {
		return "", false
	}

	return value, true
}

// contextKey is a typed key for storing the subject in a context.
type contextKey string

// subjectContextKey is the key under which the request-scoped Subject is stored.
const subjectContextKey contextKey = "feature.subject"

// WithSubject returns a context carrying the subject, merged over any subject
// already in the context. Middleware typically calls this once per request so
// that code deeper in the stack can call Flag.Enabled(ctx) without having to
// thread user or tenant IDs through just for flags.
func WithSubject(ctx context.Context, subject Subject) context.Context {
	existing := SubjectFrom(ctx)

	return contexts.WithValue(ctx, subjectContextKey, existing.Merge(subject))
}

// WithAttribute returns a context carrying the given attribute in addition to
// any subject already in the context.
func WithAttribute(ctx context.Context, dim Dimension, value string) context.Context {
	existing := SubjectFrom(ctx)

	return contexts.WithValue(ctx, subjectContextKey, existing.With(dim, value))
}

// SubjectFrom returns the subject stored in the context, or nil if there is
// none. The returned map must not be modified.
func SubjectFrom(ctx context.Context) Subject {
	subject, _ := contexts.GetValue[contextKey, Subject](ctx, subjectContextKey)

	return subject
}
