package dns

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/amp-labs/amp-common/retry"
)

// LookupCoordinator turns a "host:port" address into the set of IP addresses a
// caller should attempt to connect to. It owns the resolution pipeline that
// [Dialer] uses -- strategy-coordinated queries across the configured
// resolvers, TTL-based caching, and network-family filtering -- but performs no
// dialing itself, so it can also be used standalone wherever resolved IPs are
// needed (e.g., custom dialers or connection pools). Build one with
// [NewLookupCoordinator].
type LookupCoordinator struct {
	// resolvers is the list of DNS resolvers we'll query (e.g., UDP resolvers for 8.8.8.8, 1.1.1.1)
	resolvers []Resolver

	// filter vets IP literals passed directly to Lookup. Hostname lookups don't
	// use it here: those are filtered inside the resolver stack, where each
	// resolver is wrapped by a filterResolver (see createLookupCoordinator). IP
	// literals bypass that stack entirely, so this is the only place they can be
	// checked.
	filter Filter

	// strategy determines how we coordinate queries (Race, Fallback, Consensus, Compare)
	strategy Strategy

	// cache stores DNS lookup results with TTL-based expiration, disabled by default
	cache *dnsCache

	// retryOptions configures how a failed lookup is retried (see
	// WithLookupRetryOptions); empty means no retries
	retryOptions []retry.Option
}

// NewLookupCoordinator builds a [LookupCoordinator] from the given options. It
// accepts the same options as [NewDialer] (dialer-only options such as
// [WithDialer] are ignored) and returns [ErrNoResolvers] if no resolvers were
// configured via [WithResolvers].
func NewLookupCoordinator(opts ...Option) (*LookupCoordinator, error) {
	o := newOptions()

	for _, opt := range opts {
		opt(o)
	}

	return o.createLookupCoordinator()
}

// Lookup resolves addr ("host:port") into the IP addresses suitable for the
// requested network, plus the parsed port. The network selects the address
// family: "tcp4"/"udp4" return only IPv4, "tcp6"/"udp6" only IPv6, and
// anything else (typically "tcp"/"udp") returns all addresses ordered IPv4
// first. If host is already an IP literal it is returned as-is after passing
// the configured filter (no DNS queries are made); note that literals are not
// checked against the network's address family -- a mismatch surfaces at dial
// time instead.
func (l *LookupCoordinator) Lookup(ctx context.Context, network, addr string) ([]net.IP, string, error) {
	host, port, err := parseHostAndPort(addr)
	if err != nil {
		return nil, "", err
	}

	// If host is already an IP address, use it directly without DNS lookup. No point
	// in doing DNS resolution for something that's already an IP.
	if ip := net.ParseIP(host); ip != nil {
		return l.lookupLiteralIP(ip, port, network)
	}

	ips, err := retry.DoValue[[]net.IP](ctx, func(ctx context.Context) ([]net.IP, error) {
		vals, lookupErr := l.lookupIPs(ctx, host)
		if lookupErr != nil {
			return nil, fmt.Errorf("DNS lookup failed for %q: %w", host, lookupErr)
		}

		return vals, nil
	}, l.retryOptions...)
	if err != nil {
		return nil, "", fmt.Errorf("DNS lookup failed for %q: %w", host, err)
	}

	filteredIPs := filterIPs(ips, network)

	if len(filteredIPs) == 0 {
		return nil, "", fmt.Errorf("%w for %s (network: %s)", errNoSuitableIPs, host, network)
	}

	return filteredIPs, port, nil
}

// lookup queries A, AAAA, and CNAME records for host concurrently and returns
// the union of all records found. Per-type failures are logged and skipped
// rather than failing the whole lookup, so an IPv4-only or IPv6-only host still
// resolves.
func (l *LookupCoordinator) lookup(ctx context.Context, host string) []Record {
	queryTypes := []RecordType{TypeA, TypeAAAA, TypeCNAME}

	type result struct {
		records []Record
		err     error
		qtype   RecordType
	}

	results := make(chan result, len(queryTypes))

	for _, qtype := range queryTypes {
		go func(qt RecordType) {
			records, err := l.strategy.ResolveType(ctx, host, qt, l.resolvers)
			results <- result{
				records: records,
				err:     err,
				qtype:   qt,
			}
		}(qtype)
	}

	allRecords := make([]Record, 0, len(queryTypes)*recordsPerTypeHint)

	for range queryTypes {
		res := <-results
		if res.err != nil {
			logDebug(ctx, "query type failed",
				"type", res.qtype.String(),
				"error", res.err.Error())

			continue
		}

		allRecords = append(allRecords, res.records...)
	}

	return allRecords
}

// lookupIPs returns the IP addresses for host, serving from cache when
// possible. On a miss it performs a full lookup, extracts the A/AAAA addresses,
// and caches them using the smallest record TTL (capped at 300s and then
// clamped by the cache's own bounds). It returns an error if no addresses are
// found.
func (l *LookupCoordinator) lookupIPs(ctx context.Context, host string) ([]net.IP, error) {
	if cached := l.cache.getIPs(host); cached != nil {
		logDebug(ctx, "IP cache hit",
			"host", host,
			"ips", len(cached))

		return cached, nil
	}

	logDebug(ctx, "IP cache miss",
		"host", host)

	records := l.lookup(ctx, host)

	ips := make([]net.IP, 0, len(records))
	minTTL := uint32(maxCachedTTLSeconds)

	for _, record := range records {
		if record.Type == TypeA || record.Type == TypeAAAA {
			ip := net.ParseIP(record.Value)
			if ip != nil {
				ips = append(ips, ip)

				if record.TTL < minTTL {
					minTTL = record.TTL
				}
			}
		}
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("%w for %s", errNoIPAddresses, host)
	}

	l.cache.setIPs(host, ips, time.Duration(minTTL)*time.Second)

	return ips, nil
}

// lookupLiteralIP handles the case where the caller passed an IP literal
// instead of a hostname. There is nothing to resolve, but the configured
// filter (if any) still gets a say: the IP is converted to a synthetic A/AAAA
// record so the same Accept predicate used for resolved records can vet it.
// Rejection is reported as errNoSuitableIPs, matching the hostname path.
func (l *LookupCoordinator) lookupLiteralIP(ip net.IP, port string, network string) ([]net.IP, string, error) {
	if l.filter != nil {
		record, ok := ipToRecord(ip)
		if !ok {
			return nil, "", fmt.Errorf("%w for %s (network: %s)", errNoSuitableIPs, ip.String(), network)
		}

		if !l.filter.Accept(ip.String(), record) {
			return nil, "", fmt.Errorf("%w for %s (network: %s)", errNoSuitableIPs, ip.String(), network)
		}
	}

	return []net.IP{ip}, port, nil
}
