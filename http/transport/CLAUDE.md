# Package: http/transport

HTTP transport configuration with DNS caching and connection pooling.

## Usage

```go
import "github.com/amp-labs/amp-common/http/transport"

// Create with defaults
t := transport.New(ctx)

// Get singleton with options
rt := transport.Get(transport.EnableDNSCache)

// Context override
ctx = transport.WithTransport(ctx, customTransport)
rt := transport.GetContext(ctx)
```

## Common Patterns

- `New()` - Create transport with defaults
- `Get()` - Singleton instances with options
- Options: DisableConnectionPooling, EnableDNSCache, InsecureTLS, WithProvider
- Environment variable configuration (HTTP_TRANSPORT_*)
- Per-provider settings: `HTTP_TRANSPORT_<SETTING>_<PROVIDER>` (e.g. `HTTP_TRANSPORT_MAX_IDLE_CONNS_PER_HOST_ACCULYNX`)
  wins over `HTTP_TRANSPORT_<SETTING>` for transports obtained with `WithProvider`
- Reuse transport instances for connection pooling

## Gotchas

- Singleton transports cached by configuration
- A provider with any of its own settings gets its own transport and connection pool (net/http applies
  one transport's settings to every host)
- Idle connections per host default to net/http's 2; raise HTTP_TRANSPORT_MAX_IDLE_CONNS_PER_HOST for
  hosts with many concurrent requests, or connections are closed and re-dialed after each burst
- DNS caching reduces DNS traffic
- HTTP/2 disabled by default (set FORCE_ATTEMPT_HTTP2)

## Related

- See godoc for detailed configuration options
