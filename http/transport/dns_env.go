package transport

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/amp-labs/amp-common/dns"
	"github.com/amp-labs/amp-common/envtypes"
	"github.com/amp-labs/amp-common/envutil"
	"github.com/amp-labs/amp-common/lazy"
	"github.com/amp-labs/amp-common/xform"
)

// Defaults applied when the corresponding AMP_PUBLIC_DNS_* env vars are unset or fail to parse.
const (
	defaultDNSPort         = 53
	defaultDNSConnPoolSize = 4
	defaultDNSCacheSize    = 1000

	defaultDNSMinCacheTTL = 10 * time.Second
	defaultDNSMaxCacheTTL = 24 * time.Hour
)

// defaultPublicDNSResolvers is the "host:port" fallback resolver list (Google and Cloudflare public DNS).
var defaultPublicDNSResolvers = []string{
	"8.8.8.8:53",
	"1.1.1.1:53",
}

// getDefaultPublicDNSResolvers returns the fallback resolvers as parsed HostPort values, used
// whenever AMP_PUBLIC_DNS_RESOLVERS is unset or any configured entry fails to parse.
func getDefaultPublicDNSResolvers() []envtypes.HostPort {
	return []envtypes.HostPort{
		{
			Host: "8.8.8.8",
			Port: defaultDNSPort,
		},
		{
			Host: "1.1.1.1",
			Port: defaultDNSPort,
		},
	}
}

// dnsPublicResolvers lazily parses AMP_PUBLIC_DNS_RESOLVERS, a comma-separated list of "host:port"
// entries, into HostPort values. It falls back to the public defaults if the variable is unset or
// any entry fails to parse, so a misconfiguration never leaves the dialer without resolvers.
var dnsPublicResolvers = lazy.NewCtx[[]envtypes.HostPort](func(ctx context.Context) []envtypes.HostPort {
	// Read the env var
	r := envutil.String(ctx, "AMP_PUBLIC_DNS_RESOLVERS")

	// Split it by comma in to a list
	s := envutil.Map(r, xform.SplitString(","))

	// Fall back to a default list if not set
	s = s.WithDefault(defaultPublicDNSResolvers)

	// Convert "host:port" format to proper envtypes.HostPort types.
	hps := envutil.Map[[]string, []envtypes.HostPort](
		s, func(values []string) ([]envtypes.HostPort, error) {
			var out []envtypes.HostPort

			for _, value := range values {
				value = strings.TrimSpace(value)

				// Skip empty values
				if len(value) == 0 {
					continue
				}

				// Parse "host:port" to a struct
				hp, err := xform.HostAndPort(value)
				if err != nil {
					return nil, fmt.Errorf("error parsing host and port %q: %w", value, err)
				}

				out = append(out, hp)
			}

			// Corner case: we were given an empty list
			if len(out) == 0 {
				return nil, envutil.ErrUnsetValue
			}

			return out, nil
		},
	)

	// Do all the parsing, and if anything fails fall back to a safe list.
	return hps.ValueOrElseFunc(getDefaultPublicDNSResolvers)
})

// getDNSPublicResolvers returns the configured public resolvers as "host:port" strings ready to
// hand to the dnsdialer, substituting the defaults if the resolved list is somehow empty.
func getDNSPublicResolvers(ctx context.Context) []string {
	resolvers := dnsPublicResolvers.Get(ctx)

	if len(resolvers) == 0 {
		resolvers = getDefaultPublicDNSResolvers()
	}

	out := make([]string, 0, len(resolvers))

	for _, resolver := range resolvers {
		out = append(out, resolver.String())
	}

	return out
}

var errUnknownLogLevel = errors.New("unknown log level")

// dnsLogging lazily reads AMP_PUBLIC_DNS_LOGGING ("none", "errors", or "verbose") and resolves it
// to a dns.LogLevel, defaulting to LogLevelNone for unset or unrecognized values.
var dnsLogging = lazy.NewCtx[dns.LogLevel](func(ctx context.Context) dns.LogLevel {
	s := envutil.String(ctx, "AMP_PUBLIC_DNS_LOGGING", envutil.Default("none"))

	lvl := envutil.Map[string, dns.LogLevel](s, func(s string) (dns.LogLevel, error) {
		s = strings.TrimSpace(s)
		s = strings.ToLower(s)

		switch s {
		case "none":
			return dns.LogLevelNone, nil
		case "errors":
			return dns.LogLevelErrorOnly, nil
		case "verbose":
			return dns.LogLevelVerbose, nil
		default:
			return dns.LogLevelNone, fmt.Errorf("%w %q", errUnknownLogLevel, s)
		}
	})

	return lvl.ValueOrElse(dns.LogLevelNone)
})

// dnsConnPoolSize is the per-resolver connection pool size, from AMP_PUBLIC_DNS_CONNECTION_POOL_SIZE.
var dnsConnPoolSize = lazy.NewCtx[int](func(ctx context.Context) int {
	return envutil.Int[int](ctx, "AMP_PUBLIC_DNS_CONNECTION_POOL_SIZE",
		envutil.Default(defaultDNSConnPoolSize)).
		ValueOrElse(defaultDNSConnPoolSize)
})

// dnsCacheSize is the maximum number of cached DNS entries, from AMP_PUBLIC_DNS_CACHE_SIZE.
var dnsCacheSize = lazy.NewCtx[int](func(ctx context.Context) int {
	return envutil.Int[int](ctx, "AMP_PUBLIC_DNS_CACHE_SIZE",
		envutil.Default(defaultDNSCacheSize)).
		ValueOrElse(defaultDNSCacheSize)
})

// dnsMinCacheTTL is the floor applied to cached entry TTLs, from AMP_PUBLIC_DNS_MIN_CACHE_TTL.
var dnsMinCacheTTL = lazy.NewCtx[time.Duration](func(ctx context.Context) time.Duration {
	return envutil.Duration(ctx, "AMP_PUBLIC_DNS_MIN_CACHE_TTL",
		envutil.Default(defaultDNSMinCacheTTL)).
		ValueOrElse(defaultDNSMinCacheTTL)
})

// dnsMaxCacheTTL is the ceiling applied to cached entry TTLs, from AMP_PUBLIC_DNS_MAX_CACHE_TTL.
var dnsMaxCacheTTL = lazy.NewCtx[time.Duration](func(ctx context.Context) time.Duration {
	return envutil.Duration(ctx, "AMP_PUBLIC_DNS_MAX_CACHE_TTL",
		envutil.Default(defaultDNSMaxCacheTTL)).
		ValueOrElse(defaultDNSMaxCacheTTL)
})
