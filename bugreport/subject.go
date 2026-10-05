package bugreport

import (
	"context"
	"log/slog"
	"maps"

	"github.com/amp-labs/amp-common/contexts"
)

// Key names a subject attribute: something that identifies who reported the
// bug or what it concerns. Any string is a valid Key. Programs define the keys
// that exist in their own domain, typically as constants:
//
//	const (
//		Org     bugreport.Key = "org_id"
//		Project bugreport.Key = "project_id"
//	)
type Key string

// Subject identifies who or what a bug report concerns: a value for each key
// that is known. Empty values are treated as absent and are not logged.
type Subject map[Key]string

// With returns a copy of the subject with the given attribute set.
// The receiver is not modified; a nil receiver is allowed.
func (s Subject) With(key Key, value string) Subject {
	out := make(Subject, len(s)+1)
	maps.Copy(out, s)
	out[key] = value

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

// attrs returns the non-empty attributes as slog attributes.
func (s Subject) attrs() []any {
	attrs := make([]any, 0, len(s))

	for key, value := range s {
		if value != "" {
			attrs = append(attrs, slog.String(string(key), value))
		}
	}

	return attrs
}

// contextKey is a typed key for storing the subject in a context.
type contextKey string

// subjectContextKey is the key under which the request-scoped Subject is stored.
const subjectContextKey contextKey = "bugreport.subject"

// WithSubject returns a context carrying the subject, merged over any subject
// already in the context. Middleware typically calls this as identifiers
// become known, so that a report submitted deeper in the stack is filled in
// automatically.
func WithSubject(ctx context.Context, subject Subject) context.Context {
	existing := SubjectFrom(ctx)

	return contexts.WithValue(ctx, subjectContextKey, existing.Merge(subject))
}

// WithAttribute returns a context carrying the given attribute in addition to
// any subject already in the context.
func WithAttribute(ctx context.Context, key Key, value string) context.Context {
	existing := SubjectFrom(ctx)

	return contexts.WithValue(ctx, subjectContextKey, existing.With(key, value))
}

// SubjectFrom returns the subject stored in the context, or nil if there is
// none. The returned map must not be modified.
func SubjectFrom(ctx context.Context) Subject {
	subject, _ := contexts.GetValue[contextKey, Subject](ctx, subjectContextKey)

	return subject
}
