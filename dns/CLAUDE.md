# dns package

Configurable, caching DNS client with a drop-in `net.Dialer`-compatible
`DialContext`. Resolves hostnames against explicitly configured DNS servers
(e.g. 8.8.8.8, 1.1.1.1) instead of the host's default resolver, so outbound
traffic gets predictable, resolver-independent name resolution. See `doc.go`
for the full package overview and usage example.

## Commands

```bash
go test -v ./dns/                  # Run package tests
go test -v -run TestName ./dns/    # Run a single test
go test -race ./dns/               # Race detection (resolvers/strategies are heavily concurrent)
```

## Architecture

### Two entry points

- **`Dialer`** (`dialer.go`) — `NewDialer(opts...)`. `DialContext` resolves the
  host then dials each resulting IP until one succeeds. Assign directly to
  `http.Transport.DialContext`.
- **`LookupCoordinator`** (`lookup.go`) — `NewLookupCoordinator(opts...)`.
  The resolution pipeline alone (no dialing), usable standalone. `Dialer` is a
  thin wrapper around it. Accepts the same `Option` set; dialer-only options
  are ignored.

### Resolver stack (assembled in `option.go:createLookupCoordinator`)

Each configured address gets this onion, innermost first:

```
udpResolver / tcpResolver     wire-level query over a pooled connection
  └ metricsResolver           Prometheus metrics, labeled by server + protocol
    └ unifiedResolver         UDP first, transparent TCP retry on truncation (TC bit)
      └ cnameResolver         follows CNAME chains (max 16 hops, loop detection)
        └ filterResolver      drops records the caller's Filter rejects (only if WithFilter set)
```

Note: `metricsResolver` wraps the UDP/TCP transports *inside*
`unifiedResolver` (see `unified_resolver.go`), so a truncated-UDP→TCP retry
counts once per protocol. CNAME chasing happens *before* filtering so the
filter sees flattened terminal A/AAAA records, not bare CNAMEs.

All `Resolver` implementations must be safe for concurrent use — strategies
query several resolvers in parallel.

### Strategies (`strategy.go`, one file per implementation)

`Strategy.ResolveType` receives the whole resolver set and decides how to
query/reconcile:

- `Race` (default) — first successful answer wins, rest canceled (`race.go`)
- `Fallback` — in-order, first success (`fallback.go`)
- `Consensus` — N resolvers must agree; guards against a poisoned resolver (`consensus.go`)
- `Compare` — queries all, returns first, reports discrepancies via callback (`compare.go`)

### Lookup flow (`lookup.go`)

1. IP literals skip DNS entirely but still pass the configured `Filter`
   (synthesized into a fake A/AAAA record — the resolver-stack filter never
   sees literals, so `LookupCoordinator.filter` is the only place to vet them).
2. A, AAAA, and CNAME queries run concurrently; per-type failures are logged
   and skipped (an IPv4-only host still resolves), not fatal.
3. Results cached by host using the smallest record TTL, capped at 300s
   (`maxCachedTTLSeconds`), then clamped to the cache's `[minTTL, maxTTL]`.
4. Network selects address family: `tcp4`/`udp4` → IPv4 only, `tcp6`/`udp6` →
   IPv6 only, generic `tcp`/`udp` → IPv4 first then IPv6.

### Caching (`cache.go`)

`dnsCache` is an expirable LRU keyed by host. **Disabled by default** — a
zero/negative size produces a no-op cache so call sites never nil-check.
Enable with `WithCache(size, minTTL, maxTTL)`.

## Conventions and gotchas

- **Sentinel errors** (`errors.go`): exported ones (`ErrNoRecords`,
  `ErrNoConsensus`, `ErrNoResolvers`, `ErrCNAMELoop`, `ErrCNAMEChainTooLong`)
  are part of the API — match with `errors.Is`. Internal ones stay lowercase.
- **Logging is opt-in per request** (`context.go`, `logging.go`): resolution
  is a hot path, silent by default. Callers enable it with
  `dns.WithLogLevel(ctx, dns.LogLevelVerbose)`. Use `logDebug`/`logError`
  helpers inside the package, never `slog` directly.
- **The underlying wire library is `codeberg.org/miekg/dns`** and must not
  leak into the public API — `RecordType` and `Record` (`record_type.go`)
  exist precisely to wrap it. Keep new APIs free of that import.
- **Tracing**: `Dialer.DialContext` and `Lookup` emit OTel spans via the
  `spans` package (`dialAddress`, `dialIP`, `dnsLookup`); follow that pattern
  for new externally-visible operations.
- **Metrics** (`metrics.go`): `dns_lookups_total`, `dns_lookup_errors_total`,
  `dns_lookup_duration_millis`, labeled by `server` ("host:port") and
  `protocol` ("udp"/"tcp"). Cancellation (losing a Race) is not counted as an
  error; timeouts are.
- **Retry layers are distinct**: `WithLookupRetryOptions` retries the whole
  resolution attempt; `WithDialerRetryOptions` retries dialing each individual
  IP. Both default to no retries.
- Bare resolver addresses default to port 53 (`util.go:parseHostAndPort`).

## Testing

- Tests are white-box (same package). `stub_test.go` provides the shared
  fakes: `stubResolver` (fixed answer + optional delay + call counter, named
  so Consensus/Compare can group by resolver) and `fakeStrategy` (canned
  records per query type, call counter for cache assertions). Reuse these
  instead of inventing new stubs.
- Strategy and timing-sensitive tests rely on `stubResolver.delay` plus
  context cancellation — keep delays small and assert via call counters, not
  wall-clock sleeps.
- No test touches the real network; anything wire-level is exercised through
  the stub layer.
