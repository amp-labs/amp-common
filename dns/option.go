package dns

import (
	"net"
	"time"
)

const (
	// defaultTimeout is the per-query timeout when none is configured.
	defaultTimeout = 5 * time.Second
	// defaultPoolSize is the per-resolver connection pool size when none is
	// configured.
	defaultPoolSize = 4
)

// options holds the accumulated configuration produced by the [Option]
// functions before a [Dialer] is built.
type options struct {
	resolvers []string
	filter    Filter
	strategy  Strategy
	dialer    *net.Dialer
	timeout   time.Duration
	poolSize  int
	cache     *dnsCache
}

// newOptions returns the default configuration: race strategy, a plain dialer,
// the default timeout and pool size, and caching disabled.
func newOptions() *options {
	return &options{
		strategy: Race{},
		dialer:   &net.Dialer{},
		timeout:  defaultTimeout,
		poolSize: defaultPoolSize,
		cache:    newDNSCache(0, 0, 0), // disabled by default
	}
}

// createDialer assembles the resolver stack and returns a ready [Dialer]. Each
// configured address is wrapped in a unifiedResolver (UDP with TCP fallback),
// then a cnameResolver, and finally a filterResolver when a filter is set. It
// returns [ErrNoResolvers] if no addresses were configured.
func (o *options) createDialer() (*Dialer, error) {
	if len(o.resolvers) == 0 {
		return nil, ErrNoResolvers
	}

	resolvers := make([]Resolver, 0, len(o.resolvers))

	for _, addr := range o.resolvers {
		var r Resolver = newUnifiedResolver(addr, o.timeout, o.poolSize)

		// Follow CNAME chains using this resolver before filtering, so the
		// filter sees the flattened result (including any terminal A/AAAA we had
		// to chase) rather than a bare CNAME.
		r = newCNameResolver(addr, r)

		if o.filter != nil {
			r = newFilterResolver(addr, r, o.filter)
		}

		resolvers = append(resolvers, r)
	}

	return &Dialer{
		resolvers: resolvers,
		strategy:  o.strategy,
		timeout:   o.timeout,
		poolSize:  o.poolSize,
		dialer:    o.dialer,
		cache:     o.cache,
	}, nil
}

// Option configures a [Dialer] built by [NewDialer]. Options are applied in
// order, so a later option of the same kind overrides an earlier one.
type Option func(*options)

// WithResolvers adds DNS server addresses to query. Each address may be a bare
// host (port 53 is assumed) or "host:port". At least one resolver is required;
// the option may be given more than once and the addresses accumulate.
func WithResolvers(addrs ...string) Option {
	return func(r *options) {
		r.resolvers = append(r.resolvers, addrs...)
	}
}

// WithFilter installs a predicate that decides which resolved records to keep.
// A nil predicate is ignored, leaving all records.
func WithFilter(f func(host string, record Record) bool) Option {
	return func(r *options) {
		if f != nil {
			r.filter = newFilter(f)
		}
	}
}

// WithDialer sets the [net.Dialer] used to open the final connection to a
// resolved IP. It does not affect how DNS queries themselves are dialed.
func WithDialer(dialer *net.Dialer) Option {
	return func(r *options) {
		r.dialer = dialer
	}
}

// WithStrategy selects how answers from multiple resolvers are combined. The
// default is [Race].
func WithStrategy(s Strategy) Option {
	return func(r *options) {
		r.strategy = s
	}
}

// WithTimeout sets the per-query timeout applied to each DNS query.
func WithTimeout(d time.Duration) Option {
	return func(r *options) {
		r.timeout = d
	}
}

// WithConnPoolSize sets the maximum number of connections pooled per resolver.
// Non-positive values are ignored, keeping the default.
func WithConnPoolSize(size int) Option {
	return func(r *options) {
		if size > 0 {
			r.poolSize = size
		}
	}
}

// WithCache enables IP caching for up to size hosts, clamping each entry's TTL
// to [minTTL, maxTTL]. A non-positive size disables caching (the default).
func WithCache(size int, minTTL, maxTTL time.Duration) Option {
	return func(r *options) {
		r.cache = newDNSCache(size, minTTL, maxTTL)
	}
}
