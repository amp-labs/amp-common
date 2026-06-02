package dns

import (
	"context"
	"fmt"
	"net"
	"time"
)

// Dialer resolves hostnames using its configured resolvers and strategy, then
// dials the resulting IP addresses. Its [Dialer.DialContext] method matches the
// signature of [net.Dialer.DialContext], so it can be assigned directly to an
// [net/http.Transport]'s DialContext field. Build one with [NewDialer].
type Dialer struct {
	// resolvers is the list of DNS resolvers we'll query (e.g., UDP resolvers for 8.8.8.8, 1.1.1.1)
	resolvers []Resolver

	// strategy determines how we coordinate queries (Race, Fallback, Consensus, Compare)
	strategy Strategy

	// timeout is the per-query timeout we apply to individual DNS queries
	timeout time.Duration

	// poolSize is the max connections to pool per resolver, defaults to 4
	poolSize int

	// dialer is reused for TCP/UDP connections to avoid allocating a new one each time
	dialer *net.Dialer

	// cache stores DNS lookup results with TTL-based expiration, disabled by default
	cache *dnsCache
}

// NewDialer builds a [Dialer] from the given options. It returns
// [ErrNoResolvers] if no resolvers were configured via [WithResolvers].
func NewDialer(opts ...Option) (*Dialer, error) {
	o := newOptions()

	for _, opt := range opts {
		opt(o)
	}

	return o.createDialer()
}

// DialContext resolves the host portion of addr (unless it is already an IP)
// and dials the resulting addresses for the requested network, returning the
// first connection that succeeds. The network is honored when selecting between
// IPv4 and IPv6 results: "tcp4"/"udp4" use only IPv4, "tcp6"/"udp6" only IPv6,
// and the generic "tcp"/"udp" try IPv4 first then IPv6. It mirrors the
// signature of [net.Dialer.DialContext].
func (r *Dialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	// Split addr into host and port (standard net package format)
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid address %q: %w", addr, err)
	}

	// If host is already an IP address, use it directly without DNS lookup. No point
	// in doing DNS resolution for something that's already an IP.
	if ip := net.ParseIP(host); ip != nil {
		return r.dialer.DialContext(ctx, network, addr)
	}

	// Perform DNS lookup using whichever strategy is configured
	ips, err := r.lookupIPs(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("DNS lookup failed for %s: %w", host, err)
	}

	// Filter IPs based on network type
	var filteredIPs []net.IP
	switch network {
	case "tcp4", "udp4":
		// Only use IPv4 addresses, the caller explicitly asked for v4
		for _, ip := range ips {
			if ip.To4() != nil {
				filteredIPs = append(filteredIPs, ip)
			}
		}
	case "tcp6", "udp6":
		// Only use IPv6 addresses, the caller explicitly asked for v6
		for _, ip := range ips {
			if ip.To4() == nil && ip.To16() != nil {
				filteredIPs = append(filteredIPs, ip)
			}
		}
	default:
		// For "tcp" and "udp", use all IPs we got. Try IPv4 first for better compatibility,
		// more things support IPv4 than IPv6 in practice.
		filteredIPs = make([]net.IP, 0, len(ips))
		// Add IPv4 addresses first
		for _, ip := range ips {
			if ip.To4() != nil {
				filteredIPs = append(filteredIPs, ip)
			}
		}
		// Then add IPv6 addresses
		for _, ip := range ips {
			if ip.To4() == nil && ip.To16() != nil {
				filteredIPs = append(filteredIPs, ip)
			}
		}
	}

	if len(filteredIPs) == 0 {
		return nil, fmt.Errorf("no suitable IP addresses found for %s (network: %s)", host, network)
	}

	var lastErr error

	for _, ip := range filteredIPs {
		ipAddr := net.JoinHostPort(ip.String(), portStr)
		conn, err := r.dialer.DialContext(ctx, network, ipAddr)
		if err == nil {
			return conn, nil
		}

		lastErr = err

		logDebug(ctx, "connection failed, trying next IP",
			"ip", ip.String(),
			"error", err.Error())
	}

	return nil, fmt.Errorf("failed to connect to %s: %w", host, lastErr)
}

// lookup queries A, AAAA, and CNAME records for host concurrently and returns
// the union of all records found. Per-type failures are logged and skipped
// rather than failing the whole lookup, so an IPv4-only or IPv6-only host still
// resolves.
func (r *Dialer) lookup(ctx context.Context, host string) ([]Record, error) {
	queryTypes := []RecordType{TypeA, TypeAAAA, TypeCNAME}

	type result struct {
		records []Record
		err     error
		qtype   RecordType
	}

	results := make(chan result, len(queryTypes))

	for _, qtype := range queryTypes {
		go func(qt RecordType) {
			records, err := r.strategy.ResolveType(ctx, host, qt, r.resolvers)
			results <- result{
				records: records,
				err:     err,
				qtype:   qt,
			}
		}(qtype)
	}

	allRecords := make([]Record, 0, len(queryTypes)*4)

	for i := 0; i < len(queryTypes); i++ {
		res := <-results
		if res.err != nil {
			logDebug(ctx, "query type failed",
				"type", res.qtype.String(),
				"error", res.err.Error())

			continue
		}

		allRecords = append(allRecords, res.records...)
	}

	return allRecords, nil
}

// lookupIPs returns the IP addresses for host, serving from cache when
// possible. On a miss it performs a full lookup, extracts the A/AAAA addresses,
// and caches them using the smallest record TTL (capped at 300s and then
// clamped by the cache's own bounds). It returns an error if no addresses are
// found.
func (r *Dialer) lookupIPs(ctx context.Context, host string) ([]net.IP, error) {
	if cached := r.cache.getIPs(host); cached != nil {
		logDebug(ctx, "IP cache hit",
			"host", host,
			"ips", len(cached))

		return cached, nil
	}

	logDebug(ctx, "IP cache miss",
		"host", host)

	records, err := r.lookup(ctx, host)
	if err != nil {
		return nil, err
	}

	ips := make([]net.IP, 0, len(records))
	minTTL := uint32(300)

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
		return nil, fmt.Errorf("no IP addresses found for %s", host)
	}

	r.cache.setIPs(host, ips, time.Duration(minTTL)*time.Second)

	return ips, nil
}
