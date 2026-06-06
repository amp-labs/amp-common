// Package dns provides a configurable, caching DNS client and a drop-in
// [net.Dialer]-compatible DialContext that resolves hostnames using public
// resolvers rather than the host's default resolver.
//
// # Overview
//
// The package was built to give outbound HTTP traffic predictable,
// resolver-independent name resolution: it queries one or more explicitly
// configured DNS servers (for example Google's 8.8.8.8 and Cloudflare's
// 1.1.1.1) and combines their answers according to a pluggable [Strategy].
// This sidesteps a misconfigured or hijacked local resolver and makes name
// resolution behave the same across every environment the binary runs in.
//
// The central type is [Dialer]. Its [Dialer.DialContext] method has the same
// signature as [net.Dialer.DialContext], so it can be plugged straight into an
// [net/http.Transport]:
//
//	d, err := dns.NewDialer(
//	    dns.WithResolvers("8.8.8.8:53", "1.1.1.1:53"),
//	    dns.WithStrategy(dns.Race{}),
//	    dns.WithCache(1000, 10*time.Second, time.Hour),
//	)
//	if err != nil {
//	    return err
//	}
//
//	transport := &http.Transport{DialContext: d.DialContext}
//
// # Resolvers
//
// A [Resolver] performs a single typed query (A, AAAA, CNAME, ...) against one
// DNS server. The package layers several resolver implementations:
//
//   - udpResolver / tcpResolver issue the wire-level query over a pooled
//     connection.
//   - unifiedResolver tries UDP first and transparently retries over TCP when
//     the response is truncated.
//   - metricsResolver records Prometheus metrics (lookup count, error count,
//     and latency) labeled by the server's "host:port" address.
//   - cnameResolver follows CNAME chains so callers always see the terminal
//     address records, even from a non-recursive server.
//   - filterResolver drops records the caller's [Filter] rejects.
//
// [NewDialer] assembles this stack for each configured address.
//
// # Strategies
//
// A [Strategy] decides how the answers from multiple resolvers are combined:
//
//   - [Race] returns the first successful answer (lowest latency wins).
//   - [Fallback] tries resolvers in order until one succeeds.
//   - [Consensus] requires a minimum number of resolvers to agree.
//   - [Compare] queries every resolver and reports discrepancies.
//
// # Caching
//
// Resolved IP addresses are cached with TTL-based expiration (see [WithCache]),
// honoring the record TTL clamped to a configurable minimum and maximum.
// Caching is disabled by default.
package dns
