package dns

import (
	"net"
	"time"

	"github.com/amp-labs/amp-common/retry"
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
	resolvers          []string
	filter             Filter
	strategy           Strategy
	dialer             *net.Dialer
	timeout            time.Duration
	poolSize           int
	cache              *dnsCache
	lookupRetryOptions []retry.Option
	dialerRetryOptions []retry.Option
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

// createLookupCoordinator assembles the resolver stack and returns a ready
// [LookupCoordinator]. Each configured address is wrapped in a unifiedResolver
// (UDP with TCP fallback), then a metricsResolver, then a cnameResolver, and
// finally a filterResolver when a filter is set. It returns [ErrNoResolvers]
// if no addresses were configured.
func (o *options) createLookupCoordinator() (*LookupCoordinator, error) {
	if len(o.resolvers) == 0 {
		return nil, ErrNoResolvers
	}

	resolvers := make([]Resolver, 0, len(o.resolvers))

	for _, addr := range o.resolvers {
		var resolver Resolver = newUnifiedResolver(addr, o.timeout, o.poolSize)

		// Follow CNAME chains using this resolver before filtering, so the
		// filter sees the flattened result (including any terminal A/AAAA we had
		// to chase) rather than a bare CNAME.
		resolver = newCNameResolver(addr, resolver)

		if o.filter != nil {
			resolver = newFilterResolver(addr, resolver, o.filter)
		}

		resolvers = append(resolvers, resolver)
	}

	return &LookupCoordinator{
		resolvers:    resolvers,
		filter:       o.filter,
		strategy:     o.strategy,
		cache:        o.cache,
		retryOptions: o.lookupRetryOptions,
	}, nil
}

// createDialer builds the [LookupCoordinator] (see createLookupCoordinator)
// and pairs it with the configured net.Dialer to produce a ready [Dialer]. It
// returns [ErrNoResolvers] if no addresses were configured.
func (o *options) createDialer() (*Dialer, error) {
	lookup, err := o.createLookupCoordinator()
	if err != nil {
		return nil, err
	}

	return &Dialer{
		lookup:       lookup,
		dialer:       o.dialer,
		retryOptions: o.dialerRetryOptions,
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

// WithLookupRetryOptions sets the [retry.Option] set applied to DNS lookups
// (the whole resolution attempt, not individual resolver queries). By default
// lookups are not retried. Later calls replace earlier ones.
func WithLookupRetryOptions(opts ...retry.Option) Option {
	return func(r *options) {
		r.lookupRetryOptions = opts
	}
}

// WithDialerRetryOptions sets the [retry.Option] set applied when dialing each
// resolved IP. By default each IP is dialed once before moving to the next.
// Later calls replace earlier ones.
func WithDialerRetryOptions(opts ...retry.Option) Option {
	return func(r *options) {
		r.dialerRetryOptions = opts
	}
}
