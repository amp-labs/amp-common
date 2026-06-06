package transport

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/amp-labs/amp-common/dns"
	"github.com/amp-labs/amp-common/utils"
	"github.com/rs/dnscache"
)

// dnsResolver is a package-level DNS cache resolver used to reduce DNS lookup overhead.
// It is initialized once and shared across all transports that enable DNS caching.
var dnsResolver *dnscache.Resolver

func init() {
	dnsResolver = &dnscache.Resolver{}
}

// useAmpersandDNSDialer modifies the given http.Transport to dial through the Ampersand DNS
// dialer, which resolves names using its own configured resolvers (public ones by default,
// e.g. 8.8.8.8, 1.1.1.1) and refuses to connect to private/RFC 1918 addresses. This stops
// callers from reaching internal services via private DNS names or private IPs. When cache
// is true, resolved results are cached to reduce DNS traffic.
func useAmpersandDNSDialer(
	ctx context.Context,
	trans *http.Transport,
	cache, publicOnly bool,
	timeout time.Duration,
) error {
	opts := []dns.Option{
		dns.WithConnPoolSize(dnsConnPoolSize.Get(ctx)),
		dns.WithStrategy(dns.Fallback{}),
		dns.WithResolvers(getAmpersandDNSResolvers(ctx)...),
	}

	if publicOnly {
		opts = append(
			opts, dns.WithFilter(func(host string, record dns.Record) bool {
				// Leave non-IP records alone
				if record.Type != dns.TypeA && record.Type != dns.TypeAAAA {
					return true
				}

				public, valid := utils.IsPublicIPString(record.Value)

				// Has to be both public and a valid IP string to be allowed
				return public && valid
			}),
		)
	}

	lookupOpts := dnsLookupRetryOpts.Get(ctx)

	if len(lookupOpts) > 0 {
		opts = append(opts, dns.WithLookupRetryOptions(lookupOpts...))
	}

	dialOpts := dnsDialRetryOpts.Get(ctx)

	if len(dialOpts) > 0 {
		opts = append(opts, dns.WithDialerRetryOptions(dialOpts...))
	}

	if timeout > 0 {
		opts = append(opts, dns.WithTimeout(timeout))
	}

	if cache {
		opts = append(opts, dns.WithCache(
			dnsCacheSize.Get(ctx),
			dnsMinCacheTTL.Get(ctx),
			dnsMaxCacheTTL.Get(ctx),
		))
	}

	dialer, err := dns.NewDialer(opts...)
	if err != nil {
		return fmt.Errorf("could not create a new dialer: %w", err)
	}

	// Resolve the configured log level once; the dialer reads it from the
	// per-request context via dns.WithLogLevel.
	logLevel := dnsLogging.Get(ctx)

	trans.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return dialer.DialContext(dns.WithLogLevel(ctx, logLevel), network, addr)
	}

	return nil
}

// useDNSCacheDialer modifies the given http.Transport to use a DNS caching dialer.
// This helps reduce DNS lookups and improve performance, especially under load.
func useDNSCacheDialer(trans *http.Transport, timeout, keepAlive time.Duration) {
	dialer := &net.Dialer{
		Timeout:   timeout,
		KeepAlive: keepAlive,
	}

	// Under load the DNS resolver can sometimes time out (resulting in "DNS timeout errors" when sending webhooks),
	// so we'll use a caching resolver to reduce the amount of DNS traffic.
	trans.DialContext = func(ctx context.Context, network string, addr string) (conn net.Conn, err error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}

		ips, err := dnsResolver.LookupHost(ctx, host)
		if err != nil {
			return nil, err
		}

		for _, ip := range ips {
			conn, err = dialer.DialContext(ctx, network, net.JoinHostPort(ip, port))
			if err == nil {
				break
			}
		}

		return
	}
}
