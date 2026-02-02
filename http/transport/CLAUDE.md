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
- Options: DisableConnectionPooling, EnableDNSCache, InsecureTLS
- Environment variable configuration (HTTP_TRANSPORT_*)
- Reuse transport instances for connection pooling

## Gotchas

- Singleton transports cached by configuration
- DNS caching reduces DNS traffic
- HTTP/2 disabled by default (set FORCE_ATTEMPT_HTTP2)

## Related

- See godoc for detailed configuration options
