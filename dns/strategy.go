package dns

import "context"

// Strategy decides how the answers from several resolvers are combined into a
// single result. The [Dialer] calls ResolveType once per query type, passing
// the full resolver set; the strategy is responsible for querying them (in
// parallel or in sequence) and reconciling their answers.
type Strategy interface {
	// ResolveType resolves host for the given record type using the supplied
	// resolvers, returning the reconciled records or an error when no acceptable
	// answer was obtained.
	ResolveType(ctx context.Context, host string, qtype RecordType, resolvers []Resolver) ([]Record, error)
}

// Race queries every resolver concurrently and returns the first successful
// answer, canceling the rest. It optimizes for latency and is the default
// strategy.
type Race struct{}

// Consensus requires at least MinAgreement resolvers to return identical
// answers before accepting a result, guarding against a single rogue or
// poisoned resolver.
type Consensus struct {
	// MinAgreement is the number of resolvers that must agree. When zero or
	// negative it defaults to a strict majority of the resolver set.
	MinAgreement int
	// IgnoreTTL compares records by value only, treating differing TTLs as equal.
	IgnoreTTL bool
}

// Fallback tries resolvers in the order given and returns the first successful
// answer, moving on only when one fails. It favors a preferred resolver while
// tolerating its outages.
type Fallback struct{}

// Compare queries every resolver and returns the first answer, but reports when
// the resolvers disagree. It is intended for monitoring/auditing rather than
// for picking a winner.
type Compare struct {
	// OnDiscrepancy, if set, is invoked with every resolver's answer whenever the
	// resolvers do not all agree.
	OnDiscrepancy func(host string, qtype RecordType, results map[string][]Record)
	// IgnoreTTL compares records by value only, treating differing TTLs as equal.
	IgnoreTTL bool
}
