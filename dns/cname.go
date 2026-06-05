package dns

import (
	"context"
	"fmt"
	"net"

	"codeberg.org/miekg/dns/dnsutil"
)

// cnameResolver decorates another Resolver to follow CNAME chains itself.
//
// A query for an address record (A/AAAA) is often answered with a CNAME that
// points at another name. A recursive resolver chases that chain for us and
// returns the terminal address records in the same response, but we can't assume
// the wrapped resolver is recursive, so we follow the chain ourselves.
//
// We do so lazily: if the response already contains the next link in the chain
// (another CNAME) or the terminal record we asked for, no extra query is made.
// Only a genuinely missing link triggers a follow-up query. In the common case
// of a recursive resolver returning the whole chain at once, this adds no DNS
// traffic at all.
type cnameResolver struct {
	addr     string
	resolver Resolver
}

// newCNameResolver wraps resolver in a cnameResolver. addr (defaulting to port
// 53 when none is given) is used only as the resolver's Name; queries go
// through the wrapped resolver.
func newCNameResolver(addr string, resolver Resolver) *cnameResolver {
	_, _, err := net.SplitHostPort(addr)
	if err != nil {
		addr = net.JoinHostPort(addr, "53")
	}

	return &cnameResolver{
		addr:     addr,
		resolver: resolver,
	}
}

// ResolveType resolves host via the wrapped resolver, then follows any CNAME
// chain in the answer (see followChain) so the result includes the terminal
// records of the requested type whenever they are reachable.
func (c *cnameResolver) ResolveType(
	ctx context.Context,
	host string,
	qtype RecordType,
) ([]Record, TruncationStatus, error) {
	records, trunc, err := c.resolver.ResolveType(ctx, host, qtype)
	if err != nil {
		return nil, trunc, err
	}

	return c.followChain(ctx, host, qtype, records, trunc)
}

// Name returns the resolver address, identifying the underlying server.
func (c *cnameResolver) Name() string {
	return c.addr
}

// followChain walks the CNAME chain starting at host, returning the original
// records plus any additional records fetched while chasing CNAMEs toward the
// requested type. It never drops records: the walk only decides whether more
// queries are needed and detects loops/over-long chains.
func (c *cnameResolver) followChain(
	ctx context.Context,
	host string,
	qtype RecordType,
	records []Record,
	trunc TruncationStatus,
) ([]Record, TruncationStatus, error) {
	// all accumulates every record we've seen across hops, so the caller gets
	// the whole flattened chain.
	all := records

	// target is the name we're currently trying to resolve; it advances down the
	// chain each time we follow a CNAME. Names are compared in canonical form
	// (lowercase, fully-qualified) because DNS names are case-insensitive and
	// responses are fully-qualified while host may not be.
	target := dnsutil.Canonical(host)

	// visited guards against CNAME loops (a -> b -> a) and re-querying a name we
	// have already expanded.
	visited := map[string]bool{target: true}

	for range maxCNAMEDepth {
		// If we already hold a record of the requested type for the current
		// target, we're done. This covers a direct answer, a recursive resolver
		// that returned the terminal record, and a CNAME query (whose answer is
		// the CNAME itself, so it returns here without chasing further).
		if hasRecordOfType(all, target, qtype) {
			return all, trunc, nil
		}

		// Otherwise look for a CNAME telling us where target points next.
		next, ok := cnameTarget(all, target)
		if !ok {
			// No terminal record and no CNAME to follow: nothing more we can do.
			// Return what we have and let the caller decide if it's usable.
			return all, trunc, nil
		}

		if visited[next] {
			return nil, trunc, fmt.Errorf("%w: %s -> %s", ErrCNAMELoop, target, next)
		}

		visited[next] = true
		target = next

		// If the response already carried records for the next name -- either the
		// terminal record or the subsequent CNAME -- don't query again. This is
		// the common case with a recursive resolver that returns the full chain.
		if hasRecordsFor(all, target) {
			continue
		}

		// The link is missing, so the wrapped resolver wasn't recursive (or chose
		// not to chase). Fetch the next hop ourselves.
		more, hopTrunc, err := c.resolver.ResolveType(ctx, target, qtype)
		if err != nil {
			return nil, hopTrunc, fmt.Errorf("following CNAME to %q: %w", target, err)
		}

		all = append(all, more...)
		trunc = hopTrunc
	}

	return nil, trunc, fmt.Errorf("%w: exceeded %d hops resolving %q", ErrCNAMEChainTooLong, maxCNAMEDepth, host)
}

// hasRecordOfType reports whether records holds a record of the given type whose
// name matches target (compared canonically).
func hasRecordOfType(records []Record, target string, qtype RecordType) bool {
	for _, r := range records {
		if r.Type == qtype && dnsutil.Canonical(r.Name) == target {
			return true
		}
	}

	return false
}

// hasRecordsFor reports whether records holds any record whose name matches
// target (compared canonically).
func hasRecordsFor(records []Record, target string) bool {
	for _, r := range records {
		if dnsutil.Canonical(r.Name) == target {
			return true
		}
	}

	return false
}

// cnameTarget returns the canonical target of the CNAME record for target, if
// one is present.
func cnameTarget(records []Record, target string) (string, bool) {
	for _, r := range records {
		if r.Type == TypeCNAME && dnsutil.Canonical(r.Name) == target {
			return dnsutil.Canonical(r.Value), true
		}
	}

	return "", false
}
